package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/sshjesus/internal/sshconfig"
)

// ViewMode represents the current view state.
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewDetail
)

// Model is the root Bubbletea model for the TUI.
type Model struct {
	viewMode   ViewMode
	list       list.Model
	viewport   viewport.Model
	detailHost *sshconfig.SSHHost
	spinner    spinner.Model
	loading    bool
	configPath string
	hosts      []sshconfig.SSHHost
	err        error
	width      int
	height     int
	ready      bool
}

// New creates a new TUI model.
func New(configPath string) Model {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accentColor)

	// Initialize empty list
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.Styles.SelectedTitle = selectedStyle
	delegate.Styles.SelectedDesc = selectedStyle

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "SSH Connections"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // Enable in Phase 3

	return Model{
		viewMode:   ViewList,
		list:       l,
		spinner:    s,
		loading:    true,
		configPath: configPath,
	}
}

// Init initializes the model and starts async config loading.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadConfigCmd(m.configPath))
}

// loadConfigCmd returns a command that parses the SSH config file asynchronously.
func loadConfigCmd(path string) tea.Cmd {
	return func() tea.Msg {
		hosts, err := sshconfig.ParseSSHConfig(path)
		if err != nil {
			return configLoadedMsg{
				hosts: nil,
				items: nil,
				err:   err,
			}
		}

		// Organize hosts: regular + separator + wildcards
		regular, wildcards := sshconfig.OrganizeHosts(hosts)

		// Build list items
		items := make([]list.Item, 0, len(regular)+len(wildcards)+1)

		// Add regular hosts
		for _, host := range regular {
			items = append(items, hostItem{host: host})
		}

		// Add separator if there are wildcards
		if len(wildcards) > 0 {
			items = append(items, separatorItem{})

			// Add wildcard hosts
			for _, host := range wildcards {
				items = append(items, hostItem{host: host})
			}
		}

		return configLoadedMsg{
			hosts: hosts,
			items: items,
			err:   nil,
		}
	}
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update list dimensions (subtract space for title and help)
		headerHeight := 2 // title + padding
		footerHeight := 2 // help text
		m.list.SetSize(msg.Width, msg.Height-headerHeight-footerHeight)

		// Update viewport dimensions if in detail mode
		if m.viewMode == ViewDetail && m.detailHost != nil {
			m.viewport = viewport.New(msg.Width, msg.Height)
			content := renderDetailView(m.detailHost, m.width, m.height)
			m.viewport.SetContent(content)
		}

	case configLoadedMsg:
		m.loading = false
		m.hosts = msg.hosts
		m.err = msg.err

		if msg.items != nil {
			cmd := m.list.SetItems(msg.items)
			cmds = append(cmds, cmd)
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		// Global quit keys
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Ignore other keys while loading
		if m.loading {
			return m, tea.Batch(cmds...)
		}

		// View-specific key handling
		switch m.viewMode {
		case ViewList:
			switch msg.String() {
			case "q":
				return m, tea.Quit

			case "enter":
				// Get selected item
				selectedItem := m.list.SelectedItem()
				if selectedItem == nil {
					return m, tea.Batch(cmds...)
				}

				// Only switch to detail view for hostItem (not separator)
				if item, ok := selectedItem.(hostItem); ok {
					m.viewMode = ViewDetail
					m.detailHost = &item.host

					// Initialize viewport with detail content
					m.viewport = viewport.New(m.width, m.height)
					content := renderDetailView(m.detailHost, m.width, m.height)
					m.viewport.SetContent(content)

					return m, tea.Batch(cmds...)
				}

			default:
				// Delegate to list for navigation
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
			}

		case ViewDetail:
			switch msg.String() {
			case "esc", "q":
				// Return to list view
				m.viewMode = ViewList
				m.detailHost = nil

			case "up", "down", "pgup", "pgdown":
				// Delegate to viewport for scrolling
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the current view.
func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Loading spinner
	if m.loading {
		return fmt.Sprintf("\n  %s Loading SSH config...\n", m.spinner.View())
	}

	// Error state (no hosts loaded)
	if m.err != nil && len(m.hosts) == 0 {
		return emptyStateStyle.Render(fmt.Sprintf(`
Error loading SSH config:
  %v

Check your SSH config file for issues.

Press 'q' to quit
`, m.err))
	}

	// Empty state (no hosts found)
	if len(m.hosts) == 0 {
		return emptyStateStyle.Render(`
No SSH connections found

Create or edit ~/.ssh/config with Host entries:

  Host myserver
    HostName example.com
    User username
    Port 22

Press 'q' to quit
`)
	}

	// View routing
	switch m.viewMode {
	case ViewList:
		return m.list.View()

	case ViewDetail:
		if m.detailHost == nil {
			m.viewMode = ViewList
			return m.list.View()
		}
		return m.viewport.View()

	default:
		return "Unknown view mode"
	}
}
