package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/ssherpa/internal/update"
)

// UpdateOverlay shows a scrollable changelog with update/dismiss actions.
type UpdateOverlay struct {
	viewport       viewport.Model
	width          int
	height         int
	currentVersion string
	latestVersion  string
	updating       bool // True while update is in progress
	updateErr      error
}

// NewUpdateOverlay creates a new update overlay.
func NewUpdateOverlay(width, height int, currentVersion, latestVersion string, changes []update.VersionChanges) UpdateOverlay {
	vp := viewport.New(68, height-10) // Room for border, header, and footer
	vp.SetContent(renderChangelog(changes))

	return UpdateOverlay{
		viewport:       vp,
		width:          width,
		height:         height,
		currentVersion: currentVersion,
		latestVersion:  latestVersion,
	}
}

// Update handles viewport scrolling in the overlay.
func (o UpdateOverlay) Update(msg tea.Msg) (UpdateOverlay, tea.Cmd) {
	var cmd tea.Cmd
	o.viewport, cmd = o.viewport.Update(msg)
	return o, cmd
}

// View renders the overlay with border, header, body, and footer.
func (o UpdateOverlay) View() string {
	header := formTitleStyle.Render(
		fmt.Sprintf("What's new (v%s → v%s)", o.currentVersion, o.latestVersion),
	)

	var footer string
	if o.updating {
		footer = formSavingStyle.Render("Updating...")
	} else if o.updateErr != nil {
		footer = lipgloss.JoinVertical(lipgloss.Left,
			formErrorStyle.Render(fmt.Sprintf("Update failed: %v", o.updateErr)),
			"",
			renderHintRow([]shortcutHint{
				{key: "esc", desc: "close"},
			}),
		)
	} else {
		footer = renderHintRow([]shortcutHint{
			{key: "enter", desc: "update"},
			{key: "x", desc: "dismiss"},
			{key: "esc", desc: "close"},
			{key: "↑/↓", desc: "scroll"},
		})
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		o.viewport.View(),
		"",
		footer,
	)

	return updateOverlayStyle.Render(content)
}

// renderChangelog renders []VersionChanges as styled markdown-like content.
func renderChangelog(changes []update.VersionChanges) string {
	if len(changes) == 0 {
		return secondaryStyle.Render("No changelog available.")
	}

	var sections []string
	for _, vc := range changes {
		// Version header
		versionHeader := fmt.Sprintf("v%s", vc.Version)
		if vc.Date != "" {
			versionHeader += fmt.Sprintf(" (%s)", vc.Date)
		}
		sections = append(sections, hostnameStyle.Render(versionHeader))
		sections = append(sections, "")

		for _, sec := range vc.Sections {
			// Category header
			sections = append(sections, formLabelStyle.Render(sec.Category))
			for _, entry := range sec.Entries {
				sections = append(sections, fmt.Sprintf("  • %s", entry))
			}
			sections = append(sections, "")
		}
	}

	return strings.Join(sections, "\n")
}
