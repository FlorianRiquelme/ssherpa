# Phase 1: Foundation & Architecture - Research

**Researched:** 2026-02-14
**Domain:** Go backend architecture, domain modeling, pluggable interfaces
**Confidence:** HIGH

## Summary

Phase 1 establishes the pluggable backend architecture for sshjesus using Go best practices for domain-driven design and interface-based extensibility. The research confirms that Go's standard library patterns (particularly `database/sql`) provide proven blueprints for implementing pluggable backends with optional methods via type assertions. The domain models (Server, Project, Credential) should be completely database-agnostic, with backends handling only storage CRUD operations. Go's ecosystem offers mature tooling for configuration management (XDG base directory specification), SSH config parsing, and mock-based testing.

**Primary recommendation:** Follow the `database/sql` package pattern: create a thin user-facing API layer that wraps a pluggable backend interface, support optional backend capabilities via type assertions, and keep domain types completely separate from storage concerns.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Domain model scope
- **Server** = SSH config fields (host, user, port, identity file, proxy) + metadata (tags, notes, last connected timestamp, favorite flag, display name, VPN requirement flag)
- VPN requirement flag lets the TUI warn users before connecting to a server that needs VPN
- **Project** = named group of servers (e.g., "payments-api"). Git remote URL is one detection method, but servers can also be manually assigned to projects
- A server can belong to **multiple projects** (shared infra spanning teams)
- **Credential** = auth reference, not a secret store. Points to a key file path, SSH agent, or marks "password auth". Actual secrets live in the filesystem, 1Password, or agent — not in sshjesus

#### Backend capabilities
- Backends handle **storage only** (CRUD for servers, projects, credentials). Operational tasks (connectivity checks, import, sync) live outside the backend interface
- Backends can be **read-only** — the interface has optional write methods. SSH config backend may be read-only; 1Password supports full CRUD
- **Querying/filtering is optional** in the backend interface. Backends that support it can filter server-side; others return everything and the app layer filters in-memory
- **Request/response only** — no change notifications, no file watchers, no push events

#### Multi-backend strategy
- **One backend active at a time** — user picks ssh config OR 1Password, not both simultaneously
- Backend selection via **config file** (~/.config/sshjesus or similar)
- **First run with no config → interactive setup wizard** prompts user to pick a backend and creates the config file (DEFERRED to Phase 2+)
- **Switching backends via TUI settings screen** — discoverable, not just config file editing (DEFERRED to Phase 2+)

#### Error handling
- **Backend unavailable at startup → error and exit** with clear message explaining what's wrong
- **Mid-use operation failure → show error, keep user data** so they can retry without re-entering
- **Errors are technical** — target audience is developers, surface actual error messages (SDK errors, file system errors, etc.) directly. No consumer-friendly abstraction layer
- **Malformed config file → show error, offer to reset** by re-running the setup wizard

### Claude's Discretion

- Go package structure and module layout
- Exact interface method signatures and return types
- Mock backend implementation details
- Error type hierarchy design
- Config file format choice (TOML, YAML, JSON)

### Deferred Ideas (OUT OF SCOPE)

