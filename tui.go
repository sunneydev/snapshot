package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenMenu screen = iota
	screenSave
	screenList
	screenRestorePath
	screenRestorePick
	screenRestoreDiff
	screenDiffPath
	screenDiffView
	screenWs
	screenAdd
	screenRm
	screenResult
)

type WsInfo struct {
	Path      string
	SnapCount int
	LastSnap  time.Time
}

type saveMsg struct {
	id  string
	err error
}

type snapshotsMsg struct {
	snaps []Snapshot
	err   error
}

type restoreDoneMsg struct {
	diff string
	err  error
}

type diffMsg struct {
	content string
	err     error
}

type addDoneMsg struct {
	path string
	err  error
}

type rmDoneMsg struct{ err error }
type wsInfoMsg struct{ info []WsInfo }

var menuItems = [][2]string{
	{"save", "create a snapshot"},
	{"list", "browse snapshots"},
	{"restore", "recover a file"},
	{"diff", "compare with snapshot"},
	{"workspaces", "manage workspaces"},
}

type model struct {
	screen  screen
	width   int
	height  int
	loading bool

	workspaces  []string
	workspace   string
	menuCursor  int
	spinner     spinner.Model
	input       textinput.Model
	vp          viewport.Model
	snapTable   table.Model
	snapshots   []Snapshot
	snapCursor  int
	restorePath string
	wsInfo      []WsInfo
	wsCursor    int
	result      string
	err         error
}

func newTUI() model {
	workspaces := loadWorkspaces()
	ws, _ := resolveWorkspace("")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = accentStyle

	ti := textinput.New()
	ti.CharLimit = 256

	m := model{
		workspaces: workspaces,
		workspace:  ws,
		spinner:    s,
		input:      ti,
	}

	if len(workspaces) == 0 {
		m.screen = screenAdd
		m.input.Placeholder = "workspace path (e.g. ~/work)"
		m.input.Focus()
	} else {
		if ws == "" {
			m.workspace = workspaces[0]
		}
		m.screen = screenMenu
	}

	return m
}

func (m model) Init() tea.Cmd { return m.spinner.Tick }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenSave:
		return m.updateSave(msg)
	case screenList:
		return m.updateList(msg)
	case screenRestorePath:
		return m.updateRestorePath(msg)
	case screenRestorePick:
		return m.updateRestorePick(msg)
	case screenRestoreDiff:
		return m.updateRestoreDiff(msg)
	case screenDiffPath:
		return m.updateDiffPath(msg)
	case screenDiffView:
		return m.updateDiffView(msg)
	case screenWs:
		return m.updateWs(msg)
	case screenAdd:
		return m.updateAdd(msg)
	case screenRm:
		return m.updateRm(msg)
	case screenResult:
		if _, ok := msg.(tea.KeyMsg); ok {
			return m.toMenu()
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenMenu:
		return m.viewMenu()
	case screenSave:
		return m.viewSave()
	case screenList:
		return m.viewList()
	case screenRestorePath:
		return m.viewInput("snapshot · restore", "file or directory to restore:")
	case screenRestorePick:
		return m.viewRestorePick()
	case screenRestoreDiff:
		return m.viewRestoreDiff()
	case screenDiffPath:
		return m.viewInput("snapshot · diff", "file to compare:")
	case screenDiffView:
		return m.viewDiffView()
	case screenWs:
		return m.viewWs()
	case screenAdd:
		if m.loading {
			return page("snapshot · add", "", "  "+m.spinner.View()+" initializing...", "")
		}
		return m.viewInput("snapshot · add workspace", "path:")
	case screenRm:
		return m.viewRm()
	case screenResult:
		return m.viewResult()
	}
	return ""
}

func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "down", "j":
		if m.menuCursor < len(menuItems)-1 {
			m.menuCursor++
		}
	case "enter":
		switch m.menuCursor {
		case 0:
			m.screen = screenSave
			m.loading = true
			return m, doSave(m.workspace)
		case 1:
			m.screen = screenList
			m.loading = true
			return m, doListSnaps(m.workspace)
		case 2:
			m.screen = screenRestorePath
			m.input.Placeholder = "path relative to " + shortenHome(m.workspace)
			m.input.SetValue("")
			m.input.Focus()
		case 3:
			m.screen = screenDiffPath
			m.input.Placeholder = "path relative to " + shortenHome(m.workspace)
			m.input.SetValue("")
			m.input.Focus()
		case 4:
			m.screen = screenWs
			m.wsInfo = make([]WsInfo, len(m.workspaces))
			for i, ws := range m.workspaces {
				m.wsInfo[i] = WsInfo{Path: ws}
			}
			return m, doLoadWsInfo()
		}
	}
	return m, nil
}

