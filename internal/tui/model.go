package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/sshjesus/internal/history"
	"github.com/florianriquelme/sshjesus/internal/ssh"
	"github.com/florianriquelme/sshjesus/internal/sshconfig"
	"github.com/sahilm/fuzzy"
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

	// Phase 3 additions:
	searchInput   textinput.Model      // Always-on filter bar
	searchFocused bool                 // Whether search bar has keyboard focus
	allHosts      []sshconfig.SSHHost  // Unfiltered hosts (original order)
	filteredIdx   []int                // Indices into allHosts after fuzzy filter
	keys          KeyMap               // Key bindings
	help          help.Model           // Help footer component
	searchKeys    SearchKeyMap         // Key bindings for search mode help
	historyPath   string               // Path to history file
	returnToTUI   bool                 // Config: return to TUI after SSH (default false)
	recentHosts   map[string]time.Time // Recent connections for star indicator
	lastConnHost  string               // Last connected host from cwd (for preselection)
}

// New creates a new TUI model.
func New(configPath, historyPath string, returnToTUI bool) Model {
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
	l.SetFilteringEnabled(false) // We handle filtering manually with fuzzy search

	// Initialize search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search servers..."
	searchInput.CharLimit = 100

	// Initialize key bindings and help
	keys := DefaultKeyMap()
	searchKeys := SearchKeyMap{
		ClearSearch: keys.ClearSearch,
	}
	helpModel := help.New()

	return Model{
		viewMode:      ViewList,
		list:          l,
		spinner:       s,
		loading:       true,
		configPath:    configPath,
		searchInput:   searchInput,
		searchFocused: false,
		keys:          keys,
		help:          helpModel,
		searchKeys:    searchKeys,
		historyPath:   historyPath,
		returnToTUI:   returnToTUI,
		recentHosts:   make(map[string]time.Time),
	}
}

// Init initializes the model and starts async config and history loading.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadConfigCmd(m.configPath),
		loadHistoryCmd(m.historyPath),
	)
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

		// NOTE: We don't build list items here anymore.
		// The model will rebuild them after applying history data.
		return configLoadedMsg{
			hosts: hosts,
			items: nil,
			err:   nil,
		}
	}
}

// loadHistoryCmd returns a command that loads connection history asynchronously.
func loadHistoryCmd(historyPath string) tea.Cmd {
	return func() tea.Msg {
		if historyPath == "" {
			return historyLoadedMsg{
				lastConnectedHost: "",
				recentHosts:       make(map[string]time.Time),
			}
		}

		// Get current working directory for preselection
		cwd, err := os.Getwd()
		if err != nil {
			cwd = ""
		}

		// Get last connected host for this directory
		lastConnectedHost := ""
		if cwd != "" {
			entry, err := history.GetLastConnectedForPath(historyPath, cwd)
			if err == nil && entry != nil {
				lastConnectedHost = entry.HostName
			}
		}

		// Get recent hosts (last 50 unique)
		recentHosts, err := history.GetRecentHosts(historyPath, 50)
		if err != nil {
			recentHosts = make(map[string]time.Time)
		}

		return historyLoadedMsg{
			lastConnectedHost: lastConnectedHost,
			recentHosts:       recentHosts,
		}
	}
}

// hostSource implements fuzzy.Source for SSHHost slices.
type hostSource []sshconfig.SSHHost

func (h hostSource) String(i int) string {
	// Fuzzy match against Name, Hostname, and User
	return h[i].Name + " " + h[i].Hostname + " " + h[i].User
}

func (h hostSource) Len() int {
	return len(h)
}

// filterHosts applies fuzzy search to hosts and updates filteredIdx.
func (m *Model) filterHosts() {
	query := m.searchInput.Value()
	if query == "" {
		// Show all hosts
		m.filteredIdx = make([]int, len(m.allHosts))
		for i := range m.allHosts {
			m.filteredIdx[i] = i
		}
	} else {
		// Fuzzy search across Name + Hostname + User
		source := hostSource(m.allHosts)
		matches := fuzzy.FindFrom(query, source)
		m.filteredIdx = make([]int, len(matches))
		for i, match := range matches {
			m.filteredIdx[i] = match.Index
		}
	}

	m.rebuildListItems()
}

