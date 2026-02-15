# Phase 03: Connection & Navigation - Research

**Researched:** 2026-02-14
**Domain:** Fuzzy search, SSH connection handoff, keyboard navigation, connection history
**Confidence:** HIGH

## Summary

Phase 3 adds search, connection, and navigation capabilities to the Phase 2 TUI. The critical technical challenges are: (1) implementing fuzzy search across multiple fields (Host, Hostname, User), (2) handing off terminal control to SSH seamlessly via `tea.ExecProcess`, (3) supporting comprehensive keyboard navigation (Vim + arrow keys), and (4) tracking connection history for "last connected" preselection.

The Go ecosystem provides battle-tested solutions for all domains: `sahilm/fuzzy` for fuzzy matching (used by Bubbles list internally, optimized for interactive search), `tea.ExecProcess` for blocking external command execution with automatic terminal restoration, Bubbles `textinput` + `key` + `help` components for search bar and keyboard navigation, and standard library `encoding/json` for append-only history files.

**Key insight:** The "silent handoff" to SSH is achieved by `tea.ExecProcess` + `exec.Command` with stdin/stdout/stderr connected to `os.Stdin`/`os.Stdout`/`os.Stderr`. This creates the illusion of running `ssh` directly — the TUI disappears, SSH takes over, and when SSH exits, Bubbletea automatically restores the TUI state. Phase 2's Enter-for-details is reassigned to Tab (and a Vim alternative like `i` or `l`) to free up Enter for connection.

**Primary recommendation:** Use `sahilm/fuzzy` with custom `Source` interface for multi-field search, `textinput` component for filter bar, `key.Binding` + `help` component for persistent footer, `tea.ExecProcess` for SSH handoff, and JSON append-only file for connection history. Structure search as always-on filter (not a separate mode) with Esc to clear/defocus.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Search & filtering:** Always-on filter bar (visible at top or bottom), fuzzy matching (e.g., "prd" matches "production-server"), matches Host/Hostname/User fields, Esc clears and defocuses search bar returning to list navigation, "No matches" empty state when search returns zero results
- **Connection flow:** Enter on selected server connects immediately (no confirmation), silent handoff to SSH (TUI disappears, SSH takes over terminal), Phase 2's Enter-for-details reassigned to Tab (and a Vim alternative like `i`, `l`, or `o`), SSH connection failure shows native SSH error output then returns to TUI on keypress
- **Post-connection experience:** Default: exit to shell after SSH session ends (configurable option to return to TUI instead), on relaunch: preselect last server connected from current working directory path, last-connected indicator shown next to recently-connected servers (subtle marker or timestamp), connection history stored in separate history file (not in TOML app config)
- **Keyboard navigation:** Full Vim navigation (j/k, g/G, Ctrl+d/u for half-page scroll), arrow keys/Page Up/Down/Home/End for non-Vim users, q quits from list view (when search not focused), Ctrl+C/Esc works everywhere as fallback, persistent footer bar showing key hints (e.g., "Enter: connect | Tab: details | /: search | q: quit")

