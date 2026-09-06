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

func main() {
	if len(os.Args) < 3 || os.Args[1] != "ask" {
		fmt.Fprintln(os.Stderr, "usage: grillme ask <question> [question...]")
		os.Exit(2)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	if err := openGrillMePane(cwd); err != nil {
		fatal(err)
	}

	path := filepath.Join(sessionDir, sessionFile)
	ids := make([]string, len(os.Args)-2)
	for index, text := range os.Args[2:] {
		ids[index] = fmt.Sprintf("%d-%d-%d", os.Getpid(), time.Now().UnixNano(), index)
		if err := addQuestion(path, question{Type: "question", ID: ids[index], Text: strings.TrimSpace(text)}); err != nil {
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
