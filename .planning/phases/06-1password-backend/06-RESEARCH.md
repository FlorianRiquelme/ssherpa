# Phase 6: 1Password Backend - Research

**Researched:** 2026-02-14
**Domain:** 1Password SDK integration, credential syncing, offline fallback, team vault sharing
**Confidence:** HIGH

## Summary

Phase 6 implements a 1Password backend that stores SSH server configurations in 1Password items, syncing them to local storage (both SSH config include file and ssherpa TOML) for offline availability. The 1Password Go SDK (v0.4.0-beta.2) supports two authentication methods: desktop app integration (with biometric/OS-level auth prompts) and service accounts (for automation). The architecture uses tag-based discovery (`ssherpa` tag) to identify managed items across all accessible vaults, enabling vault-per-customer organization without hardcoded vault configuration.

**Primary recommendation:** Use 1Password SDK with desktop app integration for primary authentication (better UX, no token management), implement continuous sync to local storage (SSH config include file + TOML), detect when 1Password becomes unavailable and gracefully fall back to cached servers with clear UI indicators.

**Key insight:** 1Password is the source of truth; local storage is a read-only cache. When 1Password is unavailable (locked, not running, auth expired), immediately show ssh-config servers + persistent banner prompting to unlock. Auto-detect when 1Password becomes available and reload.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Setup & onboarding
- First launch triggers an interactive wizard prompting which backend to use (ssh-config or 1Password for v1)
- Wizard walks user through 1Password setup step-by-step (detect app, pick vaults, confirm)
- Multiple backends can be active simultaneously — servers from all backends merge into one unified list
- No visual distinction between backends in the TUI — source is an implementation detail

#### What gets stored
- Each 1Password item = one full server config (hostname, user, port, key path, project tags, remote project path)
- Remote project path stored per server — enables `ssh user@host -t 'cd /path && $SHELL'` to land in the right directory
- Connection history stays local (personal to each machine, not synced)
- When 1Password is the backend, project-to-server assignments are stored in the 1Password item (not local TOML) — team sees same project groupings
- When a server exists in both ssh-config AND 1Password, 1Password wins

#### Team sharing model
- ssherpa scans ALL accessible vaults (not configured to specific vaults)
- Items are discoverable via a specific tag (e.g., `ssherpa`) — that's how ssherpa identifies its managed items
- Existing vault-per-customer organization stays as-is — ssherpa works with whatever vault structure exists
- Migration wizard offered to convert existing unstructured SSH items to ssherpa format with proper tags
- Real-time sync — changes by team members appear immediately
- Personal vault items are supported — users can tag items in their Private vault for personal-only servers

#### Sync to local storage
- 1Password is the source of truth; synced down to local for offline/fallback
- Sync targets: both `~/.ssh/ssherpa_config` (include file) AND ssherpa local TOML config
- SSH config sync uses a separate include file (`~/.ssh/ssherpa_config`) with an `Include` directive added to `~/.ssh/config` — fully isolated, never touches user's existing SSH entries
- Local TOML gets the extra ssherpa-specific fields (project path, project tags, custom metadata)
- Sync triggers: on launch + on every change (add/edit/remove in 1Password)
- Conflict detection: if a server exists in both 1Password and user's original ssh-config (not the synced include file), show a warning and let the user decide

#### Fallback behavior
- When 1Password is unavailable (not running, locked): immediately show ssh-config servers + persistent banner prompting to unlock 1Password
- Auto-detect when 1Password becomes available mid-session and automatically load its servers
- Clear warning bar when auth fails (expired token, revoked access) — keep working with available backends
- When no backend is configured at all: show empty TUI with setup prompt (call-to-action to configure)

### Claude's Discretion
- 1Password SDK authentication approach (desktop app integration, service accounts)
- Item field mapping and custom field naming in 1Password
- Tag naming convention for ssherpa-managed items
- Sync conflict resolution UI details
- Include directive placement strategy in ssh-config
- Polling interval for auto-detect when 1Password becomes available

</user_constraints>

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/1Password/onepassword-sdk-go` | v0.4.0-beta.2 | 1Password SDK integration | Official SDK, supports desktop app auth (macOS/Windows/Linux), vault/item CRUD, tag-based filtering |
| `github.com/kevinburke/ssh_config` | v1.4.0+ | Write SSH config include file | Already in use (Phase 2), preserves formatting, handles Include directives |
| `github.com/google/renameio/v2` | v2.0+ | Atomic file writes | Already in use (Phase 5), prevents corruption during sync |
| `github.com/BurntSushi/toml` | Latest | Parse/write TOML config | Already in use (Phase 1), for local ssherpa TOML storage |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `time.Ticker` (stdlib) | Go 1.21+ | Periodic 1Password availability polling | Detect when locked app becomes available |
| `context.Context` (stdlib) | Go 1.21+ | SDK call timeouts and cancellation | All 1Password SDK operations |
| `sync.RWMutex` (stdlib) | Go 1.21+ | Protect cached server state | Concurrent reads during sync |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Desktop app auth | Service accounts | Service accounts require token management and don't support biometric auth; desktop app is better UX for individual developers |
| Desktop app auth | 1Password Connect (self-hosted) | Connect requires infrastructure setup; overkill for individual/small team use case |
| Tag-based discovery | Vault-specific config | Tag-based discovery works with existing vault structures; vault config is rigid and breaks when vaults are reorganized |
| Continuous sync | On-demand sync (user-triggered) | Continuous sync enables "real-time" team collaboration; on-demand requires manual refresh |

**Installation:**
```bash
go get github.com/1Password/onepassword-sdk-go@v0.4.0-beta.2
```

---

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── backend/
│   ├── interface.go          # Existing backend interface
│   ├── onepassword/          # NEW: 1Password backend
│   │   ├── backend.go        # Backend implementation
│   │   ├── client.go         # SDK wrapper
│   │   ├── sync.go           # Sync to local storage
│   │   ├── mapping.go        # Item ↔ Server conversion
│   │   └── auth.go           # Desktop app integration
│   └── sshconfig/            # Existing SSH config backend
├── sync/                      # NEW: Local cache management
│   ├── ssh_include.go        # Write ~/.ssh/ssherpa_config
│   ├── toml_cache.go         # Write local TOML cache
│   └── conflict.go           # Conflict detection logic
└── ui/                        # TUI (existing)
    └── status_bar.go          # UPDATE: Add 1Password status indicator
```

