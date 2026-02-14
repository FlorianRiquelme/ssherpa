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

// formCancelledMsg is sent when the user cancels the add/edit form.
type formCancelledMsg struct{}

// serverSavedMsg is sent after a server is successfully added or edited.
type serverSavedMsg struct {
	alias string
}

// serverDeletedMsg is sent after a server is successfully deleted.
type serverDeletedMsg struct {
	alias        string
	removedLines []string
}

// deleteErrorMsg is sent when deletion fails.
type deleteErrorMsg struct {
	err error
}

// deleteConfirmCancelledMsg is sent when the user cancels deletion.
type deleteConfirmCancelledMsg struct{}

// undoCompletedMsg is sent after a successful undo operation.
type undoCompletedMsg struct {
	alias string
}

// undoErrorMsg is sent when undo operation fails.
type undoErrorMsg struct {
	err error
}
