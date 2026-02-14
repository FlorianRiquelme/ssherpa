package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette using AdaptiveColor for light/dark terminal support
var (
	// Primary accent color for hostnames and structural elements
	accentColor = lipgloss.AdaptiveColor{
		Light: "#5A67D8", // Indigo-600
		Dark:  "#818CF8", // Indigo-400
	}

	// Secondary color for user/port info
	secondaryColor = lipgloss.AdaptiveColor{
		Light: "#64748B", // Slate-500
		Dark:  "#94A3B8", // Slate-400
	}

	// Warning color for malformed entries
	warningColor = lipgloss.AdaptiveColor{
		Light: "#D97706", // Amber-600
		Dark:  "#FBBF24", // Amber-400
	}

	// Border color for panels
	borderColor = lipgloss.AdaptiveColor{
		Light: "#CBD5E1", // Slate-300
		Dark:  "#475569", // Slate-600
	}
)

// Reusable styles
var (
	// Title style for main list header
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			Padding(0, 1)

	// Selected list item highlight
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accentColor).
			Bold(true)

	// Hostname style (accent color + bold)
	hostnameStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	// Secondary style for user/port text
	secondaryStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	// Warning style for parse error indicators
	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	// Detail view header
	detailHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				Padding(1, 0)

	// Detail view option labels
	detailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(secondaryColor)

	// Detail view option values
	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	// Empty state style
	emptyStateStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true).
			Padding(1, 0)

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true)

	// Separator line style
	separatorStyle = lipgloss.NewStyle().
			Foreground(borderColor)
)
