package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sftppkg "github.com/candratama/sshm/internal/sftp"
)

type Pane int

const (
	PaneLocal Pane = iota
	PaneRemote
)

type SftpQuitMsg struct{}
type SftpErrorMsg struct{ Err error }
type SftpRefreshMsg struct{}

type SftpModel struct {
	Client         *sftppkg.Client
	LocalDir       string
	RemoteDir      string
	LocalEntries   []sftppkg.Entry
	RemoteEntries  []sftppkg.Entry
	LocalCursor    int
	RemoteCursor   int
	LocalScroll    int
	RemoteScroll   int
	LocalSelected  map[string]bool
	RemoteSelected map[string]bool
	LocalFilter    string
	RemoteFilter   string
	Filtering      bool
	ShowHidden     bool
	Active         Pane
	Width          int
	Height         int
	Err            string
	Info           string
}

func NewSftpModel(client *sftppkg.Client, localDir, remoteDir string) SftpModel {
	m := SftpModel{
		Client:         client,
		LocalDir:       localDir,
		RemoteDir:      remoteDir,
		LocalSelected:  map[string]bool{},
		RemoteSelected: map[string]bool{},
		Active:         PaneLocal,
		Width:          80,
		Height:         24,
	}
	m.refreshLocal()
	m.refreshRemote()
	return m
}

func (m *SftpModel) refreshLocal() {
	infos, err := os.ReadDir(m.LocalDir)
	if err != nil {
		m.Err = err.Error()
		m.LocalEntries = nil
		return
	}
	entries := make([]sftppkg.Entry, 0, len(infos))
	for _, fi := range infos {
		info, _ := fi.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		entries = append(entries, sftppkg.Entry{Name: fi.Name(), IsDir: fi.IsDir(), Size: size})
	}
	sortEntries(entries)
	m.LocalEntries = entries
	m.LocalSelected = map[string]bool{}
	m.LocalCursor = 0
	m.LocalScroll = 0
}

func (m *SftpModel) refreshRemote() {
	if m.Client == nil {
		return
	}
	entries, err := m.Client.List(m.RemoteDir)
	if err != nil {
		m.Err = err.Error()
		m.RemoteEntries = nil
		return
	}
	sortEntries(entries)
	m.RemoteEntries = entries
	m.RemoteSelected = map[string]bool{}
	m.RemoteCursor = 0
	m.RemoteScroll = 0
}

func sortEntries(entries []sftppkg.Entry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
}