### Pattern 1: Desktop App Integration (Primary Authentication)

**What:** Use 1Password desktop app for authentication with biometric/OS-level prompts

**When to use:** Default for individual developers and small teams

**Example:**
```go
// Source: https://developer.1password.com/docs/sdks/desktop-app-integrations/
import (
    "context"
    "github.com/1Password/onepassword-sdk-go"
)

func NewDesktopAppClient(accountName string) (*onepassword.Client, error) {
    ctx := context.Background()

    // Desktop app integration - user authorizes via biometric/password prompt
    client, err := onepassword.NewClient(
        ctx,
        onepassword.WithDesktopAppIntegration(accountName),
        onepassword.WithIntegrationInfo("ssherpa", "v0.1.0"),
    )
    if err != nil {
        return nil, fmt.Errorf("create 1Password client: %w", err)
    }

    return client, nil
}

// Get account name from user's 1Password app
// User sees this at top-left of 1Password sidebar
func GetAccountName() string {
    // Example: "john@example.com" or "ACME Corp"
    // Can also use account UUID from `op account list --format json`
    return os.Getenv("OP_ACCOUNT_NAME")
}
```

**Key insight:** Desktop app integration requires 1Password app to be unlocked. When locked, client operations fail with `DesktopSessionExpiredError`. This is the trigger for showing "unlock 1Password" banner.

### Pattern 2: Tag-Based Item Discovery

**What:** Filter 1Password items by tag (`ssherpa`) to identify managed servers

**When to use:** To discover servers across ALL accessible vaults without hardcoding vault IDs

**Example:**
```go
// Source: https://developer.1password.com/docs/sdks/list-vaults-items
import (
    "context"
    "github.com/1Password/onepassword-sdk-go"
)

func ListManagedServers(client *onepassword.Client) ([]*domain.Server, error) {
    ctx := context.Background()

    // List all vaults user has access to
    vaults, err := client.Vaults().List(ctx)
    if err != nil {
        return nil, fmt.Errorf("list vaults: %w", err)
    }

    var servers []*domain.Server

    // Search each vault for items with "ssherpa" tag
    for {
        vault, err := vaults.Next()
        if errors.Is(err, onepassword.ErrorIteratorDone) {
            break
        }
        if err != nil {
            return nil, fmt.Errorf("iterate vaults: %w", err)
        }

        // List items in vault (filters by tag on SDK side if supported)
        items, err := client.Items().ListAll(ctx, vault.ID)
        if err != nil {
            // Log error but continue (vault might be inaccessible)
            log.Printf("skip vault %s: %v", vault.Name, err)
            continue
        }

        for {
            item, err := items.Next()
            if errors.Is(err, onepassword.ErrorIteratorDone) {
                break
            }
            if err != nil {
                continue
            }

            // Check for "ssherpa" tag
            if hasTag(item.Tags, "ssherpa") {
                server, err := itemToServer(item)
                if err != nil {
                    log.Printf("skip item %s: %v", item.Title, err)
                    continue
                }
                servers = append(servers, server)
            }
        }
    }

    return servers, nil
}

func hasTag(tags []string, target string) bool {
    for _, tag := range tags {
        if strings.EqualFold(tag, target) {
            return true
        }
    }
    return false
}
```

**Key insight:** Tag-based discovery avoids hardcoding vault IDs/names. Works with existing vault structures (vault-per-customer). Personal vault items are automatically included if tagged.

### Pattern 3: Item to Server Mapping

**What:** Store SSH server config in 1Password item using custom fields

**When to use:** Converting between 1Password item format and ssherpa domain model