func (m model) viewMenu() string {
	var b strings.Builder
	for i, item := range menuItems {
		prefix := "    "
		style := dimStyle
		if i == m.menuCursor {
			prefix = "  " + accentStyle.Render(">") + " "
			style = selectedStyle
		}
		fmt.Fprintf(&b, "%s%s%s\n", prefix, style.Render(fmt.Sprintf("%-14s", item[0])), dimStyle.Render(item[1]))
	}
	return page("snapshot", shortenHome(m.workspace), b.String(), "↑/↓ navigate · enter select · q quit")
}

func (m model) updateSave(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case saveMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.result = msg.id
		}
	case tea.KeyMsg:
		if !m.loading {
			return m.toMenu()
		}
	}
	return m, nil
}

func (m model) viewSave() string {
	if m.loading {
		return page("snapshot · save", "", "  "+m.spinner.View()+" saving "+shortenHome(m.workspace)+"...", "")
	}
	if m.err != nil {
		return page("snapshot · save", "", "  "+errorStyle.Render(m.err.Error()), "press any key")
	}
	return page("snapshot · save", "", "  "+successStyle.Render("✓")+" snapshot "+accentStyle.Render(m.result)+" saved", "press any key")
}

func (m model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case snapshotsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.snapshots = msg.snaps
		m.snapTable = makeSnapTable(msg.snaps)
	case tea.KeyMsg:
		if !m.loading {
			if msg.String() == "esc" || msg.String() == "q" {
				return m.toMenu()
			}
			var cmd tea.Cmd
			m.snapTable, cmd = m.snapTable.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) viewList() string {
	if m.loading {
		return page("snapshot · list", "", "  "+m.spinner.View()+" loading...", "")
	}
	if m.err != nil {
		return page("snapshot · list", "", "  "+errorStyle.Render(m.err.Error()), "esc back")
	}
	if len(m.snapshots) == 0 {
		return page("snapshot · list", shortenHome(m.workspace), "  "+dimStyle.Render("no snapshots"), "esc back")
	}
	return page("snapshot · list", shortenHome(m.workspace), m.snapTable.View(), "↑/↓ navigate · esc back")
}

func (m model) updateRestorePath(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.Type {
		case tea.KeyEsc:
			return m.toMenu()
		case tea.KeyEnter:
			val := strings.TrimSpace(m.input.Value())
			if val == "" {
				return m, nil
			}
			m.restorePath = val
			m.screen = screenRestorePick
			m.loading = true
			m.input.Blur()
			return m, doListSnaps(m.workspace)
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateRestorePick(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case snapshotsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.screen = screenResult
			return m, nil
		}
		m.snapshots = msg.snaps
		m.snapCursor = 0
	case tea.KeyMsg:
		if !m.loading {
			switch msg.String() {
			case "esc":
				return m.toMenu()
			case "up", "k":
				if m.snapCursor > 0 {
					m.snapCursor--
				}
			case "down", "j":
				if m.snapCursor < len(m.snapshots)-1 {
					m.snapCursor++
				}
			case "enter":
				if len(m.snapshots) > 0 {
					m.screen = screenRestoreDiff
					m.loading = true
					return m, doRestore(m.workspace, m.snapshots[m.snapCursor].ID, m.restorePath)
				}
			}
		}
	}
	return m, nil
}

func (m model) viewRestorePick() string {
	if m.loading {
		return page("snapshot · restore", "", "  "+m.spinner.View()+" loading snapshots...", "")
	}
	if len(m.snapshots) == 0 {
		return page("snapshot · restore", "", "  "+dimStyle.Render("no snapshots"), "esc back")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "  select snapshot for: %s\n\n", accentStyle.Render(m.restorePath))
	for i, snap := range m.snapshots {
		prefix := "    "
		style := lipgloss.NewStyle()
		if i == m.snapCursor {
			prefix = "  " + accentStyle.Render(">") + " "
			style = selectedStyle
		}
		fmt.Fprintf(&b, "%s%s  %s\n", prefix, style.Render(snap.ID), dimStyle.Render(snap.Time.Format("2006-01-02 15:04")))
	}
	return page("snapshot · restore", "", b.String(), "↑/↓ navigate · enter select · esc back")
}

func (m model) updateRestoreDiff(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case restoreDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.screen = screenResult
			return m, nil
		}
		m.vp = viewport.New(max(m.width-6, 40), max(m.height-10, 10))
		content := dimStyle.Render("files are identical")
		if msg.diff != "" {
			content = msg.diff
		}
		m.vp.SetContent(content)
	case tea.KeyMsg:
		if !m.loading {
			switch msg.String() {
			case "y":
				src := filepath.Join("/tmp/restic-restore", m.workspace, m.restorePath)
				dst := filepath.Join(m.workspace, m.restorePath)
				info, statErr := os.Stat(src)
				if statErr != nil {
					m.err = statErr
					m.screen = screenResult
					return m, nil
				}
				if info.IsDir() {
					exec.Command("cp", "-R", src+"/", dst+"/").Run()
				} else {
					exec.Command("cp", src, dst).Run()
				}
				m.result = "restored " + m.restorePath
				m.screen = screenResult
				return m, nil
			case "n", "esc":
				return m.toMenu()
			default:
				var cmd tea.Cmd
				m.vp, cmd = m.vp.Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m model) viewRestoreDiff() string {
	if m.loading {
		return page("snapshot · restore", "", "  "+m.spinner.View()+" restoring...", "")
	}
	return page("snapshot · restore", m.restorePath, "  "+m.vp.View(), "y restore · n cancel · ↑/↓ scroll")
}

func (m model) updateDiffPath(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.Type {
		case tea.KeyEsc:
			return m.toMenu()
		case tea.KeyEnter:
			val := strings.TrimSpace(m.input.Value())
			if val == "" {
				return m, nil
			}
			m.screen = screenDiffView
			m.loading = true
			m.input.Blur()
			return m, doDiff(m.workspace, val)
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateDiffView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case diffMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.screen = screenResult
			return m, nil
		}
		m.vp = viewport.New(max(m.width-6, 40), max(m.height-10, 10))
		content := dimStyle.Render("files are identical")
		if msg.content != "" {
			content = msg.content
		}
		m.vp.SetContent(content)
	case tea.KeyMsg:
		if !m.loading {
			if msg.String() == "esc" || msg.String() == "q" {
				return m.toMenu()
			}
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) viewDiffView() string {
	if m.loading {
		return page("snapshot · diff", "", "  "+m.spinner.View()+" loading...", "")
	}
	return page("snapshot · diff", "", "  "+m.vp.View(), "↑/↓ scroll · esc back")
}

func (m model) updateWs(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case wsInfoMsg:
		m.wsInfo = msg.info
	case tea.KeyMsg:
		if !m.loading {
			switch msg.String() {
			case "esc", "q":
				return m.toMenu()
			case "a":
				m.screen = screenAdd
				m.input.Placeholder = "workspace path (e.g. ~/projects/app)"
				m.input.SetValue("")
				m.input.Focus()
			case "d":
				if len(m.wsInfo) > 0 {
					m.screen = screenRm
					m.wsCursor = 0
				}
			}
		}
	}
	return m, nil
}

func (m model) viewWs() string {
	if len(m.wsInfo) == 0 {
		return page("snapshot · workspaces", "", "  "+dimStyle.Render("no workspaces"), "a add · esc back")
	}
	var b strings.Builder
	for _, ws := range m.wsInfo {
		name := accentStyle.Render(fmt.Sprintf("%-35s", shortenHome(ws.Path)))
		info := dimStyle.Render("no snapshots")
		if ws.SnapCount > 0 {
			info = dimStyle.Render(fmt.Sprintf("%d snapshots, last: %s", ws.SnapCount, timeAgo(ws.LastSnap)))
		}
		fmt.Fprintf(&b, "  %s%s\n", name, info)
	}
	return page("snapshot · workspaces", "", b.String(), "a add · d remove · esc back")
}

func (m model) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case addDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.result = "added " + shortenHome(msg.path)
			m.workspaces = loadWorkspaces()
			if m.workspace == "" {
				m.workspace = msg.path
			}
		}
		m.screen = screenResult
		return m, nil
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyEsc:
			return m.toMenu()
		case tea.KeyEnter:
			val := strings.TrimSpace(m.input.Value())
			if val == "" {
				return m, nil
			}
			m.loading = true
			m.input.Blur()
			return m, doAdd(val)
		}
	}
	if !m.loading {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updateRm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case rmDoneMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.result = "removed " + shortenHome(m.wsInfo[m.wsCursor].Path)
			m.workspaces = loadWorkspaces()
			if len(m.workspaces) > 0 {
				m.workspace = m.workspaces[0]
			}
		}
		m.screen = screenResult
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			return m, doRm(m.wsInfo[m.wsCursor].Path)
		case "n", "esc":
			m.screen = screenWs
		}
	}
	return m, nil
}