### Claude's Discretion
- Filter bar placement (top vs bottom)
- Fuzzy matching algorithm/library choice
- Exact footer bar content and styling
- History file format and location
- Last-connected indicator visual design (icon, timestamp, or both)
- Vim alternative key for detail view (e.g., `i`, `l`, or `o`)
- How "return to TUI" config option is named and structured

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| sahilm/fuzzy | latest | Fuzzy string matching | Used internally by Bubbles list, optimized for filenames/code symbols (like server names), millisecond performance, rank-sorted results, external-dependency-free |
| charmbracelet/bubbles/textinput | v1.0.0 | Search filter bar | Official Charm component, supports focus/blur, validation, placeholder text, Vim-style cursor movement |
| charmbracelet/bubbles/key | v1.0.0 | Keyboard shortcut management | Official Charm component, user-definable keymaps, integrates with help component |
| charmbracelet/bubbles/help | v1.0.0 | Persistent footer help view | Official Charm component, auto-generates from keymaps, single/multi-line modes, graceful truncation |
| tea.ExecProcess (Bubbletea) | built-in | SSH handoff | Built-in function for spawning interactive commands (SSH, vim, shells), blocking execution, automatic terminal restoration |
| encoding/json (stdlib) | stdlib | Connection history | Standard library, append-mode file I/O for log-like history, structured data with timestamps |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| os/exec (stdlib) | stdlib | SSH command construction | Always — `exec.Command("ssh", args...)` passed to `tea.ExecProcess` |
| os.Stdin/Stdout/Stderr (stdlib) | stdlib | Terminal I/O connection | Set `cmd.Stdin = os.Stdin` etc. for silent handoff to SSH |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| sahilm/fuzzy | lithammer/fuzzysearch | Simpler API but less optimized for interactive search, no rank scoring |
| sahilm/fuzzy | Custom strings.Contains | No fuzzy matching, poor UX ("prd" won't match "production-server") |
| textinput component | Custom input handling | Reinventing cursor movement, text selection, clipboard support — not worth it |
| JSON history file | SQLite database | Overkill for append-only logs, adds dependency, complicates deployment (single binary goal) |
| JSON history file | TOML history file | TOML not optimized for append-mode (requires full file rewrite), JSON is standard for log-like data |

**Installation:**
```bash
go get github.com/sahilm/fuzzy@latest
go get github.com/charmbracelet/bubbles/textinput@v1.0.0
go get github.com/charmbracelet/bubbles/key@v1.0.0
go get github.com/charmbracelet/bubbles/help@v1.0.0
# tea.ExecProcess, os/exec, encoding/json are built-in
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── config/              # SSH config parsing (from Phase 2)
│   ├── parser.go
│   └── host.go
├── history/             # Connection history tracking
│   ├── history.go      # History file management
│   └── entry.go        # History entry model
├── tui/                 # TUI presentation layer
│   ├── model.go        # Bubbletea root model
│   ├── list_view.go    # Server list view (from Phase 2)
│   ├── detail_view.go  # Detail view (from Phase 2)
│   ├── search.go       # Search/filter logic
│   ├── keys.go         # Key bindings (Vim + standard)
│   ├── styles.go       # Lipgloss styles
│   └── messages.go     # Custom Bubbletea messages
└── ssh/                 # SSH connection handling
    └── connect.go       # SSH command construction
```

### Pattern 1: Always-On Filter with Fuzzy Matching
**What:** Search bar always visible, typing immediately filters list with fuzzy matching
**When to use:** Real-time search UX, no mode switching required
**Example:**
```go
// internal/tui/search.go
import (
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/sahilm/fuzzy"
)

type SearchableHost struct {
    Host     config.SSHHost
}

// Implement fuzzy.Source interface for multi-field matching
type HostSource []SearchableHost

func (h HostSource) String(i int) string {
    // Concatenate all searchable fields
    // Search matches on: Host name, Hostname, User
    host := h[i].Host
    return host.Name + " " + host.Hostname + " " + host.User
}

func (h HostSource) Len() int {
    return len(h)
}

// In your model:
type Model struct {
    allHosts       []SearchableHost  // All hosts from config
    filteredHosts  []SearchableHost  // Hosts after fuzzy filter
    searchInput    textinput.Model
    searchFocused  bool
}

func (m Model) filterHosts() {
    query := m.searchInput.Value()

    if query == "" {
        m.filteredHosts = m.allHosts
        return
    }

    // Fuzzy search across all fields
    source := HostSource(m.allHosts)
    matches := fuzzy.FindFrom(query, source)

    m.filteredHosts = make([]SearchableHost, len(matches))
    for i, match := range matches {
        m.filteredHosts[i] = m.allHosts[match.Index]
    }
}
```

### Pattern 2: Silent SSH Handoff with tea.ExecProcess
**What:** TUI disappears, SSH takes over terminal, TUI resumes on SSH exit
**When to use:** Always, for SSH connections
**Example:**
```go
// internal/ssh/connect.go
import (
    "os"
    "os/exec"
    tea "github.com/charmbracelet/bubbletea"
)

type SSHFinishedMsg struct {
    err error
}

func ConnectSSH(host config.SSHHost) tea.Cmd {
    // Construct SSH command using host's config name
    // This leverages user's existing SSH config for all options
    c := exec.Command("ssh", host.Name)

    // CRITICAL: Connect to terminal stdin/stdout/stderr for silent handoff
    c.Stdin = os.Stdin
    c.Stdout = os.Stdout
    c.Stderr = os.Stderr

    // ExecProcess blocks until SSH exits, then sends message
    return tea.ExecProcess(c, func(err error) tea.Msg {
        return SSHFinishedMsg{err: err}
    })
}

// In your Update function:
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if !m.searchFocused && msg.String() == "enter" {
            // Connect to selected server
            if i, ok := m.list.SelectedItem().(hostItem); ok {
                return m, ConnectSSH(i.host)
            }
        }

    case SSHFinishedMsg:
        // SSH session ended
        if msg.err != nil {
            // SSH connection failed - error already printed by SSH
            // User saw native SSH error output
            // Just return to TUI (no need to show error again)
            return m, nil
        }

        // SSH session ended normally
        // Check config option: exit or return to TUI
        if m.config.ExitAfterSSH {
            return m, tea.Quit
        }

        // Otherwise, stay in TUI
        return m, nil
    }

    return m, nil
}
```

### Pattern 3: Comprehensive Keyboard Navigation (Vim + Standard)
**What:** Vim keybindings for power users, standard keys for everyone else
**When to use:** Always, for inclusive UX
**Example:**
```go
// internal/tui/keys.go
import (
    "github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
    // Navigation (Vim + Standard)
    Up         key.Binding
    Down       key.Binding
    PageUp     key.Binding
    PageDown   key.Binding
    HalfPageUp key.Binding
    HalfPageDown key.Binding
    Top        key.Binding
    Bottom     key.Binding

    // Actions
    Connect    key.Binding
    Details    key.Binding
    Search     key.Binding
    Quit       key.Binding

    // Search mode
    ClearSearch key.Binding
}

var DefaultKeyMap = KeyMap{
    Up: key.NewBinding(
        key.WithKeys("k", "up"),
        key.WithHelp("↑/k", "up"),
    ),
    Down: key.NewBinding(
        key.WithKeys("j", "down"),
        key.WithHelp("↓/j", "down"),
    ),
    PageUp: key.NewBinding(
        key.WithKeys("pgup"),
        key.WithHelp("pgup", "page up"),
    ),
    PageDown: key.NewBinding(
        key.WithKeys("pgdown"),
        key.WithHelp("pgdn", "page down"),
    ),
    HalfPageUp: key.NewBinding(
        key.WithKeys("ctrl+u"),
        key.WithHelp("ctrl+u", "½ page up"),
    ),
    HalfPageDown: key.NewBinding(
        key.WithKeys("ctrl+d"),
        key.WithHelp("ctrl+d", "½ page down"),
    ),
    Top: key.NewBinding(
        key.WithKeys("g", "home"),
        key.WithHelp("g/home", "top"),
    ),
    Bottom: key.NewBinding(
        key.WithKeys("G", "end"),
        key.WithHelp("G/end", "bottom"),
    ),
    Connect: key.NewBinding(
        key.WithKeys("enter"),
        key.WithHelp("enter", "connect"),
    ),
    Details: key.NewBinding(
        key.WithKeys("tab", "i"),  // Tab + Vim alternative
        key.WithHelp("tab/i", "details"),
    ),
    Search: key.NewBinding(
        key.WithKeys("/"),
        key.WithHelp("/", "search"),
    ),
    Quit: key.NewBinding(
        key.WithKeys("q", "ctrl+c"),
        key.WithHelp("q", "quit"),
    ),
    ClearSearch: key.NewBinding(
        key.WithKeys("esc"),
        key.WithHelp("esc", "clear search"),
    ),
}

// Implement help.KeyMap interface
func (k KeyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Connect, k.Details, k.Search, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Up, k.Down, k.PageUp, k.PageDown},
        {k.HalfPageUp, k.HalfPageDown, k.Top, k.Bottom},
        {k.Connect, k.Details, k.Search, k.Quit},
    }
}
```

### Pattern 4: Persistent Footer with Help Component
**What:** Always-visible footer showing key hints, auto-generated from keymaps
**When to use:** Always, for discoverability
**Example:**
```go
// In your model:
type Model struct {
    keys KeyMap
    help help.Model
    // ... other fields
}

func initialModel() Model {
    return Model{
        keys: DefaultKeyMap,
        help: help.New(),
    }
}

func (m Model) View() string {
    var view strings.Builder

    // Search bar (top or bottom - Claude's discretion)
    if m.searchFocused {
        view.WriteString("Search: ")
    } else {
        view.WriteString("Filter: ")
    }
    view.WriteString(m.searchInput.View())
    view.WriteString("\n")

    // Main list
    view.WriteString(m.list.View())
    view.WriteString("\n")

    // Persistent footer with help
    // Set help width to match terminal
    m.help.Width = m.width
    view.WriteString(m.help.View(m.keys))

    return view.String()
}
```

### Pattern 5: Connection History Tracking
**What:** Append-only JSON file tracking connections per working directory
**When to use:** For "last connected from this path" preselection
**Example:**
```go
// internal/history/entry.go
type HistoryEntry struct {
    Timestamp   time.Time `json:"timestamp"`
    WorkingDir  string    `json:"working_dir"`
    HostName    string    `json:"host_name"`
    Hostname    string    `json:"hostname"`
    User        string    `json:"user"`
}

// internal/history/history.go
import (
    "encoding/json"
    "os"
    "path/filepath"
)

func GetHistoryPath() string {
    home, _ := os.UserHomeDir()
    // Store alongside SSH config
    return filepath.Join(home, ".ssh", "ssherpa_history.json")
}

func RecordConnection(host config.SSHHost) error {
    cwd, err := os.Getwd()
    if err != nil {
        cwd = "" // Fallback to empty if can't get working dir
    }

    entry := HistoryEntry{
        Timestamp:  time.Now(),
        WorkingDir: cwd,
        HostName:   host.Name,
        Hostname:   host.Hostname,
        User:       host.User,
    }

    // Open in append mode (create if doesn't exist)
    f, err := os.OpenFile(
        GetHistoryPath(),
        os.O_CREATE|os.O_APPEND|os.O_WRONLY,
        0600,
    )
    if err != nil {
        return fmt.Errorf("open history: %w", err)
    }
    defer f.Close()

    // Append JSON line
    enc := json.NewEncoder(f)
    if err := enc.Encode(entry); err != nil {
        return fmt.Errorf("write history: %w", err)
    }

    return nil
}

func GetLastConnectedForPath(path string) (*HistoryEntry, error) {
    f, err := os.Open(GetHistoryPath())
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil // No history yet
        }
        return nil, fmt.Errorf("open history: %w", err)
    }
    defer f.Close()

    // Read all entries (file is append-only, not huge)
    var entries []HistoryEntry
    dec := json.NewDecoder(f)
    for {
        var entry HistoryEntry
        if err := dec.Decode(&entry); err != nil {
            if err == io.EOF {
                break
            }
            // Skip malformed lines
            continue
        }
        entries = append(entries, entry)
    }

    // Find most recent entry for this path
    var latest *HistoryEntry
    for i := len(entries) - 1; i >= 0; i-- {
        if entries[i].WorkingDir == path {
            latest = &entries[i]
            break
        }
    }

    return latest, nil
}

// Usage in TUI initialization:
func (m Model) Init() tea.Cmd {
    cwd, _ := os.Getwd()
    lastConn, err := history.GetLastConnectedForPath(cwd)

    if err == nil && lastConn != nil {
        // Preselect this host in the list
        m.preselectHost(lastConn.HostName)
    }

    return m.list.StartSpinner()
}
```

### Pattern 6: Search Focus Management
**What:** Esc clears search and returns focus to list, slash focuses search
**When to use:** Always, for intuitive search UX
**Example:**
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Search mode handling
        if m.searchFocused {
            switch {
            case key.Matches(msg, m.keys.ClearSearch):
                // Esc: clear search and defocus
                m.searchInput.SetValue("")
                m.searchInput.Blur()
                m.searchFocused = false
                m.filterHosts() // Re-show all hosts
                return m, nil

            default:
                // Pass other keys to textinput
                var cmd tea.Cmd
                m.searchInput, cmd = m.searchInput.Update(msg)

                // Re-filter on every keystroke
                m.filterHosts()
                m.updateListItems()

                return m, cmd
            }
        }

        // List mode handling
        switch {
        case key.Matches(msg, m.keys.Search):
            // Slash: focus search bar
            m.searchFocused = true
            return m, m.searchInput.Focus()

        case key.Matches(msg, m.keys.Quit):
            // q: quit (only works when search not focused)
            return m, tea.Quit

        case key.Matches(msg, m.keys.Connect):
            // Enter: connect to selected server
            if i, ok := m.list.SelectedItem().(hostItem); ok {
                // Record connection before handing off
                history.RecordConnection(i.host)
                return m, ConnectSSH(i.host)
            }

        case key.Matches(msg, m.keys.Details):
            // Tab or 'i': show details
            if i, ok := m.list.SelectedItem().(hostItem); ok {
                m.viewMode = ViewDetail
                m.detailHost = &i.host
                return m, nil
            }
        }
    }

    // Delegate to list component
    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}
