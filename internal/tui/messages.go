package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/florianriquelme/sshjesus/internal/sshconfig"
)

// configLoadedMsg is sent after async SSH config parsing completes.
// Carries the parsed hosts and any error that occurred during parsing.
type configLoadedMsg struct {
	hosts []sshconfig.SSHHost
	items []list.Item
	err   error
}
