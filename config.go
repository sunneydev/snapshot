package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	homeDir   = os.Getenv("HOME")
	configDir = initConfigDir()
	dataDir   = initDataDir()
	passFile  = filepath.Join(configDir, "password")
	wsFile    = filepath.Join(configDir, "workspaces")
	backupDir = dataDir
)

func initConfigDir() string {
	oldDir := filepath.Join(homeDir, ".config", "restic")
	if _, err := os.Stat(filepath.Join(oldDir, "workspaces")); err == nil {
		return oldDir
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "snapshot")
	}
	return filepath.Join(homeDir, ".config", "snapshot")
}

func initDataDir() string {
	oldDir := filepath.Join(homeDir, "backups")
	if _, err := os.Stat(oldDir); err == nil {
		return oldDir
	}
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "snapshot")
	}
	return filepath.Join(homeDir, ".local", "share", "snapshot")
}

func repoFor(ws string) string {
	name := strings.TrimPrefix(ws, homeDir+"/")
	return filepath.Join(backupDir, strings.ReplaceAll(name, "/", "-"))
}

func loadWorkspaces() []string {
	f, err := os.Open(wsFile)
	if err != nil {
		return nil
	}
	defer f.Close()
	var result []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		if line := strings.TrimSpace(s.Text()); line != "" {
			result = append(result, line)
		}
	}
	return result
}

func resolveWorkspace(explicit string) (string, error) {
	workspaces := loadWorkspaces()
	if len(workspaces) == 0 {
		return "", fmt.Errorf("no workspaces registered")
	}
	if explicit != "" {
		abs, err := filepath.EvalSymlinks(expandHome(explicit))
		if err != nil {
			return "", fmt.Errorf("path not found: %s", explicit)
		}
		for _, ws := range workspaces {
			if ws == abs {
				return abs, nil
			}
		}
		return "", fmt.Errorf("not a registered workspace: %s", abs)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cwd, _ = filepath.EvalSymlinks(cwd)
	for _, ws := range workspaces {
		if strings.HasPrefix(cwd, ws) {
			return ws, nil
		}
	}
	if len(workspaces) == 1 {
		return workspaces[0], nil
	}
	return "", fmt.Errorf("not inside a registered workspace")
}

func addWorkspace(absPath string) error {
	for _, ws := range loadWorkspaces() {
		if ws == absPath {
			return fmt.Errorf("already registered: %s", shortenHome(absPath))
		}
	}
	f, err := os.OpenFile(wsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, absPath)
	return err
}

func removeWorkspace(path string) error {
	workspaces := loadWorkspaces()
	var remaining []string
	found := false
	for _, ws := range workspaces {
		if ws == path {
			found = true
			continue
		}
		remaining = append(remaining, ws)
	}
	if !found {
		return fmt.Errorf("not registered: %s", path)
	}
	content := strings.Join(remaining, "\n")
	if len(remaining) > 0 {
		content += "\n"
	}
	return os.WriteFile(wsFile, []byte(content), 0644)
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

func shortenHome(path string) string {
	if strings.HasPrefix(path, homeDir) {
		return "~" + path[len(homeDir):]
	}
	return path
}
