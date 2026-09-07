package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	sessionDir  = ".grillme"
	sessionFile = "session.jsonl"
	pollDelay   = 150 * time.Millisecond
)

// Release builds can set these with -ldflags, for example:
// -X main.version=v1.2.3 -X main.commit=abc1234
var (
	version = "devel"
	commit  = ""
)

func parseQuestions(args []string) ([]question, error) {
	var questions []question
	for len(args) > 0 {
		if args[0] == "--recommended" {
			return nil, fmt.Errorf("--recommended must follow a question")
		}
		if strings.HasPrefix(args[0], "-") {
			return nil, fmt.Errorf("unknown option %q", args[0])
		}
		item := question{Text: strings.TrimSpace(args[0])}
		if item.Text == "" {
			return nil, fmt.Errorf("question cannot be empty")
		}
		args = args[1:]
		if len(args) > 0 && args[0] == "--recommended" {
			if len(args) < 2 || strings.HasPrefix(args[1], "--") {
				return nil, fmt.Errorf("--recommended requires an answer")
			}
			item.RecommendedAnswer = strings.TrimSpace(args[1])
			if item.RecommendedAnswer == "" {
				return nil, fmt.Errorf("recommended answer cannot be empty")
			}
			args = args[2:]
		}
		questions = append(questions, item)
	}
	if len(questions) == 0 {
		return nil, fmt.Errorf("at least one question is required")
	}
	return questions, nil
}

func printTopHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: grillme <command> [options]

Ask questions through Neovim and wait for their answers.

Commands:
  ask      Ask one or more questions
  clean    Remove completed questions and answers
  version  Print version information

Run "grillme <command> --help" for command help.
`)
}

func printAskHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: grillme ask <question> [--recommended <answer>] [question...]

Ask one or more questions. Put --recommended and its answer directly after
the question it belongs to.

GrillMe requires Neovim 0.10 or later and Herdr 0.7.3 or later. The herdr
executable must be on PATH unless HERDR_BIN_PATH is set.

Examples:
  grillme ask "Which database?"
  grillme ask "Which database?" --recommended "Postgres"
  grillme ask "Database?" --recommended "Postgres" "Enable backups?" --recommended "Yes"
`)
}

func printCleanHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: grillme clean [--older-than <duration>]

Remove completed question and answer pairs. Durations accept Go syntax or days,
such as 24h or 7d.
`)
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

func versionString() string {
	buildVersion, buildCommit := version, commit
	if info, ok := debug.ReadBuildInfo(); ok {
		if buildVersion == "devel" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			buildVersion = info.Main.Version
		}
		if buildCommit == "" {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					buildCommit = setting.Value
					break
				}
			}
		}
	}
	if buildCommit != "" {
		return fmt.Sprintf("grillme %s (commit %s)", buildVersion, buildCommit)
	}
	return "grillme " + buildVersion
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "grillme: a command is required")
		fmt.Fprintln(stderr, "Run \"grillme --help\" for usage.")
		return 2
	}

	switch args[0] {
	case "-h", "--help", "help":
		if len(args) != 1 {
			return usageError(stderr, "help does not accept arguments")
		}
		printTopHelp(stdout)
		return 0
	case "version":
		if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
			fmt.Fprintln(stdout, "Usage: grillme version")
			return 0
		}
		if len(args) != 1 {
			return usageError(stderr, "version does not accept arguments")
		}
		fmt.Fprintln(stdout, versionString())
		return 0
	case "ask":
		if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
			printAskHelp(stdout)
			return 0
		}
		questions, err := parseQuestions(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, "grillme ask:", err)
			fmt.Fprintln(stderr, "Run \"grillme ask --help\" for usage.")
			return 2
		}
		if err := ask(questions, stdout); err != nil {
			fmt.Fprintln(stderr, "grillme:", err)
			return 1
		}
		return 0
	case "clean":
		if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
			printCleanHelp(stdout)
			return 0
		}
		var olderThan *time.Duration
		if len(args) == 3 && args[1] == "--older-than" {
			duration, err := parseRetentionDuration(args[2])
			if err != nil || duration < 0 {
				return commandUsageError(stderr, "clean", fmt.Sprintf("invalid --older-than duration %q", args[2]))
			}
			olderThan = &duration
		} else if len(args) != 1 {
			return commandUsageError(stderr, "clean", "invalid arguments")
		}
		removed, err := cleanSession(filepath.Join(sessionDir, sessionFile), olderThan, time.Now())
		if err != nil {
			fmt.Fprintln(stderr, "grillme:", err)
			return 1
		}
		fmt.Fprintf(stdout, "removed %d events\n", removed)
		return 0
	default:
		return usageError(stderr, fmt.Sprintf("unknown command %q", args[0]))
	}
}

func usageError(w io.Writer, message string) int {
	fmt.Fprintln(w, "grillme:", message)
	fmt.Fprintln(w, "Run \"grillme --help\" for usage.")
	return 2
}

func commandUsageError(w io.Writer, command, message string) int {
	fmt.Fprintf(w, "grillme %s: %s\n", command, message)
	fmt.Fprintf(w, "Run \"grillme %s --help\" for usage.\n", command)
	return 2
}

func ask(questions []question, stdout io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := openGrillMePane(cwd); err != nil {
		return err
	}

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
			answer, err := findAnswer(path, id)
			if err != nil {
				return err
			}
			if answer != "" {
				fmt.Fprintln(stdout, answer)
				break
			}
			time.Sleep(pollDelay)
		}
	}
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