- TUI settings screen for switching backends — Phase 2+ (needs TUI first)
- Interactive setup wizard — Phase 2+ (needs TUI or at minimum a CLI prompt flow)
</user_constraints>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library | 1.24+ | Core language features, error handling, interfaces | Go 1.24 (Feb 2025) added 2-3% CPU performance improvement, `testing/synctest` for concurrent testing |
| `github.com/adrg/xdg` | Latest | XDG Base Directory Specification for config file paths | Standard Linux ecosystem pattern, cross-platform user directory support |
| `github.com/kevinburke/ssh_config` | Latest | Parse SSH config files | Designed to work with `golang.org/x/crypto/ssh`, preserves comments, handles multi-value directives |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify` | 1.8.4+ | Mock generation and assertions | Mock backend testing, table-driven tests |
| `github.com/BurntSushi/toml` | Latest | TOML config parsing | If choosing TOML for config format |
| `gopkg.in/yaml.v3` | Latest | YAML config parsing | If choosing YAML for config format |
| `encoding/json` | Standard library | JSON config parsing | If choosing JSON for config format |
| `golang.org/x/crypto/ssh` | Latest | SSH protocol implementation | Future phases (actual SSH connection), not Phase 1 |
| `github.com/1Password/onepassword-sdk-go` | v0.x (beta) | 1Password SDK integration | Future backend implementation (not Phase 1) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| testify/mock | gomock | gomock requires code generation but enforces stricter contracts; testify is lighter for simple mocks |
| TOML | YAML | YAML has broader DevOps adoption but indentation-sensitivity causes subtle bugs; TOML eliminates type coercion ambiguity |
| TOML | JSON | JSON lacks comments and human-friendliness; only choose if programmatic parsing is primary use case |

**Installation:**
```bash
go get github.com/adrg/xdg
go get github.com/kevinburke/ssh_config
go get github.com/stretchr/testify
go get github.com/BurntSushi/toml  # or yaml.v3, depending on config format choice
```

## Architecture Patterns

### Recommended Project Structure
```
sshjesus/
├── cmd/
│   └── sshjesus/        # Main executable entry point (minimal logic)
├── internal/
│   ├── domain/          # Domain models (Server, Project, Credential)
│   ├── backend/         # Backend interface + implementations
│   │   ├── backend.go   # Core Backend interface
│   │   ├── mock/        # Mock backend for testing
│   │   ├── sshconfig/   # SSH config backend (future)
│   │   └── onepassword/ # 1Password backend (future)
│   ├── config/          # App configuration management
│   └── errors/          # Custom error types
├── pkg/                 # (Optional) Reusable libraries if needed
├── go.mod
└── go.sum
```

**Rationale:**
- `/cmd/sshjesus` contains minimal `main()` that imports from `/internal`
- `/internal` prevents external packages from importing private code
- Domain models live in `/internal/domain` — completely database-agnostic
- Backend implementations are separate packages under `/internal/backend`
- Config management in `/internal/config` handles XDG paths and file parsing

**Source:** [Standard Go Project Layout](https://github.com/golang-standards/project-layout), [Go Modules Layout](https://go.dev/doc/modules/layout)

### Pattern 1: Database/SQL-Inspired Backend Interface

**What:** Split user-facing API from backend interface, support optional methods via type assertions

**When to use:** When building pluggable backends with varying capabilities (read-only vs read-write, filtering vs no filtering)

**Example:**
```go
// Source: https://eli.thegreenplace.net/2019/design-patterns-in-gos-databasesql-package/
// Adapted for sshjesus domain

package backend

import "context"

// Backend is the minimal interface all backends must implement
type Backend interface {
    // GetServer retrieves a server by ID
    GetServer(ctx context.Context, id string) (*domain.Server, error)

    // ListServers returns all servers (backends filter server-side if capable)
    ListServers(ctx context.Context) ([]*domain.Server, error)

    // Close releases any backend resources
    Close() error
}

// Writer is an optional interface for backends that support write operations
type Writer interface {
    CreateServer(ctx context.Context, server *domain.Server) error
    UpdateServer(ctx context.Context, server *domain.Server) error
    DeleteServer(ctx context.Context, id string) error
}

// Filterer is an optional interface for backends that support server-side filtering
type Filterer interface {
    // FilterServers applies filters server-side
    FilterServers(ctx context.Context, filters map[string]string) ([]*domain.Server, error)
}

