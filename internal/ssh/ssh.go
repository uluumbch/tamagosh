package ssh

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/config"
)

// BuildCommand returns (binary, args) for the SSH invocation.
//
// Password auth: returns ("sshpass", ["-e", ssh, "-p", port, "-o", opt, user@host]).
// Caller MUST set SSHPASS env on the resulting cmd (see ConnectCmd).
//
// Key auth: returns (ssh, ["-i", keypath, "-p", port, "-o", opts..., user@host]).
// If the key has a passphrase, ConnectCmd wires SSH_ASKPASS — that lands in Task 5.
func BuildCommand(c config.Connection) (string, []string) {
	port := c.Port
	if port == 0 {
		port = 22
	}
	sshBin := "ssh"
	if p, err := exec.LookPath("ssh"); err == nil {
		sshBin = p
	}
	target := fmt.Sprintf("%s@%s", c.User, c.Host)
	portStr := fmt.Sprintf("%d", port)

	if c.AuthMethod == "key" {
		args := []string{
			"-i", c.KeyPath,
			"-p", portStr,
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "IdentitiesOnly=yes",
			target,
		}
		return sshBin, args
	}

	args := []string{
		"-e",
		sshBin,
		"-p", portStr,
		"-o", "StrictHostKeyChecking=accept-new",
		target,
	}
	return "sshpass", args
}

type ExitMsg struct {
	Err error
}

// ConnectCmd builds the exec.Cmd with env wired for the auth method:
//   - password: SSHPASS env var (no plaintext in args)
//   - key: just exec for now (passphrase handling lands in Task 5)
//
// `secret` is the password for password auth, passphrase for key auth (empty if no passphrase).
func ConnectCmd(c config.Connection, secret string) tea.Cmd {
	name, args := BuildCommand(c)
	cmd := exec.Command(name, args...)
	env := os.Environ()

	switch c.AuthMethod {
	case "key":
		// Task 5 will wire SSH_ASKPASS for passphrase-protected keys.
		// For now, exec ssh directly; if the key is encrypted ssh will fail or prompt on TTY.
	default:
		env = append(env, "SSHPASS="+secret)
	}

	cmd.Env = env
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return ExitMsg{Err: err}
	})
}
