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

	// Search bar container style
	searchBarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	// Search label style (accent color)
	searchLabelStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	// Star indicator for last-connected server
	starIndicatorStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	// No matches empty state
	noMatchesStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true).
			Padding(2, 4)

	// Project separator between current and other projects
	projectSeparatorStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Italic(true)

	// Picker overlay styles
	pickerBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Padding(1, 2).
				Width(50)

	pickerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor)

	pickerSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(accentColor).
				Bold(true)

	pickerSuggestedStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Italic(true)

	pickerCheckmarkStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"}).
				Bold(true)

	pickerHelpStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Italic(true)

	pickerLabelStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true)

	// Form styles
	formTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			Padding(1, 0)

	formLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(secondaryColor)

	formRequiredStyle = lipgloss.NewStyle().
				Foreground(warningColor)

	formErrorStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Italic(true)

	formDnsWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}). // Amber
				Italic(true)

	formHelpStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true)

	formSavingStyle = lipgloss.NewStyle().
			Foreground(accentColor)
)
