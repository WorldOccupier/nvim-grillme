package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	lockRetryInterval = 10 * time.Millisecond
	lockTimeout       = 10 * time.Second
	incompleteLockAge = 2 * time.Second
)

type sessionLock struct {
	path string
}

func acquireSessionLock(sessionPath string) (*sessionLock, error) {
	lockPath := sessionPath + ".lock"
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}

	deadline := time.Now().Add(lockTimeout)
	for {
		if err := os.Mkdir(lockPath, 0o700); err == nil {
			owner := []byte(strconv.Itoa(os.Getpid()) + "\n")
			if err := os.WriteFile(filepath.Join(lockPath, "owner"), owner, 0o600); err != nil {
				_ = os.Remove(lockPath)
				return nil, fmt.Errorf("write session lock owner: %w", err)
			}
			return &sessionLock{path: lockPath}, nil
		} else if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("acquire session lock: %w", err)
		}

		stale, err := staleSessionLock(lockPath)
		if err != nil {
			return nil, fmt.Errorf("inspect session lock: %w", err)
		}
		if stale {
			if err := os.RemoveAll(lockPath); err != nil {
				return nil, fmt.Errorf("remove stale session lock: %w", err)
			}
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for session lock %s", lockPath)
		}
		time.Sleep(lockRetryInterval)
	}
}

func staleSessionLock(lockPath string) (bool, error) {
	data, err := os.ReadFile(filepath.Join(lockPath, "owner"))
	if errors.Is(err, os.ErrNotExist) {
		info, statErr := os.Stat(lockPath)
		if errors.Is(statErr, os.ErrNotExist) {
			return false, nil
		}
		if statErr != nil {
			return false, statErr
		}
		return time.Since(info.ModTime()) > incompleteLockAge, nil
	}
	if err != nil {
		return false, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		info, statErr := os.Stat(lockPath)
		if statErr != nil {
			return false, statErr
		}
		return time.Since(info.ModTime()) > incompleteLockAge, nil
	}
	err = syscall.Kill(pid, 0)
	return errors.Is(err, syscall.ESRCH), nil
}

func (lock *sessionLock) release() error {
	if err := os.Remove(filepath.Join(lock.path, "owner")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove session lock owner: %w", err)
	}
	if err := os.Remove(lock.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("release session lock: %w", err)
	}
	return nil
}

func withSessionLock(path string, mutate func() error) (err error) {
	lock, err := acquireSessionLock(path)
	if err != nil {
		return err
	}
	defer func() {
		if releaseErr := lock.release(); err == nil && releaseErr != nil {
			err = releaseErr
		}
	}()
	return mutate()
}