```

### Anti-Patterns to Avoid
- **❌ Modal search (separate search mode):** Slows down UX, requires mode switching — use always-on filter instead
- **❌ Synchronous SSH connection (blocking main thread):** TUI will freeze — always use `tea.ExecProcess` which handles blocking correctly
- **❌ Showing custom error UI after SSH fails:** SSH already printed its error to terminal — don't duplicate, just return to TUI
- **❌ Concatenating history entries in memory before writing:** Inefficient for large histories — use append-mode file I/O
- **❌ Hardcoded "return to TUI" behavior:** Some users want to return to shell immediately — make it configurable

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Fuzzy matching | Custom substring search or regexp | sahilm/fuzzy | Rank scoring (first char match, camel case, separator-following, adjacency), Unicode handling, optimized for interactive search (millisecond results), highlight-matched-characters support |
| Text input with cursor movement | Manual cursor tracking, text buffer manipulation | bubbles/textinput | Clipboard support, word-level navigation (Ctrl+A, Ctrl+E, Alt+Left/Right), Vim-style movement, validation, suggestions/autocomplete, echo modes (password input) |
| Keyboard shortcut management | switch statement on raw key strings | bubbles/key | User-definable keymaps, multiple keys per binding, disabled-state handling, automatic integration with help component |
| Help footer generation | Manual string formatting of key hints | bubbles/help | Auto-generated from keymaps, graceful truncation, single/multi-line modes, respects disabled bindings |
| Terminal handoff to SSH | os/exec.Run with manual terminal control | tea.ExecProcess | Automatic terminal state save/restore, blocking execution with TUI pause, cross-platform (Windows signal handling), integrates with Bubbletea message loop |
| Connection history | In-memory tracking, session-based history | JSON append-only file | Persists across app restarts, simple append-mode I/O (no locking needed), structured data, easy inspection/debugging |

**Key insight:** `tea.ExecProcess` is the secret to "silent handoff" — it handles all terminal state management (switching to raw mode, restoring after command exits) automatically. Trying to hand-roll this with manual signal handling and terminal control is error-prone and non-portable (Windows vs Unix differences). The fuzzy search experience depends on ranking — simple substring matching doesn't feel "fuzzy" because results aren't sorted by relevance.

## Common Pitfalls

### Pitfall 1: Not Connecting SSH Command to Terminal Stdin/Stdout/Stderr
**What goes wrong:** SSH prompts (password, host key verification) don't work, connection hangs or fails
**Why it happens:** `exec.Command` defaults to `/dev/null` for stdin/stdout/stderr — SSH can't interact with user
**How to avoid:** **ALWAYS** set `cmd.Stdin = os.Stdin`, `cmd.Stdout = os.Stdout`, `cmd.Stderr = os.Stderr` before passing to `tea.ExecProcess`
**Warning signs:** SSH connection hangs silently, password prompts not visible, host key verification fails
**Example:**
```go
// ❌ BAD: Stdin/Stdout/Stderr not connected
func ConnectSSH(host string) tea.Cmd {
    c := exec.Command("ssh", host)
    return tea.ExecProcess(c, sshFinishedCallback)
}

