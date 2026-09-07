package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: grillme ask <question> [--recommended <answer>] [question...]")
	fmt.Fprintln(os.Stderr, "       grillme clean [--older-than <duration>]")
}

func parseRetentionDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") {
		days, err := strconv.ParseFloat(strings.TrimSuffix(value, "d"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(days * float64(24*time.Hour)), nil
	}
	return time.ParseDuration(value)
}

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "clean" {
		var olderThan *time.Duration
		if len(os.Args) == 4 && os.Args[2] == "--older-than" {
			duration, err := parseRetentionDuration(os.Args[3])
			if err != nil || duration < 0 {
				fatal(fmt.Errorf("invalid --older-than duration %q", os.Args[3]))
			}
			olderThan = &duration
		} else if len(os.Args) != 2 {
			printUsage()
			os.Exit(2)
		}
		removed, err := cleanSession(filepath.Join(sessionDir, sessionFile), olderThan, time.Now())
		if err != nil {
			fatal(err)
		}
		fmt.Printf("removed %d events\n", removed)
		return
	}
	if len(os.Args) < 3 || os.Args[1] != "ask" {
		printUsage()
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
		item.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
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