// In the application layer, check capabilities via type assertion:
func SaveServer(ctx context.Context, backend Backend, server *domain.Server) error {
    writer, ok := backend.(Writer)
    if !ok {
        return fmt.Errorf("backend does not support write operations")
    }
    return writer.UpdateServer(ctx, server)
}
```

**Key insight:** This pattern allows backends to advertise capabilities without requiring all backends to implement every method. Use `interface{}` type assertion with the `ok` idiom to gracefully handle missing capabilities.

**Source:** [Design Patterns in Go's database/sql](https://eli.thegreenplace.net/2019/design-patterns-in-gos-databasesql-package/), [Go driver interface](https://pkg.go.dev/database/sql/driver)

### Pattern 2: Domain-First Repository Pattern

**What:** Domain entities are completely separate from database models; repositories use closures for transactions

**When to use:** When building storage abstractions that need to support multiple backends (mock, SQL, NoSQL, external API)

**Example:**
```go
// Source: https://threedots.tech/post/repository-pattern-in-go/
// Domain entity (NO database tags, validation, or storage logic)
package domain

type Server struct {
    ID           string
    Host         string
    User         string
    Port         int
    IdentityFile string
    Proxy        string
    Tags         []string
    Notes        string
    LastConnected *time.Time
    Favorite      bool
    DisplayName   string
    VPNRequired   bool
}

// Backend implementation maintains separate transport types
package mock

type serverModel struct {
    ID            string    `json:"id"`
    Host          string    `json:"host"`
    User          string    `json:"user"`
    // ... storage-specific fields
}

func (m *serverModel) toDomain() *domain.Server {
    return &domain.Server{
        ID:   m.ID,
        Host: m.Host,
        // ... field mapping
    }
}

func fromDomain(s *domain.Server) *serverModel {
    return &serverModel{
        ID:   s.ID,
        Host: s.Host,
        // ... field mapping
    }
}
```

**Key insight:** Never add database tags (`json:"..."`, `db:"..."`) to domain types. Maintain separate transport models for each backend and convert at the boundary.

**Source:** [Repository Pattern in Go - Three Dots Labs](https://threedots.tech/post/repository-pattern-in-go/)

### Pattern 3: XDG Configuration Management

**What:** Use XDG Base Directory Specification for cross-platform config file locations

**When to use:** When storing application configuration files that should follow platform conventions

**Example:**
```go
// Source: https://github.com/adrg/xdg
package config

import (
    "github.com/adrg/xdg"
    "github.com/BurntSushi/toml"
)

type AppConfig struct {
    Backend string `toml:"backend"` // "sshconfig", "onepassword", etc.
    // ... other config fields
}

func Load() (*AppConfig, error) {
    // Search for existing config file across XDG search paths
    configPath, err := xdg.SearchConfigFile("sshjesus/config.toml")
    if err != nil {
        // Config doesn't exist — return default or trigger setup wizard
        return nil, ErrConfigNotFound
    }

    var cfg AppConfig
    if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
        return nil, fmt.Errorf("malformed config file: %w", err)
    }
    return &cfg, nil
}

func Save(cfg *AppConfig) error {
    // ConfigFile creates directories if they don't exist
    configPath, err := xdg.ConfigFile("sshjesus/config.toml")
    if err != nil {
        return fmt.Errorf("failed to create config path: %w", err)
    }

    f, err := os.Create(configPath)
    if err != nil {
        return err
    }
    defer f.Close()

    return toml.NewEncoder(f).Encode(cfg)
}
```

**Key insight:** Use `xdg.SearchConfigFile()` to locate existing configs, `xdg.ConfigFile()` to create new ones. Automatically handles platform differences (Linux `~/.config`, macOS `~/Library/Application Support`, Windows `%APPDATA%`).

**Source:** [adrg/xdg GitHub](https://github.com/adrg/xdg), [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir/latest/)

### Pattern 4: Custom Error Types with Context

**What:** Define custom error types that implement `error` interface and `Unwrap()` for error chains

**When to use:** When errors need to carry metadata (error codes, fields) or when callers need to distinguish error types

**Example:**
```go
// Source: https://go.dev/blog/error-handling-and-go
// Adapted with Go 1.13+ wrapping patterns
package errors

import (
    "errors"
    "fmt"
)

// Sentinel errors for common conditions
var (
    ErrBackendUnavailable = errors.New("backend unavailable")
    ErrConfigNotFound     = errors.New("config file not found")
    ErrServerNotFound     = errors.New("server not found")
    ErrReadOnlyBackend    = errors.New("backend does not support write operations")
)

