package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type question struct {
	Type              string `json:"type"`
	ID                string `json:"id,omitempty"`
	QuestionID        string `json:"question_id,omitempty"`
	SessionID         string `json:"session_id,omitempty"`
	Text              string `json:"text"`
	RecommendedAnswer string `json:"recommended_answer,omitempty"`
	Timestamp         string `json:"timestamp,omitempty"`
}

func addQuestion(path string, question question) error {
	if strings.TrimSpace(question.Text) == "" {
		return errors.New("text cannot be empty")
	}
	data, err := json.Marshal(question)
	if err != nil {
		return err
	}
	return withSessionLock(path, func() error {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("open session file: %w", err)
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			_ = f.Close()
			return fmt.Errorf("append question: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close session file: %w", err)
		}
		return nil
	})
}

func findAnswer(path, id, sessionID string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var value question
		if json.Unmarshal(scanner.Bytes(), &value) == nil && value.Type == "answer" && value.QuestionID == id && value.SessionID == sessionID {
			return value.Text, nil
		}
	}
	return "", scanner.Err()
}
