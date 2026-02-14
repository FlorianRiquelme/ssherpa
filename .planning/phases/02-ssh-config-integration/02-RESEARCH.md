# Phase 02: SSH Config Integration - Research

**Researched:** 2026-02-14
**Domain:** SSH config parsing, TUI list/detail views, Go libraries
**Confidence:** HIGH

## Summary

Phase 2 requires parsing SSH config files (including Include directives) and displaying connections in a navigable TUI with detail views. The standard Go ecosystem provides mature libraries for both SSH config parsing and TUI components. The recommended stack is `kevinburke/ssh_config` for parsing (stable, widely-used, preserves comments) paired with Bubbles `list` component for navigation and Lipgloss for layout/styling.

**Key insight:** Don't hand-roll SSH config parsing or TUI components — both domains have deceptively complex edge cases (wildcard matching, Include recursion, terminal resize handling, color profile detection). Use battle-tested libraries.

**Primary recommendation:** Start with `kevinburke/ssh_config` + Bubbles `list` + Lipgloss layout. Structure code to isolate config parsing from TUI rendering to enable testing and future backend switching.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Server list layout:** Two lines per server entry — name + hostname on first line, user/port on second. Show: Name, Hostname, User, Port (key path reserved for detail view). Sorted alphabetically by Host name. Wildcard entries (Host *) displayed in a separate section at the bottom.
- **Config parsing scope:** Follow Include directives recursively. Ignore Match blocks entirely — only parse Host blocks. Use an existing Go SSH config parser library (e.g., kevinburke/ssh_config). Malformed/unreadable entries shown in the list with a warning indicator (not silently skipped).
- **Detail view behavior:** Enter key opens detail view (does NOT connect — that's Phase 3). Detail view shows ALL SSH config options set for the host (IdentityFile, ProxyJump, ForwardAgent, etc.). Includes which config file the entry was defined in (source tracking).
- **Initial launch experience:** Missing/empty `~/.ssh/config` shows friendly empty state message with guidance. Show loading indicator (spinner or status text) while parsing. Use accent colors to distinguish structural elements (hostnames, users, ports). Read sshjesus TOML app config to determine which backend to use (integrate with Phase 1 config system).

### Claude's Discretion
- Detail view layout style (right panel vs bottom panel vs inline expansion)
- Exact color palette and accent color choices
- Loading spinner implementation details
- Keyboard shortcut assignments beyond arrow keys and Enter
- How to display the warning indicator for malformed entries

### Deferred Ideas (OUT OF SCOPE)
- None — discussion stayed within phase scope

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| kevinburke/ssh_config | v1.4.0 | SSH config parsing | 511 projects use it, preserves comments, stable v1+ release, tracks source positions |
| charmbracelet/bubbles/list | v1.0.0 | List component | Official Charm component, built-in filtering/pagination/help, 14k+ stars |
| charmbracelet/lipgloss | latest | Styling/layout | Official Charm styling lib, CSS-like declarative API, automatic color profile detection |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| charmbracelet/bubbles/spinner | v1.0.0 | Loading indicator | During config file parsing |
| charmbracelet/bubbles/viewport | v1.0.0 | Scrollable detail view | If detail content exceeds terminal height |
| winder/bubblelayout | latest | Declarative layout manager | For complex split-panel layouts (optional, may be overkill) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| kevinburke/ssh_config | k0sproject/rig/v2/sshconfig | More features (partial Match support, token expansion) but pre-release (v2.0.0-alpha.3), not tested outside dev environment |
| kevinburke/ssh_config | mikkeloscar/sshconfig | Simpler API but less maintained, no comment preservation |
| Bubbles list | Custom list component | Reinventing filtering, pagination, help — not worth the effort |
| Lipgloss layout | Manual string concatenation | Loses responsive sizing, color profile detection, style inheritance |

**Installation:**
```bash
go get github.com/kevinburke/ssh_config@v1.4.0
go get github.com/charmbracelet/bubbles/list@v1.0.0
go get github.com/charmbracelet/bubbles/spinner@v1.0.0
go get github.com/charmbracelet/lipgloss@latest
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── config/              # SSH config parsing (domain logic)
│   ├── parser.go       # kevinburke/ssh_config wrapper
│   ├── host.go         # Host/connection models
│   └── parser_test.go
├── tui/                 # TUI presentation layer
│   ├── model.go        # Bubbletea root model
│   ├── list_view.go    # Server list view
│   ├── detail_view.go  # Connection detail view
│   ├── styles.go       # Lipgloss style definitions
│   └── messages.go     # Custom Bubbletea messages
└── backend/             # Backend integration (from Phase 1)
    └── interface.go     # Backend interface
```

### Pattern 1: Config Parser Wrapper
**What:** Wrap `kevinburke/ssh_config` to return domain models, not library types
**When to use:** Isolates parsing library from rest of codebase, easier testing
**Example:**
```go
// internal/config/parser.go
type SSHHost struct {
    Name         string   // Host pattern from config
    Hostname     string
    User         string
    Port         string
    IdentityFile []string
    AllOptions   map[string][]string // All config options
    SourceFile   string   // Which file defined this host
    SourceLine   int      // Line number in source file
    IsWildcard   bool     // True for Host * entries
    ParseError   error    // Non-nil if malformed
}

func ParseSSHConfig(path string) ([]SSHHost, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("open config: %w", err)
    }
    defer f.Close()

    cfg, err := ssh_config.Decode(f)
    if err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }

    var hosts []SSHHost
    for _, host := range cfg.Hosts {
        // Convert library types to domain models
        h := SSHHost{
            Name:       strings.Join(host.Patterns, " "),
            IsWildcard: containsWildcard(host.Patterns),
            SourceFile: path,
            AllOptions: make(map[string][]string),
        }

        // Extract all config values
        for _, node := range host.Nodes {
            if kv, ok := node.(*ssh_config.KV); ok {
                h.AllOptions[kv.Key] = append(h.AllOptions[kv.Key], kv.Value)

                // Populate known fields
                switch kv.Key {
                case "HostName":
                    h.Hostname = kv.Value
                case "User":
                    h.User = kv.Value
                case "Port":
                    h.Port = kv.Value
                case "IdentityFile":
                    h.IdentityFile = append(h.IdentityFile, kv.Value)
                }
            }
        }

        hosts = append(hosts, h)
    }

    return hosts, nil
}
```

### Pattern 2: List Item Implementation
**What:** Implement Bubbles `list.Item` and `list.DefaultItem` interfaces
**When to use:** Always, for custom list rendering
**Example:**
```go
// internal/tui/list_view.go
type hostItem struct {
    host config.SSHHost
}

func (h hostItem) FilterValue() string {
    return h.host.Name
}

func (h hostItem) Title() string {
    // Line 1: Name + Hostname
    return fmt.Sprintf("%s (%s)", h.host.Name, h.host.Hostname)
}

func (h hostItem) Description() string {
    // Line 2: User + Port
    user := h.host.User
    if user == "" {
        user = "default"
    }
    port := h.host.Port
    if port == "" {
        port = "22"
    }

    desc := fmt.Sprintf("User: %s | Port: %s", user, port)

    // Add warning indicator for parse errors
    if h.host.ParseError != nil {
        desc = "⚠ " + desc
    }

    return desc
}
```

### Pattern 3: View State Machine
**What:** Use enum-style view states to manage list vs detail view
**When to use:** Multiple view modes in same TUI
**Example:**
```go
// internal/tui/model.go
type ViewMode int

const (
    ViewList ViewMode = iota
    ViewDetail
)

type Model struct {
    viewMode     ViewMode
    list         list.Model
    detailHost   *config.SSHHost
    spinner      spinner.Model
    loading      bool
    width        int
    height       int
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.list.SetSize(msg.Width, msg.Height)
        return m, nil

    case tea.KeyMsg:
        if m.viewMode == ViewList {
            switch msg.String() {
            case "enter":
                // Switch to detail view
                if i, ok := m.list.SelectedItem().(hostItem); ok {
                    m.viewMode = ViewDetail
                    m.detailHost = &i.host
                    return m, nil
                }
            }
        } else if m.viewMode == ViewDetail {
            switch msg.String() {
            case "esc", "q":
                // Return to list view
                m.viewMode = ViewList
                m.detailHost = nil
                return m, nil
            }
        }
    }

    // Delegate to current view
    if m.viewMode == ViewList {
        var cmd tea.Cmd
        m.list, cmd = m.list.Update(msg)
        return m, cmd
    }

    return m, nil
}

func (m Model) View() string {
    if m.loading {
        return m.spinner.View() + " Loading SSH config..."
    }

    switch m.viewMode {
    case ViewList:
        return m.list.View()
    case ViewDetail:
        return m.renderDetailView()
    default:
        return "Unknown view"
    }
}
```

### Pattern 4: Include Directive Recursion
**What:** `kevinburke/ssh_config` handles Include recursion automatically up to depth 5
**When to use:** Default behavior — no manual recursion needed
**Example:**
```go
// Library handles this automatically:
// ~/.ssh/config contains: Include ~/.ssh/config.d/*.conf
// Library will parse main file + all included files
cfg, err := ssh_config.Decode(f) // Automatically follows Include directives
```

### Pattern 5: Detail View Layout (Right Panel)
**What:** Use Lipgloss JoinHorizontal to create side-by-side panels
**When to use:** Detail view shows config alongside list
**Example:**
```go
// internal/tui/detail_view.go
func (m Model) renderDetailView() string {
    if m.detailHost == nil {
        return "No host selected"
    }

    // Left panel: abbreviated list
    listPanel := lipgloss.NewStyle().
        Width(m.width / 3).
        Height(m.height).
        BorderStyle(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Render(m.list.View())

    // Right panel: detail content
    var details strings.Builder
    details.WriteString(lipgloss.NewStyle().Bold(true).Render(m.detailHost.Name))
    details.WriteString("\n\n")
    details.WriteString(fmt.Sprintf("Source: %s:%d\n", m.detailHost.SourceFile, m.detailHost.SourceLine))
    details.WriteString(fmt.Sprintf("Hostname: %s\n", m.detailHost.Hostname))
    details.WriteString(fmt.Sprintf("User: %s\n", m.detailHost.User))
    details.WriteString(fmt.Sprintf("Port: %s\n", m.detailHost.Port))
    details.WriteString("\nAll Options:\n")

    for key, values := range m.detailHost.AllOptions {
        for _, v := range values {
            details.WriteString(fmt.Sprintf("  %s %s\n", key, v))
        }
    }

    detailPanel := lipgloss.NewStyle().
        Width(2*m.width/3 - 2). // Subtract border width
        Height(m.height - 2).   // Subtract border height
        BorderStyle(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Padding(1, 2).
        Render(details.String())

    return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailPanel)
}
```

### Anti-Patterns to Avoid
- **❌ Parsing SSH config with regexp:** Wildcard matching, Include directives, multi-value keys, and escape sequences make this error-prone
- **❌ Manually tracking terminal size:** Bubbletea sends `tea.WindowSizeMsg` automatically, recalculate layouts on this message
- **❌ Hardcoded ANSI colors:** Use Lipgloss `AdaptiveColor` to support light/dark backgrounds and terminal profiles
- **❌ Filtering/pagination from scratch:** Bubbles `list` component handles this with fuzzy search built-in
- **❌ Blocking I/O in Update():** SSH config parsing should happen before TUI starts or use Bubbletea Cmd for async loading

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SSH config parsing | Custom parser with regexp | kevinburke/ssh_config | Wildcard matching, Include recursion (5 levels deep), Match blocks, multi-value keys (IdentityFile), escape sequences, comment preservation |
| Host wildcard matching | strings.Contains or basic glob | ssh_config library pattern matching | SSH uses `*` (zero or more chars) and `?` (exactly one char), plus multi-pattern support (`Host *.example.com *.test.org`) |
| List filtering | Manual string matching loop | Bubbles list DefaultFilter | Built-in fuzzy search with ranked results, highlight matched characters, handles Unicode correctly |
| Terminal resize handling | SIGWINCH signal handlers | Bubbletea WindowSizeMsg | Cross-platform (Windows doesn't have SIGWINCH), automatic message dispatch, integrates with component lifecycle |
| Color profile detection | Manual terminfo parsing | Lipgloss ColorProfile() | Auto-detects 4-bit, 8-bit, 24-bit (TrueColor) support, gracefully degrades colors, dark/light background detection |
| Layout calculations | Manual width/height math | Lipgloss PlaceHorizontal/Vertical, JoinHorizontal/Vertical | Handles borders (subtract 2 from height/width), padding, margins, alignment, prevents text wrapping in bordered panels |

**Key insight:** SSH config parsing is deceptively complex. OpenSSH's parser handles ~50 directives, multi-value keys, wildcards, includes, and escape sequences. Even simple configs can have edge cases (e.g., `Host * !internal` negation patterns). TUI layout math is similarly subtle — forgetting to subtract border sizes causes off-by-one rendering errors.

## Common Pitfalls

### Pitfall 1: Not Tracking Source File for Included Configs
**What goes wrong:** User has 10 included config files, sees malformed entry, can't find which file to edit
**Why it happens:** `kevinburke/ssh_config` provides `Position` (line/col) but you must track which file is being parsed
**How to avoid:** When parsing, track the source file path and store it in your domain model alongside Position
**Warning signs:** User asks "which file is this from?" and you can't answer
**Example:**
```go
// ❌ BAD: No source tracking
type SSHHost struct {
    Name     string
    Hostname string
}

// ✅ GOOD: Track source file
type SSHHost struct {
    Name       string
    Hostname   string
    SourceFile string  // Full path to config file
    SourceLine int     // Line number in that file
}

// When parsing includes:
for _, include := range cfg.Hosts {
    if inc, ok := include.(*ssh_config.Include); ok {
        for _, path := range inc.Files() {
            // Track path for all hosts parsed from this file
            includedHosts := parseConfig(path)
            for i := range includedHosts {
                includedHosts[i].SourceFile = path
            }
        }
    }
}
```

### Pitfall 2: Terminal Resize Doesn't Update Layout
**What goes wrong:** User resizes terminal, list stays at original size (truncated or with wasted space)
**Why it happens:** Not handling `tea.WindowSizeMsg` or not updating child component sizes
**How to avoid:** Always handle `WindowSizeMsg` in Update(), propagate to all child components (list, viewport), recalculate panel widths
**Warning signs:** Layout looks wrong after resize, scrollbar disappears, panels overlap
**Example:**
```go
// ❌ BAD: Ignoring resize
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle keys...
    }
    return m, nil
}

// ✅ GOOD: Handling resize
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height

        // Update child components
        m.list.SetSize(msg.Width, msg.Height)
        if m.viewport != nil {
            m.viewport.Width = msg.Width
            m.viewport.Height = msg.Height
        }

        return m, nil
    case tea.KeyMsg:
        // Handle keys...
    }
    return m, nil
}
```

### Pitfall 3: Not Accounting for Border Width in Layout Math
**What goes wrong:** Detail panel is 2 cells too wide, causes line wrapping or rendering artifacts
**Why it happens:** Lipgloss borders add 2 chars (1 left + 1 right) to width, 2 lines (1 top + 1 bottom) to height
**How to avoid:** **ALWAYS** subtract 2 from width and height before rendering content in bordered panels
**Warning signs:** Text wraps unexpectedly, panels don't fit side-by-side, vertical scrolling when content should fit
**Example:**
```go
// ❌ BAD: Not accounting for borders
detailPanel := lipgloss.NewStyle().
    Width(m.width / 2).              // WRONG: Border adds 2 to this
    Height(m.height).                // WRONG: Border adds 2 to this
    BorderStyle(lipgloss.RoundedBorder()).
    Render(content)

// ✅ GOOD: Subtract border size
detailPanel := lipgloss.NewStyle().
    Width(m.width/2 - 2).            // Subtract 2 for left+right borders
    Height(m.height - 2).            // Subtract 2 for top+bottom borders
    BorderStyle(lipgloss.RoundedBorder()).
    Render(content)

// Also account for padding if used:
detailPanel := lipgloss.NewStyle().
    Width(m.width/2 - 2 - 4).        // Subtract border (2) + horizontal padding (4 = 2 left + 2 right)
    Height(m.height - 2 - 2).        // Subtract border (2) + vertical padding (2 = 1 top + 1 bottom)
    BorderStyle(lipgloss.RoundedBorder()).
    Padding(1, 2).
    Render(content)
```

### Pitfall 4: Wildcard Hosts (Host *) Mixed with Regular Hosts
**What goes wrong:** User sees `Host *` entries mixed in alphabetically, making list hard to scan
**Why it happens:** Sorting all hosts alphabetically without separating wildcards
**How to avoid:** Separate regular hosts from wildcard hosts, display wildcards in a separate section at bottom
**Warning signs:** User feedback "hard to find specific servers", wildcard catch-all configs clutter list
**Example:**
```go
// ❌ BAD: All hosts mixed together
func sortHosts(hosts []SSHHost) {
    sort.Slice(hosts, func(i, j int) bool {
        return hosts[i].Name < hosts[j].Name
    })
}

// ✅ GOOD: Separate wildcards
func organizeHosts(hosts []SSHHost) (regular, wildcards []SSHHost) {
    for _, h := range hosts {
        if h.IsWildcard {
            wildcards = append(wildcards, h)
        } else {
            regular = append(regular, h)
        }
    }

    // Sort each group separately
    sort.Slice(regular, func(i, j int) bool {
        return regular[i].Name < regular[j].Name
    })
    sort.Slice(wildcards, func(i, j int) bool {
        return wildcards[i].Name < wildcards[j].Name
    })

    return regular, wildcards
}

// Render with separator:
func (m Model) View() string {
    regularItems := convertToListItems(m.regularHosts)

    if len(m.wildcardHosts) > 0 {
        // Add separator
        regularItems = append(regularItems, separatorItem{text: "--- Wildcard Entries ---"})
        regularItems = append(regularItems, convertToListItems(m.wildcardHosts)...)
    }

    m.list.SetItems(regularItems)
    return m.list.View()
}
```

### Pitfall 5: Malformed Config Entries Crash Parser
**What goes wrong:** User has one typo in config, entire app fails to start
**Why it happens:** Not handling parse errors gracefully, assuming all configs are valid
**How to avoid:** Wrap parse errors in domain model, display malformed entries with warning indicator, allow user to see what's broken
**Warning signs:** Parser returns error on any malformed entry, user can't use app until they fix config
**Example:**
```go
// ❌ BAD: Parse error fails entire operation
func ParseSSHConfig(path string) ([]SSHHost, error) {
    cfg, err := ssh_config.Decode(f)
    if err != nil {
        return nil, err  // User can't use app now!
    }
    // ...
}

// ✅ GOOD: Capture parse errors per-host
func ParseSSHConfig(path string) ([]SSHHost, error) {
    var hosts []SSHHost

    cfg, err := ssh_config.Decode(f)
    if err != nil {
        // File-level parse error: create error entry
        hosts = append(hosts, SSHHost{
            Name:       filepath.Base(path),
            ParseError: fmt.Errorf("failed to parse config: %w", err),
            SourceFile: path,
        })
        return hosts, nil  // Return partial results
    }

    for _, host := range cfg.Hosts {
        h := SSHHost{
            Name:       strings.Join(host.Patterns, " "),
            SourceFile: path,
        }

        // Validate required fields
        if h.Hostname == "" && !h.IsWildcard {
            h.ParseError = fmt.Errorf("missing Hostname directive")
        }

        hosts = append(hosts, h)
    }

    return hosts, nil
}

// In TUI, show warning indicator:
func (h hostItem) Description() string {
    if h.host.ParseError != nil {
        return "⚠ " + h.host.ParseError.Error()
    }
    return fmt.Sprintf("User: %s | Port: %s", h.host.User, h.host.Port)
}
```

### Pitfall 6: Empty Config File Shows Blank Screen
**What goes wrong:** User launches app with no SSH config, sees empty terminal with no guidance
**Why it happens:** Not handling empty state, assuming config always exists
**How to avoid:** Detect empty/missing config, show friendly message with actionable guidance
**Warning signs:** User asks "is the app working?", no visual feedback for empty state
**Example:**
```go
// ❌ BAD: Empty list with no context
func (m Model) View() string {
    return m.list.View()  // Shows nothing if list is empty
}

// ✅ GOOD: Friendly empty state
func (m Model) View() string {
    if len(m.list.Items()) == 0 {
        emptyStyle := lipgloss.NewStyle().
            Foreground(lipgloss.Color("241")).
            Italic(true).
            Padding(2)

        message := "No SSH config found\n\n" +
            "Create ~/.ssh/config with Host entries:\n\n" +
            "  Host myserver\n" +
            "    HostName example.com\n" +
            "    User username\n\n" +
            "Press 'q' to quit"

        return emptyStyle.Render(message)
    }

    return m.list.View()
}
```

### Pitfall 7: Match Blocks Crash Parser
**What goes wrong:** User has `Match` directive in config, parser fails
**Why it happens:** `kevinburke/ssh_config` v1.4.0 explicitly does NOT support Match directives
**How to avoid:** Document limitation in error messages, suggest removing Match blocks, or pre-process config to skip Match blocks
**Warning signs:** Parse error mentioning "Match directive unsupported"
**Example:**
```go
// Documented limitation — can't fix, must communicate clearly:
func ParseSSHConfig(path string) ([]SSHHost, error) {
    cfg, err := ssh_config.Decode(f)
    if err != nil {
        // Check if error is about Match directive
        if strings.Contains(err.Error(), "Match") {
            return nil, fmt.Errorf(
                "SSH config contains Match directive which is not supported.\n" +
                "Please remove Match blocks from %s or use Host directives instead.\n" +
                "Original error: %w", path, err,
            )
        }
        return nil, fmt.Errorf("parse config: %w", err)
    }
    // ...
}
```

## Code Examples

Verified patterns from official sources:

### Basic SSH Config Parsing
```go
// Source: https://pkg.go.dev/github.com/kevinburke/ssh_config
import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/kevinburke/ssh_config"
)

func LoadSSHConfig() (*ssh_config.Config, error) {
    home, _ := os.UserHomeDir()
    configPath := filepath.Join(home, ".ssh", "config")

    f, err := os.Open(configPath)
    if err != nil {
        return nil, fmt.Errorf("open config: %w", err)
    }
    defer f.Close()

    cfg, err := ssh_config.Decode(f)
    if err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }

    return cfg, nil
}

func GetAllHosts(cfg *ssh_config.Config) []string {
    var hosts []string
    for _, host := range cfg.Hosts {
        for _, pattern := range host.Patterns {
            hosts = append(hosts, pattern.String())
        }
    }
    return hosts
}

func GetHostConfig(cfg *ssh_config.Config, alias string) map[string]string {
    config := make(map[string]string)

    // Common SSH config keys
    keys := []string{"HostName", "User", "Port", "IdentityFile", "ProxyJump", "ForwardAgent"}

    for _, key := range keys {
        val, err := cfg.Get(alias, key)
        if err == nil && val != "" {
            config[key] = val
        }
    }

    return config
}
```

### Bubbles List Component Setup
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbles/list
import (
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
)

type item struct {
    title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
    list list.Model
}

func initialModel() model {
    items := []list.Item{
        item{title: "server1", desc: "user@example.com:22"},
        item{title: "server2", desc: "admin@test.org:2222"},
    }

    l := list.New(items, list.NewDefaultDelegate(), 80, 24)
    l.Title = "SSH Connections"

    return model{list: l}
}

func (m model) Init() tea.Cmd {
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.list.SetSize(msg.Width, msg.Height)
        return m, nil

    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        case "enter":
            i, ok := m.list.SelectedItem().(item)
            if ok {
                // Handle selection
                fmt.Printf("Selected: %s\n", i.title)
            }
        }
    }

    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return m.list.View()
}
```

### Lipgloss Side-by-Side Layout
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/lipgloss
import (
    "github.com/charmbracelet/lipgloss"
)

func renderSplitView(leftContent, rightContent string, width, height int) string {
    leftWidth := width / 3
    rightWidth := width - leftWidth

    leftStyle := lipgloss.NewStyle().
        Width(leftWidth - 2).
        Height(height - 2).
        BorderStyle(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Padding(1)

    rightStyle := lipgloss.NewStyle().
        Width(rightWidth - 2).
        Height(height - 2).
        BorderStyle(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Padding(1)

    leftPanel := leftStyle.Render(leftContent)
    rightPanel := rightStyle.Render(rightContent)

    return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}
```

