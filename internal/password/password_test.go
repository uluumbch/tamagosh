package password

import (
	"errors"
	"strings"
	"testing"
)

type fakeRunner struct {
	lastArgs []string
	lastIn   string
	out      string
	err      error
}

func (f *fakeRunner) Run(stdin string, name string, args ...string) ([]byte, error) {
	f.lastArgs = append([]string{name}, args...)
	f.lastIn = stdin
	return []byte(f.out), f.err
}

func TestGet(t *testing.T) {
	r := &fakeRunner{out: "secret\n"}
	p := &Pass{Runner: r}
	got, err := p.Get("ssh/atlantic")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "secret" {
		t.Fatalf("got %q want %q", got, "secret")
	}
	if !equal(r.lastArgs, []string{"pass", "show", "ssh/atlantic"}) {
		t.Fatalf("bad args: %v", r.lastArgs)
	}
}

func TestGetNotFound(t *testing.T) {
	r := &fakeRunner{out: "", err: errors.New("exit status 1")}
	p := &Pass{Runner: r}
	if _, err := p.Get("ssh/missing"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSet(t *testing.T) {
	r := &fakeRunner{}
	p := &Pass{Runner: r}
	if err := p.Set("ssh/atlantic", "hunter2"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !equal(r.lastArgs, []string{"pass", "insert", "-m", "-f", "ssh/atlantic"}) {
		t.Fatalf("bad args: %v", r.lastArgs)
	}
	if strings.TrimSpace(r.lastIn) != "hunter2" {
		t.Fatalf("bad stdin: %q", r.lastIn)
	}
}

func TestDelete(t *testing.T) {
	r := &fakeRunner{}
	p := &Pass{Runner: r}
	if err := p.Delete("ssh/atlantic"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !equal(r.lastArgs, []string{"pass", "rm", "-f", "ssh/atlantic"}) {
		t.Fatalf("bad args: %v", r.lastArgs)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
