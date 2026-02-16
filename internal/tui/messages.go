package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/sshconfig"
	"github.com/florianriquelme/ssherpa/internal/sshkey"
)

// configLoadedMsg is sent after async SSH config parsing completes.
// Carries the parsed hosts and any error that occurred during parsing.
type configLoadedMsg struct {
	hosts   []sshconfig.SSHHost
	items   []list.Item
	sources map[string]string // Maps host name to source (e.g., "ssh-config", "1password")
	err     error
}

// historyLoadedMsg is sent after async history loading completes.
// Carries last-connected host for current directory and recent hosts map.
type historyLoadedMsg struct {
	lastConnectedHost string
	recentHosts       map[string]time.Time
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

// OnePasswordStatusMsg is sent when 1Password status changes.
type OnePasswordStatusMsg struct {
	Status backend.BackendStatus
}

// BackendServersUpdatedMsg is sent when backend servers are refreshed (e.g., after 1P sync).
type BackendServersUpdatedMsg struct{}

// keyPickerClosedMsg is sent when the key picker is closed without selection.
type keyPickerClosedMsg struct{}

// keySelectedMsg is sent when a key is selected from the picker.
type keySelectedMsg struct {
	path    string         // Path to the selected key file
	key     *sshkey.SSHKey // Selected key (nil if "None" was selected)
	cleared bool           // True if "None (SSH default)" was selected
}

// keysDiscoveredMsg is sent after async key discovery completes.
type keysDiscoveredMsg struct {
	keys []sshkey.SSHKey
	err  error
}

// hostKeyUpdatedMsg is sent after a host's IdentityFile is updated.
type hostKeyUpdatedMsg struct {
	host    sshconfig.SSHHost
	cleared bool
	keyPath string
}

// formRequestKeyPickerMsg is sent when the form requests the key picker to open.
type formRequestKeyPickerMsg struct {
	currentKeyPath string
}
