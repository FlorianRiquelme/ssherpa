package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/sshjesus/internal/backend"
	"github.com/florianriquelme/sshjesus/internal/config"
	"github.com/florianriquelme/sshjesus/internal/domain"
	"github.com/florianriquelme/sshjesus/internal/history"
	"github.com/florianriquelme/sshjesus/internal/project"
	"github.com/florianriquelme/sshjesus/internal/ssh"
	"github.com/florianriquelme/sshjesus/internal/sshconfig"
	"github.com/florianriquelme/sshjesus/internal/sshkey"
	"github.com/sahilm/fuzzy"
)

// ViewMode represents the current view state.
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewDetail
	ViewAdd
	ViewEdit
	ViewDelete
)

// hostWithProject pairs a host with its project configurations
type hostWithProject struct {
	host     sshconfig.SSHHost
	projects []config.ProjectConfig
}

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
	searchNavKeys key.Binding          // Arrow-only navigation for search mode
	historyPath   string               // Path to history file
	returnToTUI   bool                 // Config: return to TUI after SSH (default false)
	recentHosts   map[string]time.Time // Recent connections for star indicator
	lastConnHost  string               // Last connected host from cwd (for preselection)

	// Phase 4 additions:
	currentProjectID string                           // Detected from git, empty if not in repo
	projects         []config.ProjectConfig           // Project configs from TOML
	projectMap       map[string]config.ProjectConfig  // Project ID -> config, for fast lookup
	picker           *ProjectPicker                   // Project picker overlay (nil when not showing)
	showingPicker    bool                             // Whether picker is visible
	configFilePath   string                           // Path to config file for saving

	// Phase 5 additions:
	serverForm    *ServerForm    // Add/edit form (nil when not showing)
	deleteConfirm *DeleteConfirm // Delete confirmation (nil when not showing)
	undoBuffer    *UndoBuffer    // Session-scoped undo buffer
	statusMsg     string         // Temporary status message (e.g. "Deleted X, press u to undo")

	// Phase 6 additions:
	opStatus   backend.BackendStatus // Current 1Password status
	opStatusBar string               // Rendered status bar (cached)
	appBackend backend.Backend       // Backend interface (nil for sshconfig-only mode)

	// Phase 7 additions:
	discoveredKeys    []sshkey.SSHKey  // All discovered SSH keys (from file/agent/1Password)
	keyPicker         *SSHKeyPicker    // SSH key picker overlay (nil when not showing)
	showingKeyPicker  bool             // Whether key picker is visible
	hostSources       map[string]string // Maps host name to source (e.g., "ssh-config", "1password")
	detailSource      string           // Source of the currently displayed detail host
}

// New creates a new TUI model.
func New(configPath, historyPath string, returnToTUI bool, currentProjectID string, projects []config.ProjectConfig, appConfigPath string, opStatus backend.BackendStatus, appBackend backend.Backend) Model {
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
	// Enable sign-in keybinding if 1Password needs authentication
	keys.SignIn.SetEnabled(opStatus == backend.StatusNotSignedIn || opStatus == backend.StatusLocked)
	searchKeys := SearchKeyMap{
		ClearSearch: keys.ClearSearch,
	}
	// Arrow-only navigation for search mode (excludes j/k which are valid search chars)
	searchNavKeys := key.NewBinding(key.WithKeys("up", "down"))
	helpModel := help.New()

	// Build project map for fast lookup
	projectMap := make(map[string]config.ProjectConfig)
	for _, p := range projects {
		projectMap[p.ID] = p
		// Also map by name for flexible matching
		projectMap[p.Name] = p
	}

	return Model{
		viewMode:         ViewList,
		list:             l,
		spinner:          s,
		loading:          true,
		configPath:       configPath,
		searchInput:      searchInput,
		searchFocused:    false,
		keys:             keys,
		help:             helpModel,
		searchKeys:       searchKeys,
		searchNavKeys:    searchNavKeys,
		historyPath:      historyPath,
		returnToTUI:      returnToTUI,
		recentHosts:      make(map[string]time.Time),
		currentProjectID: currentProjectID,
		projects:         projects,
		projectMap:       projectMap,
		configFilePath:   appConfigPath, // App config path for saving project assignments
		undoBuffer:       NewUndoBuffer(10),
		opStatus:    opStatus,   // Initial 1Password status
		opStatusBar: "",         // Will be rendered on first draw
		appBackend:  appBackend, // Backend interface (may be nil)
	}
}

// Init initializes the model and starts async config and history loading.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.spinner.Tick,
		loadHistoryCmd(m.historyPath),
		discoverKeysCmd(nil), // Async SSH key discovery (re-runs after hosts load)
	}

	// Load from backend if available, otherwise fallback to SSH config
	if m.appBackend != nil {
		cmds = append(cmds, loadBackendServersCmd(m.appBackend))
	} else {
		cmds = append(cmds, loadConfigCmd(m.configPath))
	}

	return tea.Batch(cmds...)
}