func (m SftpModel) visible(pane Pane) []sftppkg.Entry {
	var all []sftppkg.Entry
	var filter string
	if pane == PaneLocal {
		all = m.LocalEntries
		filter = m.LocalFilter
	} else {
		all = m.RemoteEntries
		filter = m.RemoteFilter
	}
	out := make([]sftppkg.Entry, 0, len(all))
	q := strings.ToLower(filter)
	for _, e := range all {
		if !m.ShowHidden && strings.HasPrefix(e.Name, ".") {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(e.Name), q) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (m SftpModel) activeFilter() string {
	if m.Active == PaneLocal {
		return m.LocalFilter
	}
	return m.RemoteFilter
}

func (m SftpModel) Init() tea.Cmd { return nil }

func (m SftpModel) paneBodyHeight() int {
	h := m.Height - 6
	if h < 3 {
		h = 3
	}
	return h
}

func (m *SftpModel) clampScroll() {
	body := m.paneBodyHeight()
	vl := len(m.visible(PaneLocal))
	vr := len(m.visible(PaneRemote))
	if m.LocalCursor >= vl {
		m.LocalCursor = vl - 1
	}
	if m.LocalCursor < 0 {
		m.LocalCursor = 0
	}
	if m.RemoteCursor >= vr {
		m.RemoteCursor = vr - 1
	}
	if m.RemoteCursor < 0 {
		m.RemoteCursor = 0
	}
	if m.LocalCursor < m.LocalScroll {
		m.LocalScroll = m.LocalCursor
	} else if m.LocalCursor >= m.LocalScroll+body {
		m.LocalScroll = m.LocalCursor - body + 1
	}
	if m.LocalScroll < 0 {
		m.LocalScroll = 0
	}
	if m.RemoteCursor < m.RemoteScroll {
		m.RemoteScroll = m.RemoteCursor
	} else if m.RemoteCursor >= m.RemoteScroll+body {
		m.RemoteScroll = m.RemoteCursor - body + 1
	}
	if m.RemoteScroll < 0 {
		m.RemoteScroll = 0
	}
}

func (m *SftpModel) appendFilter(s string) {
	if m.Active == PaneLocal {
		m.LocalFilter += s
		m.LocalCursor = 0
		m.LocalScroll = 0
	} else {
		m.RemoteFilter += s
		m.RemoteCursor = 0
		m.RemoteScroll = 0
	}
}

func (m *SftpModel) popFilter() {
	if m.Active == PaneLocal {
		if len(m.LocalFilter) > 0 {
			m.LocalFilter = m.LocalFilter[:len(m.LocalFilter)-1]
		}
	} else {
		if len(m.RemoteFilter) > 0 {
			m.RemoteFilter = m.RemoteFilter[:len(m.RemoteFilter)-1]
		}
	}
}

func (m *SftpModel) clearFilter() {
	if m.Active == PaneLocal {
		m.LocalFilter = ""
	} else {
		m.RemoteFilter = ""
	}
	m.LocalCursor = 0
	m.RemoteCursor = 0
	m.LocalScroll = 0
	m.RemoteScroll = 0
}

func (m SftpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.clampScroll()
		return m, nil
	case SftpRefreshMsg:
		m.refreshLocal()
		m.refreshRemote()
		return m, nil
	case tea.KeyMsg:
		if m.Filtering {
			switch msg.Type {
			case tea.KeyEsc:
				m.clearFilter()
				m.Filtering = false
			case tea.KeyEnter:
				m.Filtering = false
			case tea.KeyBackspace:
				m.popFilter()
			case tea.KeyRunes:
				m.appendFilter(string(msg.Runes))
			case tea.KeySpace:
				m.appendFilter(" ")
			}
			m.clampScroll()
			return m, nil
		}
		switch msg.Type {
		case tea.KeyTab:
			if m.Active == PaneLocal {
				m.Active = PaneRemote
			} else {
				m.Active = PaneLocal
			}
		case tea.KeyUp:
			if m.Active == PaneLocal && m.LocalCursor > 0 {
				m.LocalCursor--
			} else if m.Active == PaneRemote && m.RemoteCursor > 0 {
				m.RemoteCursor--
			}
			m.clampScroll()
		case tea.KeyDown:
			vis := m.visible(m.Active)
			if m.Active == PaneLocal && m.LocalCursor < len(vis)-1 {
				m.LocalCursor++
			} else if m.Active == PaneRemote && m.RemoteCursor < len(vis)-1 {
				m.RemoteCursor++
			}
			m.clampScroll()
		case tea.KeyPgUp:
			body := m.paneBodyHeight()
			if m.Active == PaneLocal {
				m.LocalCursor -= body
			} else {
				m.RemoteCursor -= body
			}
			m.clampScroll()
		case tea.KeyPgDown:
			body := m.paneBodyHeight()
			if m.Active == PaneLocal {
				m.LocalCursor += body
			} else {
				m.RemoteCursor += body
			}
			m.clampScroll()
		case tea.KeyHome:
			if m.Active == PaneLocal {
				m.LocalCursor = 0
			} else {
				m.RemoteCursor = 0
			}
			m.clampScroll()
		case tea.KeyEnd:
			vis := m.visible(m.Active)
			if m.Active == PaneLocal {
				m.LocalCursor = len(vis) - 1
			} else {
				m.RemoteCursor = len(vis) - 1
			}
			m.clampScroll()
		case tea.KeyEnter:
			m.descend()
			m.clampScroll()
		case tea.KeyBackspace:
			m.ascend()
			m.clampScroll()
		case tea.KeySpace:
			m.toggleSelect()
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				return m, func() tea.Msg { return SftpQuitMsg{} }
			case "c":
				m.copy()
			case "d":
				m.delete()
			case "r":
				m.refreshLocal()
				m.refreshRemote()
			case ".":
				m.ShowHidden = !m.ShowHidden
				m.LocalCursor = 0
				m.RemoteCursor = 0
				m.LocalScroll = 0
				m.RemoteScroll = 0
			case "/":
				m.Filtering = true
				m.clearFilter()
			case "a":
				m.selectAll()
			case "A":
				m.clearSelection()
			}
		}
	}
	return m, nil
}

