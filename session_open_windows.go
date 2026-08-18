//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func openSessionBg(s Session) error {
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not found: %w", err)
	}

	if _, err := exec.LookPath("wt"); err == nil {
		cmd := exec.Command("wt", "nt", "-d", s.Directory, bin, "-s", s.ID)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		return cmd.Start()
	}

	cmd := exec.Command("cmd", "/c", "start", "", bin, "-s", s.ID)
	cmd.Dir = s.Directory
	return cmd.Start()
}

func spawnMonitor(sessionID string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get self path: %w", err)
	}

	args := []string{self, "-m"}
	if sessionID != "" {
		args = append(args, sessionID)
	}

	if _, err := exec.LookPath("wt"); err == nil {
		cmd := exec.Command("wt", append([]string{"nt"}, args...)...)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		return cmd.Start()
	}

	cmd := exec.Command("cmd", append([]string{"/c", "start", ""}, args...)...)
	return cmd.Start()
}
