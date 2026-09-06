package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	sessionDir  = ".grillme"
	sessionFile = "session.jsonl"
	pollDelay   = 150 * time.Millisecond
)

func parseQuestions(args []string) ([]question, error) {
	var questions []question
	for len(args) > 0 {
		if args[0] == "--recommended" {
			return nil, fmt.Errorf("--recommended must follow a question")
		}
		item := question{Text: strings.TrimSpace(args[0])}
		args = args[1:]
		if len(args) > 0 && args[0] == "--recommended" {
			if len(args) < 2 {
				return nil, fmt.Errorf("--recommended requires an answer")
			}
			item.RecommendedAnswer = strings.TrimSpace(args[1])
			args = args[2:]
		}
		questions = append(questions, item)
	}
	return questions, nil
}

func main() {
	if len(os.Args) < 3 || os.Args[1] != "ask" {
		fmt.Fprintln(os.Stderr, "usage: grillme ask <question> [--recommended <answer>] [question...]")
		os.Exit(2)
	}
	questions, err := parseQuestions(os.Args[2:])
	if err != nil {
		fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	if err := openGrillMePane(cwd); err != nil {
		fatal(err)
	}

	path := filepath.Join(sessionDir, sessionFile)
	ids := make([]string, len(questions))
	for index, item := range questions {
		ids[index] = fmt.Sprintf("%d-%d-%d", os.Getpid(), time.Now().UnixNano(), index)
		item.Type = "question"
		item.ID = ids[index]
		if err := addQuestion(path, item); err != nil {
			fatal(err)
		}
	}

	for _, id := range ids {
		for {
			answer, err := findAnswer(path, id)
			if err != nil {
				fatal(err)
			}
			if answer != "" {
				fmt.Println(answer)
				break
			}
			time.Sleep(pollDelay)
		}
	}
}
