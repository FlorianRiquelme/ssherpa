package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MigrationWizard is a Bubbletea model for migrating existing 1Password items.
type MigrationWizard struct {
	client   interface{}          // SDK client (will be onepassword.Client when integrated)
	items    []MigrationCandidate // Discovered unmanaged items
	selected map[int]bool         // Selection state for each item
	cursor   int                  // Cursor position in list
	step     int                  // Current step: scanning, selecting, migrating, done
	spinner  spinner.Model        // Loading spinner
	results  MigrationResults     // Migration results
	width    int
	height   int
	err      error // Error message if any
}

// MigrationCandidate represents a 1Password item that can be migrated.
type MigrationCandidate struct {
	ItemID      string // 1Password item ID
	Title       string // Item title
	VaultName   string // Vault the item belongs to
	HasHostname bool   // Whether hostname field exists
	HasUser     bool   // Whether user field exists
	Complete    bool   // Has all required fields (hostname + user)
}

// MigrationResults tracks the outcome of the migration.
type MigrationResults struct {
	Migrated int      // Number of successfully migrated items
	Skipped  int      // Number of items skipped by user
	Errors   []string // Error messages for failed migrations
}

// Migration steps
const (
	stepScanning = iota
	stepSelecting
	stepMigrating
	stepDone
)

// NewMigrationWizard creates a new migration wizard.
func NewMigrationWizard(client interface{}) MigrationWizard {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accentColor)

	return MigrationWizard{
		client:   client,
		selected: make(map[int]bool),
		spinner:  s,
		step:     stepScanning,
	}
}

// Init starts the scanning process.
func (m MigrationWizard) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.scanForItems(),
	)
}

// Update handles messages.
func (m MigrationWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch m.step {
		case stepScanning:
			// No input during scan
			return m, nil

		case stepSelecting:
			return m.updateSelecting(msg)

		case stepMigrating:
			// No input during migration
			return m, nil

		case stepDone:
			// Enter to exit
			if msg.String() == "enter" {
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		if m.step == stepScanning || m.step == stepMigrating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case scanCompleteMsg:
		// Scan completed
		m.items = msg.items
		m.err = msg.err

		if msg.err != nil {
			m.step = stepDone
			return m, nil
		}

		if len(m.items) == 0 {
			// No items found - skip to done
			m.step = stepDone
			return m, nil
		}

		// Move to selection
		m.step = stepSelecting

		// Pre-select all complete items
		for i, item := range m.items {
			if item.Complete {
				m.selected[i] = true
			}
		}

	case migrationCompleteMsg:
		// Migration completed
		m.results = msg.results
		m.step = stepDone
	}

	return m, nil
}

// updateSelecting handles input during selection step.
func (m MigrationWizard) updateSelecting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.cursor = (m.cursor + 1) % len(m.items)
	case "k", "up":
		m.cursor = (m.cursor - 1 + len(m.items)) % len(m.items)
	case " ":
		// Toggle selection
		m.selected[m.cursor] = !m.selected[m.cursor]
	case "a":
		// Select all
		for i := range m.items {
			m.selected[i] = true
		}
	case "n":
		// Deselect all
		for i := range m.items {
			m.selected[i] = false
		}
	case "enter":
		// Start migration
		m.step = stepMigrating
		return m, tea.Batch(m.spinner.Tick, m.migrateSelected())
	case "esc":
		// Skip migration
		m.results.Skipped = len(m.items)
		m.step = stepDone
	}
	return m, nil
}

// View renders the current step.
func (m MigrationWizard) View() string {
	switch m.step {
	case stepScanning:
		return m.renderScanning()
	case stepSelecting:
		return m.renderSelecting()
	case stepMigrating:
		return m.renderMigrating()
	case stepDone:
		return m.renderDone()
	default:
		return "Unknown step"
	}
}

// renderScanning renders the scanning screen.
func (m MigrationWizard) renderScanning() string {
	var b strings.Builder

	title := titleStyle.Render("Migration Wizard")
	b.WriteString(title + "\n\n")

	b.WriteString(fmt.Sprintf("  %s Scanning 1Password vaults for SSH items...\n", m.spinner.View()))

	return wizardBoxStyle.Render(b.String())
}

