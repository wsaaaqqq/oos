//go:build !windows

package main

import (
	"fmt"
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
