package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/florianriquelme/ssherpa/internal/update"
	"github.com/florianriquelme/ssherpa/internal/version"
)

// checkForUpdateCmd runs the update check asynchronously.
func checkForUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		info, err := update.CheckForUpdate(version.Short())
		if err != nil || info == nil {
			return updateAvailableMsg{info: nil}
		}
		return updateAvailableMsg{info: info}
	}
}

// performUpdateCmd downloads and installs the update.
func performUpdateCmd(targetVersion string) tea.Cmd {
	return func() tea.Msg {
		err := update.PerformUpdate(targetVersion)
		// PerformUpdate only returns on error (exec replaces process on success)
		return updateFinishedMsg{err: err}
	}
}

// dismissUpdateCmd persists the dismissed version to cache.
func dismissUpdateCmd(ver string) tea.Cmd {
	return func() tea.Msg {
		update.DismissVersion(ver)
		return updateDismissedMsg{version: ver}
	}
}
