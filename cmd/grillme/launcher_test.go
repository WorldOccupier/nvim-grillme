package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type recordedCommand struct {
	name string
	args []string
}

type fakeCommandRunner struct {
	output    []byte
	outputErr error
	runErr    error
	commands  []recordedCommand
}

func (r *fakeCommandRunner) Output(name string, args ...string) ([]byte, error) {
	r.commands = append(r.commands, recordedCommand{name, append([]string(nil), args...)})
	return r.output, r.outputErr
}

func (r *fakeCommandRunner) Run(name string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{name, append([]string(nil), args...)})
	if len(args) >= 2 && args[1] == "run" {
		return r.runErr
	}
	return nil
}

func TestConfiguredLauncherDefaultsToHerdr(t *testing.T) {
	t.Setenv(launcherEnv, "")
	t.Setenv(herdrBinPathEnv, "/custom/herdr")
	selected, err := configuredLauncher()
	if err != nil {
		t.Fatal(err)
	}
	herdr, ok := selected.(*herdrLauncher)
	if !ok || herdr.executable != "/custom/herdr" {
		t.Fatalf("got %#v", selected)
	}
}

func TestConfiguredLauncherRejectsUnsupportedValue(t *testing.T) {
	t.Setenv(launcherEnv, "tmux")
	_, err := configuredLauncher()
	if err == nil || !strings.Contains(err.Error(), `unsupported launcher "tmux"`) {
		t.Fatalf("got %v", err)
	}
}

func TestHerdrLauncherPreservesPanePlacementAndEnvironment(t *testing.T) {
	t.Setenv(herdrPaneIDEnv, "parent")
	t.Setenv(grillMePluginPathEnv, "/plugin")
	runner := &fakeCommandRunner{output: []byte(`{"result":{"pane":{"pane_id":"w1:p2"}}}`)}
	selected := &herdrLauncher{executable: "herdr-test", runner: runner}

	id, err := selected.Open("/project", "session-1")
	if err != nil || id != "w1:p2" {
		t.Fatalf("got %q, %v", id, err)
	}
	want := []recordedCommand{
		{"herdr-test", []string{"pane", "split", "--current", "--direction", "right", "--ratio", "0.35", "--cwd", "/project", "--no-focus", "--env", "GRILLME_SESSION_ID=session-1", "--env", "GRILLME_PLUGIN_PATH=/plugin"}},
		{"herdr-test", []string{"pane", "run", "w1:p2", localNvimCommand}},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}
}

func TestHerdrLauncherClosesPaneWhenNeovimFails(t *testing.T) {
	t.Setenv(herdrPaneIDEnv, "parent")
	t.Setenv(grillMePluginPathEnv, "")
	runner := &fakeCommandRunner{
		output: []byte(`{"result":{"pane":{"pane_id":"w1:p2"}}}`),
		runErr: errors.New("failed"),
	}
	selected := &herdrLauncher{executable: "herdr-test", runner: runner}

	if _, err := selected.Open("/project", "session-1"); err == nil {
		t.Fatal("expected an error")
	}
	last := runner.commands[len(runner.commands)-1]
	if !reflect.DeepEqual(last.args, []string{"pane", "close", "w1:p2"}) {
		t.Fatalf("last command = %#v", last)
	}
}