// ✅ GOOD: Connect to terminal I/O
func ConnectSSH(host string) tea.Cmd {
    c := exec.Command("ssh", host)
    c.Stdin = os.Stdin    // SSH can read password input
    c.Stdout = os.Stdout  // SSH output visible
    c.Stderr = os.Stderr  // SSH errors visible
    return tea.ExecProcess(c, sshFinishedCallback)
}
```

### Pitfall 2: Fuzzy Search Not Filtering on Every Keystroke
**What goes wrong:** User types "prod" but list doesn't update until they press Enter or Tab
**Why it happens:** Only filtering on specific key events instead of every textinput change
**How to avoid:** Call `filterHosts()` in the **default** case after updating textinput, not just on Enter/Tab
**Warning signs:** Search feels laggy, users press Enter expecting search to execute
**Example:**
```go
// ❌ BAD: Only filtering on Enter
case tea.KeyMsg:
    if m.searchFocused {
        if msg.String() == "enter" {
            m.filterHosts() // TOO LATE!
        }
        m.searchInput, cmd = m.searchInput.Update(msg)
    }

// ✅ GOOD: Filter on every keystroke
case tea.KeyMsg:
    if m.searchFocused {
        if msg.String() == "esc" {
            m.searchInput.SetValue("")
            m.searchFocused = false
            m.filterHosts() // Clear filter
            return m, nil
        }

        // Update input
        var cmd tea.Cmd
        m.searchInput, cmd = m.searchInput.Update(msg)

        // Re-filter IMMEDIATELY
        m.filterHosts()
        m.updateListItems() // Update list with new filtered results

        return m, cmd
    }
