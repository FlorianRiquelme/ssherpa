package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/ssherpa/internal/sshconfig"
)

// badgeData represents a project badge to render inline.
type badgeData struct {
	name  string
	color lipgloss.AdaptiveColor
}

// hostItem wraps an SSHHost for display in the list.
// Implements list.Item interface.
type hostItem struct {
	host            sshconfig.SSHHost
	lastConnectedAt *time.Time  // Timestamp of last connection (nil if never connected)
	projectBadges   []badgeData // Project badges to render inline
}

// FilterValue returns the value used for filtering/searching.
// Returns concatenated Name + Hostname + User for multi-field search.
func (h hostItem) FilterValue() string {
	return h.host.Name + " " + h.host.Hostname + " " + h.host.User
}

// Title returns the first line of the list item.
// Format: Name (hostname) [badge1] [badge2] with warning indicator if ParseError is set.
func (h hostItem) Title() string {
	title := fmt.Sprintf("%s (%s)",
		hostnameStyle.Render(h.host.Name),
		h.host.Hostname)

	// Append project badges
	for _, badge := range h.projectBadges {
		title += " " + RenderProjectBadge(badge.name, badge.color)
	}

	// Add warning indicator if there's a parse error
	if h.host.ParseError != nil {
		title = warningStyle.Render("⚠ ") + title
	}

	return title
}

// Description returns the second line of the list item.
// Format: "User: {user} | Port: {port} | Last used at {timestamp}" or error message if ParseError is set.
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

	desc := fmt.Sprintf("User: %s | Port: %s", user, port)

	// Add "Last used at" timestamp if available
	if h.lastConnectedAt != nil {
		relativeTime := formatRelativeTime(*h.lastConnectedAt)
		desc += fmt.Sprintf(" | Last used %s", relativeTime)
	}

	return secondaryStyle.Render(desc)
}

// formatRelativeTime formats a timestamp as a relative time string (e.g., "2h ago", "yesterday")
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	duration := now.Sub(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if duration < 30*24*time.Hour {
		weeks := int(duration.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / (24 * 30))
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	} else {
		years := int(duration.Hours() / (24 * 365))
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}
