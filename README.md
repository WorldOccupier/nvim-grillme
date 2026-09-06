# nvim-grill-me

A minimal question-and-answer bridge between a coding agent and Neovim.

## Requirements

- Go 1.26+
- Neovim 0.10+
- Herdr 0.7.3+

## Install

Install the command:

```sh
go install github.com/vatsa/nvim-grill-me@latest
```

Ensure Go's bin directory is on `PATH`, then install the Neovim plugin with
Lazy:

```lua
{
  "vatsa/nvim-grill-me",
  cmd = { "GrillMeOpen", "GrillMeSubmit" },
}
```

Tell your coding agent to ask questions with:

```sh
grillme ask "What should happen when authentication expires?"
```

The command opens a right-hand Herdr pane containing your configured Neovim,
then waits until you submit an answer. The answer is printed to stdout so the
agent's command continues automatically.

## Manual test

Run the complete demo with:

```sh
make demo
```

Or provide the question:

```sh
make demo QUESTIONS='"Which authentication method should we use?" "Should it be configurable?"'
```

`make demo` sets `GRILLME_PLUGIN_PATH` for local development. An installed
plugin needs no path: `grillme ask` launches `nvim -c GrillMeOpen` directly.

From this repository inside a Herdr pane, ask a question:

```sh
go run . ask "What should happen when authentication expires?"
```

The command creates a right-hand Herdr pane and starts your configured Neovim
with `GrillMeOpen`. When running from this plugin repository, it loads the local
plugin automatically; once installed, Neovim loads it through your plugin
manager.

Type beneath `Answer:` in Neovim, press `Esc`, then run:

```vim
:GrillMeSubmit
```

The waiting `go run` command prints the answer and exits. The dedicated Herdr
pane closes after submission. Questions and answers are appended to
`.grillme/session.jsonl`.
