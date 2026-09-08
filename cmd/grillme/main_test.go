package main

import (
	"bytes"
	"context"
	"errors"
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
	answer, err := findAnswer(path, "q1", "")
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
	for _, args := range [][]string{{"unknown"}, {"ask"}, {"ask", "Question?", "--recommended"}, {"ask", "--timeout", "invalid", "Question?"}} {
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

func TestParseAskTimeout(t *testing.T) {
	questions, timeout, err := parseAskArgs([]string{"--timeout", "10m", "Question?"})
	if err != nil || timeout != 10*time.Minute || len(questions) != 1 {
		t.Fatalf("got %#v, %v, %v", questions, timeout, err)
	}
	for _, value := range []string{"nope", "0s", "-1s"} {
		if _, _, err := parseAskArgs([]string{"--timeout", value, "Question?"}); err == nil {
			t.Errorf("expected %q to fail", value)
		}
	}
}

func TestAskTimeoutAndCancellationCloseCreatedPane(t *testing.T) {
	for _, test := range []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
		want string
	}{
		{"timeout", func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), time.Millisecond)
		}, "timed out"},
		{"cancellation", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, func() {}
		}, "cancelled"},
	} {
		t.Run(test.name, func(t *testing.T) {
			withTempWorkingDirectory(t)
			ctx, cancel := test.ctx()
			defer cancel()
			closed := ""
			deps := askDependencies{
				launcher: &fakeLauncher{
					open:  func(string, string) (string, error) { return "created-pane", nil },
					close: func(id string) error { closed = id; return errors.New("cleanup failed") },
				},
				find:       func(string, string, string) (string, error) { return "", nil },
				newSession: func() (string, error) { return "session-1", nil },
				poll:       time.Millisecond,
			}
			err := askWithDependencies(ctx, []question{{Text: "Question?"}}, &bytes.Buffer{}, deps)
			if err == nil || !bytes.Contains([]byte(err.Error()), []byte(test.want)) {
				t.Fatalf("got error %v", err)
			}
			if closed != "created-pane" {
				t.Fatalf("closed pane %q", closed)
			}
		})
	}
}

func TestAskPrintsAnswersInOrderWithoutCleanup(t *testing.T) {
	withTempWorkingDirectory(t)
	answers := []string{"First answer.", "Second answer."}
	closed := false
	var stdout bytes.Buffer
	deps := askDependencies{
		launcher: &fakeLauncher{
			open:  func(string, string) (string, error) { return "created-pane", nil },
			close: func(string) error { closed = true; return nil },
		},
		newSession: func() (string, error) { return "session-1", nil },
		find: func(string, string, string) (string, error) {
			answer := answers[0]
			answers = answers[1:]
			return answer, nil
		},
		poll: time.Millisecond,
	}
	if err := askWithDependencies(context.Background(), []question{{Text: "First?"}, {Text: "Second?"}}, &stdout, deps); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "First answer.\nSecond answer.\n" {
		t.Fatalf("got %q", got)
	}
	if closed {
		t.Fatal("successful ask closed the pane")
	}
}

func TestAskPassesSessionToPaneAndEvents(t *testing.T) {
	withTempWorkingDirectory(t)
	var openedSession string
	deps := askDependencies{
		launcher: &fakeLauncher{
			open: func(_ string, sessionID string) (string, error) {
				openedSession = sessionID
				return "created-pane", nil
			},
			close: func(string) error { return nil },
		},
		newSession: func() (string, error) { return "session-1", nil },
		find: func(path, id, sessionID string) (string, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			if !bytes.Contains(data, []byte(`"session_id":"session-1"`)) || sessionID != "session-1" || id == "" {
				return "", errors.New("question did not carry the generated session")
			}
			return "Answer.", nil
		},
		poll: time.Millisecond,
	}
	if err := askWithDependencies(context.Background(), []question{{Text: "Question?"}}, &bytes.Buffer{}, deps); err != nil {
		t.Fatal(err)
	}
	if openedSession != "session-1" {
		t.Fatalf("pane received session %q", openedSession)
	}
}

func TestFindAnswerRequiresMatchingSession(t *testing.T) {
	path := writeSession(t, "{\"type\":\"answer\",\"question_id\":\"q1\",\"session_id\":\"other\",\"text\":\"Wrong.\"}\n"+
		"{\"type\":\"answer\",\"question_id\":\"q1\",\"session_id\":\"wanted\",\"text\":\"Right.\"}\n")
	answer, err := findAnswer(path, "q1", "wanted")
	if err != nil || answer != "Right." {
		t.Fatalf("got %q, %v", answer, err)
	}
}

type fakeLauncher struct {
	open  func(string, string) (string, error)
	close func(string) error
}

func (l *fakeLauncher) Open(cwd, sessionID string) (string, error) {
	return l.open(cwd, sessionID)
}

func (l *fakeLauncher) Close(id string) error {
	return l.close(id)
}

func withTempWorkingDirectory(t *testing.T) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}