// BackendError wraps backend-specific errors with context
type BackendError struct {
    Op      string // Operation that failed (e.g., "ListServers", "CreateServer")
    Backend string // Backend type (e.g., "sshconfig", "onepassword")
    Err     error  // Underlying error
}

func (e *BackendError) Error() string {
    return fmt.Sprintf("%s: %s backend: %v", e.Op, e.Backend, e.Err)
}

func (e *BackendError) Unwrap() error {
    return e.Err
}

// Usage in backend implementation:
func (b *MockBackend) GetServer(ctx context.Context, id string) (*domain.Server, error) {
    server, exists := b.servers[id]
    if !exists {
        return nil, &BackendError{
            Op:      "GetServer",
            Backend: "mock",
            Err:     ErrServerNotFound,
        }
    }
    return server, nil
}

// Callers can use errors.Is() and errors.As():
if errors.Is(err, ErrServerNotFound) {
    // Handle not found case
}

var backendErr *BackendError
if errors.As(err, &backendErr) {
    log.Printf("Backend operation failed: %s", backendErr.Op)
}
```

**Key insight:** Always implement `Unwrap()` for custom error types to enable `errors.Is()` and `errors.As()`. Include operational context (what operation failed, which backend, etc.) in error messages for developer-friendly debugging.

**Source:** [Error Handling and Go](https://go.dev/blog/error-handling-and-go), [Creating Custom Errors in Go](https://www.digitalocean.com/community/tutorials/creating-custom-errors-in-go), [OneUpTime Error Handling Guide](https://oneuptime.com/blog/post/2026-01-30-how-to-create-custom-error-types-with-stack-traces-in-go/view)

### Anti-Patterns to Avoid

- **Don't add database tags to domain types** — keep domain models completely storage-agnostic
- **Don't return concrete error types from interfaces** — return `error` interface, use `Unwrap()` for type inspection
- **Don't use hardcoded config paths** — always use XDG base directory specification
- **Don't implement all optional methods on all backends** — use type assertions to gracefully handle missing capabilities
- **Don't log errors at every layer** — log once at the top of the call stack with full context
- **Don't skip error context** — always include what operation failed and why in error messages

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SSH config parsing | Custom parser for `~/.ssh/config` | `github.com/kevinburke/ssh_config` | Handles edge cases (multi-value directives, comments, includes), preserves formatting, tested against OpenSSH behavior |
| XDG config paths | Manual path construction (`~/.config/appname`) | `github.com/adrg/xdg` | Cross-platform (Linux, macOS, Windows), handles environment variable overrides, creates directories automatically |
| Error wrapping | Custom error formatting | `fmt.Errorf()` with `%w` and `errors.Is()`/`errors.As()` | Standard library since Go 1.13, enables error chain inspection, works with all third-party errors |
| Config file parsing | String manipulation | `github.com/BurntSushi/toml` or `gopkg.in/yaml.v3` | Production-tested, handles encoding edge cases, proper error reporting |
| Mock implementations | Manual test doubles | `github.com/stretchr/testify/mock` | Assertions built-in, method call tracking, argument matchers, less boilerplate |

**Key insight:** Go's ecosystem is mature for backend/infrastructure tooling. Prefer well-tested libraries over custom implementations, especially for parsing and cross-platform concerns.

## Common Pitfalls

### Pitfall 1: Interface Pollution (Defining Interfaces Too Early)

**What goes wrong:** Creating backend interfaces before understanding what operations are actually needed leads to overly broad interfaces with unused methods.

**Why it happens:** Coming from languages where interfaces are defined upfront (Java, C#), developers define all possible methods in a single large interface.

**How to avoid:**
- Start with minimal interface (read operations only)
- Add optional interfaces as capabilities are discovered
- Define interfaces in the package that *uses* them, not where they're implemented (Go proverb: "Accept interfaces, return structs")

**Warning signs:**
- Backend interface has >5 methods in Phase 1
- Methods like `Subscribe()`, `Watch()`, `Notify()` appear (violates "request/response only" constraint)
- All backends implement every method (means capabilities aren't actually optional)

**Source:** [Go Interfaces Best Practices](https://blog.boot.dev/golang/golang-interfaces/), [Design Patterns in Go](https://refactoring.guru/design-patterns/go)

### Pitfall 2: Indentation Bugs in YAML Config

**What goes wrong:** If YAML is chosen for config format, a single misplaced space completely changes data structure without causing parse errors.

**Why it happens:** YAML's indentation-sensitive syntax has implicit type coercion (`yes`/`no`/`on`/`off` become booleans, not strings).

**How to avoid:**
- Choose TOML over YAML for application config (explicit types, no indentation sensitivity)
- If YAML is required, use strict parsing mode and validate against schema
- Document config format examples clearly

**Warning signs:**
- Config file works in development but breaks in production (different editors/spacing)
- Boolean fields mysteriously become strings
- Nested structures flatten unexpectedly

**Source:** [JSON vs YAML vs TOML Comparison](https://devtoolbox.dedyn.io/blog/json-vs-yaml-vs-toml), [HOCON vs YAML vs TOML](https://singhajit.com/configuration-file-formats-comparison/)

### Pitfall 3: Tight Coupling Between Domain and Storage

**What goes wrong:** Adding database tags directly to domain types (`json:"..."`, `db:"..."`) makes domain logic dependent on storage implementation.

**Why it happens:** It's faster to add tags inline than maintain separate transport types and conversion functions.

**How to avoid:**
- Define domain types in `internal/domain` with zero external dependencies
- Create backend-specific transport types (e.g., `serverModel`, `sshConfigEntry`)
- Write `toDomain()` and `fromDomain()` conversion functions at backend boundaries
- Keep validation logic in domain layer, not storage layer

**Warning signs:**
- Import statements in domain package reference backend libraries (`encoding/json`, `database/sql`)
- Tests for domain logic require mocking storage
- Changing backend implementation requires changing domain types

**Source:** [Repository Pattern in Go](https://threedots.tech/post/repository-pattern-in-go/), [DDD in Go](https://programmingpercy.tech/blog/how-to-domain-driven-design-ddd-golang/)

### Pitfall 4: Missing `Unwrap()` on Custom Errors

**What goes wrong:** Custom error types don't work with `errors.Is()` or `errors.As()`, breaking error handling patterns.

**Why it happens:** Forgetting to implement `Unwrap() error` method when wrapping errors.

**How to avoid:**
- Always implement `Unwrap()` for error types that wrap other errors
- Test error handling with `errors.Is()` and `errors.As()` in unit tests
- Use `fmt.Errorf("context: %w", err)` for simple wrapping

**Warning signs:**
- `errors.Is()` returns false for errors that should match
- Error chains can't be inspected by calling code
- Losing context when errors bubble up through layers

**Source:** [Error Handling and Go](https://go.dev/blog/error-handling-and-go), [Custom Errors in Go](https://www.digitalocean.com/community/tutorials/creating-custom-errors-in-go)

### Pitfall 5: Performance Cost of Type Assertions in Hot Paths

**What goes wrong:** Using type assertions to check optional interfaces in performance-critical loops degrades performance.

**Why it happens:** Interface conversions and type assertions have non-trivial CPU cost.

**How to avoid:**
- Cache type assertion results at backend initialization, not per-operation
- Avoid type assertions in tight loops
- For frequently-called operations, consider storing capability flags on a wrapper type

**Example:**
```go
// Bad: type assertion on every call
func ListServers(backend Backend, filters map[string]string) ([]*Server, error) {
    if filterer, ok := backend.(Filterer); ok {
        return filterer.FilterServers(context.TODO(), filters)
    }
    // fallback path
}