// rebuildListItems rebuilds the list items from filteredIdx with history indicators.
func (m *Model) rebuildListItems() {
	if len(m.filteredIdx) == 0 {
		m.list.SetItems([]list.Item{})
		return
	}

	// Organize filtered hosts
	var regular, wildcards []sshconfig.SSHHost
	for _, idx := range m.filteredIdx {
		host := m.allHosts[idx]
		if host.IsWildcard {
			wildcards = append(wildcards, host)
		} else {
			regular = append(regular, host)
		}
	}

	// Build list items
	items := make([]list.Item, 0, len(regular)+len(wildcards)+1)

	// Add regular hosts with star indicators
	for _, host := range regular {
		_, isRecent := m.recentHosts[host.Name]
		items = append(items, hostItem{
			host:          host,
			lastConnected: isRecent,
		})
	}

	// Add separator if there are wildcards
	if len(wildcards) > 0 {
		items = append(items, separatorItem{})

		// Add wildcard hosts with star indicators
		for _, host := range wildcards {
			_, isRecent := m.recentHosts[host.Name]
			items = append(items, hostItem{
				host:          host,
				lastConnected: isRecent,
			})
		}
	}

	m.list.SetItems(items)

	// Preselect last-connected host if applicable
	if m.lastConnHost != "" {
		for i, item := range items {
			if hostItem, ok := item.(hostItem); ok {
				if hostItem.host.Name == m.lastConnHost {
					m.list.Select(i)
					m.lastConnHost = "" // Clear so we only preselect once
					break
				}
			}
		}
	}
}