```

### Pitfall 3: Search Bar Not Defocused on Esc
**What goes wrong:** User presses Esc expecting to return to list navigation, but search bar still active
**Why it happens:** Handling Esc in textinput's Update (which does nothing) instead of in your own Update logic
**How to avoid:** Check for Esc **before** passing message to textinput.Update
**Warning signs:** After pressing Esc, typing still goes to search bar instead of triggering shortcuts (like 'q' to quit)
**Example:**
```go
// ❌ BAD: Esc passed to textinput (does nothing)
if m.searchFocused {
    m.searchInput, cmd = m.searchInput.Update(msg)
    // Esc key was consumed by textinput, no defocus happened
    return m, cmd
}

// ✅ GOOD: Intercept Esc before textinput sees it
if m.searchFocused {
    if msg.String() == "esc" {
        m.searchInput.SetValue("")
        m.searchInput.Blur()
        m.searchFocused = false
        m.filterHosts() // Clear filter
        return m, nil
    }

    // Other keys go to textinput
    m.searchInput, cmd = m.searchInput.Update(msg)
    m.filterHosts()
    return m, cmd
}
```

### Pitfall 4: Empty Search Results Show Blank List
**What goes wrong:** User types search that matches nothing, sees empty list with no explanation
**Why it happens:** Not detecting zero-results case and showing empty state message
**How to avoid:** After filtering, check if `len(m.filteredHosts) == 0` and query is not empty — render "No matches" message
**Warning signs:** User asks "did search break?" when no results, unclear if search is working or app crashed
**Example:**
```go
// ❌ BAD: No empty state for search
func (m Model) View() string {
    return m.list.View() // Empty if no matches, no context
}

// ✅ GOOD: Show empty state for zero results
func (m Model) View() string {
    var view strings.Builder

    view.WriteString(m.searchInput.View())
    view.WriteString("\n\n")

    // Check for empty search results
    if m.searchInput.Value() != "" && len(m.filteredHosts) == 0 {
        emptyStyle := lipgloss.NewStyle().
            Foreground(lipgloss.Color("241")).
            Italic(true).
            Padding(1)

        view.WriteString(emptyStyle.Render(
            "No matches for \"" + m.searchInput.Value() + "\"\n" +
            "Press Esc to clear search",
        ))
    } else {
        view.WriteString(m.list.View())
    }

    view.WriteString("\n")
    view.WriteString(m.help.View(m.keys))

    return view.String()
}
```

### Pitfall 5: Not Recording Connection History Before SSH Handoff
**What goes wrong:** User connects to server, app exits or crashes, connection not recorded in history
**Why it happens:** Recording history after SSH session (which might never return if app exits)
**How to avoid:** Record connection **before** calling `tea.ExecProcess`, not in the SSH finished callback
**Warning signs:** History file empty even after connections, "last connected" preselection not working
**Example:**
```go
// ❌ BAD: Recording after SSH session (may never execute if app exits)
case SSHFinishedMsg:
    history.RecordConnection(m.lastConnectedHost)
    if m.config.ExitAfterSSH {
        return m, tea.Quit
    }

// ✅ GOOD: Record BEFORE handing off to SSH
case tea.KeyMsg:
    if msg.String() == "enter" && !m.searchFocused {
        if i, ok := m.list.SelectedItem().(hostItem); ok {
            // Record NOW, before SSH handoff
            if err := history.RecordConnection(i.host); err != nil {
                // Log error but don't block connection
                log.Printf("failed to record history: %v", err)
            }

            return m, ConnectSSH(i.host)
        }
    }
```

### Pitfall 6: Vim Keybindings Conflict with Search Input
**What goes wrong:** User types in search bar, presses 'j' or 'k', list scrolls instead of typing character
**Why it happens:** Not checking search focus state before processing Vim keys
**How to avoid:** When search is focused, **only** handle Esc — all other keys go to textinput
**Warning signs:** Can't type certain letters in search bar, search input feels broken
**Example:**
```go
// ❌ BAD: Vim keys always active
case tea.KeyMsg:
    switch {
    case key.Matches(msg, m.keys.Up):
        // Moves list selection even when typing in search!
        m.list.CursorUp()
    }

// ✅ GOOD: Disable Vim keys when search is focused
case tea.KeyMsg:
    // Search mode has priority
    if m.searchFocused {
        if msg.String() == "esc" {
            // Only Esc handled in search mode
            m.searchInput.SetValue("")
            m.searchFocused = false
            m.searchInput.Blur()
            m.filterHosts()
            return m, nil
        }

        // Everything else goes to textinput (including j/k)
        var cmd tea.Cmd
        m.searchInput, cmd = m.searchInput.Update(msg)
        m.filterHosts()
        return m, cmd
    }

    // List mode: Vim keys active
    switch {
    case key.Matches(msg, m.keys.Up):
        m.list.CursorUp()
    }