// Good: cache capability at initialization
type BackendWrapper struct {
    backend  Backend
    canFilter bool
}

func NewBackendWrapper(backend Backend) *BackendWrapper {
    _, canFilter := backend.(Filterer)
    return &BackendWrapper{
        backend: backend,
        canFilter: canFilter,
    }
}
```

**Warning signs:**
- Type assertions inside `for` loops
- Same interface check repeated across multiple function calls
- Profiling shows type assertion overhead

**Source:** [Plugin Architecture with Interface Extension](https://www.dolthub.com/blog/2022-09-12-golang-interface-extension/), [Go Type Assertions Performance](https://www.slingacademy.com/article/type-assertions-and-type-switches-with-interfaces-in-go/)

## Code Examples

Verified patterns from official sources:

### SSH Config Parsing

```go
// Source: https://github.com/kevinburke/ssh_config
package sshconfig

import (
    "github.com/kevinburke/ssh_config"
    "os"
    "path/filepath"
)

// GetSSHConfigValue retrieves a single config value for a host
func GetSSHConfigValue(host, key string) (string, error) {
    // Reads from $HOME/.ssh/config by default
    return ssh_config.Get(host, key), nil
}

// GetAllIdentityFiles retrieves all identity files for a host
func GetAllIdentityFiles(host string) []string {
    return ssh_config.GetAll(host, "IdentityFile")
}

