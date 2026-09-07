package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestParseQuestionsWithRecommendations(t *testing.T) {
	questions, err := parseQuestions([]string{
		"First?", "--recommended", "Use the default.",
		"Second?", "--recommended", "No.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(questions) != 2 || questions[0].RecommendedAnswer != "Use the default." || questions[1].Text != "Second?" {
		t.Fatalf("got %#v", questions)
	}
}

func TestParseQuestionsRejectsOrphanRecommendation(t *testing.T) {
	if _, err := parseQuestions([]string{"--recommended", "No."}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestParseRetentionDurationInDays(t *testing.T) {
	duration, err := parseRetentionDuration("7d")
	if err != nil || duration != 7*24*time.Hour {
		t.Fatalf("got %v, %v", duration, err)
	}
}

func TestHelpListsCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code %d, stderr %q", code, stderr.String())
	}
	for _, command := range []string{"ask", "clean", "version"} {
		if !bytes.Contains(stdout.Bytes(), []byte(command)) {
			t.Fatalf("help does not list %q:\n%s", command, stdout.String())
		}
	}
}

func TestAskHelpDocumentsQuestionsAndRecommendations(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"ask", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code %d, stderr %q", code, stderr.String())
	}
	for _, text := range []string{"question", "--recommended", "Herdr", "Examples:"} {
		if !bytes.Contains(stdout.Bytes(), []byte(text)) {
			t.Fatalf("ask help does not contain %q:\n%s", text, stdout.String())
		}
	}
}

func TestVersionDoesNotRequireHerdr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code %d, stderr %q", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("grillme ")) {
		t.Fatalf("unexpected version output %q", stdout.String())
	}
}

func TestUnknownCommandAndMalformedAskAreUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"unknown"}, {"ask"}, {"ask", "Question?", "--recommended"}} {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 2 {
			t.Errorf("run(%q) exit code %d, want 2", args, code)
		}
		if !bytes.Contains(stderr.Bytes(), []byte("--help")) {
			t.Errorf("run(%q) lacks usage guidance: %q", args, stderr.String())
		}
	}
}

func TestVersionStringIncludesConfiguredMetadata(t *testing.T) {
	oldVersion, oldCommit := version, commit
	version, commit = "v1.2.3", "abc123"
	t.Cleanup(func() { version, commit = oldVersion, oldCommit })
	if got := versionString(); got != "grillme v1.2.3 (commit abc123)" {
		t.Fatalf("got %q", got)
	}
}

func TestSplitPaneID(t *testing.T) {
	id, err := splitPaneID([]byte(`{"result":{"pane":{"pane_id":"w1:p2"}}}`))
	if err != nil || id != "w1:p2" {
		t.Fatalf("got %q, %v", id, err)
	}
}
