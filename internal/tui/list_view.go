package tui

import (
	"fmt"

	"github.com/florianriquelme/sshjesus/internal/sshconfig"
)

// hostItem wraps an SSHHost for display in the list.
// Implements list.Item interface.
type hostItem struct {
	host          sshconfig.SSHHost
	lastConnected bool // Whether this host was recently connected
}

// FilterValue returns the value used for filtering/searching.
// Returns concatenated Name + Hostname + User for multi-field search.
func (h hostItem) FilterValue() string {
	return h.host.Name + " " + h.host.Hostname + " " + h.host.User
}

// Title returns the first line of the list item.
// Format: [★] Name (hostname) with star for last-connected and warning indicator if ParseError is set.
func (h hostItem) Title() string {
	// Prepend star indicator if this was recently connected
	prefix := ""
	if h.lastConnected {
		prefix = starIndicatorStyle.Render("★ ")
	}

	title := fmt.Sprintf("%s (%s)",
		hostnameStyle.Render(h.host.Name),
		h.host.Hostname)

	// Add warning indicator if there's a parse error
	if h.host.ParseError != nil {
		title = warningStyle.Render("⚠ ") + title
	}

	return prefix + title
}

// Description returns the second line of the list item.
// Format: "User: {user} | Port: {port}" or error message if ParseError is set.
func (h hostItem) Description() string {
	// If there's a parse error, show it instead of user/port
	if h.host.ParseError != nil {
		return warningStyle.Render(fmt.Sprintf("Error: %v", h.host.ParseError))
	}

	// Default values for empty fields
	user := h.host.User
	if user == "" {
		user = "default"
	}

	port := h.host.Port
	if port == "" {
		port = "22"
	}

	return secondaryStyle.Render(fmt.Sprintf("User: %s | Port: %s", user, port))
}

// separatorItem is a non-interactive list item that displays a separator.
// Used to separate wildcard entries from regular hosts.
type separatorItem struct{}

// FilterValue returns empty string (excluded from search).
func (s separatorItem) FilterValue() string {
	return ""
}

// Title returns the separator text.
func (s separatorItem) Title() string {
	return separatorStyle.Render("--- Wildcard Entries ---")
}

// Description returns empty string (no second line for separator).
func (s separatorItem) Description() string {
	return ""
}
