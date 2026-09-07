package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type sessionEvent struct {
	Type       string `json:"type"`
	ID         string `json:"id,omitempty"`
	QuestionID string `json:"question_id,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
}

type sessionLine struct {
	// Keep the original bytes so cleanup does not alter unknown event fields.
	raw   []byte
	event sessionEvent
}

func cleanSession(path string, olderThan *time.Duration, now time.Time) (removed int, err error) {
	err = withSessionLock(path, func() error {
		removed, err = cleanSessionLocked(path, olderThan, now)
		return err
	})
	return removed, err
}

func cleanSessionLocked(path string, olderThan *time.Duration, now time.Time) (int, error) {
	lines, info, err := readSession(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	removeIDs := completedQuestionIDs(lines, olderThan, now)
	kept, removed := removeCompletedEvents(lines, removeIDs)
	if removed == 0 {
		return 0, nil
	}
	if err := rewriteSession(path, kept, info); err != nil {
		return 0, err
	}
	return removed, nil
}
func readSession(path string) ([]sessionLine, os.FileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	var lines []sessionLine
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		raw := append([]byte(nil), scanner.Bytes()...)
		var event sessionEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			_ = file.Close()
			return nil, nil, fmt.Errorf("malformed JSON on line %d: %w", lineNumber, err)
		}
		lines = append(lines, sessionLine{raw: raw, event: event})
	}
	if err := scanner.Err(); err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if err := file.Close(); err != nil {
		return nil, nil, err
	}
	return lines, info, nil
}

func completedQuestionIDs(lines []sessionLine, olderThan *time.Duration, now time.Time) map[string]bool {
	questions := make(map[string]bool)
	answers := make(map[string][]int)
	for index, line := range lines {
		switch line.event.Type {
		case "question":
			if line.event.ID != "" {
				questions[line.event.ID] = true
			}
		case "answer":
			if line.event.QuestionID != "" {
				answers[line.event.QuestionID] = append(answers[line.event.QuestionID], index)
			}
		}
	}

	removeIDs := make(map[string]bool)
	for id := range questions {
		indices := answers[id]
		if len(indices) == 0 {
			continue
		}
		eligible := olderThan == nil
		if olderThan != nil {
			eligible = true
			cutoff := now.Add(-*olderThan)
			for _, index := range indices {
				timestamp, err := time.Parse(time.RFC3339Nano, lines[index].event.Timestamp)
				if err != nil || timestamp.After(cutoff) {
					eligible = false
					break
				}
			}
		}
		if eligible {
			removeIDs[id] = true
		}
	}
	return removeIDs
}

func removeCompletedEvents(lines []sessionLine, removeIDs map[string]bool) ([][]byte, int) {
	removed := 0
	kept := make([][]byte, 0, len(lines))
	for _, line := range lines {
		remove := line.event.Type == "question" && removeIDs[line.event.ID] ||
			line.event.Type == "answer" && removeIDs[line.event.QuestionID]
		if remove {
			removed++
			continue
		}
		kept = append(kept, line.raw)
	}
	return kept, removed
}

func rewriteSession(path string, lines [][]byte, original os.FileInfo) error {
	directory := filepath.Dir(path)
	// A temporary file in the same directory makes the final rename atomic.
	temporary, err := os.CreateTemp(directory, ".session-*.jsonl")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	for _, line := range lines {
		if _, err := temporary.Write(append(line, '\n')); err != nil {
			_ = temporary.Close()
			return err
		}
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	current, err := os.Stat(path)
	if err != nil {
		return err
	}
	if current.Size() != original.Size() || !current.ModTime().Equal(original.ModTime()) {
		return errors.New("session changed during cleanup; no changes were made")
	}
	return os.Rename(temporaryPath, path)
}
