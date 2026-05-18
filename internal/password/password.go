package password

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(stdin string, name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(stdin string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

type Pass struct {
	Runner Runner
}

func New() *Pass {
	return &Pass{Runner: ExecRunner{}}
}

func (p *Pass) Get(key string) (string, error) {
	out, err := p.Runner.Run("", "pass", "show", key)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func (p *Pass) Set(key, value string) error {
	_, err := p.Runner.Run(value+"\n", "pass", "insert", "-m", "-f", key)
	return err
}

func (p *Pass) Delete(key string) error {
	_, err := p.Runner.Run("", "pass", "rm", "-f", key)
	return err
}