// connectToHost initiates SSH connection and records history.
func (m Model) connectToHost(host sshconfig.SSHHost) tea.Cmd {
	// Record history BEFORE handoff (app may exit after SSH)
	if m.historyPath != "" {
		_ = history.RecordConnection(m.historyPath, host.Name, host.Hostname, host.User)
		// Ignore error — don't block connection for history failure
	}

	return ssh.ConnectSSH(host.Name)
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update search input width
		m.searchInput.Width = msg.Width - 20 // Leave room for label and padding

		// Update list dimensions (subtract space for search bar, title, and help)
		searchBarHeight := 3 // search bar + border
		footerHeight := 2    // help text
		m.list.SetSize(msg.Width, msg.Height-searchBarHeight-footerHeight)

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

		// Store all hosts and initialize filtered index
		if msg.hosts != nil {
			m.allHosts = msg.hosts
			m.filterHosts() // Initial filter (shows all)
		}

	case historyLoadedMsg:
		m.lastConnHost = msg.lastConnectedHost
		m.recentHosts = msg.recentHosts

		// Rebuild list items with history indicators
		if len(m.allHosts) > 0 {
			m.rebuildListItems()
		}

	case ssh.SSHFinishedMsg:
		// SSH session ended
		if m.returnToTUI {
			// Return to list view and reload config (SSH config may have changed)
			m.viewMode = ViewList
			return m, loadConfigCmd(m.configPath)
		} else {
			// Default: exit to shell
			return m, tea.Quit
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		// Force quit always works
		if key.Matches(msg, m.keys.ForceQuit) {
			return m, tea.Quit
		}

		// Ignore other keys while loading
		if m.loading {
			return m, tea.Batch(cmds...)
		}

		// View-specific key handling
		switch m.viewMode {
		case ViewList:
			if m.searchFocused {
				// Search mode key handling
				switch {
				case key.Matches(msg, m.keys.ClearSearch):
					// Esc: clear search and return focus to list
					m.searchInput.SetValue("")
					m.searchInput.Blur()
					m.searchFocused = false
					m.filterHosts() // Show all hosts
					return m, nil

				case key.Matches(msg, m.keys.Connect):
					// Enter in search mode: connect to selected server
					selectedItem := m.list.SelectedItem()
					if selectedItem == nil {
						return m, nil
					}

					if item, ok := selectedItem.(hostItem); ok {
						return m, m.connectToHost(item.host)
					}

				default:
					// Pass all other keys to search input
					var cmd tea.Cmd
					m.searchInput, cmd = m.searchInput.Update(msg)
					cmds = append(cmds, cmd)

					// Re-filter on every keystroke
					m.filterHosts()
				}

			} else {
				// List mode key handling
				switch {
				case key.Matches(msg, m.keys.Search):
					// "/" - focus search
					m.searchFocused = true
					m.searchInput.Focus()
					return m, textinput.Blink

				case key.Matches(msg, m.keys.Quit):
					return m, tea.Quit

				case key.Matches(msg, m.keys.Connect):
					// Enter: connect to selected server
					selectedItem := m.list.SelectedItem()
					if selectedItem == nil {
						return m, nil
					}

					if item, ok := selectedItem.(hostItem); ok {
						return m, m.connectToHost(item.host)
					}

				case key.Matches(msg, m.keys.Details):
					// Tab or 'i': open detail view
					selectedItem := m.list.SelectedItem()
					if selectedItem == nil {
						return m, nil
					}

					if item, ok := selectedItem.(hostItem); ok {
						m.viewMode = ViewDetail
						m.detailHost = &item.host

						m.viewport = viewport.New(m.width, m.height)
						content := renderDetailView(m.detailHost, m.width, m.height)
						m.viewport.SetContent(content)
					}

				case key.Matches(msg, m.keys.GoToTop):
					// g or Home: jump to top
					m.list.Select(0)

				case key.Matches(msg, m.keys.GoToBottom):
					// G or End: jump to bottom
					m.list.Select(len(m.list.Items()) - 1)

				case key.Matches(msg, m.keys.HalfPageUp):
					// Ctrl+u: half page up
					listHeight := m.list.Height()
					halfPage := listHeight / 2
					for i := 0; i < halfPage; i++ {
						m.list.CursorUp()
					}

				case key.Matches(msg, m.keys.HalfPageDown):
					// Ctrl+d: half page down
					listHeight := m.list.Height()
					halfPage := listHeight / 2
					for i := 0; i < halfPage; i++ {
						m.list.CursorDown()
					}

				default:
					// Delegate to list for standard navigation (j/k, arrows, pgup/pgdn)
					var cmd tea.Cmd
					m.list, cmd = m.list.Update(msg)
					cmds = append(cmds, cmd)
				}
			}

		case ViewDetail:
			switch {
			case key.Matches(msg, m.keys.ClearSearch): // Esc
				// Return to list view
				m.viewMode = ViewList
				m.detailHost = nil

			case key.Matches(msg, m.keys.Quit):
				// q: quit from detail view
				return m, tea.Quit

			case msg.String() == "up" || msg.String() == "down" || msg.String() == "pgup" || msg.String() == "pgdown":
				// Delegate scrolling to viewport
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
		// Build search bar
		searchLabel := searchLabelStyle.Render("Filter: ")
		searchBar := searchLabel + m.searchInput.View()

		// Build main content
		var mainContent string
		if m.searchInput.Value() != "" && len(m.filteredIdx) == 0 {
			// No matches for search query
			mainContent = noMatchesStyle.Render(fmt.Sprintf(
				"No matches for \"%s\"\n\nPress Esc to clear search",
				m.searchInput.Value(),
			))
		} else {
			mainContent = m.list.View()
		}

		// Build help footer (context-sensitive)
		var helpView string
		if m.searchFocused {
			helpView = m.help.View(m.searchKeys)
		} else {
			helpView = m.help.View(m.keys)
		}

		// Combine all parts
		return lipgloss.JoinVertical(
			lipgloss.Left,
			searchBar,
			separatorStyle.Render("─"),
			mainContent,
			helpView,
		)

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
