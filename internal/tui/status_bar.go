package tui

import (
	"github.com/florianriquelme/ssherpa/internal/backend"
)

// renderStatusBar renders the 1Password status bar based on availability.
// Returns empty string when status is Available (clean UI, no banner needed).
func renderStatusBar(status backend.BackendStatus, width int) string {
	switch status {
	case backend.StatusAvailable:
		// No bar shown - clean UI when everything is working
		return ""

	case backend.StatusLocked:
		// Yellow warning bar: 1Password is locked
		return statusBarWarningStyle.
			Width(width).
			Render("⚠️  1Password is locked. Press 's' to authenticate.")

	case backend.StatusNotSignedIn:
		// Yellow warning bar: op CLI not signed in
		return statusBarWarningStyle.
			Width(width).
			Render("⚠️  1Password CLI not signed in. Press 's' to authenticate.")

	case backend.StatusUnavailable:
		// Orange warning bar: 1Password not available
		return statusBarWarningStyle.
			Width(width).
			Render("⚠️  1Password is not available. Using cached servers.")

	case backend.StatusUnknown:
		// Gray info bar: checking status
		return statusBarInfoStyle.
			Width(width).
			Render("Checking 1Password status...")

	default:
		return ""
	}
}
