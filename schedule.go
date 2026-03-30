package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var (
	scheduleFile = filepath.Join(configDir, "schedule")
	plistPath    = filepath.Join(homeDir, "Library", "LaunchAgents", "dev.sunneydev.snapshot.plist")
)

var intervalOptions = []string{"10m", "30m", "1h", "6h"}

func isAutoEnabled() bool {
	data, err := os.ReadFile(scheduleFile)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) != ""
}

func getScheduleInterval() string {
	data, err := os.ReadFile(scheduleFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func enableAuto(interval string) error {
	bin, err := os.Executable()
	if err != nil {
		bin = "snapshot"
	}

	if err := os.MkdirAll(filepath.Dir(scheduleFile), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(scheduleFile, []byte(interval+"\n"), 0644); err != nil {
		return err
	}

	if runtime.GOOS == "darwin" {
		return setupLaunchd(bin, intervalToSeconds(interval))
	}
	return setupCron(bin, intervalToCron(interval))
}

func disableAuto() error {
	os.Remove(scheduleFile)
	if runtime.GOOS == "darwin" {
		exec.Command("launchctl", "unload", plistPath).Run()
		os.Remove(plistPath)
		return nil
	}
	return removeCron()
}

func autoStatus() string {
	interval := getScheduleInterval()
	if interval == "" {
		return "off"
	}
	return "on, every " + interval
}

func intervalToSeconds(interval string) int {
	switch interval {
	case "10m":
		return 600
	case "30m":
		return 1800
	case "1h":
		return 3600
	case "6h":
		return 21600
	default:
		n, _ := strconv.Atoi(strings.TrimSuffix(interval, "m"))
		if n > 0 {
			return n * 60
		}
		return 1800
	}
}

func intervalToCron(interval string) string {
	switch interval {
	case "10m":
		return "*/10 * * * *"
	case "30m":
		return "*/30 * * * *"
	case "1h":
		return "0 * * * *"
	case "6h":
		return "0 */6 * * *"
	default:
		return "*/30 * * * *"
	}
}

func setupLaunchd(bin string, seconds int) error {
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>dev.sunneydev.snapshot</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>save</string>
    </array>
    <key>StartInterval</key>
    <integer>%d</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/snapshot.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/snapshot.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin</string>
    </dict>
</dict>
</plist>`, bin, seconds)

	os.MkdirAll(filepath.Dir(plistPath), 0755)
	exec.Command("launchctl", "unload", plistPath).Run()
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}
	return exec.Command("launchctl", "load", plistPath).Run()
}

func setupCron(bin, schedule string) error {
	out, _ := exec.Command("crontab", "-l").Output()
	lines := strings.Split(string(out), "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, "# snapshot-auto") {
			filtered = append(filtered, line)
		}
	}
	entry := fmt.Sprintf("%s %s save >> /tmp/snapshot.log 2>&1 # snapshot-auto", schedule, bin)
	filtered = append(filtered, entry)
	content := strings.Join(filtered, "\n")
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}

func removeCron() error {
	out, _ := exec.Command("crontab", "-l").Output()
	lines := strings.Split(string(out), "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, "# snapshot-auto") {
			filtered = append(filtered, line)
		}
	}
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(strings.Join(filtered, "\n"))
	return cmd.Run()
}
