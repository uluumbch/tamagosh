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
	Client        *sftppkg.Client
	LocalDir      string
	RemoteDir     string
	LocalEntries  []sftppkg.Entry
	RemoteEntries []sftppkg.Entry
	LocalCursor   int
	RemoteCursor  int
	Active        Pane
	Err           string
}

func NewSftpModel(client *sftppkg.Client, localDir, remoteDir string) SftpModel {
	m := SftpModel{
		Client:    client,
		LocalDir:  localDir,
		RemoteDir: remoteDir,
		Active:    PaneLocal,
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
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	m.LocalEntries = entries
	if m.LocalCursor >= len(entries) {
		m.LocalCursor = 0
	}
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
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})
	m.RemoteEntries = entries
	if m.RemoteCursor >= len(entries) {
		m.RemoteCursor = 0
	}
}

func (m SftpModel) Init() tea.Cmd { return nil }

func (m SftpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SftpRefreshMsg:
		m.refreshLocal()
		m.refreshRemote()
		return m, nil
	case tea.KeyMsg:
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
		case tea.KeyDown:
			if m.Active == PaneLocal && m.LocalCursor < len(m.LocalEntries)-1 {
				m.LocalCursor++
			} else if m.Active == PaneRemote && m.RemoteCursor < len(m.RemoteEntries)-1 {
				m.RemoteCursor++
			}
		case tea.KeyEnter:
			return m, m.descend()
		case tea.KeyBackspace:
			return m, m.ascend()
		case tea.KeyRunes:
			switch string(msg.Runes) {
			case "q":
				return m, func() tea.Msg { return SftpQuitMsg{} }
			case "c":
				return m, m.copy()
			case "d":
				return m, m.delete()
			case "r":
				m.refreshLocal()
				m.refreshRemote()
			}
		}
	}
	return m, nil
}

func (m *SftpModel) descend() tea.Cmd {
	if m.Active == PaneLocal {
		if m.LocalCursor >= len(m.LocalEntries) {
			return nil
		}
		e := m.LocalEntries[m.LocalCursor]
		if !e.IsDir {
			return nil
		}
		m.LocalDir = filepath.Join(m.LocalDir, e.Name)
		m.LocalCursor = 0
		m.refreshLocal()
	} else {
		if m.RemoteCursor >= len(m.RemoteEntries) {
			return nil
		}
		e := m.RemoteEntries[m.RemoteCursor]
		if !e.IsDir {
			return nil
		}
		m.RemoteDir = sftppkg.Join(m.RemoteDir, e.Name)
		m.RemoteCursor = 0
		m.refreshRemote()
	}
	return nil
}

func (m *SftpModel) ascend() tea.Cmd {
	if m.Active == PaneLocal {
		m.LocalDir = filepath.Dir(m.LocalDir)
		m.LocalCursor = 0
		m.refreshLocal()
	} else {
		m.RemoteDir = sftppkg.Parent(m.RemoteDir)
		m.RemoteCursor = 0
		m.refreshRemote()
	}
	return nil
}

func (m *SftpModel) copy() tea.Cmd {
	if m.Client == nil {
		return nil
	}
	if m.Active == PaneLocal {
		if m.LocalCursor >= len(m.LocalEntries) {
			return nil
		}
		e := m.LocalEntries[m.LocalCursor]
		if e.IsDir {
			m.Err = "directory copy not supported"
			return nil
		}
		src := filepath.Join(m.LocalDir, e.Name)
		dst := sftppkg.Join(m.RemoteDir, e.Name)
		if err := m.Client.Upload(src, dst); err != nil {
			m.Err = err.Error()
			return nil
		}
		m.refreshRemote()
	} else {
		if m.RemoteCursor >= len(m.RemoteEntries) {
			return nil
		}
		e := m.RemoteEntries[m.RemoteCursor]
		if e.IsDir {
			m.Err = "directory copy not supported"
			return nil
		}
		src := sftppkg.Join(m.RemoteDir, e.Name)
		dst := filepath.Join(m.LocalDir, e.Name)
		if err := m.Client.Download(src, dst); err != nil {
			m.Err = err.Error()
			return nil
		}
		m.refreshLocal()
	}
	m.Err = ""
	return nil
}

func (m *SftpModel) delete() tea.Cmd {
	if m.Active == PaneLocal {
		if m.LocalCursor >= len(m.LocalEntries) {
			return nil
		}
		e := m.LocalEntries[m.LocalCursor]
		target := filepath.Join(m.LocalDir, e.Name)
		var err error
		if e.IsDir {
			err = os.Remove(target)
		} else {
			err = os.Remove(target)
		}
		if err != nil {
			m.Err = err.Error()
			return nil
		}
		m.refreshLocal()
	} else {
		if m.Client == nil || m.RemoteCursor >= len(m.RemoteEntries) {
			return nil
		}
		e := m.RemoteEntries[m.RemoteCursor]
		target := sftppkg.Join(m.RemoteDir, e.Name)
		if err := m.Client.Delete(target); err != nil {
			m.Err = err.Error()
			return nil
		}
		m.refreshRemote()
	}
	m.Err = ""
	return nil
}

func (m SftpModel) View() string {
	left := renderPane("Local: "+m.LocalDir, m.LocalEntries, m.LocalCursor, m.Active == PaneLocal)
	right := renderPane("Remote: "+m.RemoteDir, m.RemoteEntries, m.RemoteCursor, m.Active == PaneRemote)
	joined := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	help := StyleHelp.Render("[Tab] switch  [Enter] open  [Bksp] up  [c] copy  [d] del  [r] refresh  [q] back")
	if m.Err != "" {
		help = StyleError.Render(m.Err) + "\n" + help
	}
	return joined + "\n" + help
}

func renderPane(title string, entries []sftppkg.Entry, cursor int, active bool) string {
	var b strings.Builder
	b.WriteString(StyleTitle.Render(title))
	b.WriteString("\n\n")
	for i, e := range entries {
		name := e.Name
		if e.IsDir {
			name += "/"
		}
		line := fmt.Sprintf("  %s", name)
		if i == cursor {
			line = StyleSelected.Render("> " + name)
		} else {
			line = StyleNormal.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	if len(entries) == 0 {
		b.WriteString(StyleHelp.Render("  (empty)"))
		b.WriteString("\n")
	}
	if active {
		return StylePaneActive.Width(40).Render(b.String())
	}
	return StylePaneInactive.Width(40).Render(b.String())
}
