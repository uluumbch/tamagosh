package ssh

import (
	"strings"
	"testing"

	"github.com/Candratama/tamagosh/internal/config"
)

func TestBuildCommand(t *testing.T) {
	conn := config.Connection{Name: "atlantic", Host: "43.228.213.209", Port: 2255, User: "candra"}
	name, args := BuildCommand(conn, "hunter2")
	if name != "sshpass" {
		t.Fatalf("name=%q", name)
	}
	if len(args) != 8 {
		t.Fatalf("args len=%d want 8 (%v)", len(args), args)
	}
	if args[0] != "-p" || args[1] != "hunter2" {
		t.Fatalf("password arg wrong: %v", args[:2])
	}
	if !strings.HasSuffix(args[2], "ssh") {
		t.Fatalf("ssh binary path doesn't end with 'ssh': %q", args[2])
	}
	if args[3] != "-p" || args[4] != "2255" {
		t.Fatalf("port arg wrong: %v", args[3:5])
	}
	if args[5] != "-o" || args[6] != "StrictHostKeyChecking=accept-new" {
		t.Fatalf("strict host key opt wrong: %v", args[5:7])
	}
	if args[7] != "candra@43.228.213.209" {
		t.Fatalf("user@host wrong: %q", args[7])
	}
}

func TestBuildCommandDefaultPort(t *testing.T) {
	conn := config.Connection{Name: "x", Host: "h", Port: 0, User: "u"}
	_, args := BuildCommand(conn, "p")
	found := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-p" && args[i+1] == "22" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected default port 22 in args: %v", args)
	}
}
