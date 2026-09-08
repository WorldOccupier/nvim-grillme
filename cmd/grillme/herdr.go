package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

const (
	herdrPaneIDEnv       = "HERDR_PANE_ID"
	herdrBinPathEnv      = "HERDR_BIN_PATH"
	grillMePluginPathEnv = "GRILLME_PLUGIN_PATH"
	grillMeSessionIDEnv  = "GRILLME_SESSION_ID"
	herdrExecutable      = "herdr"
	installedNvimCommand = "nvim -c GrillMeOpen"
	localNvimCommand     = `nvim -c 'execute "set runtimepath+=" . fnameescape($GRILLME_PLUGIN_PATH)' -c 'runtime plugin/grillme.lua' -c GrillMeOpen`
)

type commandRunner interface {
	Output(name string, args ...string) ([]byte, error)
	Run(name string, args ...string) error
}

type execCommandRunner struct{}

func (execCommandRunner) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (execCommandRunner) Run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

type herdrLauncher struct {
	executable string
	runner     commandRunner
}

func newHerdrLauncher() launcher {
	executable := os.Getenv(herdrBinPathEnv)
	if executable == "" {
		executable = herdrExecutable
	}
	return &herdrLauncher{executable: executable, runner: execCommandRunner{}}
}

func (l *herdrLauncher) Open(cwd, sessionID string) (string, error) {
	if os.Getenv(herdrPaneIDEnv) == "" {
		return "", errors.New("not running inside a Herdr pane")
	}

	args := []string{"pane", "split", "--current", "--direction", "right", "--ratio", "0.35", "--cwd", cwd, "--no-focus", "--env", grillMeSessionIDEnv + "=" + sessionID}
	pluginPath := os.Getenv(grillMePluginPathEnv)
	if pluginPath != "" {
		args = append(args, "--env", grillMePluginPathEnv+"="+pluginPath)
	}

	output, err := l.runner.Output(l.executable, args...)
	if err != nil {
		return "", fmt.Errorf("split Herdr pane: %w", err)
	}
	paneID, err := splitPaneID(output)
	if err != nil {
		return "", err
	}

	command := installedNvimCommand
	if pluginPath != "" {
		command = localNvimCommand
	}
	if err := l.runner.Run(l.executable, "pane", "run", paneID, command); err != nil {
		_ = l.runner.Run(l.executable, "pane", "close", paneID)
		return "", fmt.Errorf("start Neovim in Herdr pane: %w", err)
	}
	return paneID, nil
}

func (l *herdrLauncher) Close(paneID string) error {
	if err := l.runner.Run(l.executable, "pane", "close", paneID); err != nil {
		return fmt.Errorf("close Herdr pane %s: %w", paneID, err)
	}
	return nil
}

func splitPaneID(data []byte) (string, error) {
	var response herdrResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("read Herdr response: %w", err)
	}
	if response.Result.Pane.ID == "" {
		return "", errors.New("Herdr response did not include a pane ID")
	}
	return response.Result.Pane.ID, nil
}
