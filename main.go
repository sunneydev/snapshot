package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--help", "-h":
			printUsage()
			return
		case "--version", "-v":
			fmt.Println("snapshot " + version)
			return
		}
	}

	checkDeps()

	if len(os.Args) < 2 {
		p := tea.NewProgram(newTUI(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fatal(err.Error())
		}
		return
	}
	switch os.Args[1] {
	case "save":
		cmdSave(arg(2))
	case "list":
		cmdList(arg(2))
	case "restore":
		requireArg(2, "snapshot restore <path> [snapshot-id]")
		cmdRestore(os.Args[2], arg(3))
	case "diff":
		requireArg(2, "snapshot diff <path>")
		cmdDiff(os.Args[2])
	case "ws":
		cmdWs()
	case "add":
		requireArg(2, "snapshot add <path>")
		cmdAdd(os.Args[2])
	case "rm":
		requireArg(2, "snapshot rm <path>")
		cmdRm(os.Args[2])
	case "auto":
		cmdAuto(arg(2), arg(3))
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`snapshot - automatic workspace backups with a beautiful tui
https://github.com/sunneydev/snapshot

usage:
  snapshot              interactive TUI
  snapshot save [ws]    create a snapshot
  snapshot list [ws]    list snapshots
  snapshot restore <path> [id]  restore a file
  snapshot diff <path>  compare with last snapshot
  snapshot ws           list workspaces
  snapshot add <path>   register workspace
  snapshot rm <path>    unregister workspace
  snapshot auto [on|off] [interval]  manage automatic backups

flags:
  --help, -h     show this help
  --version, -v  show version
`)
}

func checkDeps() {
	if _, err := os.Stat(resticBin); err == nil {
		return
	}
	if _, err := exec.LookPath("restic"); err != nil {
		fmt.Fprintln(os.Stderr, "missing dependency. run:")
		fmt.Fprintln(os.Stderr, "  brew install restic    (macOS)")
		fmt.Fprintln(os.Stderr, "  apt install restic     (linux)")
		os.Exit(1)
	}
}

func arg(i int) string {
	if i < len(os.Args) {
		return os.Args[i]
	}
	return ""
}

func requireArg(i int, usage string) {
	if i >= len(os.Args) {
		fmt.Fprintf(os.Stderr, "usage: %s\n", usage)
		os.Exit(1)
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, errorStyle.Render(msg))
	os.Exit(1)
}

func cmdSave(wsPath string) {
	ws, err := resolveWorkspace(wsPath)
	if err != nil {
		fatal(err.Error())
	}
	args := append([]string{"backup", ws}, defaultExcludes...)
	cmd := resticCmd(repoFor(ws), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

func cmdList(wsPath string) {
	ws, err := resolveWorkspace(wsPath)
	if err != nil {
		fatal(err.Error())
	}
	cmd := resticCmd(repoFor(ws), "snapshots")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func cmdRestore(path, snapID string) {
	ws, err := resolveWorkspace("")
	if err != nil {
		fatal(err.Error())
	}
	if snapID == "" {
		snapID = "latest"
	}
	os.RemoveAll("/tmp/restic-restore")
	if err := resticRestore(ws, snapID, path); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("restored to: /tmp/restic-restore%s/%s\n", ws, path)
}

func cmdDiff(path string) {
	ws, err := resolveWorkspace("")
	if err != nil {
		fatal(err.Error())
	}
	content, err := resticDump(ws, path)
	if err != nil {
		fatal(err.Error())
	}
	tmp := "/tmp/restic-dump-file"
	os.WriteFile(tmp, []byte(content), 0644)
	defer os.Remove(tmp)
	cmd := exec.Command("diff", "-u", "--label", "snapshot", "--label", "current",
		tmp, filepath.Join(ws, path))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func cmdWs() {
	workspaces := loadWorkspaces()
	if len(workspaces) == 0 {
		fmt.Println(dimStyle.Render("no workspaces registered"))
		return
	}
	for _, ws := range workspaces {
		fmt.Println("  " + shortenHome(ws))
	}
}

func cmdAdd(path string) {
	abs, err := filepath.EvalSymlinks(expandHome(path))
	if err != nil {
		fatal("path not found: " + path)
	}
	if err := addWorkspace(abs); err != nil {
		fatal(err.Error())
	}
	repo := repoFor(abs)
	if _, statErr := os.Stat(repo); os.IsNotExist(statErr) {
		if err := resticInit(repo); err != nil {
			fatal("failed to init repo")
		}
	}
	fmt.Println(successStyle.Render("added: " + shortenHome(abs)))

	if !isAutoEnabled() {
		fmt.Println()
		fmt.Print("  set up automatic backups? [10m/30m/1h/6h/skip] ")
		var choice string
		fmt.Scanln(&choice)
		choice = strings.TrimSpace(strings.ToLower(choice))
		if choice != "" && choice != "skip" && choice != "s" && choice != "n" && choice != "no" {
			if err := enableAuto(choice); err != nil {
				fmt.Fprintln(os.Stderr, dimStyle.Render("  failed to set up automatic backups: "+err.Error()))
				return
			}
			fmt.Println(successStyle.Render("  automatic backups enabled (every " + choice + ")"))
		}
	}
}

func cmdAuto(action, interval string) {
	switch action {
	case "":
		fmt.Println("  automatic backups: " + autoStatus())
	case "on":
		if interval == "" {
			interval = "30m"
		}
		if err := enableAuto(interval); err != nil {
			fatal(err.Error())
		}
		fmt.Println(successStyle.Render("automatic backups enabled (every " + interval + ")"))
	case "off":
		if err := disableAuto(); err != nil {
			fatal(err.Error())
		}
		fmt.Println(successStyle.Render("automatic backups disabled"))
	default:
		fmt.Fprintln(os.Stderr, "usage: snapshot auto [on|off] [10m|30m|1h|6h]")
		os.Exit(1)
	}
}

func cmdRm(path string) {
	abs, _ := filepath.EvalSymlinks(expandHome(path))
	if abs == "" {
		abs = expandHome(path)
	}
	if err := removeWorkspace(abs); err != nil {
		fatal(err.Error())
	}
	fmt.Println(successStyle.Render("removed: " + shortenHome(abs)))
}