func (m *SftpModel) selectedAt(pane Pane) map[string]bool {
	if pane == PaneLocal {
		return m.LocalSelected
	}
	return m.RemoteSelected
}

func (m *SftpModel) toggleSelect() {
	vis := m.visible(m.Active)
	var cursor int
	if m.Active == PaneLocal {
		cursor = m.LocalCursor
	} else {
		cursor = m.RemoteCursor
	}
	if cursor < 0 || cursor >= len(vis) {
		return
	}
	e := vis[cursor]
	if e.IsDir {
		return
	}
	sel := m.selectedAt(m.Active)
	if sel[e.Name] {
		delete(sel, e.Name)
	} else {
		sel[e.Name] = true
	}
}

func (m *SftpModel) selectAll() {
	vis := m.visible(m.Active)
	sel := m.selectedAt(m.Active)
	for _, e := range vis {
		if !e.IsDir {
			sel[e.Name] = true
		}
	}
}

func (m *SftpModel) clearSelection() {
	if m.Active == PaneLocal {
		m.LocalSelected = map[string]bool{}
	} else {
		m.RemoteSelected = map[string]bool{}
	}
}

func (m *SftpModel) descend() {
	vis := m.visible(m.Active)
	if m.Active == PaneLocal {
		if m.LocalCursor >= len(vis) {
			return
		}
		e := vis[m.LocalCursor]
		if !e.IsDir {
			return
		}
		m.LocalDir = filepath.Join(m.LocalDir, e.Name)
		m.LocalFilter = ""
		m.refreshLocal()
	} else {
		if m.RemoteCursor >= len(vis) {
			return
		}
		e := vis[m.RemoteCursor]
		if !e.IsDir {
			return
		}
		m.RemoteDir = sftppkg.Join(m.RemoteDir, e.Name)
		m.RemoteFilter = ""
		m.refreshRemote()
	}
}

func (m *SftpModel) ascend() {
	if m.Active == PaneLocal {
		m.LocalDir = filepath.Dir(m.LocalDir)
		m.LocalFilter = ""
		m.refreshLocal()
	} else {
		m.RemoteDir = sftppkg.Parent(m.RemoteDir)
		m.RemoteFilter = ""
		m.refreshRemote()
	}
}

func (m *SftpModel) copy() {
	if m.Client == nil {
		return
	}
	sel := m.selectedAt(m.Active)
	vis := m.visible(m.Active)

	targets := []sftppkg.Entry{}
	if len(sel) > 0 {
		for _, e := range vis {
			if sel[e.Name] && !e.IsDir {
				targets = append(targets, e)
			}
		}
	} else {
		var cursor int
		if m.Active == PaneLocal {
			cursor = m.LocalCursor
		} else {
			cursor = m.RemoteCursor
		}
		if cursor < 0 || cursor >= len(vis) {
			return
		}
		e := vis[cursor]
		if e.IsDir {
			m.Err = "directory copy not supported"
			return
		}
		targets = append(targets, e)
	}

	ok := 0
	for _, e := range targets {
		if m.Active == PaneLocal {
			src := filepath.Join(m.LocalDir, e.Name)
			dst := sftppkg.Join(m.RemoteDir, e.Name)
			if err := m.Client.Upload(src, dst); err != nil {
				m.Err = err.Error()
				return
			}
		} else {
			src := sftppkg.Join(m.RemoteDir, e.Name)
			dst := filepath.Join(m.LocalDir, e.Name)
			if err := m.Client.Download(src, dst); err != nil {
				m.Err = err.Error()
				return
			}
		}
		ok++
	}
	m.Err = ""
	m.Info = fmt.Sprintf("copied %d file(s)", ok)
	m.clearSelection()
	if m.Active == PaneLocal {
		m.refreshRemote()
	} else {
		m.refreshLocal()
	}
}

