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
	herdrExecutable      = "herdr"
	installedNvimCommand = "nvim -c GrillMeOpen"
	localNvimCommand     = `nvim -c 'execute "set runtimepath+=" . fnameescape($GRILLME_PLUGIN_PATH)' -c 'runtime plugin/grillme.lua' -c GrillMeOpen`
)

func openGrillMePane(cwd string) error {
	if os.Getenv(herdrPaneIDEnv) == "" {
		return errors.New("not running inside a Herdr pane")
	}

	herdr := os.Getenv(herdrBinPathEnv)
	if herdr == "" {
		herdr = herdrExecutable
	}
	args := []string{"pane", "split", "--current", "--direction", "right", "--ratio", "0.35", "--cwd", cwd, "--no-focus"}
	pluginPath := os.Getenv(grillMePluginPathEnv)
	if pluginPath != "" {
		args = append(args, "--env", grillMePluginPathEnv+"="+pluginPath)
	}

	output, err := exec.Command(herdr, args...).Output()
	if err != nil {
		return fmt.Errorf("split Herdr pane: %w", err)
	}
	paneID, err := splitPaneID(output)
	if err != nil {
		return err
	}

	command := installedNvimCommand
	if pluginPath != "" {
		command = localNvimCommand
	}
	if err := exec.Command(herdr, "pane", "run", paneID, command).Run(); err != nil {
		_ = exec.Command(herdr, "pane", "close", paneID).Run()
		return fmt.Errorf("start Neovim in Herdr pane: %w", err)
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
