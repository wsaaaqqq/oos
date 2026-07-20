package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type releaseInfo struct {
	Tag    string       `json:"tag_name"`
	Name   string       `json:"name"`
	Assets []assetInfo  `json:"assets"`
}

type assetInfo struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type giteeAssetInfo struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

type giteeReleaseInfo struct {
	Tag    string            `json:"tag_name"`
	Name   string            `json:"name"`
	Assets []giteeAssetInfo  `json:"assets"`
}

func showVersion() {
	fmt.Println("oos", version)
}

func doUpgrade() {
	current := strings.TrimPrefix(version, "v")
	fmt.Printf("Current: %s\n\n", version)

	fetchTimeout := 15 * time.Second
	ch := make(chan *releaseInfo, 2)

	go func() { r, _ := fetchFromGitHub(fetchTimeout); ch <- r }()
	go func() { r, _ := fetchFromGitee(fetchTimeout); ch <- r }()

	var best *releaseInfo
	for i := 0; i < 2; i++ {
		select {
		case r := <-ch:
			if r == nil {
				continue
			}
			latest := strings.TrimPrefix(r.Tag, "v")
		if best == nil || compareVersion(latest, strings.TrimPrefix(best.Tag, "v")) > 0 {
			best = r
		}
		case <-time.After(fetchTimeout):
		}
	}

	if best == nil {
		fmt.Fprintln(os.Stderr, "Error: could not reach GitHub or Gitee")
		os.Exit(1)
	}

	latest := strings.TrimPrefix(best.Tag, "v")
	if current != "dev" && compareVersion(latest, current) <= 0 {
		fmt.Printf("Already up to date (%s)\n", best.Tag)
		os.Exit(0)
	}

	fmt.Printf("Latest:  %s\n", best.Tag)

	asset := findAsset(best)
	if asset == nil {
		fmt.Fprintf(os.Stderr, "Error: no asset for %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(1)
	}

	if err := downloadAndReplace(asset.URL); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Upgraded to %s\n", best.Tag)
}

func fetchFromGitHub(timeout time.Duration) (*releaseInfo, error) {
	url := "https://api.github.com/repos/wsaaaqqq/oos/releases/latest"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "oos-upgrader")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func fetchFromGitee(timeout time.Duration) (*releaseInfo, error) {
	url := "https://gitee.com/api/v5/repos/haitao666/oos/releases/latest"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "oos-upgrader")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gr giteeReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, err
	}

	r := releaseInfo{
		Tag:  gr.Tag,
		Name: gr.Name,
	}
	for _, a := range gr.Assets {
		r.Assets = append(r.Assets, assetInfo{
			Name: a.Name,
			URL:  fmt.Sprintf("https://gitee.com/haitao666/oos/attach_files/%d/download", a.ID),
		})
	}
	return &r, nil
}

func findAsset(r *releaseInfo) *assetInfo {
	suffix := fmt.Sprintf("_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		suffix += ".exe"
	}
	for _, a := range r.Assets {
		if strings.HasSuffix(a.Name, suffix) {
			return &a
		}
	}
	return nil
}

func compareVersion(a, b string) int {
	pa := splitVersion(a)
	pb := splitVersion(b)
	for i := 0; i < 3; i++ {
		if pa[i] > pb[i] {
			return 1
		}
		if pa[i] < pb[i] {
			return -1
		}
	}
	return 0
}

func splitVersion(v string) [3]int {
	var parts [3]int
	fmt.Sscanf(v, "%d.%d.%d", &parts[0], &parts[1], &parts[2])
	return parts
}

func downloadAndReplace(url string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get exe path: %w", err)
	}
	tmp := exe + ".new"

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("write: %w", err)
	}

	if runtime.GOOS == "windows" {
		return replaceSelfWindows(exe, tmp)
	}
	return replaceSelfUnix(tmp, exe)
}

func replaceSelfWindows(exe, tmp string) error {
	dir := filepath.Dir(exe)
	bat := filepath.Join(dir, "upgrade.bat")
	script := fmt.Sprintf("@echo off\r\ntimeout /t 1 >nul\r\nmove /y \"%s\" \"%s\"\r\ndel \"%%~f0\"\r\n", tmp, exe)
	if err := os.WriteFile(bat, []byte(script), 0644); err != nil {
		return fmt.Errorf("write bat: %w", err)
	}
	cmd := exec.Command("cmd", "/c", "start", "/b", bat)
	cmd.Start()
	os.Exit(0)
	return nil
}

func replaceSelfUnix(tmp, target string) error {
	if err := os.Rename(tmp, target); err != nil {
		return fmt.Errorf("replace: %w", err)
	}
	if err := os.Chmod(target, 0755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	os.Exit(0)
	return nil
}