### Loading Spinner During Config Parse
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/bubbles/spinner
import (
    "github.com/charmbracelet/bubbles/spinner"
    tea "github.com/charmbracelet/bubbletea"
)

type loadingMsg struct {
    hosts []SSHHost
    err   error
}

type model struct {
    spinner spinner.Model
    loading bool
    hosts   []SSHHost
    err     error
}

func initialModel() model {
    s := spinner.New()
    s.Spinner = spinner.Dot

    return model{
        spinner: s,
        loading: true,
    }
}

func (m model) Init() tea.Cmd {
    return tea.Batch(
        m.spinner.Tick,
        loadSSHConfigCmd(),
    )
}

func loadSSHConfigCmd() tea.Cmd {
    return func() tea.Msg {
        hosts, err := ParseSSHConfig("~/.ssh/config")
        return loadingMsg{hosts: hosts, err: err}
    }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case loadingMsg:
        m.loading = false
        m.hosts = msg.hosts
        m.err = msg.err
        return m, nil

    case spinner.TickMsg:
        if m.loading {
            var cmd tea.Cmd
            m.spinner, cmd = m.spinner.Update(msg)
            return m, cmd
        }
    }

    return m, nil
}

func (m model) View() string {
    if m.loading {
        return m.spinner.View() + " Loading SSH config..."
    }

    if m.err != nil {
        return fmt.Sprintf("Error: %v", m.err)
    }

    return fmt.Sprintf("Loaded %d hosts", len(m.hosts))
}
```

### Adaptive Color Styling
```go
// Source: https://pkg.go.dev/github.com/charmbracelet/lipgloss
import (
    "github.com/charmbracelet/lipgloss"
)

