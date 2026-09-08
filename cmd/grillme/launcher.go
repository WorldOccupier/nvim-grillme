package main

import (
	"fmt"
	"os"
)

const (
	launcherEnv         = "GRILLME_LAUNCHER"
	defaultLauncherName = "herdr"
)

type launcher interface {
	Open(cwd, sessionID string) (string, error)
	Close(id string) error
}

func configuredLauncher() (launcher, error) {
	name := os.Getenv(launcherEnv)
	if name == "" {
		name = defaultLauncherName
	}

	switch name {
	case "herdr":
		return newHerdrLauncher(), nil
	default:
		return nil, fmt.Errorf("unsupported launcher %q (supported: herdr)", name)
	}
}