// renderSelecting renders the item selection screen.
func (m MigrationWizard) renderSelecting() string {
	var b strings.Builder

	title := titleStyle.Render(fmt.Sprintf("Migration: %d items found", len(m.items)))
	b.WriteString(title + "\n\n")

	// Count selected
	selectedCount := 0
	for _, selected := range m.selected {
		if selected {
			selectedCount++
		}
	}

	// Show items with selection checkboxes
	for i, item := range m.items {
		checkbox := "[ ]"
		if m.selected[i] {
			checkbox = "[x]"
		}

		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		// Status indicator
		status := "✓ Complete"
		statusStyle := wizardSuccessStyle
		if !item.Complete {
			if !item.HasHostname {
				status = "✗ Missing hostname"
				statusStyle = wizardErrorStyle
			} else if !item.HasUser {
				status = "⚠ Missing user"
				statusStyle = warningStyle
			}
		}

		line := fmt.Sprintf("%s%s %-30s (%-20s) %s",
			cursor,
			checkbox,
			item.Title,
			item.VaultName,
			statusStyle.Render(status),
		)

		if i == m.cursor {
			line = selectedStyle.Render(line)
		}

		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(wizardDimStyle.Render("Space: toggle, a: select all, n: deselect all\n"))
	b.WriteString(wizardDimStyle.Render("Enter: migrate selected, Esc: skip\n"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s/%s selected",
		wizardSuccessStyle.Render(fmt.Sprintf("%d", selectedCount)),
		fmt.Sprintf("%d", len(m.items))))

	return wizardBoxStyle.Render(b.String())
}

// renderMigrating renders the migration progress screen.
func (m MigrationWizard) renderMigrating() string {
	var b strings.Builder

	title := titleStyle.Render("Migrating Items")
	b.WriteString(title + "\n\n")

	selectedCount := 0
	for _, selected := range m.selected {
		if selected {
			selectedCount++
		}
	}

	b.WriteString(fmt.Sprintf("  %s Migrating %d items...\n", m.spinner.View(), selectedCount))

	return wizardBoxStyle.Render(b.String())
}

// renderDone renders the results screen.
func (m MigrationWizard) renderDone() string {
	var b strings.Builder

	title := titleStyle.Render("Migration Complete")
	b.WriteString(title + "\n\n")

	if m.err != nil {
		b.WriteString(wizardErrorStyle.Render(fmt.Sprintf("Error: %v\n", m.err)))
	} else if len(m.items) == 0 {
		b.WriteString(wizardDimStyle.Render("No unmanaged SSH items found in 1Password\n"))
	} else {
		b.WriteString(fmt.Sprintf("  Migrated:  %s items\n", wizardSuccessStyle.Render(fmt.Sprintf("%d", m.results.Migrated))))
		b.WriteString(fmt.Sprintf("  Skipped:   %d items\n", m.results.Skipped))
		if len(m.results.Errors) > 0 {
			b.WriteString(fmt.Sprintf("  Errors:    %s items\n", wizardErrorStyle.Render(fmt.Sprintf("%d", len(m.results.Errors)))))
			b.WriteString("\n")
			b.WriteString(wizardErrorStyle.Render("Errors:\n"))
			for _, errMsg := range m.results.Errors {
				b.WriteString(fmt.Sprintf("  - %s\n", errMsg))
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(wizardDimStyle.Render("Press Enter to continue"))

	return wizardBoxStyle.Render(b.String())
}

// scanForItems scans 1Password vaults for unmanaged SSH items.
func (m MigrationWizard) scanForItems() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement actual 1Password scanning using the client
		// For now, simulate no items found (client integration pending)

		// In real implementation:
		// 1. List all vaults
		// 2. For each vault, list items with category "Server" or "SSH"
		// 3. Filter items WITHOUT "ssherpa" tag
		// 4. Parse fields to determine completeness (hostname + user)

		return scanCompleteMsg{
			items: []MigrationCandidate{}, // Empty for now (placeholder)
			err:   nil,
		}
	}
}

// migrateSelected performs migration on selected items.
func (m MigrationWizard) migrateSelected() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		results := MigrationResults{}

		// Count selected items
		var selectedItems []MigrationCandidate
		for i, item := range m.items {
			if m.selected[i] {
				selectedItems = append(selectedItems, item)
			}
		}

		// TODO: Implement actual migration using the client
		// For now, simulate successful migration

		// In real implementation:
		// 1. For each selected item:
		//    a. If incomplete, prompt for missing fields (inline form)
		//    b. Add "ssherpa" tag to item
		//    c. Normalize field labels (hostname, user, port, etc.)
		//    d. Call client.UpdateItem()
		// 2. Track results (success/failure per item)

		_ = ctx // Suppress unused warning

		results.Migrated = len(selectedItems)
		results.Skipped = len(m.items) - len(selectedItems)

		return migrationCompleteMsg{
			results: results,
		}
	}
}

// scanCompleteMsg is sent when scanning completes.
type scanCompleteMsg struct {
	items []MigrationCandidate
	err   error
}

// migrationCompleteMsg is sent when migration completes.
type migrationCompleteMsg struct {
	results MigrationResults
}