// LoadCustomConfig parses a custom config file
func LoadCustomConfig(configPath string) (*ssh_config.Config, error) {
    f, err := os.Open(configPath)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    return ssh_config.Decode(f)
}

// Example usage for building domain.Server from SSH config:
func ServerFromSSHConfig(host string) (*domain.Server, error) {
    cfg, err := LoadCustomConfig(filepath.Join(os.Getenv("HOME"), ".ssh", "config"))
    if err != nil {
        return nil, err
    }

    return &domain.Server{
        ID:           host, // or generate UUID
        Host:         cfg.Get(host, "HostName"),
        User:         cfg.Get(host, "User"),
        Port:         parsePort(cfg.Get(host, "Port")),
        IdentityFile: cfg.Get(host, "IdentityFile"),
        Proxy:        cfg.Get(host, "ProxyJump"),
        DisplayName:  host,
        // ... other fields
    }, nil
}
```

### XDG Config File Management

```go
// Source: https://github.com/adrg/xdg
package config

import (
    "fmt"
    "os"

    "github.com/adrg/xdg"
    "github.com/BurntSushi/toml"
)

type Config struct {
    Backend string `toml:"backend"`
    // Future: backend-specific config sections
}

func LoadOrDefault() (*Config, error) {
    configPath, err := xdg.SearchConfigFile("sshjesus/config.toml")
    if err != nil {
        // Config doesn't exist yet — return default config
        return &Config{
            Backend: "", // Empty means setup wizard should run
        }, nil
    }

    var cfg Config
    if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
        return nil, fmt.Errorf("malformed config file at %s: %w", configPath, err)
    }

    return &cfg, nil
}

func (c *Config) Save() error {
    configPath, err := xdg.ConfigFile("sshjesus/config.toml")
    if err != nil {
        return fmt.Errorf("failed to create config directory: %w", err)
    }

    f, err := os.Create(configPath)
    if err != nil {
        return fmt.Errorf("failed to create config file: %w", err)
    }
    defer f.Close()

    if err := toml.NewEncoder(f).Encode(c); err != nil {
        return fmt.Errorf("failed to encode config: %w", err)
    }

    return nil
}
```

### Mock Backend Implementation

```go
// Source: Testify patterns from https://pkg.go.dev/github.com/stretchr/testify/mock
package mock

import (
    "context"
    "sync"

    "github.com/yourorg/sshjesus/internal/domain"
    "github.com/yourorg/sshjesus/internal/errors"
)

type Backend struct {
    mu       sync.RWMutex
    servers  map[string]*domain.Server
    projects map[string]*domain.Project
    closed   bool
}

func NewBackend() *Backend {
    return &Backend{
        servers:  make(map[string]*domain.Server),
        projects: make(map[string]*domain.Project),
    }
}