// loadConfigCmd returns a command that parses the SSH config file asynchronously.
func loadConfigCmd(path string) tea.Cmd {
	return func() tea.Msg {
		hosts, err := sshconfig.ParseSSHConfig(path)
		if err != nil {
			return configLoadedMsg{
				hosts:   nil,
				items:   nil,
				sources: nil,
				err:     err,
			}
		}

		// Build sources map for all hosts (all from ssh-config)
		sources := make(map[string]string, len(hosts))
		for _, host := range hosts {
			sources[host.Name] = "ssh-config"
		}

		// NOTE: We don't build list items here anymore.
		// The model will rebuild them after applying history data.
		return configLoadedMsg{
			hosts:   hosts,
			items:   nil,
			sources: sources,
			err:     nil,
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

// discoverKeysCmd returns a command that discovers SSH keys asynchronously.
// Accepts optional hosts to extract IdentityFile references.
// Always parses ~/.ssh/config directly for IdentityAgent directives (e.g. 1Password agent).
func discoverKeysCmd(hosts []sshconfig.SSHHost) tea.Cmd {
	return func() tea.Msg {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return keysDiscoveredMsg{keys: nil, err: err}
		}
		sshDir := filepath.Join(homeDir, ".ssh")
		configPath := filepath.Join(sshDir, "config")

		// Extract IdentityFile references from hosts
		var servers []*domain.Server
		for _, host := range hosts {
			for _, idFile := range host.IdentityFile {
				servers = append(servers, &domain.Server{IdentityFile: idFile})
			}
		}

		// Always parse SSH config directly for IdentityAgent directives.
		// These may be on Host * wildcards that aren't in the backend's server list.
		var identityAgents []sshkey.IdentityAgentSource
		seenAgents := make(map[string]bool)

		if configHosts, err := sshconfig.ParseSSHConfig(configPath); err == nil {
			for _, host := range configHosts {
				if agentPaths, ok := host.AllOptions["IdentityAgent"]; ok {
					for _, agentPath := range agentPaths {
						expanded := expandTilde(agentPath, homeDir)
						if seenAgents[expanded] {
							continue
						}
						seenAgents[expanded] = true

						source := sshkey.SourceAgent
						if strings.Contains(strings.ToLower(expanded), "1password") {
							source = sshkey.Source1Password
						}
						identityAgents = append(identityAgents, sshkey.IdentityAgentSource{
							SocketPath: expanded,
							Source:     source,
						})
					}
				}
			}
		}

		keys, err := sshkey.DiscoverKeys(sshDir, servers, identityAgents...)
		if err != nil {
			return keysDiscoveredMsg{keys: nil, err: err}
		}

		return keysDiscoveredMsg{keys: keys, err: nil}
	}
}

// expandTilde strips surrounding quotes and replaces leading ~ with the home directory path.
func expandTilde(path string, homeDir string) string {
	// Strip surrounding quotes (SSH config values may be quoted)
	path = strings.Trim(path, "\"'")
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	if path == "~" {
		return homeDir
	}
	return path
}

// has1PasswordKeys returns true if any discovered keys are from 1Password.
func has1PasswordKeys(keys []sshkey.SSHKey) bool {
	for _, k := range keys {
		if k.Source == sshkey.Source1Password {
			return true
		}
	}
	return false
}

// loadBackendServersCmd returns a command that loads servers from backend asynchronously.
func loadBackendServersCmd(backend backend.Backend) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		servers, err := backend.ListServers(ctx)
		if err != nil {
			return configLoadedMsg{
				hosts: nil,
				items: nil,
				err:   err,
			}
		}

		// Convert domain.Server to SSHHost at TUI boundary
		hosts, sources := serversToSSHHosts(servers)
		return configLoadedMsg{
			hosts:   hosts,
			items:   nil,
			sources: sources,
			err:     nil,
		}
	}
}

// syncBackendCmd triggers a backend sync and returns the new status.
// Used after sign-in to immediately refresh data and status.
func syncBackendCmd(b backend.Backend) tea.Cmd {
	return func() tea.Msg {
		if syncer, ok := b.(backend.Syncer); ok {
			ctx := context.Background()
			syncer.SyncFromBackend(ctx)
			// Error is OK - status is set appropriately by SyncFromBackend
			return OnePasswordStatusMsg{Status: syncer.GetStatus()}
		}
		// Backend doesn't support sync - no-op
		return nil
	}
}

// syncBackendWithTimeoutCmd triggers a backend sync with a custom timeout.
// Used for authentication: the op CLI triggers the native 1Password biometric dialog,
// and the user needs time to approve it.
func syncBackendWithTimeoutCmd(b backend.Backend, timeout time.Duration) tea.Cmd {
	return func() tea.Msg {
		if syncer, ok := b.(backend.Syncer); ok {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			syncer.SyncFromBackend(ctx)
			return OnePasswordStatusMsg{Status: syncer.GetStatus()}
		}
		return nil
	}
}

