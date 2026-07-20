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

	if _, err := exec.LookPath("wt"); err == nil {
		cmd := exec.Command("wt", "nt", "-d", s.Directory, bin, "-s", s.ID)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		return cmd.Start()
	}

	cmd := exec.Command(bin, "-s", s.ID)
	cmd.Dir = s.Directory
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000010}
	return cmd.Start()
}