**Example:**
```go
// Source: https://developer.1password.com/docs/sdks/manage-items
import "github.com/1Password/onepassword-sdk-go"

// 1Password item structure:
// - Category: Server (or Login for SSH keys)
// - Title: Server alias (e.g., "prod-web-01")
// - Fields:
//   - hostname (text)
//   - user (text)
//   - port (text, default "22")
//   - identity_file (text, path to key)
//   - remote_project_path (text, e.g., "/var/www/app")
//   - project_tags (text, comma-separated, e.g., "payments-api,backend")
//   - proxy_jump (text, optional)
//   - forward_agent (text, "yes" or "no")
//   - extra_config (concealed, free-form SSH directives)
// - Tags: ["ssherpa"]

func itemToServer(item *onepassword.Item) (*domain.Server, error) {
    server := &domain.Server{
        ID:          item.ID,
        DisplayName: item.Title,
        Tags:        item.Tags,
    }

    // Extract fields by label
    for _, field := range item.Fields {
        switch strings.ToLower(field.Label) {
        case "hostname":
            server.Host = field.Value
        case "user":
            server.User = field.Value
        case "port":
            if field.Value != "" {
                port, err := strconv.Atoi(field.Value)
                if err != nil {
                    return nil, fmt.Errorf("invalid port: %w", err)
                }
                server.Port = port
            } else {
                server.Port = 22
            }
        case "identity_file":
            server.IdentityFile = field.Value
        case "remote_project_path":
            server.RemoteProjectPath = field.Value
        case "project_tags":
            if field.Value != "" {
                server.ProjectTags = strings.Split(field.Value, ",")
            }
        case "proxy_jump":
            server.Proxy = field.Value
        case "forward_agent":
            server.ForwardAgent = field.Value == "yes"
        case "extra_config":
            server.ExtraConfig = field.Value
        }
    }

    // Validation
    if server.Host == "" {
        return nil, fmt.Errorf("missing hostname field")
    }
    if server.User == "" {
        return nil, fmt.Errorf("missing user field")
    }

    return server, nil
}

func serverToItem(server *domain.Server, vaultID string) *onepassword.Item {
    item := &onepassword.Item{
        ID:       server.ID,
        Title:    server.DisplayName,
        Category: onepassword.Server,
        VaultID:  vaultID,
        Tags:     append(server.Tags, "ssherpa"),
        Fields: []*onepassword.ItemField{
            {Label: "hostname", Value: server.Host, Type: onepassword.FieldTypeText},
            {Label: "user", Value: server.User, Type: onepassword.FieldTypeText},
            {Label: "port", Value: strconv.Itoa(server.Port), Type: onepassword.FieldTypeText},
            {Label: "identity_file", Value: server.IdentityFile, Type: onepassword.FieldTypeText},
            {Label: "remote_project_path", Value: server.RemoteProjectPath, Type: onepassword.FieldTypeText},
            {Label: "project_tags", Value: strings.Join(server.ProjectTags, ","), Type: onepassword.FieldTypeText},
        },
    }

    if server.Proxy != "" {
        item.Fields = append(item.Fields, &onepassword.ItemField{
            Label: "proxy_jump",
            Value: server.Proxy,
            Type:  onepassword.FieldTypeText,
        })
    }

    if server.ExtraConfig != "" {
        item.Fields = append(item.Fields, &onepassword.ItemField{
            Label: "extra_config",
            Value: server.ExtraConfig,
            Type:  onepassword.FieldTypeConcealed,
        })
    }

    return item
}
```

### Pattern 4: Sync to Local Storage (SSH Include + TOML)

**What:** Write 1Password servers to both SSH config include file and local TOML

**When to use:** On startup and after every 1Password change

**Example:**
```go
// Source: Combines Phase 2 (ssh_config) and Phase 1 (TOML) patterns
import (
    "github.com/kevinburke/ssh_config"
    "github.com/BurntSushi/toml"
    "github.com/google/renameio/v2"
)

func SyncToLocalStorage(servers []*domain.Server) error {
    if err := syncToSSHConfig(servers); err != nil {
        return fmt.Errorf("sync SSH config: %w", err)
    }

    if err := syncToTOML(servers); err != nil {
        return fmt.Errorf("sync TOML: %w", err)
    }

    return nil
}

// Write ~/.ssh/ssherpa_config (SSH include file)
func syncToSSHConfig(servers []*domain.Server) error {
    home, _ := os.UserHomeDir()
    includePath := filepath.Join(home, ".ssh", "ssherpa_config")

    var buf bytes.Buffer
    buf.WriteString("# Generated by ssherpa - DO NOT EDIT MANUALLY\n")
    buf.WriteString("# Source: 1Password\n\n")

    for _, server := range servers {
        buf.WriteString(fmt.Sprintf("Host %s\n", server.DisplayName))
        buf.WriteString(fmt.Sprintf("    HostName %s\n", server.Host))
        buf.WriteString(fmt.Sprintf("    User %s\n", server.User))

        if server.Port != 22 {
            buf.WriteString(fmt.Sprintf("    Port %d\n", server.Port))
        }

        if server.IdentityFile != "" {
            buf.WriteString(fmt.Sprintf("    IdentityFile %s\n", server.IdentityFile))
        }

        if server.Proxy != "" {
            buf.WriteString(fmt.Sprintf("    ProxyJump %s\n", server.Proxy))
        }

        if server.ForwardAgent {
            buf.WriteString("    ForwardAgent yes\n")
        }

        // Extra config directives
        if server.ExtraConfig != "" {
            for _, line := range strings.Split(server.ExtraConfig, "\n") {
                if strings.TrimSpace(line) != "" {
                    buf.WriteString(fmt.Sprintf("    %s\n", line))
                }
            }
        }

        buf.WriteString("\n")
    }

    // Atomic write
    return renameio.WriteFile(includePath, buf.Bytes(), 0600)
}

// Write local TOML cache (ssherpa-specific fields)
func syncToTOML(servers []*domain.Server) error {
    configDir, _ := xdg.ConfigFile("ssherpa")
    cachePath := filepath.Join(configDir, "1password_cache.toml")

    type TOMLCache struct {
        LastSync time.Time
        Servers  []*domain.Server
    }

    cache := TOMLCache{
        LastSync: time.Now(),
        Servers:  servers,
    }

    var buf bytes.Buffer
    if err := toml.NewEncoder(&buf).Encode(cache); err != nil {
        return fmt.Errorf("encode TOML: %w", err)
    }

    return renameio.WriteFile(cachePath, buf.Bytes(), 0600)
}

// Ensure Include directive exists in ~/.ssh/config
func EnsureSSHInclude() error {
    home, _ := os.UserHomeDir()
    configPath := filepath.Join(home, ".ssh", "config")
    includePath := filepath.Join(home, ".ssh", "ssherpa_config")

    // Read current config
    content, err := os.ReadFile(configPath)
    if err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("read SSH config: %w", err)
    }

    includeDirective := fmt.Sprintf("Include %s", includePath)

    // Check if Include already exists
    if bytes.Contains(content, []byte(includeDirective)) {
        return nil // Already configured
    }

    // Prepend Include directive (ensures it's evaluated first)
    var buf bytes.Buffer
    buf.WriteString("# ssherpa 1Password integration\n")
    buf.WriteString(includeDirective + "\n\n")
    buf.Write(content)

    return renameio.WriteFile(configPath, buf.Bytes(), 0600)
}
```

