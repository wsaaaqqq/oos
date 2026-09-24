//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func openSessionBg(s Session) error {
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not found: %w", err)
	}
	cmd := exec.Command(bin, "-s", s.ID)
	cmd.Dir = s.Directory
	return cmd.Start()
}

func startSessionTabWithServer(s Session, port int, username, password string) error {
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not found: %w", err)
	}
	args := []string{"-s", s.ID, "--hostname", "127.0.0.1", "--port", fmt.Sprint(port)}
	cmd := exec.Command(bin, args...)
	cmd.Dir = s.Directory
	cmd.Env = replaceEnv(os.Environ(), map[string]string{
		"OPENCODE_SERVER_USERNAME": username,
		"OPENCODE_SERVER_PASSWORD": password,
	})
	return cmd.Start()
}

func spawnMonitor(sessionID string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get self path: %w", err)
	}
	args := []string{"-m"}
	if sessionID != "" {
		args = append(args, sessionID)
	}
	cmd := exec.Command(self, args...)
	return cmd.Start()
}
