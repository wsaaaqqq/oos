package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ForkSessionAtMessage uses OpenCode's own session fork API. A temporary
// loopback-only server avoids relying on the random, authenticated port of an
// already-running TUI instance.
func ForkSessionAtMessage(directory, sessionID, messageID string) (string, error) {
	bin, err := exec.LookPath("opencode")
	if err != nil {
		return "", fmt.Errorf("opencode not found: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("reserve server port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	password, err := randomServerPassword()
	if err != nil {
		return "", fmt.Errorf("create temporary server credentials: %w", err)
	}
	cmd := exec.Command(bin, "serve", "--hostname", "127.0.0.1", "--port", strconv.Itoa(port))
	cmd.Dir = directory
	cmd.Env = replaceEnv(os.Environ(), map[string]string{
		"OPENCODE_SERVER_USERNAME": "oos",
		"OPENCODE_SERVER_PASSWORD": password,
	})
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start temporary opencode server: %w", err)
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	defer stopServerProcess(cmd, waitDone)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: 5 * time.Second}
	if err := waitForServer(baseURL, client, password, waitDone); err != nil {
		return "", err
	}

	body, _ := json.Marshal(map[string]string{"messageID": messageID})
	endpoint := baseURL + "/session/" + url.PathEscape(sessionID) + "/fork"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create fork request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("oos", password)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fork session: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read fork response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fork session: opencode returned %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}
	var result struct {
		ID   string `json:"id"`
		Data *struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("decode fork response: %w", err)
	}
	if result.ID != "" {
		return result.ID, nil
	}
	if result.Data != nil && result.Data.ID != "" {
		return result.Data.ID, nil
	}
	return "", fmt.Errorf("fork response did not contain a session id")
}

// OpenForkedSessionWithPrompt opens the fork in a new terminal tab, then uses
// the TUI's append-prompt endpoint to restore the selected question without
// submitting it to a model.
func OpenForkedSessionWithPrompt(session Session, prompt string) error {
	port, err := freeLoopbackPort()
	if err != nil {
		return err
	}
	password, err := randomServerPassword()
	if err != nil {
		return fmt.Errorf("create TUI credentials: %w", err)
	}
	if err := startSessionTabWithServer(session, port, "oos", password); err != nil {
		return err
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: 5 * time.Second}
	if err := waitForServer(baseURL, client, password, nil); err != nil {
		return fmt.Errorf("fork opened, but its TUI server did not become ready: %w", err)
	}

	// The HTTP server starts before the TUI has necessarily subscribed to the
	// event bus. Give the new TUI time to initialize before appending the draft.
	time.Sleep(2 * time.Second)
	body, _ := json.Marshal(map[string]string{"text": prompt})
	req, err := http.NewRequest(http.MethodPost, baseURL+"/tui/append-prompt", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create prompt restore request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("oos", password)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fork opened, but could not restore the question: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return fmt.Errorf("fork opened, but prompt restore returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func freeLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("reserve loopback port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return 0, fmt.Errorf("release loopback port: %w", err)
	}
	return port, nil
}

func randomServerPassword() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func replaceEnv(env []string, values map[string]string) []string {
	out := make([]string, 0, len(env)+len(values))
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			out = append(out, entry)
			continue
		}
		if _, replace := values[strings.ToUpper(key)]; replace {
			continue
		}
		out = append(out, entry)
	}
	for key, value := range values {
		out = append(out, key+"="+value)
	}
	return out
}

func waitForServer(baseURL string, client *http.Client, password string, waitDone <-chan error) error {
	// Instance bootstrap may initialize plugins, VCS and LSP. On large projects
	// it can take longer than a typical web-server health check.
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		if waitDone != nil {
			select {
			case err := <-waitDone:
				return fmt.Errorf("opencode server exited before becoming ready: %v", err)
			default:
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/global/health", nil)
		if err == nil {
			req.SetBasicAuth("oos", password)
			resp, requestErr := client.Do(req)
			if requestErr == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					cancel()
					return nil
				}
				lastErr = fmt.Errorf("health endpoint returned %s", resp.Status)
			} else {
				lastErr = requestErr
			}
		}
		cancel()
		time.Sleep(250 * time.Millisecond)
	}
	if lastErr != nil {
		return fmt.Errorf("opencode server did not become ready: %w", lastErr)
	}
	return fmt.Errorf("opencode server did not become ready before timeout")
}

func stopServerProcess(cmd *exec.Cmd, waitDone <-chan error) {
	if cmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run()
	} else {
		_ = cmd.Process.Kill()
	}
	select {
	case <-waitDone:
	case <-time.After(3 * time.Second):
	}
}
