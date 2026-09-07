# nvim-grill-me

[![CI](https://github.com/WorldOccupier/nvim-grillme/actions/workflows/ci.yml/badge.svg)](https://github.com/WorldOccupier/nvim-grillme/actions/workflows/ci.yml)

A minimal question-and-answer bridge between a coding agent and Neovim.

## Requirements

- Linux or macOS
- Go 1.26+
- Neovim 0.10+
- Herdr 0.7.3+

## Install

Install the command:

```sh
go install github.com/WorldOccupier/nvim-grillme/cmd/grillme@latest
```

Ensure Go's bin directory is on `PATH`, then check the installed build:

```sh
grillme version
```

Install the Neovim plugin with Lazy:

```lua
{
  "WorldOccupier/nvim-grillme",
  cmd = { "GrillMeOpen", "GrillMeSubmit" },
}
```

Inspect the available commands or ask-command syntax with:

```sh
grillme --help
grillme ask --help
```

Tell your coding agent to ask questions with:

```sh
grillme ask --timeout 10m \
  "What should happen when authentication expires?" \
  --recommended "Clear the session and return to the sign-in screen."
```

Pass more question arguments to ask several at once. Each `--recommended`
value applies to the question immediately before it.

The command opens a right-hand Herdr pane containing your configured Neovim,
then waits until you submit an answer. The answer is printed to stdout so the
agent's command continues automatically.

`--timeout` accepts a Go duration such as `1s`, `10m`, or `2h`. Without it, the
wait has no time limit. A timeout, `Ctrl-C`, or `SIGTERM` stops the command with
a nonzero exit code and closes only the Herdr pane created for that request.

## Manual test

Run the complete demo with:

```sh
make demo
```

Or provide the question:

```sh
make demo QUESTIONS='"Which authentication method should we use?" "Should it be configurable?"'
```

`make demo` sets `GRILLME_PLUGIN_PATH` for local development, then cleans the
completed demo entries. An installed plugin needs no path: `grillme ask`
launches `nvim -c GrillMeOpen` directly.

From this repository inside a Herdr pane, ask a question:

```sh
go run ./cmd/grillme ask "What should happen when authentication expires?"
```

The command creates a right-hand Herdr pane and starts your configured Neovim
with `GrillMeOpen`. When running from this plugin repository, it loads the local
plugin automatically; once installed, Neovim loads it through your plugin
manager.

A recommended answer appears in grey when one was provided. Press `i` to edit it, or leave it untouched to accept it. Then run:

```vim
:GrillMeSubmit
```

The waiting `go run ./cmd/grillme` command prints the answer and exits. The dedicated Herdr
pane closes after submission.

## Local data and cleanup

GrillMe stores questions, recommended answers, answers, and timestamps in
`.grillme/session.jsonl` under the current project. The directory is ignored by
this repository's Git configuration, but projects using GrillMe should also add
`.grillme/` to their ignore file if the data should stay local.

Remove all completed question and answer pairs while keeping pending questions:

```sh
grillme clean
```

Keep completed pairs from the last seven days:

```sh
grillme clean --older-than 7d
```

Durations accept days or Go syntax, such as `7d`, `24h`, or `30m`. Age is measured from
the answer timestamp. Legacy completed pairs without timestamps are removed by
`grillme clean`, but retained when `--older-than` is set because their age
cannot be determined. Unknown event types are retained. If any line contains
malformed JSON, cleanup reports its line number and leaves the file unchanged.
Cleanup replaces the file atomically and sets its permissions to `0600`.

The CLI and Neovim plugin serialize changes with an atomic
`session.jsonl.lock` directory. This prevents asks, submissions, and cleanup
from interleaving writes or replacing data another process just appended.
Locks include the owner's process ID, so a later operation recovers a lock left
by a process that was interrupted.
