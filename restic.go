package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Snapshot struct {
	ID       string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Paths    []string  `json:"paths"`
	Hostname string    `json:"hostname"`
}

var defaultExcludes = func() []string {
	base := []string{
		"--exclude-larger-than", "20M",
		"--exclude", ".git",
		"--exclude", "node_modules", "--exclude", ".next", "--exclude", ".turbo",
		"--exclude", "dist", "--exclude", "build", "--exclude", ".gradle",
		"--exclude", ".dart_tool", "--exclude", "__pycache__", "--exclude", ".venv",
		"--exclude", "venv", "--exclude", ".cache", "--exclude", ".parcel-cache",
		"--exclude", "coverage", "--exclude", ".vercel", "--exclude", ".angular",
		"--exclude", ".wrangler", "--exclude", ".worktrees",
		"--exclude", "platform-tools", "--exclude", "build-tools",
		"--exclude", "platforms/android-*",
		"--exclude", "*.mp4", "--exclude", "*.mov", "--exclude", "*.zip", "--exclude", "*.tar.gz",
		"--exclude", "*.dylib", "--exclude", "*.jar", "--exclude", "*.so",
		"--one-file-system",
	}
	f, err := os.Open(filepath.Join(configDir, "excludes"))
	if err != nil {
		return base
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if line := strings.TrimSpace(s.Text()); line != "" && !strings.HasPrefix(line, "#") {
			base = append(base, "--exclude", line)
		}
	}
	return base
}()

var resticBin = func() string {
	if exe, err := os.Executable(); err == nil {
		local := filepath.Join(filepath.Dir(exe), "restic")
		if _, err := os.Stat(local); err == nil {
			return local
		}
	}
	return "restic"
}()

func resticCmd(repo string, args ...string) *exec.Cmd {
	full := append([]string{"--repo", repo, "--password-file", passFile}, args...)
	return exec.Command(resticBin, full...)
}

func resticBackup(ws string) (string, error) {
	args := append([]string{"backup", ws, "--json"}, defaultExcludes...)
	out, err := resticCmd(repoFor(ws), args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("backup failed: %s", strings.TrimSpace(string(out)))
	}
	for _, line := range strings.Split(string(out), "\n") {
		var msg struct {
			Type string `json:"message_type"`
			ID   string `json:"snapshot_id"`
		}
		if json.Unmarshal([]byte(line), &msg) == nil && msg.Type == "summary" {
			id := msg.ID
			if len(id) > 8 {
				id = id[:8]
			}
			return id, nil
		}
	}
	return "ok", nil
}

func resticSnapshots(ws string) ([]Snapshot, error) {
	out, err := resticCmd(repoFor(ws), "snapshots", "--json").Output()
	if err != nil {
		return nil, err
	}
	var snaps []Snapshot
	json.Unmarshal(out, &snaps)
	for i, j := 0, len(snaps)-1; i < j; i, j = i+1, j-1 {
		snaps[i], snaps[j] = snaps[j], snaps[i]
	}
	return snaps, nil
}

func resticRestore(ws, snapID, path string) error {
	out, err := resticCmd(repoFor(ws), "restore", snapID,
		"--target", "/tmp/restic-restore",
		"--include", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func resticDump(ws, path string) (string, error) {
	out, err := resticCmd(repoFor(ws), "dump", "latest", filepath.Join(ws, path)).Output()
	if err != nil {
		return "", fmt.Errorf("file not found in snapshot")
	}
	return string(out), nil
}

func resticInit(repo string) error {
	return exec.Command(resticBin, "init", "--repo", repo, "--password-file", passFile).Run()
}
