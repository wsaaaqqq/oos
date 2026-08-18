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

func spawnMonitor(sessionID string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get self path: %w", err)
	}
	args := []string{"--monitor"}
	if sessionID != "" {
		args = append(args, sessionID)
	}
	cmd := exec.Command(self, args...)
	return cmd.Start()
}
