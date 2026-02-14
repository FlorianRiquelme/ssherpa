# Architecture Research

**Domain:** SSH Connection Management TUI with Pluggable Backends
**Researched:** 2026-02-14
**Confidence:** HIGH

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    TUI Layer (Bubbletea)                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│  │  Root    │  │  Server  │  │  Detail  │  │  Help    │    │
│  │  Model   │  │  List    │  │  View    │  │  View    │    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘    │
│       │ (routes)    │ (renders)   │ (shows)     │           │
├───────┴─────────────┴─────────────┴─────────────┴───────────┤
│                     Core Application                         │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              Project Service                         │    │
│  │  • Auto-detect project from git remote              │    │
│  │  • Map project → server list                        │    │
│  │  • Cache project detection                          │    │
│  └─────────────┬──────────────────┬────────────────────┘    │
│                │                  │                          │
│  ┌─────────────▼───────┐    ┌────▼────────────────────┐    │
│  │ Backend Interface   │    │  SSH Executor           │    │
│  │ • ListProjects()    │    │  • tea.ExecProcess()    │    │
│  │ • ListServers()     │    │  • Resume after exit    │    │
│  │ • GetCredentials()  │    └─────────────────────────┘    │
│  └─────────────┬───────┘                                    │
├────────────────┴──────────────────────────────────────────────┤
│                    Backend Adapters                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ 1Password    │  │   Local      │  │   Future     │      │
│  │  Adapter     │  │   Config     │  │   Backends   │      │
│  │  (via `op`)  │  │   Adapter    │  │              │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘

External Dependencies:
- go-git (git remote detection)
- charmbracelet/bubbletea (TUI framework)
- charmbracelet/bubbles (TUI components)
- charmbracelet/lipgloss (TUI styling)
- op CLI (1Password integration)
- system ssh binary
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| Root Model | Message routing, layout composition, global keys (quit, help) | Bubbletea Model with child models, handles WindowSizeMsg broadcasting |
| Child Models | Specific views (server list, details, help), handle view-specific inputs | Nested Bubbletea Models, self-contained state |
| Project Service | Auto-detect current project from git remote, cache results | Uses go-git library, implements business logic |
| Backend Interface | Abstract contract for credential/server storage backends | Go interface with 3-5 methods (ListProjects, ListServers, GetCredentials) |
| Backend Adapters | Concrete implementations of backend interface | Structs implementing Backend interface, handle external service integration |
| SSH Executor | Launch system ssh command, hand over terminal control | Uses tea.ExecProcess to block and resume after ssh exits |

## Recommended Project Structure

```
sshjesus/
├── cmd/
│   └── sshjesus/
│       └── main.go              # Entry point, wires up dependencies
├── internal/
│   ├── tui/                     # TUI layer (presentation)
│   │   ├── root.go              # Root model, message router
│   │   ├── server_list.go       # Server list view model
│   │   ├── detail.go            # Server detail view model
│   │   ├── help.go              # Help view model
│   │   ├── messages.go          # Custom message types
│   │   └── styles.go            # Lipgloss styles (colors, borders)
│   ├── core/                    # Core business logic
│   │   ├── service.go           # Project service (git detection)
│   │   ├── backend.go           # Backend interface definition
│   │   ├── ssh.go               # SSH execution logic
│   │   └── types.go             # Domain types (Project, Server, Credential)
│   └── adapters/                # Backend implementations
│       ├── onepassword/
│       │   ├── adapter.go       # 1Password adapter implementation
│       │   └── op_client.go     # `op` CLI wrapper
│       └── local/
│           └── adapter.go       # Local config file adapter
├── pkg/                         # Public interfaces (if needed)
├── .github/                     # CI/CD workflows
├── docs/                        # User documentation
├── examples/                    # Example configurations
└── README.md
```

### Structure Rationale

- **cmd/sshjesus/:** Single binary entry point, keeps main package clean and focused on wiring
- **internal/tui/:** Encapsulates all Bubbletea-specific code, models, and styling separate from business logic
- **internal/core/:** Pure business logic with no TUI dependencies, making it testable and reusable
- **internal/adapters/:** Each backend gets its own package, clear separation, easy to add new backends
- **Follows Go best practices:** Standard layout, internal/ prevents external imports, single responsibility per package

## Architectural Patterns

### Pattern 1: Hexagonal Architecture (Ports & Adapters)

