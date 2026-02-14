package ssh

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

// SSHFinishedMsg is sent when the SSH connection terminates
type SSHFinishedMsg struct {
	Err      error
	HostName string
}

// ConnectSSH creates a Bubbletea command that hands off terminal control to SSH
// using the host alias from SSH config. This leverages the user's existing
// ~/.ssh/config settings (ProxyJump, IdentityFile, Port, etc.) automatically.
func ConnectSSH(hostName string) tea.Cmd {
	cmd := exec.Command("ssh", hostName)

	// Critical: Connect terminal I/O for silent handoff
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return SSHFinishedMsg{
			Err:      err,
			HostName: hostName,
		}
	})
}