// serversToSSHHosts converts domain.Server models to TUI-internal SSHHost representations.
// This function defines the domain → TUI boundary, keeping TUI independent of domain models.
// Returns hosts and a map of host name to source (e.g., "ssh-config", "1password").
func serversToSSHHosts(servers []*domain.Server) ([]sshconfig.SSHHost, map[string]string) {
	hosts := make([]sshconfig.SSHHost, 0, len(servers))
	sources := make(map[string]string, len(servers))

	for _, srv := range servers {
		// Use DisplayName if available, otherwise fallback to Host
		name := srv.DisplayName
		if name == "" {
			name = srv.Host
		}

		// Convert Port int to string (SSHHost uses string to preserve raw config)
		portStr := ""
		if srv.Port != 0 {
			portStr = fmt.Sprintf("%d", srv.Port)
		}

		// Build AllOptions map with available data
		allOptions := make(map[string][]string)
		if srv.Host != "" {
			allOptions["HostName"] = []string{srv.Host}
		}
		if srv.User != "" {
			allOptions["User"] = []string{srv.User}
		}
		if portStr != "" {
			allOptions["Port"] = []string{portStr}
		}
		if srv.IdentityFile != "" {
			allOptions["IdentityFile"] = []string{srv.IdentityFile}
		}
		if srv.Proxy != "" {
			allOptions["ProxyJump"] = []string{srv.Proxy}
		}

		// Build IdentityFile slice
		var identityFiles []string
		if srv.IdentityFile != "" {
			identityFiles = []string{srv.IdentityFile}
		}

		// Detect wildcards in backend server names (e.g., "*" or "*.example.com")
		isWildcard := strings.Contains(name, "*") || strings.Contains(name, "?")

		host := sshconfig.SSHHost{
			Name:         name,
			Hostname:     srv.Host,
			User:         srv.User,
			Port:         portStr,
			IdentityFile: identityFiles,
			AllOptions:   allOptions,
			SourceFile:   "", // Backend servers have no source file
			SourceLine:   0,
			IsWildcard:   isWildcard,
			ParseError:   nil,
		}

		hosts = append(hosts, host)

		// Track source for this host
		if srv.Source != "" {
			sources[name] = srv.Source
		}
	}

	return hosts, sources
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
		// Use FindFromNoSort to get unsorted matches (we'll sort by project)
		matches := fuzzy.FindFromNoSort(query, source)

		if m.currentProjectID == "" || len(m.projects) == 0 {
			// No project context - just sort by score
			m.filteredIdx = make([]int, len(matches))
			for i, match := range matches {
				m.filteredIdx[i] = match.Index
			}
		} else {
			// Split matches into current project vs others
			var currentMatches, otherMatches []fuzzy.Match
			hostProjectMap := m.buildHostProjectMap()

			for _, match := range matches {
				host := m.allHosts[match.Index]
				projectConfigs := hostProjectMap[host.Name]

				// Check if this host belongs to current project
				belongsToCurrentProject := false
				for _, pc := range projectConfigs {
					if pc.ID == m.currentProjectID || pc.Name == m.currentProjectID {
						belongsToCurrentProject = true
						break
					}
				}

				if belongsToCurrentProject {
					currentMatches = append(currentMatches, match)
				} else {
					otherMatches = append(otherMatches, match)
				}
			}

			// Build filteredIdx: current project first, then others
			m.filteredIdx = make([]int, 0, len(matches))
			for _, match := range currentMatches {
				m.filteredIdx = append(m.filteredIdx, match.Index)
			}
			for _, match := range otherMatches {
				m.filteredIdx = append(m.filteredIdx, match.Index)
			}
		}
	}

	m.rebuildListItems()
}

// buildHostProjectMap creates a map of host name -> []ProjectConfig
func (m *Model) buildHostProjectMap() map[string][]config.ProjectConfig {
	hostProjectMap := make(map[string][]config.ProjectConfig)
	for _, project := range m.projects {
		for _, serverName := range project.ServerNames {
			hostProjectMap[serverName] = append(hostProjectMap[serverName], project)
		}
	}
	return hostProjectMap
}

// sortHostsByRecency sorts a slice of hosts by last connection time (most recent first)
// Hosts without connection history are placed after those with history
func (m *Model) sortHostsByRecency(hosts []sshconfig.SSHHost) {
	// Sort using bubble sort for simplicity (avoiding importing sort package dependencies)
	for i := 0; i < len(hosts); i++ {
		for j := i + 1; j < len(hosts); j++ {
			iTime, iHasTime := m.recentHosts[hosts[i].Name]
			jTime, jHasTime := m.recentHosts[hosts[j].Name]

			// If both have times, sort by most recent first
			// If only one has a time, it goes first
			// If neither has a time, maintain current order
			shouldSwap := false
			if jHasTime && !iHasTime {
				shouldSwap = true
			} else if jHasTime && iHasTime && jTime.After(iTime) {
				shouldSwap = true
			}

			if shouldSwap {
				hosts[i], hosts[j] = hosts[j], hosts[i]
			}
		}
	}
}

// sortHostsWithProjectByRecency sorts a slice of hostWithProject by last connection time (most recent first)
func (m *Model) sortHostsWithProjectByRecency(hosts []hostWithProject) {
	for i := 0; i < len(hosts); i++ {
		for j := i + 1; j < len(hosts); j++ {
			iTime, iHasTime := m.recentHosts[hosts[i].host.Name]
			jTime, jHasTime := m.recentHosts[hosts[j].host.Name]

			shouldSwap := false
			if jHasTime && !iHasTime {
				shouldSwap = true
			} else if jHasTime && iHasTime && jTime.After(iTime) {
				shouldSwap = true
			}

			if shouldSwap {
				hosts[i], hosts[j] = hosts[j], hosts[i]
			}
		}
	}
}

// sortHostsWithProjectAlphabetically sorts a slice of hostWithProject alphabetically by host name (case-insensitive)
func (m *Model) sortHostsWithProjectAlphabetically(hosts []hostWithProject) {
	for i := 0; i < len(hosts); i++ {
		for j := i + 1; j < len(hosts); j++ {
			// Case-insensitive comparison
			iName := strings.ToLower(hosts[i].host.Name)
			jName := strings.ToLower(hosts[j].host.Name)
			if iName > jName {
				hosts[i], hosts[j] = hosts[j], hosts[i]
			}
		}
	}
}