func (m model) viewRm() string {
	if len(m.wsInfo) == 0 {
		return page("snapshot · remove", "", "  "+dimStyle.Render("no workspaces"), "esc back")
	}
	return page("snapshot · remove", "", "  remove "+accentStyle.Render(shortenHome(m.wsInfo[m.wsCursor].Path))+"?", "y confirm · n cancel")
}

func (m model) viewResult() string {
	content := "  " + successStyle.Render("✓") + " " + m.result
	if m.err != nil {
		content = "  " + errorStyle.Render(m.err.Error())
	}
	return page("snapshot", "", content, "press any key")
}

func (m model) viewInput(title, label string) string {
	sub := ""
	if m.workspace != "" {
		sub = shortenHome(m.workspace)
	}
	return page(title, sub, "  "+label+"\n  "+accentStyle.Render("> ")+m.input.View(), "enter continue · esc back")
}

func (m model) toMenu() (tea.Model, tea.Cmd) {
	m.screen = screenMenu
	m.loading = false
	m.result = ""
	m.err = nil
	m.workspaces = loadWorkspaces()
	if m.workspace == "" && len(m.workspaces) > 0 {
		m.workspace = m.workspaces[0]
	}
	return m, nil
}

func page(title, subtitle, content, help string) string {
	var b strings.Builder
	b.WriteString("\n  " + titleStyle.Render(title) + "\n")
	if subtitle != "" {
		b.WriteString("  " + dimStyle.Render(subtitle) + "\n")
	}
	b.WriteString("\n" + content + "\n")
	if help != "" {
		b.WriteString("\n  " + helpStyle.Render(help) + "\n")
	}
	return b.String()
}

