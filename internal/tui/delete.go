package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/sshconfig"
)

// DeleteConfirm is a full-screen component for confirming server deletion
// using the "type alias to confirm" pattern (like GitHub repo deletion).
type DeleteConfirm struct {
	alias         string          // Server alias to confirm
	input         textinput.Model // Text input for confirmation
	confirmed     bool            // Whether typed text matches alias (case-insensitive)
	configPath    string          // SSH config path for RemoveHost
	backendWriter backend.Writer  // Optional: if set, routes deletes through backend instead of sshconfig
	serverID      string          // For backend delete mode: server ID
}

// NewDeleteConfirm creates a delete confirmation view for the given server alias.
func NewDeleteConfirm(alias, configPath string) DeleteConfirm {
	input := textinput.New()
	input.Placeholder = alias
	input.CharLimit = len(alias) + 5 // Some tolerance for typos
	input.Focus()

	return DeleteConfirm{
		alias:      alias,
		input:      input,
		confirmed:  false,
		configPath: configPath,
	}
}

// Update handles input and confirmation logic.
func (d DeleteConfirm) Update(msg tea.Msg) (DeleteConfirm, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Enter: delete if confirmed
			if d.confirmed {
				// If backend writer is set, route through it
				if d.backendWriter != nil {
					return d, d.performBackendDelete()
				}

				// Perform deletion via SSH config
				removedLines, err := sshconfig.RemoveHost(d.configPath, d.alias)
				if err != nil {
					return d, func() tea.Msg {
						return deleteErrorMsg{err: err}
					}
				}

				return d, func() tea.Msg {
					return serverDeletedMsg{
						alias:        d.alias,
						removedLines: removedLines,
					}
				}
			}
			// Not confirmed - ignore Enter

		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Esc: cancel
			return d, func() tea.Msg {
				return deleteConfirmCancelledMsg{}
			}

		default:
			// Pass keystroke to input
			d.input, cmd = d.input.Update(msg)

			// Check if input matches alias (case-insensitive)
			d.confirmed = strings.EqualFold(d.input.Value(), d.alias)
		}
	}

	return d, cmd
}

// performBackendDelete deletes the server through the backend writer.
func (d *DeleteConfirm) performBackendDelete() tea.Cmd {
	ctx := context.Background()
	err := d.backendWriter.DeleteServer(ctx, d.serverID)
	if err != nil {
		return func() tea.Msg {
			return deleteErrorMsg{err: err}
		}
	}

	// Success - send BackendServersUpdatedMsg to trigger reload
	return func() tea.Msg {
		return BackendServersUpdatedMsg{}
	}
}

// View renders the delete confirmation prompt.
func (d DeleteConfirm) View() string {
	// Title
	title := deleteWarningStyle.Render("Delete SSH Connection")

	// Warning message
	warningText := deleteInstructionStyle.Render(
		"This will remove '" + d.alias + "' from your SSH config.",
	)

	// Instruction
	instruction := deleteInstructionStyle.Render(
		"Type the server alias to confirm deletion:",
	)

	// Alias to type (prominently displayed)
	aliasDisplay := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		Render(d.alias)

	// Input field with confirmation state
	var inputView string
	if d.confirmed {
		// Green border when confirmed
		confirmedInput := deleteConfirmedStyle.
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Render(d.input.View())
		inputView = confirmedInput
	} else {
		// Standard border
		standardInput := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1).
			Render(d.input.View())
		inputView = standardInput
	}

	// Action hint
	var actionHint string
	if d.confirmed {
		actionHint = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"}). // Green
			Bold(true).
			Render("✓ Press Enter to delete")
	} else {
		actionHint = deleteInstructionStyle.Render("Type the alias above to enable deletion")
	}

	// Footer help
	footer := renderHintRow([]shortcutHint{
		{key: "enter", desc: "delete (when confirmed)"},
		{key: "esc", desc: "cancel"},
	})

	// Compose view
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		"",
		title,
		"",
		warningText,
		"",
		instruction,
		"",
		"  "+aliasDisplay,
		"",
		inputView,
		"",
		actionHint,
		"",
		"",
		footer,
	)

	// Center the content in a bordered box
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#EF4444"}). // Red
		Padding(2, 4).
		Width(60).
		Render(content)

	return box
}