var (
    // Adapts to light/dark terminal backgrounds
    accentColor = lipgloss.AdaptiveColor{
        Light: "62",   // Purple for light backgrounds
        Dark:  "99",   // Brighter purple for dark backgrounds
    }

    highlightColor = lipgloss.AdaptiveColor{
        Light: "235",  // Dark gray for light backgrounds
        Dark:  "252",  // Light gray for dark backgrounds
    }
)

func styleHostname(hostname string) string {
    return lipgloss.NewStyle().
        Foreground(accentColor).
        Bold(true).
        Render(hostname)
}

func styleSecondaryText(text string) string {
    return lipgloss.NewStyle().
        Foreground(highlightColor).
        Render(text)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual SSH config parsing with regexp | kevinburke/ssh_config library | 2017 (v1.0) | Standardized parsing, Include support, comment preservation |
| Custom TUI frameworks (termbox, tcell) | Bubbletea (Elm architecture) | 2020 | Declarative state management, easier testing, better composition |
| Manual ANSI color codes | Lipgloss adaptive colors | 2021 | Automatic color profile detection, graceful degradation |
| Blocking I/O in TUI update loops | Bubbletea Cmd pattern | 2020 | Non-blocking async operations, responsive UI during I/O |
| Match directive unsupported | Still unsupported in kevinburke/ssh_config | N/A | Use k0sproject/rig if Match support critical (but pre-release) |

**Deprecated/outdated:**
- **termbox-go**: Unmaintained since 2021, use Bubbletea's underlying tcell instead
- **~/.ssh/identity**: Phased out in OpenSSH 2001, modern configs use `~/.ssh/id_rsa`, `~/.ssh/id_ed25519`, etc.
- **UseKeychain (macOS)**: SSH config option introduced in macOS Sierra (2016), not in OpenSSH spec — handle as unknown option gracefully

## Open Questions

1. **Alternative SSH config parser (k0sproject/rig) worth the risk?**
   - What we know: More features (partial Match support, token expansion, env var expansion), better OpenSSH compliance
   - What's unclear: Pre-release stability (v2.0.0-alpha.3), "not tested outside dev environment" warning
   - Recommendation: Start with kevinburke/ssh_config (stable, proven), switch to rig later if Match support becomes critical user request

2. **Detail view layout: right panel vs bottom panel vs inline expansion?**
   - What we know: Right panel requires ~60+ column terminal width, bottom panel works in narrow terminals, inline expansion simplest but less context
   - What's unclear: User preference, typical terminal sizes
   - Recommendation: Start with right panel (most desktop-like), add bottom panel fallback if terminal width < 80 columns

3. **Color palette for accent colors?**
   - What we know: Need to distinguish Name, Hostname, User, Port, but not be overwhelming
   - What's unclear: Specific color choices, accessibility for colorblind users
   - Recommendation: Use Lipgloss AdaptiveColor, test with `lipgloss.HasDarkBackground()`, limit to 2-3 accent colors max (e.g., purple for hostnames, gray for secondary info)

4. **How to display warning indicator for malformed entries?**
   - What we know: Need visual indicator in list, should be obvious but not alarming
   - What's unclear: Icon choice (⚠ vs ⚡ vs •), color (yellow vs red), whether to show error message inline or only in detail view
   - Recommendation: Use ⚠ prefix in description line, yellow color if supported, full error message in detail view

## Sources

### Primary (HIGH confidence)
- [kevinburke/ssh_config v1.4.0 Go Package](https://pkg.go.dev/github.com/kevinburke/ssh_config) - API documentation, Include support, Position tracking
- [kevinburke/ssh_config GitHub](https://github.com/kevinburke/ssh_config) - Library features, limitations (Match unsupported), 511+ projects using it
- [Bubbles list component](https://pkg.go.dev/github.com/charmbracelet/bubbles/list) - v1.0.0 API, keyboard navigation, filtering, custom items
- [Bubbles GitHub](https://github.com/charmbracelet/bubbles) - All available components (list, spinner, viewport), current version v1.0.0 (Feb 2026)
- [Lipgloss Go Package](https://pkg.go.dev/github.com/charmbracelet/lipgloss) - Color definition, borders, layout functions, adaptive colors
- [SSH config man page](https://man7.org/linux/man-pages/man5/ssh_config.5.html) - Wildcard patterns, Include directive, directive list

### Secondary (MEDIUM confidence)
- [k0sproject/rig/v2/sshconfig](https://pkg.go.dev/github.com/k0sproject/rig/v2/sshconfig) - Alternative parser, partial Match support, but pre-release (alpha.3)
- [BubbleLayout GitHub](https://github.com/winder/bubblelayout) - Declarative layout manager for side-by-side panels
- [Bubbletea Layout Handling Discussion](https://github.com/charmbracelet/bubbletea/discussions/307) - Community patterns for multi-panel layouts
- [Bubbletea Resize Handling](https://github.com/charmbracelet/bubbletea/discussions/661) - Common resize pitfalls
- [Empty State UX Best Practices](https://www.pencilandpaper.io/articles/empty-states) - Friendly messaging patterns
- [Go Error Handling 2026](https://jsschools.com/golang/go_error_handling_patterns_building_robust_applications_that_fail_gracefully/) - Error wrapping with fmt.Errorf %w

### Tertiary (LOW confidence)
- [SSH Config Complete Guide 2026](https://devtoolbox.dedyn.io/blog/ssh-config-complete-guide) - Include directive tilde expansion, general SSH config guidance
- [Lipgloss terminal colors](https://github.com/charmbracelet/lipgloss) - Color trends mentioned in search but not specific technical docs
- [Bubbletea state machine pattern](https://zackproser.com/blog/bubbletea-state-machine) - Community blog post on view state management

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - kevinburke/ssh_config is v1+ stable with 511 users, Bubbles is v1.0.0 official Charm library
- Architecture: HIGH - Patterns verified from official documentation, proven in production TUIs
- Pitfalls: MEDIUM-HIGH - Border math and resize handling verified in official discussions, Match directive limitation documented in library, malformed entry handling is defensive programming best practice

**Research date:** 2026-02-14
**Valid until:** ~30 days (stable ecosystem, unlikely to change rapidly)
