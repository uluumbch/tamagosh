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
	if len(args) != 7 {
		t.Fatalf("args len=%d want 7 (%v)", len(args), args)
	}
	if args[0] != "-e" {
		t.Fatalf("expected -e (env mode), got %q", args[0])
	}
	if !strings.HasSuffix(args[1], "ssh") {
		t.Fatalf("ssh binary path doesn't end with 'ssh': %q", args[1])
	}
	if args[2] != "-p" || args[3] != "2255" {
		t.Fatalf("port arg wrong: %v", args[2:4])
	}
	if args[4] != "-o" || args[5] != "StrictHostKeyChecking=accept-new" {
		t.Fatalf("strict host key opt wrong: %v", args[4:6])
	}
	if args[6] != "candra@43.228.213.209" {
		t.Fatalf("user@host wrong: %q", args[6])
	}
	// password must NOT appear anywhere in args
	for i, a := range args {
		if a == "hunter2" {
			t.Fatalf("password leaked into args[%d]", i)
		}
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