**Key insight:** SSH include file enables standard SSH commands to work (`ssh prod-web-01`). TOML cache stores ssherpa-specific fields (project path, tags) that don't fit in SSH config.

### Pattern 5: Offline Fallback with Auto-Recovery

**What:** Detect when 1Password is unavailable, fall back to cached servers, auto-reload when available

**When to use:** Handle locked app, expired auth, or 1Password not running

**Example:**
```go
// Source: Patterns from https://pkg.go.dev/github.com/docker/go-healthcheck
import (
    "time"
    "github.com/1Password/onepassword-sdk-go"
)

type BackendStatus int

const (
    StatusAvailable BackendStatus = iota
    StatusLocked
    StatusUnavailable
    StatusUnknown
)

type OnePasswordBackend struct {
    client     *onepassword.Client
    status     BackendStatus
    statusMu   sync.RWMutex

    cachedServers []*domain.Server
    cacheMu       sync.RWMutex

    ticker        *time.Ticker
    stopCh        chan struct{}
}

func (b *OnePasswordBackend) Start() error {
    // Initial load
    if err := b.syncFromOnePassword(); err != nil {
        log.Printf("1Password unavailable, using cached servers: %v", err)
        b.setStatus(StatusUnavailable)

        // Load from local cache
        cachedServers, err := b.loadFromCache()
        if err != nil {
            return fmt.Errorf("1Password unavailable and no cache: %w", err)
        }
        b.setCachedServers(cachedServers)
    } else {
        b.setStatus(StatusAvailable)
    }

    // Start background sync (check every 5 seconds)
    b.ticker = time.NewTicker(5 * time.Second)
    b.stopCh = make(chan struct{})

    go b.pollAvailability()

    return nil
}

func (b *OnePasswordBackend) pollAvailability() {
    for {
        select {
        case <-b.ticker.C:
            if err := b.syncFromOnePassword(); err != nil {
                // 1Password unavailable
                if b.getStatus() == StatusAvailable {
                    log.Println("1Password became unavailable")
                    b.setStatus(StatusUnavailable)
                }
            } else {
                // 1Password available
                if b.getStatus() != StatusAvailable {
                    log.Println("1Password became available")
                    b.setStatus(StatusAvailable)
                }
            }
        case <-b.stopCh:
            return
        }
    }
}

func (b *OnePasswordBackend) syncFromOnePassword() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Try to list vaults (lightweight check)
    _, err := b.client.Vaults().List(ctx)
    if err != nil {
        // Check if session expired
        var sessionErr *onepassword.DesktopSessionExpiredError
        if errors.As(err, &sessionErr) {
            b.setStatus(StatusLocked)
            return fmt.Errorf("1Password locked: %w", err)
        }

        b.setStatus(StatusUnavailable)
        return fmt.Errorf("1Password unavailable: %w", err)
    }

    // Fetch servers
    servers, err := ListManagedServers(b.client)
    if err != nil {
        return fmt.Errorf("list servers: %w", err)
    }

    b.setCachedServers(servers)

    // Sync to local storage
    if err := SyncToLocalStorage(servers); err != nil {
        log.Printf("sync to local storage failed: %v", err)
    }

    return nil
}

func (b *OnePasswordBackend) ListServers(ctx context.Context) ([]*domain.Server, error) {
    b.cacheMu.RLock()
    defer b.cacheMu.RUnlock()

    // Return cached servers regardless of 1Password status
    return b.cachedServers, nil
}

func (b *OnePasswordBackend) GetStatus() (BackendStatus, error) {
    b.statusMu.RLock()
    defer b.statusMu.RUnlock()

    switch b.status {
    case StatusLocked:
        return b.status, fmt.Errorf("1Password is locked - unlock to sync")
    case StatusUnavailable:
        return b.status, fmt.Errorf("1Password is not running")
    case StatusAvailable:
        return b.status, nil
    default:
        return b.status, fmt.Errorf("unknown status")
    }
}

func (b *OnePasswordBackend) Stop() {
    if b.ticker != nil {
        b.ticker.Stop()
    }
    close(b.stopCh)
    b.client.Close()
}
```

**Key insight:** Polling every 5 seconds balances responsiveness (user unlocks app, sees servers load within 5s) with CPU usage. Status exposed to TUI for banner display.

### Pattern 6: Conflict Detection (1Password vs SSH Config)

**What:** Detect when a server exists in both 1Password AND user's original ssh-config

**When to use:** On sync, to warn users about potential conflicts

