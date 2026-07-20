//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

func openSessionBg(s Session) error {
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not found: %w", err)
	}
	cmd := exec.Command("wt", "nt", "-d", s.Directory, bin, "-s", s.ID)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
