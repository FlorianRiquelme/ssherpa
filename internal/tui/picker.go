package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/ssherpa/internal/config"
)

// ProjectPicker is a lightweight popup overlay for project assignment.
type ProjectPicker struct {
	items       []pickerItem
	selected    int
	serverName  string
	suggestions []string // Project IDs that are auto-suggested
	width       int
	height      int
	creatingNew bool            // Mode: creating a new project
	nameInput   textinput.Model // For "create new" flow
}

type pickerItem struct {
	projectID   string
	projectName string
	isNew       bool // "Create new project..." option
	isSuggested bool // Highlighted as auto-suggestion
	isAssigned  bool // Server already belongs to this project
}

// NewProjectPicker creates a new project picker overlay.
func NewProjectPicker(
	serverName string,
	projects []config.ProjectConfig,
	currentAssignments []string, // Project IDs this server already belongs to
	suggestions []string, // Auto-suggested project IDs
) ProjectPicker {
	// Build items list
	items := make([]pickerItem, 0, len(projects)+1)

	// Add existing projects
	for _, proj := range projects {
		isSuggested := false
		for _, s := range suggestions {
			if s == proj.ID {
				isSuggested = true
				break
			}
		}

		isAssigned := false
		for _, a := range currentAssignments {
			if a == proj.ID {
				isAssigned = true
				break
			}
		}

		items = append(items, pickerItem{
			projectID:   proj.ID,
			projectName: proj.Name,
			isSuggested: isSuggested,
			isAssigned:  isAssigned,
		})
	}

	// Sort: suggestions first, then assigned, then others
	// For simplicity, we'll rely on suggestions being highlighted
	// Real implementation could do more sophisticated sorting

	// Add "Create new..." option at bottom
	items = append(items, pickerItem{
		projectName: "+ Create new project...",
		isNew:       true,
	})

	// Initialize name input for create flow
	nameInput := textinput.New()
	nameInput.Placeholder = "org/repo or project name"
	nameInput.CharLimit = 100

	return ProjectPicker{
		items:       items,
		selected:    0,
		serverName:  serverName,
		suggestions: suggestions,
		width:       50,
		height:      20,
		nameInput:   nameInput,
	}
}

// Update handles picker key events.
func (p ProjectPicker) Update(msg tea.Msg) (ProjectPicker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if p.creatingNew {
			// Creating new project - handle name input
			return p.updateCreatingNew(msg)
		}

		// Normal picker mode
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Close picker without changes
			return p, func() tea.Msg { return pickerClosedMsg{} }

		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			p.selected++
			if p.selected >= len(p.items) {
				p.selected = len(p.items) - 1
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			p.selected--
			if p.selected < 0 {
				p.selected = 0
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Toggle assignment or create new
			item := p.items[p.selected]
			if item.isNew {
				// Switch to "create new" mode
				p.creatingNew = true
				p.nameInput.Focus()
				// Pre-fill with auto-detected name if available
				// (We'll pass this from the model via context)
				return p, nil
			} else {
				// Toggle assignment
				toggled := !item.isAssigned
				p.items[p.selected].isAssigned = toggled
				return p, func() tea.Msg {
					return projectAssignedMsg{
						serverName: p.serverName,
						projectID:  item.projectID,
						assigned:   toggled,
					}
				}
			}
		}
	}

	return p, nil
}

// updateCreatingNew handles the "create new project" name input flow.
func (p ProjectPicker) updateCreatingNew(msg tea.KeyMsg) (ProjectPicker, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		// Cancel creation, go back to picker
		p.creatingNew = false
		p.nameInput.Blur()
		p.nameInput.SetValue("")
		return p, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Create the project
		name := strings.TrimSpace(p.nameInput.Value())
		if name == "" {
			// Empty name, ignore
			return p, nil
		}

		// Generate project ID (simple timestamp-based ID)
		projectID := fmt.Sprintf("proj-%d", time.Now().UnixNano())

		newProject := config.ProjectConfig{
			ID:          projectID,
			Name:        name,
			ServerNames: []string{p.serverName},
		}

		// Close picker and send create message
		return p, func() tea.Msg {
			return projectCreatedMsg{project: newProject}
		}

	default:
		// Update text input
		var cmd tea.Cmd
		p.nameInput, cmd = p.nameInput.Update(msg)
		return p, cmd
	}
}

// View renders the picker overlay.
func (p ProjectPicker) View() string {
	if p.creatingNew {
		return p.viewCreatingNew()
	}

	var b strings.Builder

	// Title
	title := pickerTitleStyle.Render(fmt.Sprintf("Assign Project: %s", p.serverName))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Items
	for i, item := range p.items {
		line := p.renderItem(i, item)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Help text
	b.WriteString("\n")
	help := pickerHelpStyle.Render("↑/k ↓/j: navigate • enter: toggle • esc: close")
	b.WriteString(help)

	// Wrap in border
	content := b.String()
	bordered := pickerBorderStyle.Render(content)

	return bordered
}

// viewCreatingNew renders the "create new project" input view.
func (p ProjectPicker) viewCreatingNew() string {
	var b strings.Builder

	title := pickerTitleStyle.Render("Create New Project")
	b.WriteString(title)
	b.WriteString("\n\n")

	label := pickerLabelStyle.Render("Project name:")
	b.WriteString(label)
	b.WriteString("\n")
	b.WriteString(p.nameInput.View())
	b.WriteString("\n\n")

	help := pickerHelpStyle.Render("enter: create • esc: cancel")
	b.WriteString(help)

	content := b.String()
	bordered := pickerBorderStyle.Render(content)

	return bordered
}

// renderItem renders a single picker item.
func (p ProjectPicker) renderItem(index int, item pickerItem) string {
	var parts []string

	// Cursor indicator
	if index == p.selected {
		parts = append(parts, "> ")
	} else {
		parts = append(parts, "  ")
	}

	// Checkmark if assigned
	if item.isAssigned {
		parts = append(parts, pickerCheckmarkStyle.Render("✓ "))
	} else {
		parts = append(parts, "  ")
	}

	// Project name
	nameStyle := lipgloss.NewStyle()
	if index == p.selected {
		nameStyle = pickerSelectedStyle
	} else if item.isSuggested {
		nameStyle = pickerSuggestedStyle
	}

	nameText := item.projectName
	if item.isSuggested && !item.isNew {
		nameText = nameText + " (suggested)"
	}

	parts = append(parts, nameStyle.Render(nameText))

	return strings.Join(parts, "")
}

// Messages

type pickerClosedMsg struct{}

type projectAssignedMsg struct {
	serverName string
	projectID  string
	assigned   bool // true = added, false = removed
}

type projectCreatedMsg struct {
	project config.ProjectConfig
}