```

### Pitfall 7: Not Handling SSH Connection Failure Gracefully
**What goes wrong:** SSH fails (wrong password, connection timeout), app exits or shows confusing error
**Why it happens:** Not handling `SSHFinishedMsg.err` or trying to display error UI on top of SSH's native output
**How to avoid:** SSH already printed error to terminal — just return to TUI silently on error, optionally show brief status
**Warning signs:** Double error messages, TUI rendering corrupted after SSH error, app exits on failed connection
**Example:**
```go
// ❌ BAD: Trying to show custom error UI
case SSHFinishedMsg:
    if msg.err != nil {
        // SSH already printed error, this creates duplicate/confusing output
        m.errorMsg = "SSH connection failed: " + msg.err.Error()
        return m, nil
    }

// ✅ GOOD: Trust SSH's error output, just return to TUI
case SSHFinishedMsg:
    if msg.err != nil {
        // SSH already printed its error message
        // Just return to TUI (user saw the error)
        // Optionally could wait for keypress here
        return m, nil
    }

    // Connection succeeded and ended normally
    if m.config.ExitAfterSSH {
        return m, tea.Quit
    }

    return m, nil
```

### Pitfall 8: Help Footer Not Updating Based on Mode
**What goes wrong:** Help footer shows "q: quit" even when search is focused (pressing 'q' types 'q' in search, doesn't quit)
**Why it happens:** Using static help that doesn't change based on search focus state
**How to avoid:** Define separate keymaps for list mode vs search mode, or disable irrelevant keys when search is focused
**Warning signs:** Help hints misleading, keys shown in help don't work as described
**Example:**
```go
// ❌ BAD: Static help, same in all modes
func (m Model) View() string {
    view.WriteString(m.help.View(m.keys))
}

// ✅ GOOD: Conditional help based on mode
func (m Model) View() string {
    // Different help based on mode
    if m.searchFocused {
        // In search mode, only Esc is relevant
        searchKeys := KeyMap{
            ClearSearch: m.keys.ClearSearch,
        }
        view.WriteString(m.help.View(searchKeys))
    } else {
        // In list mode, show all keys
        view.WriteString(m.help.View(m.keys))
    }
}

// OR: Dynamically enable/disable keys
func (m Model) updateKeyBindings() {
    m.keys.Quit.SetEnabled(!m.searchFocused)
    m.keys.Connect.SetEnabled(!m.searchFocused)
    m.keys.Details.SetEnabled(!m.searchFocused)
    m.keys.Search.SetEnabled(!m.searchFocused)

    // Help component automatically excludes disabled keys
}
```

## Code Examples

Verified patterns from official sources:

### Basic Fuzzy Search with sahilm/fuzzy
```go
// Source: https://pkg.go.dev/github.com/sahilm/fuzzy
import (
    "fmt"
    "github.com/sahilm/fuzzy"
)

// Simple string slice search
func basicSearch() {
    pattern := "prd"
    data := []string{
        "production-server",
        "dev-server",
        "prod-db",
    }

    matches := fuzzy.Find(pattern, data)

    for _, match := range matches {
        fmt.Println(match.Str)           // Matched string
        fmt.Println(match.MatchedIndexes) // Character positions that matched
        fmt.Println(match.Score)          // Rank score
    }
    // Output: "production-server" (matches "p-r-d")
    //         "prod-db" (matches "pr-d")
}

// Multi-field search with custom Source
type SSHHost struct {
    Name     string
    Hostname string
    User     string
}

type HostSource []SSHHost

func (h HostSource) String(i int) string {
    // Concatenate all searchable fields
    return h[i].Name + " " + h[i].Hostname + " " + h[i].User
}

func (h HostSource) Len() int {
    return len(h)
}

func multiFieldSearch() {
    hosts := HostSource{
        {Name: "prod", Hostname: "production.example.com", User: "admin"},
        {Name: "dev", Hostname: "dev.example.com", User: "developer"},
    }

    matches := fuzzy.FindFrom("admin", hosts)

    for _, match := range matches {
        host := hosts[match.Index]
        fmt.Printf("%s@%s\n", host.User, host.Hostname)
    }
}
```

### SSH Connection with tea.ExecProcess
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbletea
import (
    "os"
    "os/exec"
    tea "github.com/charmbracelet/bubbletea"
)

type SSHFinishedMsg struct {
    err error
}

func connectToServer(hostName string) tea.Cmd {
    // Use SSH config alias (leverages user's existing config)
    c := exec.Command("ssh", hostName)

    // CRITICAL: Connect to terminal I/O for interactive session
    c.Stdin = os.Stdin
    c.Stdout = os.Stdout
    c.Stderr = os.Stderr

    // ExecProcess blocks TUI, runs command, restores TUI after
    return tea.ExecProcess(c, func(err error) tea.Msg {
        return SSHFinishedMsg{err: err}
    })
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "enter" {
            return m, connectToServer("production")
        }

    case SSHFinishedMsg:
        // SSH session ended (success or failure)
        if msg.err != nil {
            // Error already displayed by SSH
            return m, nil
        }

        // Exit or return to TUI based on config
        if m.exitAfterSSH {
            return m, tea.Quit
        }
        return m, nil
    }

    return m, nil
}
```

### Textinput Component with Focus Management
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbles/textinput
import (
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
    searchInput textinput.Model
    focused     bool
}