**What:** Core business logic defines abstract interfaces (ports), external systems connect via adapters

**When to use:** When building applications with pluggable backends or multiple integrations

**Trade-offs:**
- **Pros:** Testable (mock backends), extensible (add backends without touching core), clean separation
- **Cons:** More indirection, interface overhead (66% slower than direct calls in tight loops)

**Example:**
```go
// Port (defined in core/)
type Backend interface {
    ListProjects(ctx context.Context) ([]Project, error)
    ListServers(ctx context.Context, projectID string) ([]Server, error)
    GetCredentials(ctx context.Context, serverID string) (*Credential, error)
}

// Adapter (implemented in adapters/onepassword/)
type OnePasswordAdapter struct {
    client *OpClient
}

func (a *OnePasswordAdapter) ListProjects(ctx context.Context) ([]Project, error) {
    // Call `op` CLI, parse output
}
```

### Pattern 2: Interface Extension for Optional Features

**What:** Start with minimal required interface, extend with additional interfaces for optional features

**When to use:** When backends may support different feature sets (e.g., some backends support tags, others don't)

**Trade-offs:**
- **Pros:** Backward compatible, modular development, backends only implement what they support
- **Cons:** Requires type assertions (comma-ok pattern), can consume CPU in tight loops

**Example:**
```go
// Base interface (required)
type Backend interface {
    ListServers(ctx context.Context, projectID string) ([]Server, error)
}

// Optional interface
type TaggableBackend interface {
    ListTags(ctx context.Context) ([]string, error)
    FilterByTags(ctx context.Context, tags []string) ([]Server, error)
}

// In service layer
if tb, ok := backend.(TaggableBackend); ok {
    // Use tag filtering
    tags, _ := tb.ListTags(ctx)
} else {
    // Fall back to basic listing
}
```

### Pattern 3: Model Tree (Bubbletea Composition)

**What:** Organize TUI as hierarchical models, root routes messages, children render views

**When to use:** Always for non-trivial Bubbletea applications (>1 screen)

**Trade-offs:**
- **Pros:** Modular, testable views, clear responsibilities, easier debugging
- **Cons:** More boilerplate (each model needs Init/Update/View), message routing logic

**Example:**
```go
// Root model
type RootModel struct {
    state       AppState  // current, detail, help
    serverList  ServerListModel
    detailView  DetailModel
    helpView    HelpModel
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle global keys
        if msg.String() == "q" {
            return m, tea.Quit
        }
    case tea.WindowSizeMsg:
        // Broadcast to all children
        m.serverList, _ = m.serverList.Update(msg)
        m.detailView, _ = m.detailView.Update(msg)
    }

    // Route to active child
    switch m.state {
    case AppStateList:
        m.serverList, cmd = m.serverList.Update(msg)
    case AppStateDetail:
        m.detailView, cmd = m.detailView.Update(msg)
    }
    return m, cmd
}
```

## Data Flow

### Application Startup Flow

```
[main.go]
    ↓
[Initialize Backend Adapter] → [Load config: which backend to use]
    ↓
[Initialize Project Service] → [Inject backend adapter]
    ↓
[Initialize Root TUI Model] → [Inject service dependencies]
    ↓
[tea.NewProgram(rootModel).Run()] → [Start Bubbletea event loop]
```

### Project Detection Flow

```
[User opens TUI in terminal]
    ↓
[Project Service: DetectProject()]
    ↓
[go-git: Open repository] → [Get git remote URL]
    ↓
[Parse remote URL] → [Extract project identifier]
    ↓
[Cache project ID] → [Store in service state]
    ↓
[Backend: ListServers(projectID)] → [Query backend for servers]
    ↓
[Return server list to TUI] → [Update ServerListModel state]
```

### SSH Connection Flow

```
[User selects server in TUI]
    ↓
[Backend: GetCredentials(serverID)] → [Fetch username, host, key from backend]
    ↓
[SSH Executor: BuildSSHCommand()] → [Construct: ssh user@host -i keyfile]
    ↓
[tea.ExecProcess(sshCmd, callback)] → [Pause TUI, exec ssh, hand over terminal]
    ↓
[SSH session runs] → [User interacts with remote shell]
    ↓
[SSH exits] → [Resume TUI, callback with exit code]
    ↓
[Show "Connection closed" message] → [Return to server list]
```

### Message Flow (Bubbletea)

```
[User Input / System Event]
    ↓
[tea.Msg created by framework]
    ↓
[RootModel.Update(msg)]
    ↓ (global handling)
[Handle WindowSizeMsg, quit keys]
    ↓ (routing)
[Route to active child model]
    ↓
[ChildModel.Update(msg)]
    ↓ (may return tea.Cmd)
[Optional async command] → [Execute goroutine, return msg when done]
    ↓
[Return (newModel, cmd)]
    ↓
[Framework calls View()]
    ↓
[Render UI string]
```

### Key Data Flows

1. **Git → Project ID:** go-git opens .git, reads remote URL, service parses to extract project identifier
2. **Project ID → Servers:** Backend adapter queries external service (1Password), returns structured Server list
3. **Server → SSH:** Backend fetches credentials, SSH executor builds command, tea.ExecProcess blocks and resumes
4. **User Input → State Change:** Bubbletea message → Update() → new model state → View() renders new UI

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 0-100 projects | In-memory caching is sufficient, single backend adapter |
| 100-1000 projects | Add persistent cache (SQLite), implement pagination in server list UI |
| 1000+ projects | Consider backend pre-indexing, add search/filter, lazy load server details |

### Scaling Priorities

1. **First bottleneck:** Backend API calls (1Password CLI can be slow)
   - **Fix:** Aggressive in-memory caching (5-minute TTL), cache project detection, parallel backend queries
2. **Second bottleneck:** TUI rendering with large server lists (>100 servers)
   - **Fix:** Virtual scrolling (only render visible rows), use bubbles.viewport for efficient rendering

## Anti-Patterns

### Anti-Pattern 1: Blocking in Update()

**What people do:** Call backend directly in Update() method, wait for results synchronously

```go
// WRONG
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    servers, _ := m.backend.ListServers(ctx, m.projectID)  // BLOCKS!
    m.servers = servers
    return m, nil
}
```

**Why it's wrong:** Freezes entire TUI while waiting for backend, violates Bubbletea's async command pattern

**Do this instead:** Use tea.Cmd for async operations

```go
// CORRECT
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, m.fetchServersCmd()  // Returns immediately
}

func (m Model) fetchServersCmd() tea.Cmd {
    return func() tea.Msg {
        servers, err := m.backend.ListServers(ctx, m.projectID)
        return serversLoadedMsg{servers, err}
    }
}
```

### Anti-Pattern 2: Direct Backend Coupling in TUI

**What people do:** Import adapter packages directly in TUI models, call backend from views

```go
// WRONG
import "github.com/you/sshjesus/internal/adapters/onepassword"

type Model struct {
    backend *onepassword.OnePasswordAdapter  // Tight coupling!
}
```

**Why it's wrong:** TUI can't be tested without real backend, impossible to switch backends, violates hexagonal architecture

**Do this instead:** Depend on interfaces, inject via constructor

```go
// CORRECT
type Model struct {
    backend core.Backend  // Interface dependency
}

func NewModel(backend core.Backend) Model {
    return Model{backend: backend}
}
```

### Anti-Pattern 3: Using Goroutines Instead of tea.Cmd

**What people do:** Spawn goroutines directly in Update() for async work, update model from goroutine

```go
// WRONG
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    go func() {
        servers, _ := m.backend.ListServers(ctx, m.projectID)
        m.servers = servers  // RACE CONDITION!
    }()
    return m, nil
}
```

**Why it's wrong:** Race conditions (goroutine and Update() both mutate model), no way to return results to event loop

**Do this instead:** Use tea.Cmd (Bubbletea manages goroutines)

```go
// CORRECT
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, func() tea.Msg {
        servers, err := m.backend.ListServers(ctx, m.projectID)
        return serversLoadedMsg{servers, err}
    }
}
```

### Anti-Pattern 4: Mixing Business Logic in TUI Models

**What people do:** Put git detection, credential parsing, SSH command building inside TUI models

```go
// WRONG
func (m ServerListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Git detection in TUI model!
    repo, _ := git.PlainOpen(".")
    remote, _ := repo.Remote("origin")
    projectID := parseRemoteURL(remote.Config().URLs[0])
    // ...
}
```

**Why it's wrong:** TUI models become untestable, business logic can't be reused, violates separation of concerns

**Do this instead:** Extract to service layer, inject service

```go
// CORRECT
type ServerListModel struct {
    projectService *core.ProjectService  // Injected dependency
}

func (m ServerListModel) Init() tea.Cmd {
    return func() tea.Msg {
        project, err := m.projectService.DetectProject()
        return projectDetectedMsg{project, err}
    }
}
```

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| 1Password | CLI wrapper (`op` command via os/exec) | Requires `op` binary in PATH, uses JSON output format, session management |
| Git | Library (go-git/go-git) | Pure Go implementation, no git binary needed, read-only operations |
| SSH | CLI subprocess (system `ssh` via tea.ExecProcess) | Delegates to system ssh for compatibility, hands over full terminal control |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| TUI ↔ Core Service | Method calls (synchronous interface, async via tea.Cmd) | TUI wraps service calls in tea.Cmd to keep Update() fast |
| Core Service ↔ Backend | Interface methods (Backend interface) | Service owns business logic, delegates storage to backend |
| Backend ↔ External CLI | os/exec.Command (subprocess) | Adapters spawn processes, parse stdout, handle errors |

## Build Order Recommendations

Based on dependency analysis, recommended implementation sequence:

### Phase 1: Core Foundation
1. **Backend Interface** - Define abstract contract first (core/backend.go)
2. **Domain Types** - Project, Server, Credential structs (core/types.go)
3. **Mock Backend** - In-memory implementation for testing (adapters/mock/)

### Phase 2: Business Logic
4. **Project Service** - Git detection logic (core/service.go)
5. **SSH Executor** - Command building and execution (core/ssh.go)

### Phase 3: Real Backend
6. **1Password Adapter** - Implement Backend interface with `op` CLI (adapters/onepassword/)

### Phase 4: TUI Layer
7. **Root Model** - Basic structure, routing skeleton (tui/root.go)
8. **Server List Model** - Main view with server list (tui/server_list.go)
9. **Styles** - Lipgloss styling (tui/styles.go)
10. **Detail/Help Views** - Secondary screens (tui/detail.go, tui/help.go)

### Phase 5: Polish
11. **Error Handling** - Graceful fallbacks, user-friendly messages
12. **Caching** - In-memory cache for backend responses
13. **Config** - User preferences (default backend, cache TTL)

**Rationale:** Bottom-up approach (interfaces → logic → adapters → UI) allows testing at each layer before moving up, minimizes rework.

## Sources

### Official Documentation
- [Bubbletea GitHub](https://github.com/charmbracelet/bubbletea) - Official TUI framework repository
- [Bubbletea Package Docs](https://pkg.go.dev/github.com/charmbracelet/bubbletea) - API reference
- [tea.ExecProcess Documentation](https://pkg.go.dev/github.com/charmbracelet/bubbletea#ExecProcess) - External command execution
- [go-git Package](https://pkg.go.dev/github.com/go-git/go-git/v5) - Git operations library
- [1Password Go SDK](https://github.com/1Password/onepassword-sdk-go) - Official SDK

### Architecture Guides
- [Tips for Building Bubble Tea Programs](https://leg100.github.io/en/posts/building-bubbletea-programs/) - Architectural best practices
- [Go Clean Architecture](https://dasroot.net/posts/2026/01/go-project-structure-clean-architecture/) - Project structure patterns
- [Interface Extension Pattern](https://www.dolthub.com/blog/2022-09-12-golang-interface-extension/) - Plugin architecture approach
- [Hexagonal Architecture in Go](https://medium.com/@kemaltf_/clean-architecture-hexagonal-architecture-in-go-a-practical-guide-aca2593b7223) - Ports & adapters pattern
- [Repository Pattern in Go](https://threedots.tech/post/repository-pattern-in-go/) - Backend abstraction pattern

### Real-World Examples
- [SSHM Project](https://github.com/Gu1llaum-3/sshm) - SSH management TUI with Bubbletea
- [Managing Nested Models with Bubble Tea](https://donderom.com/posts/managing-nested-models-with-bubble-tea/) - Model composition patterns

### Recent Articles (2026)
- [How to Build Command Line Tools with Bubbletea](https://oneuptime.com/blog/post/2026-01-30-how-to-build-command-line-tools-with-bubbletea-in-go/view) - 2026 best practices
- [Terminal UI: BubbleTea vs Ratatui](https://www.glukhov.org/post/2026/02/tui-frameworks-bubbletea-go-vs-ratatui-rust/) - Framework comparison

---
*Architecture research for: sshjesus SSH management TUI*
*Researched: 2026-02-14*
