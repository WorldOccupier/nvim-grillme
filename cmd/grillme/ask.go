package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const pollDelay = 150 * time.Millisecond

func parseAskArgs(args []string) ([]question, time.Duration, error) {
	var questions []question
	var timeout time.Duration
	for len(args) > 0 {
		if args[0] == "--timeout" {
			if len(args) < 2 {
				return nil, 0, fmt.Errorf("--timeout requires a duration")
			}
			parsed, err := time.ParseDuration(args[1])
			if err != nil || parsed <= 0 {
				return nil, 0, fmt.Errorf("invalid --timeout duration %q", args[1])
			}
			timeout = parsed
			args = args[2:]
			continue
		}
		if args[0] == "--recommended" {
			return nil, 0, fmt.Errorf("--recommended must follow a question")
		}
		if strings.HasPrefix(args[0], "-") {
			return nil, 0, fmt.Errorf("unknown option %q", args[0])
		}
		item := question{Text: strings.TrimSpace(args[0])}
		if item.Text == "" {
			return nil, 0, fmt.Errorf("question cannot be empty")
		}
		args = args[1:]
		if len(args) > 0 && args[0] == "--recommended" {
			if len(args) < 2 || strings.HasPrefix(args[1], "--") {
				return nil, 0, fmt.Errorf("--recommended requires an answer")
			}
			item.RecommendedAnswer = strings.TrimSpace(args[1])
			if item.RecommendedAnswer == "" {
				return nil, 0, fmt.Errorf("recommended answer cannot be empty")
			}
			args = args[2:]
		}
		questions = append(questions, item)
	}
	if len(questions) == 0 {
		return nil, 0, fmt.Errorf("at least one question is required")
	}
	return questions, timeout, nil
}

func parseQuestions(args []string) ([]question, error) {
	questions, _, err := parseAskArgs(args)
	return questions, err
}

func printAskHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: grillme ask [--timeout <duration>] <question> [--recommended <answer>] [question...]

Ask one or more questions. Put --recommended and its answer directly after
the question it belongs to. By default, GrillMe waits without a limit. Use
--timeout with a Go duration, such as 10m, to limit the wait. SIGINT and SIGTERM
cancel the wait and close the pane created by this command.

GrillMe requires Neovim 0.10 or later and Herdr 0.7.3 or later. The herdr
executable must be on PATH unless HERDR_BIN_PATH is set.

Examples:
  grillme ask --timeout 10m "Which database?"
  grillme ask "Which database?" --recommended "Postgres"
  grillme ask "Database?" --recommended "Postgres" "Enable backups?" --recommended "Yes"
`)
}

type askDependencies struct {
	open  func(string) (string, error)
	close func(string) error
	find  func(string, string) (string, error)
	poll  time.Duration
}

func ask(ctx context.Context, questions []question, stdout io.Writer) error {
	return askWithDependencies(ctx, questions, stdout, askDependencies{openGrillMePane, closeGrillMePane, findAnswer, pollDelay})
}

func askWithDependencies(ctx context.Context, questions []question, stdout io.Writer, deps askDependencies) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	paneID, err := deps.open(cwd)
	if err != nil {
		return err
	}
	completed := false
	defer func() {
		if !completed {
			_ = deps.close(paneID)
		}
	}()

	path := filepath.Join(sessionDir, sessionFile)
	ids := make([]string, len(questions))
	for index, item := range questions {
		ids[index] = fmt.Sprintf("%d-%d-%d", os.Getpid(), time.Now().UnixNano(), index)
		item.Type = "question"
		item.ID = ids[index]
		item.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
		if err := addQuestion(path, item); err != nil {
			return err
		}
	}

	for _, id := range ids {
		for {
			answer, err := deps.find(path, id)
			if err != nil {
				return err
			}
			if answer != "" {
				fmt.Fprintln(stdout, answer)
				break
			}
			timer := time.NewTimer(deps.poll)
			select {
			case <-ctx.Done():
				timer.Stop()
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return errors.New("timed out waiting for answers")
				}
				return errors.New("cancelled while waiting for answers")
			case <-timer.C:
			}
		}
	}
	completed = true
	return nil
}
