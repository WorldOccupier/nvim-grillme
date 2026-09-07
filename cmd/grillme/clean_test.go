package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func writeSession(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCleanSessionRemovesCompletedPairsAndPreservesOtherEvents(t *testing.T) {
	path := writeSession(t, strings.Join([]string{
		`{"type":"question","id":"done","text":"Done?"}`,
		`{"type":"answer","question_id":"done","text":"Yes."}`,
		`{"type":"question","id":"pending","text":"Pending?"}`,
		`{"type":"metadata","value":42}`,
		`{"type":"answer","question_id":"missing","text":"Orphan."}`,
	}, "\n")+"\n")

	removed, err := cleanSession(path, nil, time.Now())
	if err != nil || removed != 2 {
		t.Fatalf("removed %d, err %v", removed, err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\"type\":\"question\",\"id\":\"pending\",\"text\":\"Pending?\"}\n" +
		"{\"type\":\"metadata\",\"value\":42}\n" +
		"{\"type\":\"answer\",\"question_id\":\"missing\",\"text\":\"Orphan.\"}\n"
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v, err %v", info.Mode().Perm(), err)
	}

	removed, err = cleanSession(path, nil, time.Now())
	if err != nil || removed != 0 {
		t.Fatalf("second cleanup removed %d, err %v", removed, err)
	}
}

func TestCleanSessionOlderThan(t *testing.T) {
	now := time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	path := writeSession(t, strings.Join([]string{
		`{"type":"question","id":"old","text":"Old?","timestamp":"2026-03-01T12:00:00Z"}`,
		`{"type":"answer","question_id":"old","text":"Yes.","timestamp":"2026-03-02T12:00:00Z"}`,
		`{"type":"question","id":"recent","text":"Recent?","timestamp":"2026-03-19T12:00:00Z"}`,
		`{"type":"answer","question_id":"recent","text":"Yes.","timestamp":"2026-03-19T12:00:00Z"}`,
		`{"type":"question","id":"legacy","text":"Legacy?"}`,
		`{"type":"answer","question_id":"legacy","text":"Yes."}`,
	}, "\n")+"\n")
	duration := 7 * 24 * time.Hour

	removed, err := cleanSession(path, &duration, now)
	if err != nil || removed != 2 {
		t.Fatalf("removed %d, err %v", removed, err)
	}
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), `"id":"old"`) || !strings.Contains(string(got), `"id":"recent"`) || !strings.Contains(string(got), `"id":"legacy"`) {
		t.Fatalf("unexpected cleaned data:\n%s", got)
	}
}

func TestCleanSessionRejectsMalformedDataWithoutChangingFile(t *testing.T) {
	content := "{\"type\":\"question\",\"id\":\"q1\",\"text\":\"Question?\"}\nnot json\n"
	path := writeSession(t, content)

	if _, err := cleanSession(path, nil, time.Now()); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("got error %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != content {
		t.Fatalf("file changed to %q", got)
	}
}

func TestCleanSessionCanProduceEmptyFile(t *testing.T) {
	path := writeSession(t, "{\"type\":\"question\",\"id\":\"q1\",\"text\":\"Question?\"}\n{\"type\":\"answer\",\"question_id\":\"q1\",\"text\":\"Answer.\"}\n")
	if _, err := cleanSession(path, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if len(got) != 0 {
		t.Fatalf("got %q", got)
	}
}

func TestConcurrentAppendAndCleanupKeepCompleteEvents(t *testing.T) {
	path := writeSession(t, "{\"type\":\"question\",\"id\":\"done\",\"text\":\"Done?\"}\n{\"type\":\"answer\",\"question_id\":\"done\",\"text\":\"Yes\"}\n")
	const count = 40
	var group sync.WaitGroup
	for index := 0; index < count; index++ {
		index := index
		group.Add(1)
		go func() {
			defer group.Done()
			if err := addQuestion(path, question{Type: "question", ID: fmt.Sprintf("q%d", index), Text: "Pending?"}); err != nil {
				t.Errorf("append question: %v", err)
			}
		}()
	}
	group.Add(1)
	go func() {
		defer group.Done()
		if _, err := cleanSession(path, nil, time.Now()); err != nil {
			t.Errorf("clean session: %v", err)
		}
	}()
	group.Wait()

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event sessionEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("partial or malformed event %q: %v", scanner.Text(), err)
		}
		seen[event.ID] = true
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < count; index++ {
		if !seen[fmt.Sprintf("q%d", index)] {
			t.Errorf("missing concurrently appended question q%d", index)
		}
	}
}

func TestSessionLockIsReleasedAfterMutationError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	want := fmt.Errorf("write failed")
	if err := withSessionLock(path, func() error { return want }); err != want {
		t.Fatalf("got %v, want %v", err, want)
	}
	lock, err := acquireSessionLock(path)
	if err != nil {
		t.Fatalf("lock was not released: %v", err)
	}
	if err := lock.release(); err != nil {
		t.Fatal(err)
	}
}

func TestSessionLockRecoversIncompleteLockFromInterruptedProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	lockPath := path + ".lock"
	if err := os.Mkdir(lockPath, 0o700); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-incompleteLockAge - time.Second)
	if err := os.Chtimes(lockPath, old, old); err != nil {
		t.Fatal(err)
	}
	lock, err := acquireSessionLock(path)
	if err != nil {
		t.Fatalf("recover interrupted lock: %v", err)
	}
	if err := lock.release(); err != nil {
		t.Fatal(err)
	}
}