func (m *SftpModel) delete() {
	vis := m.visible(m.Active)
	var cursor int
	if m.Active == PaneLocal {
		cursor = m.LocalCursor
	} else {
		cursor = m.RemoteCursor
	}
	if cursor < 0 || cursor >= len(vis) {
		return
	}
	e := vis[cursor]
	if m.Active == PaneLocal {
		target := filepath.Join(m.LocalDir, e.Name)
		if err := os.Remove(target); err != nil {
			m.Err = err.Error()
			return
		}
		m.refreshLocal()
	} else {
		if m.Client == nil {
			return
		}
		target := sftppkg.Join(m.RemoteDir, e.Name)
		if err := m.Client.Delete(target); err != nil {
			m.Err = err.Error()
			return
		}
		m.refreshRemote()
	}
	m.Err = ""
}

func (m SftpModel) View() string {
	paneW := (m.Width - 4) / 2
	if paneW < 24 {
		paneW = 24
	}
	paneH := m.Height - 3
	if paneH < 6 {
		paneH = 6
	}

	leftTitle := "Local: " + truncate(m.LocalDir, paneW-10)
	rightTitle := "Remote: " + truncate(m.RemoteDir, paneW-11)
	left := m.renderPane(leftTitle, m.visible(PaneLocal), m.LocalCursor, m.LocalScroll, m.LocalSelected, m.LocalFilter, m.Active == PaneLocal, paneW, paneH)
	right := m.renderPane(rightTitle, m.visible(PaneRemote), m.RemoteCursor, m.RemoteScroll, m.RemoteSelected, m.RemoteFilter, m.Active == PaneRemote, paneW, paneH)
	joined := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	hints := "[Tab] switch  [Enter] open  [Bksp] up  [Space] select  [c] copy  [d] del  [/] find  [.] hidden  [a]/[A] all/clear  [r] refresh  [q] back"
	if m.Filtering {
		hints = fmt.Sprintf("filter (%s): %s_  [Enter] apply  [Esc] cancel", paneLabel(m.Active), m.activeFilter())
	}
	help := StyleHelp.Render(hints)
	if m.Err != "" {
		help = StyleError.Render(m.Err) + "\n" + help
	} else if m.Info != "" {
		help = StyleHelp.Render(m.Info) + "\n" + help
	}
	return joined + "\n" + help
}

func paneLabel(p Pane) string {
	if p == PaneLocal {
		return "local"
	}
	return "remote"
}

func (m SftpModel) renderPane(title string, entries []sftppkg.Entry, cursor, scroll int, selected map[string]bool, filter string, active bool, width, height int) string {
	innerH := height - 2
	body := innerH - 2
	if body < 1 {
		body = 1
	}

	var b strings.Builder
	b.WriteString(StyleTitle.Render(title))
	b.WriteString("\n")

	nameW := width - 8
	if nameW < 8 {
		nameW = 8
	}

	if len(entries) == 0 {
		b.WriteString(StyleHelp.Render("  (empty)"))
		b.WriteString("\n")
	} else {
		end := scroll + body
		if end > len(entries) {
			end = len(entries)
		}
		for i := scroll; i < end; i++ {
			e := entries[i]
			name := e.Name
			if e.IsDir {
				name += "/"
			}
			name = truncate(name, nameW)
			marker := " "
			if selected[e.Name] {
				marker = "*"
			}
			line := fmt.Sprintf(" %s %s", marker, name)
			if i == cursor {
				line = StyleSelected.Render(">" + marker + " " + name)
			} else if selected[e.Name] {
				line = StyleError.Render(line)
			} else {
				line = StyleNormal.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	footer := ""
	if filter != "" {
		footer = fmt.Sprintf("filter: %s", filter)
	}
	if len(entries) > 0 {
		count := fmt.Sprintf("[%d/%d]", cursor+1, len(entries))
		if len(selected) > 0 {
			count += fmt.Sprintf(" sel:%d", len(selected))
		}
		if footer != "" {
			footer = footer + "  " + count
		} else {
			footer = count
		}
	}
	if footer != "" {
		b.WriteString(StyleHelp.Render(truncate(footer, width-4)))
	}

	style := StylePaneInactive
	if active {
		style = StylePaneActive
	}
	return style.Width(width).Height(innerH).Render(b.String())
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return "..." + s[len(s)-(n-3):]
}
