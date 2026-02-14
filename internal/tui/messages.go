package tui

import (
	"time"

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

// historyLoadedMsg is sent after async history loading completes.
// Carries last-connected host for current directory and recent hosts map.
type historyLoadedMsg struct {
	lastConnectedHost string
	recentHosts       map[string]time.Time
}