// sortHostsAlphabetically sorts a slice of SSHHost alphabetically by host name (case-insensitive)
func (m *Model) sortHostsAlphabetically(hosts []sshconfig.SSHHost) {
	for i := 0; i < len(hosts); i++ {
		for j := i + 1; j < len(hosts); j++ {
			// Case-insensitive comparison
			iName := strings.ToLower(hosts[i].Name)
			jName := strings.ToLower(hosts[j].Name)
			if iName > jName {
				hosts[i], hosts[j] = hosts[j], hosts[i]
			}
		}
	}
}

// rebuildListItems rebuilds the list items from filteredIdx with history indicators and project grouping.
func (m *Model) rebuildListItems() {
	if len(m.filteredIdx) == 0 {
		m.list.SetItems([]list.Item{})
		return
	}

	// Build host->project map for badge rendering
	hostProjectMap := m.buildHostProjectMap()

	query := m.searchInput.Value()
	hasSearch := query != ""

	// If no projects configured or in search mode, use simpler grouping
	if len(m.projects) == 0 || hasSearch {
		m.rebuildListItemsSimple(hostProjectMap, hasSearch)
		return
	}

	// Organize hosts: recently used at top, rest alphabetically
	var recentHosts []hostWithProject   // Hosts with recent connections (top of list)
	var otherHosts []hostWithProject    // All other hosts (sorted alphabetically)
	var wildcards []sshconfig.SSHHost   // Wildcards at bottom

	for _, idx := range m.filteredIdx {
		host := m.allHosts[idx]

		// Wildcards always go to bottom
		if host.IsWildcard {
			wildcards = append(wildcards, host)
			continue
		}

		projectConfigs := hostProjectMap[host.Name]
		hwp := hostWithProject{host: host, projects: projectConfigs}

		// Check if this host has recent connection history
		_, hasRecentConnection := m.recentHosts[host.Name]
		if hasRecentConnection {
			recentHosts = append(recentHosts, hwp)
		} else {
			otherHosts = append(otherHosts, hwp)
		}
	}

	// Sort recently used hosts by most recent first
	m.sortHostsWithProjectByRecency(recentHosts)

	// Sort other hosts alphabetically by name
	m.sortHostsWithProjectAlphabetically(otherHosts)

	// Build list items
	items := make([]list.Item, 0, len(m.filteredIdx)+3)

	// 1. Recently used hosts FIRST (sorted by most recent)
	for _, hwp := range recentHosts {
		items = append(items, m.createHostItem(hwp.host, hwp.projects, hostProjectMap))
	}

	// 2. All other hosts (sorted alphabetically)
	for _, hwp := range otherHosts {
		items = append(items, m.createHostItem(hwp.host, hwp.projects, hostProjectMap))
	}

	m.list.SetItems(items)
	m.preselectLastConnectedHost(items)
}

// rebuildListItemsSimple handles list building when in search mode or no projects configured
func (m *Model) rebuildListItemsSimple(hostProjectMap map[string][]config.ProjectConfig, hasSearch bool) {
	// Organize filtered hosts into recent and non-recent
	var recentHosts, otherHosts, wildcards []sshconfig.SSHHost
	for _, idx := range m.filteredIdx {
		host := m.allHosts[idx]
		if host.IsWildcard {
			wildcards = append(wildcards, host)
		} else {
			// Check if this host has recent connection history
			_, hasRecentConnection := m.recentHosts[host.Name]
			if hasRecentConnection {
				recentHosts = append(recentHosts, host)
			} else {
				otherHosts = append(otherHosts, host)
			}
		}
	}

	// Sort recent hosts by most recent first
	m.sortHostsByRecency(recentHosts)

	// Sort other hosts alphabetically
	m.sortHostsAlphabetically(otherHosts)

	// Build list items
	items := make([]list.Item, 0, len(recentHosts)+len(otherHosts)+len(wildcards))

	// 1. Add recently used hosts first (sorted by most recent)
	for _, host := range recentHosts {
		items = append(items, m.createHostItem(host, hostProjectMap[host.Name], hostProjectMap))
	}

	// 2. Add all other hosts (sorted alphabetically)
	for _, host := range otherHosts {
		items = append(items, m.createHostItem(host, hostProjectMap[host.Name], hostProjectMap))
	}

	m.list.SetItems(items)
	m.preselectLastConnectedHost(items)
}

// createHostItem creates a hostItem with project badges
func (m *Model) createHostItem(host sshconfig.SSHHost, projectConfigs []config.ProjectConfig, hostProjectMap map[string][]config.ProjectConfig) hostItem {
	// Get last connected timestamp (if any)
	var lastConnectedAt *time.Time
	if ts, exists := m.recentHosts[host.Name]; exists {
		lastConnectedAt = &ts
	}

	// Build project badges
	var badges []badgeData
	for _, pc := range projectConfigs {
		// Get color (use user override or auto-generate)
		color := m.getProjectColor(pc)
		badges = append(badges, badgeData{
			name:  pc.Name,
			color: color,
		})
	}

	return hostItem{
		host:            host,
		lastConnectedAt: lastConnectedAt,
		projectBadges:   badges,
	}
}

// getProjectColor returns the color for a project (user override or auto-generated)
func (m *Model) getProjectColor(pc config.ProjectConfig) lipgloss.AdaptiveColor {
	if pc.Color != "" {
		// User override
		return lipgloss.AdaptiveColor{
			Light: pc.Color,
			Dark:  pc.Color,
		}
	}
	// Auto-generate color from project ID
	return project.ProjectColor(pc.ID)
}

