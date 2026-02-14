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

// projectSeparatorItem is a non-interactive list item that separates
// current project results from other project results in search.
type projectSeparatorItem struct {
	label string
}

// FilterValue returns empty string (excluded from search).
func (p projectSeparatorItem) FilterValue() string {
	return ""
}

// Title returns the separator label.
func (p projectSeparatorItem) Title() string {
	return projectSeparatorStyle.Render(p.label)
}

// Description returns empty string (no second line for separator).
func (p projectSeparatorItem) Description() string {
	return ""
}