**Example:**
```go
// Source: Combine 1Password list with ssh_config parse
import "github.com/kevinburke/ssh_config"

type Conflict struct {
    ServerName   string
    Source1P     *domain.Server // From 1Password
    SourceSSH    *domain.Server // From SSH config
    WinnerSource string         // "1password" (always wins per requirement)
}

func DetectConflicts(onePasswordServers []*domain.Server) ([]Conflict, error) {
    // Parse user's original SSH config (NOT ssherpa_config include)
    home, _ := os.UserHomeDir()
    configPath := filepath.Join(home, ".ssh", "config")

    cfg, err := ssh_config.Decode(os.Open(configPath))
    if err != nil {
        return nil, fmt.Errorf("parse SSH config: %w", err)
    }

    // Build map of SSH config hosts
    sshHosts := make(map[string]*domain.Server)
    for _, host := range cfg.Hosts {
        for _, pattern := range host.Patterns {
            // Skip ssherpa_config entries (our own Include)
            if strings.Contains(pattern.String(), "ssherpa") {
                continue
            }

            sshHosts[pattern.String()] = &domain.Server{
                DisplayName: pattern.String(),
                Host:        cfg.Get(pattern.String(), "HostName"),
                User:        cfg.Get(pattern.String(), "User"),
                // ... other fields
            }
        }
    }

    // Find conflicts
    var conflicts []Conflict
    for _, onePasswordServer := range onePasswordServers {
        if sshServer, exists := sshHosts[onePasswordServer.DisplayName]; exists {
            conflicts = append(conflicts, Conflict{
                ServerName:   onePasswordServer.DisplayName,
                Source1P:     onePasswordServer,
                SourceSSH:    sshServer,
                WinnerSource: "1password", // Per requirement
            })
        }
    }

    return conflicts, nil
}

// In TUI, display conflicts as warning banner:
// "⚠ Server 'prod-web' exists in both 1Password and SSH config. Using 1Password version."
```

### Anti-Patterns to Avoid

- **Storing connection history in 1Password:** Connection timestamps are personal/local. Syncing creates noise for team members.
- **Hardcoding vault IDs:** Tag-based discovery is vault-agnostic. Hardcoding breaks when vaults are renamed/reorganized.
- **Synchronous SDK calls in TUI update loop:** Always use goroutines + channels for SDK operations. SDK calls can take 100-500ms.
- **Aggressive polling (<1s interval):** 1Password SDK has rate limits. Poll every 5 seconds minimum.
- **Editing items without checking vault permissions:** User may have read-only access to shared vaults. Check `item.Vault.Permissions` before write operations.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 1Password authentication | Custom 1Password API calls | 1Password SDK with desktop app integration | SDK handles biometric prompts, session management, token refresh, and cross-platform differences |
| Item tag filtering | Fetch all items + filter in Go | SDK's tag filtering (if available) or vault iteration | SDK can filter server-side, reducing network traffic |
| Atomic config writes | Manual temp file + os.Rename | `google/renameio` (already in use) | Already proven in Phase 5; handles fsync correctly |
| Availability detection | Process name checks (ps/tasklist) | SDK health check + error type inspection | Desktop app process may be running but locked; SDK errors (`DesktopSessionExpiredError`) are definitive |
| SSH config include management | Manual string manipulation | Phase 2 patterns with `kevinburke/ssh_config` | Already solved; reuse existing code |

**Key insight:** 1Password SDK is in beta (v0.4.0-beta.2), but it's the official/only way to integrate with desktop app. API stability is acceptable (breaking changes noted in release notes, migration path provided).

---

## Common Pitfalls

### Pitfall 1: Desktop App Must Be Unlocked for Auth

**What goes wrong:** User sees "1Password authentication failed" with no guidance. Actual issue: 1Password app is locked.

**Why it happens:** Desktop app integration requires unlocked app. When locked, SDK returns `DesktopSessionExpiredError`.

**How to avoid:**
- Catch `DesktopSessionExpiredError` specifically.
- Show banner: "🔒 1Password is locked. Unlock the 1Password app to sync servers."
- Auto-recover when user unlocks (polling detects this).

**Warning signs:** Users report "can't connect" but 1Password app is running (just locked).

---

### Pitfall 2: Tag Case Sensitivity

**What goes wrong:** User tags item with "SSHJesus" but app searches for "ssherpa". Item not found.

**Why it happens:** 1Password tags are case-sensitive in API but displayed case-insensitively in UI.

**How to avoid:**
- Use case-insensitive tag comparison: `strings.EqualFold(tag, "ssherpa")`.
- Document canonical tag name in wizard: "Tag items with: ssherpa (lowercase)".

**Warning signs:** Items tagged in 1Password app don't appear in ssherpa.

---

### Pitfall 3: Include Directive Placement

**What goes wrong:** User's SSH config has `Host *` as first entry. Include directive is appended, so 1Password servers never match.

**Why it happens:** SSH config uses first-match-wins. If `Host *` appears before `Include`, wildcard matches everything.

**How to avoid:**
- **Always prepend** Include directive to top of `~/.ssh/config`.
- Check for existing `Host *` entries above Include. Warn user if detected.
- Alternative: Insert Include **before** first `Host *` block (requires parsing existing config).

**Warning signs:** `ssh prod-web-01` connects to wrong server or fails (matches wrong Host block).

---

### Pitfall 4: Vault Permission Errors

**What goes wrong:** User tries to edit server from shared vault. Write fails with "permission denied".

**Why it happens:** Shared vaults may be read-only for some users.

**How to avoid:**
- Check `item.Vault.Permissions` before showing edit/delete options.
- Gray out edit button for read-only items.
- Show error: "This server is in a read-only vault. Contact vault owner to edit."

**Warning signs:** Edit works for personal vault items but fails for shared items.

---

### Pitfall 5: Sync Loop During Writes

**What goes wrong:** User edits server in TUI. Change syncs to 1Password. Sync triggers re-load, causing flicker/lag.

