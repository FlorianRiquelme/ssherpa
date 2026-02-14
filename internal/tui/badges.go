package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// RenderProjectBadge creates an inline GitHub-label-style badge.
// Uses white text on colored background with padding.
// Example output: [acme/backend] with colored background
func RenderProjectBadge(projectName string, color lipgloss.AdaptiveColor) string {
	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(color).
		Bold(true).
		Padding(0, 1)

	return badgeStyle.Render(projectName)
}
