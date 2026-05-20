package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/sftp"
)

func TestHitTestEntryMapsClickedRowToEntry(t *testing.T) {
	m := NewSftpModel(nil, "/local", "/remote", nil, "")
	m.Width = 100
	m.Height = 30
	m.LocalDir = "/local"
	m.LocalEntries = []sftp.Entry{
		{Name: "Downloads", IsDir: true},
		{Name: "Library", IsDir: true},
		{Name: "Movies", IsDir: true},
		{Name: "Music", IsDir: true},
	}
	// y=0 border, y=1 title, y=2 blank, y=3 entry 0 (Downloads)
	cases := []struct{ y, want int }{
		{3, 0}, {4, 1}, {5, 2}, {6, 3},
	}
	for _, c := range cases {
		_, idx, ok := m.hitTestEntry(2, c.y)
		if !ok || idx != c.want {
			t.Fatalf("y=%d idx=%d ok=%v want=%d", c.y, idx, ok, c.want)
		}
	}
}

func TestSftpTabSwitchesPane(t *testing.T) {
	m := NewSftpModel(nil, "/local", "/remote", nil, "")
	if m.Active != PaneLocal {
		t.Fatalf("active=%v", m.Active)
	}
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(SftpModel)
	if m.Active != PaneRemote {
		t.Fatalf("active=%v want remote", m.Active)
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(SftpModel)
	if m.Active != PaneLocal {
		t.Fatalf("active=%v want local", m.Active)
	}
}

func TestSftpCursorNavigatesActivePane(t *testing.T) {
	m := NewSftpModel(nil, "/local", "/remote", nil, "")
	m.LocalEntries = []sftp.Entry{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m.RemoteEntries = []sftp.Entry{{Name: "x"}, {Name: "y"}}

	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(SftpModel)
	if m.LocalCursor != 1 || m.RemoteCursor != 0 {
		t.Fatalf("local=%d remote=%d", m.LocalCursor, m.RemoteCursor)
	}

	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = nm.(SftpModel)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = nm.(SftpModel)
	if m.RemoteCursor != 1 {
		t.Fatalf("remote=%d", m.RemoteCursor)
	}
}

func TestSftpQuitEmits(t *testing.T) {
	m := NewSftpModel(nil, "/local", "/remote", nil, "")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if _, ok := cmd().(SftpQuitMsg); !ok {
		t.Fatalf("got %T want SftpQuitMsg", cmd())
	}
}
