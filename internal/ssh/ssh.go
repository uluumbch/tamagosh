package ssh

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Candratama/tamagosh/internal/config"
)

// BuildCommand returns the sshpass invocation WITHOUT the password.
// Password is passed via SSHPASS env var (see ConnectCmd) to avoid leaking
// through process listings (`ps aux`).
func BuildCommand(c config.Connection, _ string) (string, []string) {
	port := c.Port
	if port == 0 {
		port = 22
	}
	sshBin := "ssh"
	if p, err := exec.LookPath("ssh"); err == nil {
		sshBin = p
	}
	args := []string{
		"-e",
		sshBin,
		"-p", fmt.Sprintf("%d", port),
		"-o", "StrictHostKeyChecking=accept-new",
		fmt.Sprintf("%s@%s", c.User, c.Host),
	}
	return "sshpass", args
}

type ExitMsg struct {
	Err error
}

func ConnectCmd(c config.Connection, password string) tea.Cmd {
	name, args := BuildCommand(c, password)
	cmd := exec.Command(name, args...)
	// Pass password via SSHPASS env so it never appears in process args.
	cmd.Env = append(os.Environ(), "SSHPASS="+password)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return ExitMsg{Err: err}
	})
}
