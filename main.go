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
		fmt.Fprintln(os.Stderr, "usage: grillme ask <question>")
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
	id := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	question := question{Type: "question", ID: id, Text: strings.Join(os.Args[2:], " ")}
	if err := addQuestion(path, question); err != nil {
		fatal(err)
	}

	for {
		answer, err := findAnswer(path, id)
		if err != nil {
			fatal(err)
		}
		if answer != "" {
			fmt.Println(answer)
			return
		}
		time.Sleep(pollDelay)
	}
}