func initialModel() Model {
    ti := textinput.New()
    ti.Placeholder = "Search servers..."
    ti.CharLimit = 50
    ti.Width = 40

    return Model{
        searchInput: ti,
        focused:     false,
    }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if m.focused {
            // Esc: defocus and clear
            if msg.String() == "esc" {
                m.searchInput.SetValue("")
                m.searchInput.Blur()
                m.focused = false
                return m, nil
            }

            // Pass to textinput
            var cmd tea.Cmd
            m.searchInput, cmd = m.searchInput.Update(msg)
            return m, cmd
        } else {
            // '/': focus search
            if msg.String() == "/" {
                m.focused = true
                return m, m.searchInput.Focus()
            }
        }
    }

    return m, nil
}

func (m Model) View() string {
    return m.searchInput.View()
}
```

### Key Bindings with Help Component
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbles/key
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbles/help
import (
    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/bubbles/help"
)

type KeyMap struct {
    Up      key.Binding
    Down    key.Binding
    Connect key.Binding
    Quit    key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Connect, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Up, k.Down},
        {k.Connect, k.Quit},
    }
}

var DefaultKeyMap = KeyMap{
    Up: key.NewBinding(
        key.WithKeys("k", "up"),
        key.WithHelp("↑/k", "up"),
    ),
    Down: key.NewBinding(
        key.WithKeys("j", "down"),
        key.WithHelp("↓/j", "down"),
    ),
    Connect: key.NewBinding(
        key.WithKeys("enter"),
        key.WithHelp("enter", "connect"),
    ),
    Quit: key.NewBinding(
        key.WithKeys("q", "ctrl+c"),
        key.WithHelp("q", "quit"),
    ),
}

type Model struct {
    keys KeyMap
    help help.Model
}

func initialModel() Model {
    h := help.New()
    h.Width = 80

    return Model{
        keys: DefaultKeyMap,
        help: h,
    }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, m.keys.Up):
            // Handle up
        case key.Matches(msg, m.keys.Down):
            // Handle down
        case key.Matches(msg, m.keys.Connect):
            // Handle connect
        case key.Matches(msg, m.keys.Quit):
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m Model) View() string {
    return "Content\n\n" + m.help.View(m.keys)
}
```