func (b *Backend) GetServer(ctx context.Context, id string) (*domain.Server, error) {
    b.mu.RLock()
    defer b.mu.RUnlock()

    if b.closed {
        return nil, errors.ErrBackendUnavailable
    }

    server, exists := b.servers[id]
    if !exists {
        return nil, errors.ErrServerNotFound
    }

    // Return copy to prevent external mutation
    serverCopy := *server
    return &serverCopy, nil
}

func (b *Backend) ListServers(ctx context.Context) ([]*domain.Server, error) {
    b.mu.RLock()
    defer b.mu.RUnlock()

    if b.closed {
        return nil, errors.ErrBackendUnavailable
    }

    servers := make([]*domain.Server, 0, len(b.servers))
    for _, server := range b.servers {
        serverCopy := *server
        servers = append(servers, &serverCopy)
    }

    return servers, nil
}

func (b *Backend) Close() error {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.closed = true
    return nil
}

// Implement Writer interface (optional)
func (b *Backend) CreateServer(ctx context.Context, server *domain.Server) error {
    b.mu.Lock()
    defer b.mu.Unlock()

    if b.closed {
        return errors.ErrBackendUnavailable
    }

    serverCopy := *server
    b.servers[server.ID] = &serverCopy
    return nil
}

func (b *Backend) UpdateServer(ctx context.Context, server *domain.Server) error {
    b.mu.Lock()
    defer b.mu.Unlock()

    if b.closed {
        return errors.ErrBackendUnavailable
    }

    if _, exists := b.servers[server.ID]; !exists {
        return errors.ErrServerNotFound
    }

    serverCopy := *server
    b.servers[server.ID] = &serverCopy
    return nil
}

func (b *Backend) DeleteServer(ctx context.Context, id string) error {
    b.mu.Lock()
    defer b.mu.Unlock()

    if b.closed {
        return errors.ErrBackendUnavailable
    }

    if _, exists := b.servers[id]; !exists {
        return errors.ErrServerNotFound
    }

    delete(b.servers, id)
    return nil
}