// preselectLastConnectedHost preselects the last-connected host if applicable
func (m *Model) preselectLastConnectedHost(items []list.Item) {
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

		// Update list dimensions (subtract space for search bar, title, help, and optional status bar)
		searchBarHeight := 3 // search bar + border
		footerHeight := 2    // help text
		statusBarHeight := 0
		if m.opStatus != backend.StatusAvailable && m.opStatus != backend.StatusUnknown {
			statusBarHeight = 1 // status bar height when shown
		}
		m.list.SetSize(msg.Width, msg.Height-searchBarHeight-footerHeight-statusBarHeight)

		// Update viewport dimensions if in detail mode
		if m.viewMode == ViewDetail && m.detailHost != nil {
			m.viewport = viewport.New(msg.Width, msg.Height)
			content := renderDetailView(m.detailHost, m.detailSource, m.width, m.height)
			m.viewport.SetContent(content)
		}

	case configLoadedMsg:
		m.loading = false
		m.hosts = msg.hosts
		m.err = msg.err

		// Store all hosts and initialize filtered index
		if msg.hosts != nil {
			// If we already have hosts, merge new ones by appending to bottom
			if len(m.allHosts) > 0 {
				// Build a set of existing host names for quick lookup
				existingHosts := make(map[string]bool)
				for _, host := range m.allHosts {
					existingHosts[host.Name] = true
				}

				// Append only new hosts to the bottom
				for _, host := range msg.hosts {
					if !existingHosts[host.Name] {
						m.allHosts = append(m.allHosts, host)
					}
				}
			} else {
				// First load - just set the hosts
				m.allHosts = msg.hosts
			}

			m.filterHosts() // Initial filter (shows all)

			// Store source mapping
			if msg.sources != nil {
				// Merge source mappings instead of replacing
				if m.hostSources == nil {
					m.hostSources = msg.sources
				} else {
					for name, source := range msg.sources {
						m.hostSources[name] = source
					}
				}
			}

			// Re-discover keys now that hosts are loaded (includes IdentityFile references)
			cmds = append(cmds, discoverKeysCmd(m.allHosts))
		}

	case historyLoadedMsg:
		m.lastConnHost = msg.lastConnectedHost
		m.recentHosts = msg.recentHosts

		// Rebuild list items with history indicators
		if len(m.allHosts) > 0 {
			m.rebuildListItems()
		}

	case keysDiscoveredMsg:
		// Store discovered keys
		if msg.err == nil {
			m.discoveredKeys = msg.keys
		}
		// Silently ignore errors - key discovery is optional

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
		// Route spinner ticks to form when saving (for the form's spinner)
		if (m.viewMode == ViewAdd || m.viewMode == ViewEdit) && m.serverForm != nil && m.serverForm.saving {
			var cmd tea.Cmd
			*m.serverForm, cmd = m.serverForm.Update(msg)
			cmds = append(cmds, cmd)
		}

	case dnsCheckResultMsg:
		// Route DNS check results to the form
		if (m.viewMode == ViewAdd || m.viewMode == ViewEdit) && m.serverForm != nil {
			var cmd tea.Cmd
			*m.serverForm, cmd = m.serverForm.Update(msg)
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

		// If showing project picker, route all keys to picker
		if m.showingPicker && m.picker != nil {
			var cmd tea.Cmd
			*m.picker, cmd = m.picker.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		// If showing key picker, route all keys to key picker
		if m.showingKeyPicker && m.keyPicker != nil {
			var cmd tea.Cmd
			*m.keyPicker, cmd = m.keyPicker.Update(msg)
			cmds = append(cmds, cmd)
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


			// Arrow key navigation in search mode
			case key.Matches(msg, m.searchNavKeys):
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)

			// Tab to view details from search mode
			case key.Matches(msg, m.keys.Details):
				selectedItem := m.list.SelectedItem()
				if selectedItem == nil {
					return m, nil
				}
				if item, ok := selectedItem.(hostItem); ok {
					m.detailHost = &item.host
					m.detailSource = m.hostSources[item.host.Name]
					m.viewMode = ViewDetail
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
				// Clear status message on any key press
				m.statusMsg = ""

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

						// Look up source for this host
						if m.hostSources != nil {
							m.detailSource = m.hostSources[item.host.Name]
						}

						m.viewport = viewport.New(m.width, m.height)
						content := renderDetailView(m.detailHost, m.detailSource, m.width, m.height)
						m.viewport.SetContent(content)
					}

				case key.Matches(msg, m.keys.AssignProject):
					// 'p': open project picker
					selectedItem := m.list.SelectedItem()
					if selectedItem == nil {
						return m, nil
					}

					if item, ok := selectedItem.(hostItem); ok {
						// Get auto-suggestions using hostname matcher
						m.showingPicker = true
						picker := m.createPickerForHost(item.host.Name)
						m.picker = &picker
					}

				case key.Matches(msg, m.keys.AddServer):
					// 'a': open add server form
					form := NewServerForm(m.configPath)
					if has1PasswordKeys(m.discoveredKeys) {
						form.fields[4].input.Placeholder = "Default (1Password agent) - Press Enter to select key"
					}
					m.serverForm = &form
					m.viewMode = ViewAdd

				case key.Matches(msg, m.keys.EditServer):
					// 'e': open edit server form
					selectedItem := m.list.SelectedItem()
					if selectedItem == nil {
						return m, nil
					}

					if item, ok := selectedItem.(hostItem); ok {
						form := NewEditServerForm(m.configPath, item.host)
						m.serverForm = &form
						m.viewMode = ViewEdit
					}

				case key.Matches(msg, m.keys.DeleteServer):
					// 'd': open delete confirmation
					selectedItem := m.list.SelectedItem()
					if selectedItem == nil {
						return m, nil
					}

					if item, ok := selectedItem.(hostItem); ok {
						confirm := NewDeleteConfirm(item.host.Name, m.configPath)
						m.deleteConfirm = &confirm
						m.viewMode = ViewDelete
					}

				case key.Matches(msg, m.keys.Undo):
					// 'u': undo last delete
					if m.undoBuffer.IsEmpty() {
						// Nothing to undo - ignore
						return m, nil
					}

					// Pop the last deleted entry
					entry, ok := m.undoBuffer.Pop()
					if !ok {
						return m, nil
					}

					// Run undo command asynchronously
					return m, func() tea.Msg {
						err := RestoreHost(entry.ConfigPath, entry.RawLines)
						if err != nil {
							return undoErrorMsg{err: err}
						}
						return undoCompletedMsg{alias: entry.Alias}
					}

				case key.Matches(msg, m.keys.SignIn):
					// 's': trigger native 1Password biometric auth via sync
					if m.appBackend != nil {
						m.statusMsg = "Authenticating with 1Password..."
						return m, syncBackendWithTimeoutCmd(m.appBackend, 60*time.Second)
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
				// If key picker is showing, close it instead of going back to list
				if m.showingKeyPicker {
					m.showingKeyPicker = false
					m.keyPicker = nil
					return m, nil
				}
				// Return to list view
				m.viewMode = ViewList
				m.detailHost = nil

			case key.Matches(msg, m.keys.SelectKey): // K: open key picker
				if m.detailHost != nil {
					// Get current IdentityFile (first one if multiple)
					currentKeyPath := ""
					if len(m.detailHost.IdentityFile) > 0 {
						currentKeyPath = m.detailHost.IdentityFile[0]
					}
					// Open key picker
					picker := NewSSHKeyPicker(m.detailHost.Name, m.discoveredKeys, currentKeyPath)
					m.keyPicker = &picker
					m.showingKeyPicker = true
				}

			case key.Matches(msg, m.keys.Quit):
				// q: quit from detail view
				return m, tea.Quit

			case msg.String() == "up" || msg.String() == "down" || msg.String() == "pgup" || msg.String() == "pgdown":
				// Delegate scrolling to viewport
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}

		case ViewAdd, ViewEdit:
			// Route all messages to form
			if m.serverForm != nil {
				var cmd tea.Cmd
				*m.serverForm, cmd = m.serverForm.Update(msg)
				cmds = append(cmds, cmd)
			}

		case ViewDelete:
			// Route all messages to delete confirmation
			if m.deleteConfirm != nil {
				var cmd tea.Cmd
				*m.deleteConfirm, cmd = m.deleteConfirm.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case formCancelledMsg:
		// Close form and return to list
		m.viewMode = ViewList
		m.serverForm = nil

	case serverSavedMsg:
		// Server saved successfully - reload config and return to list
		m.viewMode = ViewList
		m.serverForm = nil
		return m, loadConfigCmd(m.configPath)

	case serverDeletedMsg:
		// Server deleted successfully - push to undo buffer and reload config
		m.undoBuffer.Push(UndoEntry{
			Alias:       msg.alias,
			ConfigPath:  m.configPath,
			RawLines:    msg.removedLines,
			DeletedAt:   time.Now(),
		})
		m.viewMode = ViewList
		m.deleteConfirm = nil
		m.statusMsg = fmt.Sprintf("Deleted '%s' (press 'u' to undo)", msg.alias)
		return m, loadConfigCmd(m.configPath)

	case deleteErrorMsg:
		// Delete failed - show error and return to list
		m.viewMode = ViewList
		m.deleteConfirm = nil
		m.statusMsg = fmt.Sprintf("Delete failed: %v", msg.err)
		return m, nil

	case deleteConfirmCancelledMsg:
		// User cancelled delete - return to list
		m.viewMode = ViewList
		m.deleteConfirm = nil
		return m, nil

	case undoCompletedMsg:
		// Undo successful - reload config
		m.statusMsg = fmt.Sprintf("Restored '%s'", msg.alias)
		return m, loadConfigCmd(m.configPath)

	case undoErrorMsg:
		// Undo failed - show error
		m.statusMsg = fmt.Sprintf("Undo failed: %v", msg.err)
		return m, nil

	case pickerClosedMsg:
		// Close picker without changes
		m.showingPicker = false
		m.picker = nil

	case projectAssignedMsg:
		// Handle project assignment toggle
		m.handleProjectAssignment(msg.serverName, msg.projectID, msg.assigned)
		// Keep picker open to allow multiple assignments

	case projectCreatedMsg:
		// Handle new project creation
		m.handleProjectCreation(msg.project)
		// Close picker after creation
		m.showingPicker = false
		m.picker = nil

	case keyPickerClosedMsg:
		// Close key picker without changes
		m.showingKeyPicker = false
		m.keyPicker = nil

	case keySelectedMsg:
		// Handle key selection from picker
		m.showingKeyPicker = false
		m.keyPicker = nil

		// Handle based on current view mode
		if m.viewMode == ViewDetail && m.detailHost != nil {
			// Update SSH config for the host
			return m, m.updateHostIdentityFile(m.detailHost.Name, msg.path, msg.cleared)
		} else if (m.viewMode == ViewAdd || m.viewMode == ViewEdit) && m.serverForm != nil {
			// Update form's IdentityFile field
			if msg.cleared {
				m.serverForm.fields[4].input.SetValue("")
				m.serverForm.selectedKey = nil
			} else {
				// Store the path value (used when saving)
				m.serverForm.fields[4].input.SetValue(msg.path)
				m.serverForm.selectedKey = msg.key
			}
		}

	case hostKeyUpdatedMsg:
		// Update detail view with new host data
		m.detailHost = &msg.host
		m.viewport.SetContent(renderDetailView(&msg.host, m.detailSource, m.width, m.height))

		// Show status message
		if msg.cleared {
			m.statusMsg = "Key cleared (using SSH default)"
		} else {
			// Extract filename from path
			filename := filepath.Base(msg.keyPath)
			m.statusMsg = fmt.Sprintf("Key updated: %s", filename)
		}

	case formRequestKeyPickerMsg:
		// Form is requesting to open the key picker
		if m.serverForm != nil {
			// Get server name from Alias field
			serverName := m.serverForm.fields[0].input.Value()
			if serverName == "" {
				serverName = "New Server"
			}

			// Open key picker with current key path
			picker := NewSSHKeyPicker(serverName, m.discoveredKeys, msg.currentKeyPath)
			m.keyPicker = &picker
			m.showingKeyPicker = true
		}

	case OnePasswordStatusMsg:
		// Update 1Password status and re-render status bar
		oldStatus := m.opStatus
		m.opStatus = msg.Status
		m.opStatusBar = renderStatusBar(m.opStatus, m.width)

		// Toggle sign-in keybinding based on status
		m.keys.SignIn.SetEnabled(msg.Status == backend.StatusNotSignedIn || msg.Status == backend.StatusLocked)

		// If status changed to Available, trigger server list refresh
		if oldStatus != backend.StatusAvailable && msg.Status == backend.StatusAvailable {
			if m.appBackend != nil {
				return m, loadBackendServersCmd(m.appBackend)
			}
		}

		// Trigger re-render
		return m, nil

	case BackendServersUpdatedMsg:
		// Backend servers refreshed (e.g., 1Password sync completed)
		// Reload servers from backend if available, otherwise fallback to SSH config
		if m.appBackend != nil {
			return m, loadBackendServersCmd(m.appBackend)
		}
		return m, loadConfigCmd(m.configPath)
	}

	return m, tea.Batch(cmds...)
}

// createPickerForHost creates a project picker for the given server.
func (m *Model) createPickerForHost(serverName string) ProjectPicker {
	// Build ProjectMember list for hostname matcher
	var projectMembers []project.ProjectMember
	for _, proj := range m.projects {
		projectMembers = append(projectMembers, project.ProjectMember{
			ProjectID:   proj.ID,
			ProjectName: proj.Name,
			Hostnames:   proj.ServerNames,
		})
	}

	// Get auto-suggestions based on hostname similarity
	suggestions := project.SuggestProjects(serverName, projectMembers)
	suggestionIDs := make([]string, len(suggestions))
	for i, s := range suggestions {
		suggestionIDs[i] = s.ProjectID
	}

	// Get current assignments for this server
	var currentAssignments []string
	hostProjectMap := m.buildHostProjectMap()
	if projConfigs, ok := hostProjectMap[serverName]; ok {
		for _, pc := range projConfigs {
			currentAssignments = append(currentAssignments, pc.ID)
		}
	}

	return NewProjectPicker(serverName, m.projects, currentAssignments, suggestionIDs)
}

// handleProjectAssignment adds or removes a server from a project.
func (m *Model) handleProjectAssignment(serverName, projectID string, assigned bool) {
	// Find the project
	var targetProject *config.ProjectConfig
	for i := range m.projects {
		if m.projects[i].ID == projectID {
			targetProject = &m.projects[i]
			break
		}
	}

	if targetProject == nil {
		return
	}

	if assigned {
		// Add server to project (if not already present)
		found := false
		for _, sn := range targetProject.ServerNames {
			if sn == serverName {
				found = true
				break
			}
		}
		if !found {
			targetProject.ServerNames = append(targetProject.ServerNames, serverName)
		}
	} else {
		// Remove server from project
		newServerNames := make([]string, 0, len(targetProject.ServerNames))
		for _, sn := range targetProject.ServerNames {
			if sn != serverName {
				newServerNames = append(newServerNames, sn)
			}
		}
		targetProject.ServerNames = newServerNames
	}

	// Update projectMap
	m.projectMap[projectID] = *targetProject

	// Save config
	m.saveConfig()

	// Rebuild list items to show updated badges
	m.rebuildListItems()
}

// handleProjectCreation adds a new project to the config.
func (m *Model) handleProjectCreation(newProject config.ProjectConfig) {
	// Add to projects list
	m.projects = append(m.projects, newProject)

	// Update projectMap
	m.projectMap[newProject.ID] = newProject
	m.projectMap[newProject.Name] = newProject

	// Save config
	m.saveConfig()

	// Rebuild list items to show new badge
	m.rebuildListItems()
}

// saveConfig saves the current config to disk.
func (m *Model) saveConfig() {
	// Load full config from disk, or create default if missing
	cfg, err := config.Load(m.configFilePath)
	if err != nil {
		// Config doesn't exist yet — create one with defaults
		cfg = config.DefaultConfig()
		cfg.Backend = "sshconfig"
	}

	// Update projects in config
	cfg.Projects = m.projects

	// Save back to disk (empty path triggers DefaultPath fallback)
	_ = config.Save(cfg, m.configFilePath)
}

// updateHostIdentityFile updates the IdentityFile for a host in SSH config.
// Returns a command that performs the update and refreshes the detail view.
func (m *Model) updateHostIdentityFile(alias, keyPath string, cleared bool) tea.Cmd {
	return func() tea.Msg {
		// Build updated host entry
		// First, parse the current host to get all fields
		host := m.detailHost
		if host == nil {
			return nil
		}

		entry := sshconfig.HostEntry{
			Alias:        host.Name,
			Hostname:     host.Hostname,
			User:         host.User,
			Port:         host.Port,
			IdentityFile: keyPath,
			ExtraConfig:  buildExtraConfigFromHost(*host),
		}

		// Update the host in SSH config
		err := sshconfig.EditHost(m.configPath, alias, entry)
		if err != nil {
			// Show error as status message
			m.statusMsg = fmt.Sprintf("Failed to update key: %v", err)
			return nil
		}

		// Reload config and refresh detail view
		hosts, err := sshconfig.ParseSSHConfig(m.configPath)
		if err != nil {
			return nil
		}

		// Find the updated host
		for _, h := range hosts {
			if h.Name == alias {
				// Update detail view content
				return hostKeyUpdatedMsg{host: h, cleared: cleared, keyPath: keyPath}
			}
		}

		return nil
	}
}

// buildExtraConfigFromHost extracts non-standard SSH options from host.
func buildExtraConfigFromHost(host sshconfig.SSHHost) string {
	var lines []string
	standardKeys := map[string]bool{
		"HostName":     true,
		"User":         true,
		"Port":         true,
		"IdentityFile": true,
	}

	for key, values := range host.AllOptions {
		if standardKeys[key] {
			continue
		}
		for _, val := range values {
			lines = append(lines, fmt.Sprintf("%s %s", key, val))
		}
	}

	return strings.Join(lines, "\n")
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

		// Build status bar for 1Password availability (if needed)
		// Only render when status is not Available (clean UI when working)
		var statusBarView string
		if m.opStatus != backend.StatusAvailable && m.opStatus != backend.StatusUnknown {
			statusBarView = renderStatusBar(m.opStatus, m.width)
		}

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

		// Build status message if present
		var statusView string
		if m.statusMsg != "" {
			statusView = undoStatusStyle.Render(m.statusMsg)
		}

		// Combine all parts (with optional status bar)
		var baseView string
		parts := []string{
			searchBar,
			separatorStyle.Render("─"),
		}

		// Add status bar if present (between search and main content)
		if statusBarView != "" {
			parts = append(parts, statusBarView)
		}

		parts = append(parts, mainContent)

		if statusView != "" {
			parts = append(parts, statusView)
		}

		parts = append(parts, helpView)

		baseView = lipgloss.JoinVertical(lipgloss.Left, parts...)

		// If showing project picker, overlay it on top
		if m.showingPicker && m.picker != nil {
			pickerView := m.picker.View()
			// Center the picker over the base view using lipgloss.Place
			centeredPicker := lipgloss.Place(
				m.width,
				m.height,
				lipgloss.Center,
				lipgloss.Center,
				pickerView,
			)
			// Layer base view with picker overlay
			// Simple approach: just show centered picker (base is dimmed/hidden)
			return centeredPicker
		}

		// If showing key picker, overlay it on top
		if m.showingKeyPicker && m.keyPicker != nil {
			keyPickerView := m.keyPicker.View()
			centeredKeyPicker := lipgloss.Place(
				m.width,
				m.height,
				lipgloss.Center,
				lipgloss.Center,
				keyPickerView,
			)
			return centeredKeyPicker
		}

		return baseView

	case ViewDetail:
		if m.detailHost == nil {
			m.viewMode = ViewList
			return m.list.View()
		}

		baseView := m.viewport.View()

		// If showing key picker, overlay it on top
		if m.showingKeyPicker && m.keyPicker != nil {
			keyPickerView := m.keyPicker.View()
			centeredKeyPicker := lipgloss.Place(
				m.width,
				m.height,
				lipgloss.Center,
				lipgloss.Center,
				keyPickerView,
			)
			return centeredKeyPicker
		}

		return baseView

	case ViewAdd, ViewEdit:
		if m.serverForm == nil {
			m.viewMode = ViewList
			return m.list.View()
		}

		baseView := m.serverForm.View()

		// If showing key picker, overlay it on top
		if m.showingKeyPicker && m.keyPicker != nil {
			keyPickerView := m.keyPicker.View()
			centeredKeyPicker := lipgloss.Place(
				m.width,
				m.height,
				lipgloss.Center,
				lipgloss.Center,
				keyPickerView,
			)
			return centeredKeyPicker
		}

		return baseView

	case ViewDelete:
		if m.deleteConfirm == nil {
			m.viewMode = ViewList
			return m.list.View()
		}
		// Center the delete confirmation overlay
		deleteView := m.deleteConfirm.View()
		centeredDelete := lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			deleteView,
		)
		return centeredDelete

	default:
		return "Unknown view mode"
	}
}
