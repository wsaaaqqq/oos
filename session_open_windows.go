//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

func windowsTerminalPath() (string, bool) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", false
	}
	path := filepath.Join(localAppData, "Microsoft", "WindowsApps", "wt.exe")
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

// parentShellName returns the lowercased executable name of the process that
// launched oos (the shell the user is typing in), e.g. "cmd.exe",
// "powershell.exe", "pwsh.exe". Empty string when it cannot be determined.
func parentShellName() string {
	ppid := uint32(os.Getppid())
	if ppid == 0 {
		return ""
	}

	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return ""
	}
	defer syscall.CloseHandle(snap)

	var entry syscall.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := syscall.Process32First(snap, &entry); err != nil {
		return ""
	}
	for {
		if entry.ProcessID == ppid {
			return strings.ToLower(syscall.UTF16ToString(entry.ExeFile[:]))
		}
		if err := syscall.Process32Next(snap, &entry); err != nil {
			break
		}
	}
	return ""
}

// shellCommand wraps exe+args so that the terminal tab hosts a shell instead
// of the process itself. When exe exits (Ctrl+C or normal quit), the shell
// stays at a prompt and the tab is not closed. The shell matches the one oos
// was launched from; unknown shells fall back to cmd.exe.
func shellCommand(exe string, args ...string) []string {
	shell := parentShellName()
	switch shell {
	case "powershell.exe", "pwsh.exe":
		name := "powershell"
		if shell == "pwsh.exe" {
			name = "pwsh"
		}
		// single-quote each token for PowerShell; '' escapes a literal quote
		var b strings.Builder
		b.WriteString("& '")
		b.WriteString(strings.ReplaceAll(exe, "'", "''"))
		b.WriteString("'")
		for _, a := range args {
			b.WriteString(" '")
			b.WriteString(strings.ReplaceAll(a, "'", "''"))
			b.WriteString("'")
		}
		return []string{name, "-NoExit", "-Command", b.String()}
	default:
		return append([]string{"cmd", "/k", exe}, args...)
	}
}

func openSessionBg(s Session) error {
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not found: %w", err)
	}

	shellArgs := shellCommand(bin, "-s", s.ID)

	if wt, ok := windowsTerminalPath(); ok {
		args := append([]string{"-w", "0", "nt", "-d", s.Directory}, shellArgs...)
		cmd := exec.Command(wt, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		return cmd.Start()
	}

	args := append([]string{"/c", "start", ""}, shellArgs...)
	cmd := exec.Command("cmd", args...)
	cmd.Dir = s.Directory
	return cmd.Start()
}

func startSessionTabWithServer(s Session, port int, username, password string) error {
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return fmt.Errorf("opencode not found: %w", err)
	}
	args := []string{"-s", s.ID, "--hostname", "127.0.0.1", "--port", fmt.Sprint(port)}
	var shellArgs []string
	if parentShellName() == "powershell.exe" || parentShellName() == "pwsh.exe" {
		shell := "powershell"
		if parentShellName() == "pwsh.exe" {
			shell = "pwsh"
		}
		command := "$env:OPENCODE_SERVER_USERNAME=" + psQuote(username) + "; $env:OPENCODE_SERVER_PASSWORD=" + psQuote(password) + "; & " + psQuote(bin)
		for _, arg := range args {
			command += " " + psQuote(arg)
		}
		shellArgs = []string{shell, "-NoExit", "-Command", command}
	} else {
		command := `set "OPENCODE_SERVER_USERNAME=` + cmdEnvValue(username) + `" && set "OPENCODE_SERVER_PASSWORD=` + cmdEnvValue(password) + `" && ` + cmdQuote(bin)
		for _, arg := range args {
			command += " " + cmdQuote(arg)
		}
		shellArgs = []string{"cmd", "/k", command}
	}

	if wt, ok := windowsTerminalPath(); ok {
		wtArgs := append([]string{"-w", "0", "nt", "-d", s.Directory}, shellArgs...)
		cmd := exec.Command(wt, wtArgs...)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		return cmd.Start()
	}
	cmd := exec.Command("cmd", append([]string{"/c", "start", ""}, shellArgs...)...)
	cmd.Dir = s.Directory
	return cmd.Start()
}

func psQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func cmdEnvValue(value string) string {
	// Generated credentials use URL-safe base64, so they contain no cmd metacharacters.
	return strings.ReplaceAll(value, `"`, "")
}

func cmdQuote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func spawnMonitor(sessionID string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get self path: %w", err)
	}

	exeArgs := []string{"-m"}
	if sessionID != "" {
		exeArgs = append(exeArgs, sessionID)
	}
	shellArgs := shellCommand(self, exeArgs...)

	if wt, ok := windowsTerminalPath(); ok {
		args := append([]string{"-w", "0", "nt"}, shellArgs...)
		cmd := exec.Command(wt, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
		return cmd.Start()
	}

	args := append([]string{"/c", "start", ""}, shellArgs...)
	cmd := exec.Command("cmd", args...)
	return cmd.Start()
}
