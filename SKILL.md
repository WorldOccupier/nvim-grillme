---
name: grill-me
description: Grill the user relentlessly about a plan, decision, or idea. Use when the user wants to stress-test their thinking, or uses any 'grill' trigger phrases.
---

# Grill Me

Interview the user relentlessly until you reach a shared understanding. Map this as a design tree: every decision branches into the decisions that hang off it.

Work the tree in rounds. The frontier is every decision whose prerequisites are already settled: the questions you can ask now without guessing at answers you haven't heard yet. Ask the whole frontier in one round: number each question and give your recommended answer. Then wait for the user's answers before the next round.

Each round the user answers reshapes the tree: settled decisions push the frontier outward and unblock questions that depended on them. Recompute the frontier and ask the next round. A question whose answer depends on another question still open in this round belongs to a later round, not this one.

Finding facts is your job, never the user's. When a frontier question needs a fact from the environment (filesystem, tools, etc.), dispatch a sub-agent to find it; don't ask the user for anything you could look up yourself. Don't block on it: a running exploration is an unsettled prerequisite, so only the questions downstream of it wait for the sub-agent to report; ask the rest of the frontier now. The decisions are the user's: put each to them and wait.

The session is done when the frontier is empty: every branch of the design tree visited, nothing left silently assumed. Do not act on it until the user confirms you have reached a shared understanding.

## Ask questions

Pass the whole frontier as separate quoted arguments. Number each question and include your recommended answer:

```sh
grillme ask \
  "Which authentication method should we use?" \
  "Should it be configurable?"
```

Run the command with a long or disabled timeout. It is interactive and will not exit until the user submits answers.

Answers are printed to stdout in the same order as the questions, one answer at a time. Treat the full command output as user input and use it to continue the task.

## Requirements

The command must run inside a Herdr pane with `HERDR_PANE_ID` set. The `grillme` executable, `herdr`, and Neovim must be available, and the Neovim plugin must be installed.

When developing this repository locally, use:

```sh
GRILLME_PLUGIN_PATH="$PWD" go run . ask "Your question"
```

If the command reports `not running inside a Herdr pane`, tell the user that GrillMe requires Herdr instead of retrying. If Neovim opens, wait for the user to submit rather than starting another request.