func makeSnapTable(snaps []Snapshot) table.Model {
	cols := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "Time", Width: 20},
	}
	rows := make([]table.Row, len(snaps))
	for i, s := range snaps {
		rows[i] = table.Row{s.ID, s.Time.Format("2006-01-02 15:04")}
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(snaps) + 1),
	)
	st := table.DefaultStyles()
	st.Header = st.Header.Foreground(lipgloss.Color("8")).Bold(true)
	st.Selected = st.Selected.Foreground(lipgloss.Color("12"))
	t.SetStyles(st)
	return t
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func doSave(ws string) tea.Cmd {
	return func() tea.Msg {
		id, err := resticBackup(ws)
		return saveMsg{id, err}
	}
}

func doListSnaps(ws string) tea.Cmd {
	return func() tea.Msg {
		snaps, err := resticSnapshots(ws)
		return snapshotsMsg{snaps, err}
	}
}

func doRestore(ws, snapID, path string) tea.Cmd {
	return func() tea.Msg {
		os.RemoveAll("/tmp/restic-restore")
		if err := resticRestore(ws, snapID, path); err != nil {
			return restoreDoneMsg{"", err}
		}
		out, _ := exec.Command("diff", "-u",
			"--label", "snapshot", "--label", "current",
			filepath.Join("/tmp/restic-restore", ws, path),
			filepath.Join(ws, path)).CombinedOutput()
		return restoreDoneMsg{string(out), nil}
	}
}

func doDiff(ws, path string) tea.Cmd {
	return func() tea.Msg {
		content, err := resticDump(ws, path)
		if err != nil {
			return diffMsg{"", err}
		}
		tmp := "/tmp/restic-dump-file"
		os.WriteFile(tmp, []byte(content), 0644)
		defer os.Remove(tmp)
		out, _ := exec.Command("diff", "-u",
			"--label", "snapshot", "--label", "current",
			tmp, filepath.Join(ws, path)).CombinedOutput()
		return diffMsg{string(out), nil}
	}
}

func doLoadWsInfo() tea.Cmd {
	return func() tea.Msg {
		workspaces := loadWorkspaces()
		info := make([]WsInfo, len(workspaces))
		var wg sync.WaitGroup
		for i, ws := range workspaces {
			info[i] = WsInfo{Path: ws}
			wg.Add(1)
			go func(idx int, w string) {
				defer wg.Done()
				if snaps, err := resticSnapshots(w); err == nil {
					info[idx].SnapCount = len(snaps)
					if len(snaps) > 0 {
						info[idx].LastSnap = snaps[0].Time
					}
				}
			}(i, ws)
		}
		wg.Wait()
		return wsInfoMsg{info}
	}
}

func doAdd(val string) tea.Cmd {
	return func() tea.Msg {
		abs, err := filepath.EvalSymlinks(expandHome(val))
		if err != nil {
			return addDoneMsg{"", fmt.Errorf("path not found: %s", val)}
		}
		if err := addWorkspace(abs); err != nil {
			return addDoneMsg{"", err}
		}
		repo := repoFor(abs)
		if _, err := os.Stat(repo); os.IsNotExist(err) {
			if initErr := resticInit(repo); initErr != nil {
				return addDoneMsg{"", fmt.Errorf("failed to init repo")}
			}
		}
		return addDoneMsg{abs, nil}
	}
}

func doRm(path string) tea.Cmd {
	return func() tea.Msg {
		return rmDoneMsg{removeWorkspace(path)}
	}
}
