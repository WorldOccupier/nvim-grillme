package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnswerRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	if err := addQuestion(path, question{Type: "question", ID: "q1", Text: "Question?"}); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{\"type\":\"answer\",\"question_id\":\"q1\",\"text\":\"Answer.\"}\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	answer, err := findAnswer(path, "q1")
	if err != nil || answer != "Answer." {
		t.Fatalf("got %q, %v", answer, err)
	}
}

func TestSplitPaneID(t *testing.T) {
	id, err := splitPaneID([]byte(`{"result":{"pane":{"pane_id":"w1:p2"}}}`))
	if err != nil || id != "w1:p2" {
		t.Fatalf("got %q, %v", id, err)
	}
}
