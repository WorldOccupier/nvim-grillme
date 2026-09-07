package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	sessionDir  = ".grillme"
	sessionFile = "session.jsonl"
)

// Release builds can set these with -ldflags, for example:
// -X main.version=v1.2.3 -X main.commit=abc1234
var (
	version = "devel"
	commit  = ""
)

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
	return runContext(context.Background(), args, stdout, stderr)
}

func runContext(ctx context.Context, args []string, stdout, stderr io.Writer) int {
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
		questions, timeout, err := parseAskArgs(args[1:])
		if err != nil {
			fmt.Fprintln(stderr, "grillme ask:", err)
			fmt.Fprintln(stderr, "Run \"grillme ask --help\" for usage.")
			return 2
		}
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		if err := ask(ctx, questions, stdout); err != nil {
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

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(runContext(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