// Helper method for testing
func (b *Backend) Seed(servers ...*domain.Server) {
    b.mu.Lock()
    defer b.mu.Unlock()

    for _, server := range servers {
        serverCopy := *server
        b.servers[server.ID] = &serverCopy
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `pkg/errors.Wrap()` | `fmt.Errorf()` with `%w` | Go 1.13 (2019) | Standard library error wrapping eliminates third-party dependency |
| `dep`, `glide` for dependencies | `go mod` | Go 1.13+ (standard since 1.16) | Built-in module management, reproducible builds |
| Manual mock implementations | `testify/mock` or `gomock` | Ongoing | Reduced boilerplate, better assertion tooling |
| SQLBoiler for codegen | `sqlc` | 2026 | sqlc is now recommended for new projects (SQLBoiler in maintenance mode) |
| Hardcoded config paths | XDG Base Directory Spec | Linux standard since ~2010 | Cross-platform config management, respects user preferences |
| `interface{}` for type erasure | `any` alias | Go 1.18 (2022) | Clearer intent, same behavior |
| Testing without concurrency control | `testing/synctest` | Go 1.24 (Feb 2025) | Isolated concurrent testing with fake clock |

**Deprecated/outdated:**
- **github.com/pkg/errors**: Replaced by standard library `fmt.Errorf()` with `%w` and `errors.Is()`/`errors.As()`
- **SQLBoiler**: Now in maintenance mode; use `sqlc` for new database code generation (not relevant for Phase 1)
- **Pre-modules dependency managers** (`dep`, `glide`, `godep`): Use `go mod` exclusively

## Open Questions

1. **TOML vs YAML for config format**
   - What we know: TOML eliminates indentation/type coercion bugs, YAML has broader DevOps familiarity
   - What's unclear: User preference — does the target audience expect YAML because of Docker/K8s familiarity?
   - Recommendation: Choose TOML for clarity and safety, document example config prominently. Can add YAML support later if community requests it.

2. **Should Backend.Close() return error?**
   - What we know: `database/sql.DB.Close()` returns error, but close operations rarely fail in Go
   - What's unclear: Mock backend doesn't need it, but 1Password SDK might need cleanup
   - Recommendation: Keep `Close() error` signature for consistency with `io.Closer` interface, allows future backends to handle cleanup errors

3. **Domain model validation: where should it live?**
   - What we know: Domain-first approach says validation in domain package, not storage layer
   - What's unclear: Who validates? Backend on write? Application layer before calling backend?
   - Recommendation: Validation lives in domain package as methods (e.g., `Server.Validate() error`), application layer calls before backend write operations

4. **Context usage: should all interface methods accept context.Context?**
   - What we know: Modern Go best practice is to pass context for cancellation/timeouts
   - What's unclear: Phase 1 has no network operations, but future backends (1Password SDK) will need context
   - Recommendation: Include `context.Context` in all interface methods now, prevents breaking API changes later

## Sources

### Primary (HIGH confidence)
- [Go Error Handling - Official Blog](https://go.dev/blog/error-handling-and-go)
- [Go Modules Layout - Official Docs](https://go.dev/doc/modules/layout)
- [database/sql/driver Package - Go Packages](https://pkg.go.dev/database/sql/driver)
- [Standard Go Project Layout - GitHub](https://github.com/golang-standards/project-layout)
- [Repository Pattern in Go - Three Dots Labs](https://threedots.tech/post/repository-pattern-in-go/)
- [Design Patterns in database/sql - Eli Bendersky](https://eli.thegreenplace.net/2019/design-patterns-in-gos-databasesql-package/)
- [adrg/xdg - GitHub](https://github.com/adrg/xdg)
- [kevinburke/ssh_config - GitHub](https://github.com/kevinburke/ssh_config)

### Secondary (MEDIUM confidence)
- [Go Testing Excellence 2026 - DasRoot](https://dasroot.net/posts/2026/01/go-testing-excellence-table-driven-tests-mocking/)
- [JSON vs YAML vs TOML Comparison - DevToolbox](https://devtoolbox.dedyn.io/blog/json-vs-yaml-vs-toml)
- [Plugin Architecture with Interface Extension - DoltHub](https://www.dolthub.com/blog/2022-09-12-golang-interface-extension/)
- [Creating Custom Errors in Go - DigitalOcean](https://www.digitalocean.com/community/tutorials/creating-custom-errors-in-go)
- [1Password Go SDK - GitHub](https://github.com/1Password/onepassword-sdk-go)
- [Go Interfaces Best Practices - Boot.dev](https://blog.boot.dev/golang/golang-interfaces/)
- [Custom Error Types with Stack Traces - OneUpTime](https://oneuptime.com/blog/post/2026-01-30-how-to-create-custom-error-types-with-stack-traces-in-go/view)

### Tertiary (LOW confidence)
- [Go Roadmap 2026 - TheLinuxCode](https://thelinuxcode.com/go-roadmap-a-complete-guide-for-2026-ready-backend-engineering/)
- [Configuration Format Comparison - AnBowell](https://www.anbowell.com/blog/an-in-depth-comparison-of-json-yaml-and-toml/)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Official Go packages and widely-adopted libraries with stable APIs
- Architecture: HIGH - Patterns verified from official Go blog, standard library design, and production-tested projects
- Pitfalls: MEDIUM-HIGH - Common issues documented across multiple sources, some based on community experience rather than official docs
- Config format choice: MEDIUM - Tradeoffs are well-documented, but actual choice depends on user preference

**Research date:** 2026-02-14
**Valid until:** 2026-03-14 (30 days for stable tooling; Go ecosystem changes slowly)

**Notes:**
- Go 1.24 is current (released Feb 2025), features like `testing/synctest` are cutting-edge
- 1Password SDK is in beta (v0.x), expect API changes before v1
- XDG, ssh_config libraries are mature and stable
- Backend interface pattern is proven (database/sql is 10+ years old)