### Connection History with JSON Append
```go
// Source: https://www.sohamkamani.com/golang/json/
// Source: https://www.dataset.com/blog/effective-strategies-and-best-practices-for-go-logging/
import (
    "encoding/json"
    "os"
    "time"
)

type HistoryEntry struct {
    Timestamp  time.Time `json:"timestamp"`
    WorkingDir string    `json:"working_dir"`
    HostName   string    `json:"host_name"`
    Hostname   string    `json:"hostname"`
    User       string    `json:"user"`
}

func recordConnection(hostName, hostname, user string) error {
    cwd, _ := os.Getwd()

    entry := HistoryEntry{
        Timestamp:  time.Now(),
        WorkingDir: cwd,
        HostName:   hostName,
        Hostname:   hostname,
        User:       user,
    }

    // Open in append mode
    f, err := os.OpenFile(
        "~/.ssh/ssherpa_history.json",
        os.O_CREATE|os.O_APPEND|os.O_WRONLY,
        0600,
    )
    if err != nil {
        return err
    }
    defer f.Close()

    // Append JSON line
    enc := json.NewEncoder(f)
    return enc.Encode(entry)
}

func getLastForPath(path string) (*HistoryEntry, error) {
    f, err := os.Open("~/.ssh/ssherpa_history.json")
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil
        }
        return nil, err
    }
    defer f.Close()

    var entries []HistoryEntry
    dec := json.NewDecoder(f)

    for {
        var entry HistoryEntry
        if err := dec.Decode(&entry); err != nil {
            break // EOF or malformed entry
        }
        entries = append(entries, entry)
    }

    // Find most recent for this path
    for i := len(entries) - 1; i >= 0; i-- {
        if entries[i].WorkingDir == path {
            return &entries[i], nil
        }
    }

    return nil, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Substring search only | Fuzzy matching with rank scoring | 2015+ (Sublime/VSCode popularized) | Users expect "prd" to match "production-server" — exact match feels broken |
| Modal search (press '/', type, press Enter) | Always-on filter (type immediately filters) | 2020+ (modern TUIs) | Faster UX, no mode switching, real-time feedback |
| Manual terminal control for external commands | tea.ExecProcess automatic state management | Bubbletea v0.20+ (2022) | Cross-platform, no signal handling bugs, automatic restoration |
| Vim-only keybindings | Vim + standard arrow keys | Modern TUIs (2020+) | Inclusive UX, non-Vim users don't feel lost |
| Hardcoded help text | Auto-generated from keymaps | Bubbles v0.15+ (2021) | Help stays in sync with actual bindings, single-line/multi-line modes |
| SSH with custom Go client | System SSH via exec.Command | Standard practice | Leverages user's existing config/keys, handles all auth methods, no custom crypto code |

**Deprecated/outdated:**
- **Modal search dialogs:** Modern TUIs use always-on filters (like VSCode, Sublime) — modal search feels slow
- **Manual `syscall.SIGTSTP` handling:** Bubbletea handles terminal suspend/resume automatically — manual signal handling is error-prone
- **Go SSH client libraries for interactive sessions:** System `ssh` command handles all auth methods, config files, keys, agents — custom Go SSH clients require reimplementing all of this

## Open Questions

1. **Filter bar placement: top vs bottom?**
   - What we know: Top feels like "search box" (browser pattern), bottom keeps list at consistent vertical position
   - What's unclear: User preference, which feels more natural in a server-selection TUI
   - Recommendation: Start with **top** (most familiar pattern from browsers/editors), gather feedback, make configurable if users split 50/50

2. **Last-connected indicator: icon, timestamp, or both?**
   - What we know: Icon (•, ★) is subtle and quick to scan, timestamp ("2m ago") is informative but takes space
   - What's unclear: Whether users care about "when" or just "recently connected"
   - Recommendation: Start with **icon only** (★ or • in accent color next to host name), add timestamp on hover or in detail view if users request it

3. **Vim alternative key for detail view: i, l, or o?**
   - What we know: Tab is primary, need a Vim-style alternative for consistency
   - What's unclear: Which Vim key is most intuitive (i = "inspect", l = "look", o = "open")
   - Recommendation: Use **`i`** (mnemonic: "inspect" or "info"), familiar from file managers, doesn't conflict with list navigation

4. **History file location: ~/.ssh/ vs XDG dirs?**
   - What we know: `~/.ssh/` keeps SSH-related data together, XDG is "proper" but spreads config across dirs
   - What's unclear: User preference, whether XDG compliance matters for this tool
   - Recommendation: Use **`~/.ssh/ssherpa_history.json`** — keeps everything SSH-related in one place, easier to find/backup, follows OpenSSH config pattern

5. **Return-to-TUI config option naming?**
   - What we know: Default is "exit after SSH", need a flag to stay in TUI instead
   - What's unclear: Config key naming (exit_after_ssh vs return_to_tui vs stay_open)
   - Recommendation: Use **`return_to_tui_after_disconnect: false`** (default false = exit) — clear positive phrasing, explicit about timing

## Sources

### Primary (HIGH confidence)
- [sahilm/fuzzy Go Package](https://pkg.go.dev/github.com/sahilm/fuzzy) - Fuzzy search API, Source interface, rank scoring
- [sahilm/fuzzy GitHub](https://github.com/sahilm/fuzzy) - Multi-field search examples, performance characteristics
- [Bubbletea tea.ExecProcess](https://pkg.go.dev/github.com/charmbracelet/bubbletea) - ExecProcess API, blocking execution, terminal restoration
- [Bubbles textinput](https://pkg.go.dev/github.com/charmbracelet/bubbles/textinput) - v1.0.0 API, focus/blur, validation, cursor movement
- [Bubbles key](https://pkg.go.dev/github.com/charmbracelet/bubbles/key) - v1.0.0 API, key.Binding, key.Matches, keymaps
- [Bubbles help](https://pkg.go.dev/github.com/charmbracelet/bubbles/help) - v1.0.0 API, ShortHelp/FullHelp, KeyMap interface
- [Go os/exec package](https://pkg.go.dev/os/exec) - Command struct, Stdin/Stdout/Stderr configuration
- [Go encoding/json package](https://pkg.go.dev/encoding/json) - Encoder/Decoder for append-only file I/O

### Secondary (MEDIUM confidence)
- [Advanced command execution in Go with os/exec](https://blog.kowalczyk.info/article/wOYk/advanced-command-execution-in-go-with-osexec.html) - Connecting stdin/stdout/stderr patterns
- [Some Useful Patterns for Go's os/exec](https://www.dolthub.com/blog/2022-11-28-go-os-exec-patterns/) - Terminal I/O connection examples
- [A Complete Guide to JSON in Golang](https://www.sohamkamani.com/golang/json/) - JSON encoding/decoding best practices
- [Effective Strategies for Go Logging](https://www.dataset.com/blog/effective-strategies-and-best-practices-for-go-logging/) - Append-mode file I/O for structured logs
- [Tips for building Bubble Tea programs](https://leg100.github.io/en/posts/building-bubbletea-programs/) - TUI state management patterns
- [Building TUI with Bubbletea and Lipgloss](https://www.grootan.com/blogs/building-an-awesome-terminal-user-interface-using-go-bubble-tea-and-lip-gloss/) - Component integration patterns

### Tertiary (LOW confidence)
- [Go fuzzy search library comparison](https://medium.com/@galiherlanggadev/fuzzy-search-in-go-72f4a5d0dba0) - Blog post comparing fuzzy libraries (not official docs)
- [VimTea GitHub](https://github.com/kujtimiihoxha/vimtea) - Example of Vim keybindings in Bubbletea (community project, not official)
- [Bubbletea ExecProcess issue #431](https://github.com/charmbracelet/bubbletea/issues/431) - Known issue with ExecProcess stdout output (acknowledged but not blocking)
- [Bubbletea state machine pattern](https://zackproser.com/blog/bubbletea-state-machine) - Community blog post on view state management (good pattern but not official)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries are official Charm components (v1.0.0), sahilm/fuzzy is battle-tested (used by Bubbles internally), stdlib packages are stable
- Architecture: HIGH - Patterns verified from official documentation, tea.ExecProcess usage confirmed in pkg.go.dev, fuzzy Source interface documented
- Pitfalls: MEDIUM-HIGH - Stdin/stdout connection verified in os/exec docs, focus management patterns from Bubbles examples, search filtering patterns from interactive search UX best practices

**Research date:** 2026-02-14
**Valid until:** ~30 days (stable ecosystem, Bubbles v1.0.0 released Feb 2026, unlikely to change rapidly)
