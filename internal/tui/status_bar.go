package tui

import (
	"github.com/florianriquelme/sshjesus/internal/backend"
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
			Render("⚠️  1Password is locked. Unlock the app to sync servers.")

	case backend.StatusUnavailable:
		// Orange warning bar: 1Password not running
		return statusBarWarningStyle.
			Width(width).
			Render("⚠️  1Password is not running. Using cached servers.")

	case backend.StatusUnknown:
		// Gray info bar: checking status
		return statusBarInfoStyle.
			Width(width).
			Render("Checking 1Password status...")

	default:
		return ""
	}
}