**Why it happens:** Polling detects "change" (user's own edit) and triggers reload.

**How to avoid:**
- Track last write timestamp.
- Skip sync if last write was <10 seconds ago (debounce).
- Alternative: Use SDK's event streaming (if available) to detect changes by others, not own writes.

**Warning signs:** TUI flickers or briefly shows stale data after user edit.

---

### Pitfall 6: Migration Wizard Overwrites Existing Items

**What goes wrong:** User runs migration wizard. Existing 1Password SSH items get re-created, causing duplicates.

**Why it happens:** Migration doesn't check if item already exists before creating.

**How to avoid:**
- Before creating item, search for existing item with same title.
- If found, ask user: "Item 'prod-web' already exists. Overwrite, skip, or rename?"
- Track migrated items in local state (don't re-migrate on second run).

**Warning signs:** Users report duplicate servers after running migration multiple times.

---

### Pitfall 7: Remote Project Path Not Synced to SSH Config

**What goes wrong:** User sets remote project path in 1Password. SSH config include file doesn't contain it (SSH has no directive for "change directory after login").

**Why it happens:** Remote project path is a ssherpa-specific feature. SSH config can't express "cd to path on login".

**How to avoid:**
- Store remote project path in local TOML cache only (not SSH config include).
- When user connects via ssherpa TUI, execute: `ssh user@host -t 'cd /remote/path && $SHELL'`.
- Document limitation: "Remote project path only works when connecting through ssherpa TUI."

**Warning signs:** User expects `ssh prod-web` (standard SSH command) to land in project dir, but it doesn't.

---

## Code Examples

Verified patterns from official sources:

### Desktop App Authentication

```go
// Source: https://developer.1password.com/docs/sdks/desktop-app-integrations/
import (
    "context"
    "github.com/1Password/onepassword-sdk-go"
)

func main() {
    ctx := context.Background()

    // Account name = what user sees at top-left of 1Password app
    accountName := "john@example.com"

    client, err := onepassword.NewClient(
        ctx,
        onepassword.WithDesktopAppIntegration(accountName),
        onepassword.WithIntegrationInfo("ssherpa", "v0.1.0"),
    )
    if err != nil {
        log.Fatalf("create client: %v", err)
    }
    defer client.Close()

    // List all accessible vaults
    vaults, err := client.Vaults().List(ctx)
    if err != nil {
        // Check if session expired (app locked)
        var sessionErr *onepassword.DesktopSessionExpiredError
        if errors.As(err, &sessionErr) {
            log.Fatal("1Password is locked. Please unlock the app.")
        }
        log.Fatalf("list vaults: %v", err)
    }

    for {
        vault, err := vaults.Next()
        if errors.Is(err, onepassword.ErrorIteratorDone) {
            break
        }
        if err != nil {
            log.Printf("iterate vaults: %v", err)
            continue
        }

        fmt.Printf("Vault: %s (ID: %s)\n", vault.Name, vault.ID)
    }
}
```

### Service Account Authentication (Alternative)

```go
// Source: https://developer.1password.com/docs/sdks/setup-tutorial/
import (
    "context"
    "os"
    "github.com/1Password/onepassword-sdk-go"
)

func NewServiceAccountClient() (*onepassword.Client, error) {
    token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
    if token == "" {
        return nil, fmt.Errorf("OP_SERVICE_ACCOUNT_TOKEN not set")
    }

    ctx := context.Background()

    client, err := onepassword.NewClient(
        ctx,
        onepassword.WithServiceAccountToken(token),
        onepassword.WithIntegrationInfo("ssherpa", "v0.1.0"),
    )
    if err != nil {
        return nil, fmt.Errorf("create client: %w", err)
    }

    return client, nil
}
```

### Create/Update Item with Tags

```go
// Source: https://developer.1password.com/docs/sdks/manage-items
import "github.com/1Password/onepassword-sdk-go"

func CreateServerItem(client *onepassword.Client, server *domain.Server, vaultID string) error {
    ctx := context.Background()

    item := &onepassword.Item{
        Title:    server.DisplayName,
        Category: onepassword.Server,
        VaultID:  vaultID,
        Tags:     []string{"ssherpa"},
        Fields: []*onepassword.ItemField{
            {
                Label: "hostname",
                Value: server.Host,
                Type:  onepassword.FieldTypeText,
            },
            {
                Label: "user",
                Value: server.User,
                Type:  onepassword.FieldTypeText,
            },
            {
                Label: "port",
                Value: strconv.Itoa(server.Port),
                Type:  onepassword.FieldTypeText,
            },
            {
                Label: "identity_file",
                Value: server.IdentityFile,
                Type:  onepassword.FieldTypeText,
            },
            {
                Label: "remote_project_path",
                Value: server.RemoteProjectPath,
                Type:  onepassword.FieldTypeText,
            },
            {
                Label: "project_tags",
                Value: strings.Join(server.ProjectTags, ","),
                Type:  onepassword.FieldTypeText,
            },
        },
    }

    createdItem, err := client.Items().Create(ctx, item)
    if err != nil {
        return fmt.Errorf("create item: %w", err)
    }

    log.Printf("Created item: %s (ID: %s)", createdItem.Title, createdItem.ID)
    return nil
}

func UpdateServerItem(client *onepassword.Client, server *domain.Server) error {
    ctx := context.Background()

    // Fetch existing item
    item, err := client.Items().Get(ctx, server.ID, server.VaultID)
    if err != nil {
        return fmt.Errorf("get item: %w", err)
    }

    // Update fields
    item.Title = server.DisplayName

    for i, field := range item.Fields {
        switch strings.ToLower(field.Label) {
        case "hostname":
            item.Fields[i].Value = server.Host
        case "user":
            item.Fields[i].Value = server.User
        case "port":
            item.Fields[i].Value = strconv.Itoa(server.Port)
        case "remote_project_path":
            item.Fields[i].Value = server.RemoteProjectPath
        }
    }

    updatedItem, err := client.Items().Update(ctx, item)
    if err != nil {
        return fmt.Errorf("update item: %w", err)
    }

    log.Printf("Updated item: %s", updatedItem.Title)
    return nil
}
```

### List Items with Tag Filtering

```go
// Source: https://developer.1password.com/docs/sdks/list-vaults-items
import "github.com/1Password/onepassword-sdk-go"

func ListServersWithTag(client *onepassword.Client, tag string) ([]*domain.Server, error) {
    ctx := context.Background()

    vaults, err := client.Vaults().List(ctx)
    if err != nil {
        return nil, fmt.Errorf("list vaults: %w", err)
    }

    var servers []*domain.Server

    for {
        vault, err := vaults.Next()
        if errors.Is(err, onepassword.ErrorIteratorDone) {
            break
        }
        if err != nil {
            continue
        }

        items, err := client.Items().ListAll(ctx, vault.ID)
        if err != nil {
            log.Printf("skip vault %s: %v", vault.Name, err)
            continue
        }

        for {
            item, err := items.Next()
            if errors.Is(err, onepassword.ErrorIteratorDone) {
                break
            }
            if err != nil {
                continue
            }

            // Check tags
            for _, itemTag := range item.Tags {
                if strings.EqualFold(itemTag, tag) {
                    server, err := itemToServer(item)
                    if err != nil {
                        log.Printf("skip item %s: %v", item.Title, err)
                        break
                    }
                    servers = append(servers, server)
                    break
                }
            }
        }
    }

    return servers, nil
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 1Password Connect (self-hosted) | 1Password SDK with desktop app integration | 2024 (SDK beta) | Better UX (biometric auth), no infrastructure required |
| Service accounts only | Desktop app + service accounts | 2024 | Individual developers can use without token management |
| Manual vault configuration | Tag-based discovery | Best practice | Works with any vault structure, no hardcoded IDs |
| Poll 1Password API directly | SDK health checks + error types | 2024 (SDK release) | Reliable detection of locked app vs unavailable |
| Hardcoded sync intervals | Event-based sync (if available) | Future SDK feature | Real-time team collaboration without polling |

**Deprecated/outdated:**
- **1Password CLI (`op`):** Still works but requires separate installation. SDK is all-in-one.
- **1Password Connect:** Still valid for server/CI use cases, but overkill for individual developer tools.
- **Vault-specific configuration:** Rigid, breaks on org changes. Tag-based discovery is current best practice.

---

## Open Questions

### 1. SDK Stability During Beta

**What we know:**
- SDK is v0.4.0-beta.2 (not v1 yet).
- Desktop app integration added in v0.4.0-beta.1 (Nov 2024).
- 1Password commits to migration paths for breaking changes.

**What's unclear:**
- When v1.0 will release (stable API).
- Risk of breaking changes during ssherpa v1 development.

**Recommendation:**
- **Accept beta risk** — desktop app integration is killer feature (no token management).
- Pin SDK version in go.mod: `github.com/1Password/onepassword-sdk-go@v0.4.0-beta.2`.
- Monitor SDK releases, test breaking changes in dev before updating.
- Document in README: "Uses 1Password SDK (beta) — API may change."

---

### 2. Polling Interval for Availability Detection

**What we know:**
- Requirement: "Auto-detect when 1Password becomes available mid-session."
- Polling vs event-based sync.

**What's unclear:**
- Optimal polling interval (balance responsiveness vs CPU/battery).
- Whether SDK supports event streaming (websockets, file watchers).

**Recommendation:**
- **Start with 5-second polling** (responsive enough, low overhead).
- Make interval configurable: `SSHJESUS_1PASSWORD_POLL_INTERVAL=5s` env var.
- Future: Investigate SDK event streaming when v1 releases.

---

### 3. Conflict Resolution UI

**What we know:**
- Requirement: "If a server exists in both 1Password and user's original ssh-config, show a warning and let the user decide."
- 1Password always wins per requirement.

**What's unclear:**
- What "let the user decide" means if 1Password already wins.
- Should user be able to override winner source?

**Recommendation:**
- **Show warning banner with action:**
  - "⚠ Server 'prod-web' exists in both 1Password and SSH config. Using 1Password version. [View Conflict] [Dismiss]"
  - [View Conflict] → Detail screen showing both versions side-by-side.
  - [Dismiss] → Hide warning until next sync.
- No override — 1Password is always source of truth (per requirement).

---

### 4. Migration Wizard UX

**What we know:**
- Requirement: "Migration wizard offered to convert existing unstructured SSH items to ssherpa format with proper tags."
- Existing items may lack required fields (hostname, user).

**What's unclear:**
- How to handle incomplete items (missing hostname/user).
- Whether to migrate all SSH items or let user select.

**Recommendation:**
- **Interactive migration:**
  1. Scan all vaults for items matching "SSH" or "Server" category without `ssherpa` tag.
  2. Show list with checkboxes: "Select items to migrate."
  3. For each selected item:
     - If hostname/user present: Add `ssherpa` tag, done.
     - If missing: Show form to fill in required fields, then tag.
  4. Summary: "Migrated 15 items to ssherpa format."
- Save migration state (don't re-offer on next launch).

---

## Sources

### Primary (HIGH confidence)

- [1Password SDK for Go - GitHub](https://github.com/1Password/onepassword-sdk-go) - Official SDK, v0.4.0-beta.2 release notes
- [1Password SDK Setup Tutorial](https://developer.1password.com/docs/sdks/setup-tutorial/) - Service account authentication
- [Desktop App Integration Docs](https://developer.1password.com/docs/sdks/desktop-app-integrations/) - Biometric auth, session management
- [Manage Items with SDK](https://developer.1password.com/docs/sdks/manage-items) - CRUD operations, tags, custom fields
- [List Vaults and Items](https://developer.1password.com/docs/sdks/list-vaults-items) - Tag filtering, vault iteration
- [1Password SDK Concepts](https://developer.1password.com/docs/sdks/concepts/) - Authentication methods, item categories

### Secondary (MEDIUM confidence)

- [Offline-First Architecture in Android](https://www.droidcon.com/2025/12/16/the-complete-guide-to-offline-first-architecture-in-android/) - Conflict resolution patterns, cache strategies
- [Offline File Sync Developer Guide](https://daily.dev/blog/offline-file-sync-developer-guide-2024) - Sync conflict handling
- [Go SSH Config Parsers](https://github.com/kevinburke/ssh_config) - Already in use, Phase 2 patterns
- [Atomic File Writes in Go](https://michael.stapelberg.ch/posts/2017-01-28-golang_atomically_writing/) - Renameio patterns (Phase 5)

### Tertiary (LOW confidence)

- [Health Check Patterns](https://pkg.go.dev/github.com/docker/go-healthcheck) - Availability detection concepts
- [Process Monitoring in Go](https://apipark.com/techblog/en/how-to-monitor-custom-resources-with-go/) - Polling vs event-based sync
- [1Password Community Forum](https://www.1password.community/discussions/developers) - SDK discussions, beta feedback

---

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** - Official 1Password SDK, existing libraries (ssh_config, renameio, TOML)
- Architecture: **HIGH** - Patterns verified from official SDK docs, Phase 1/2/5 precedents
- Pitfalls: **MEDIUM-HIGH** - Common issues documented in SDK GitHub issues, beta limitations known
- Sync strategy: **HIGH** - Offline-first patterns are well-established, local cache is standard practice
- Conflict resolution: **MEDIUM** - User requirement is clear (1Password wins), but UI details are discretionary

**Research date:** 2026-02-14
**Valid until:** 30 days (SDK is beta, may have updates; stable libraries unlikely to change)

**Notes:**
- 1Password SDK is beta — expect API changes before v1.0. Pin version in go.mod.
- Desktop app integration requires 1Password app version 8.10.0+ (released Nov 2024).
- Tag-based discovery is recommended approach per 1Password best practices.
- Existing Phase 1/2/5 patterns provide foundation for sync and config management.

---

## Implementation Checklist

When planning Phase 6, ensure the plan addresses:

**Authentication:**
- [ ] Desktop app integration with account name prompt
- [ ] Fallback to service account if desktop app unavailable
- [ ] Error handling for locked app (`DesktopSessionExpiredError`)

**Item Management:**
- [ ] Tag-based discovery across all vaults
- [ ] Item ↔ Server mapping with all required fields
- [ ] Support for custom fields (remote_project_path, project_tags)
- [ ] Create, update, delete operations
- [ ] Vault permission checks before writes

**Sync:**
- [ ] Sync to `~/.ssh/ssherpa_config` (SSH include file)
- [ ] Sync to local TOML cache (ssherpa-specific fields)
- [ ] Ensure Include directive in `~/.ssh/config`
- [ ] Atomic writes with renameio (reuse Phase 5 pattern)
- [ ] Sync triggers: startup + periodic polling

**Offline Fallback:**
- [ ] Load from cache when 1Password unavailable
- [ ] Persistent status banner in TUI (locked/unavailable)
- [ ] Auto-recovery when 1Password becomes available
- [ ] Polling with configurable interval (default 5s)

**Conflict Detection:**
- [ ] Compare 1Password servers with SSH config hosts
- [ ] Show warning for duplicates (1Password always wins)
- [ ] Exclude ssherpa_config from conflict detection

**Migration:**
- [ ] Wizard to tag existing SSH items
- [ ] Handle incomplete items (missing fields)
- [ ] Track migrated items (don't re-migrate)
- [ ] Summary screen showing migration results

**Testing:**
- [ ] Unit tests for item ↔ server mapping
- [ ] Integration tests with mock 1Password client
- [ ] Test locked app scenario
- [ ] Test offline fallback with cached servers
- [ ] Test conflict detection logic

Sources:
- [Releases · 1Password/onepassword-sdk-go](https://github.com/1Password/onepassword-sdk-go/releases)
- [1Password SDKs](https://developer.1password.com/docs/sdks/)
- [Use the 1Password desktop app to authenticate 1Password SDKs](https://developer.1password.com/docs/sdks/desktop-app-integrations/)
- [List vaults and items using 1Password SDKs](https://developer.1password.com/docs/sdks/list-vaults-items)
- [Implementing Data Sync & Conflict Resolution Offline in Flutter](https://vibe-studio.ai/insights/implementing-data-sync-conflict-resolution-offline-in-flutter)
- [The Complete Guide to Offline-First Architecture in Android](https://www.droidcon.com/2025/12/16/the-complete-guide-to-offline-first-architecture-in-android/)
- [Offline File Sync: Developer Guide 2024](https://daily.dev/blog/offline-file-sync-developer-guide-2024)
