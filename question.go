package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type question struct {
	Type       string `json:"type"`
	ID         string `json:"id,omitempty"`
	QuestionID string `json:"question_id,omitempty"`
	Text       string `json:"text"`
}

func addQuestion(path string, question question) error {
	if strings.TrimSpace(question.Text) == "" {
		return errors.New("text cannot be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(question)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

func findAnswer(path, id string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var value question
		if json.Unmarshal(scanner.Bytes(), &value) == nil && value.Type == "answer" && value.QuestionID == id {
			return value.Text, nil
		}
	}
	return "", scanner.Err()
}
