# Phase 5: Config Management - Research

**Researched:** 2026-02-14
**Domain:** SSH config file manipulation, TUI form design, file I/O safety
**Confidence:** HIGH

## Summary

This phase enables users to add, edit, and delete SSH connections through interactive forms within the existing Bubbletea TUI. The core challenge is safely modifying `~/.ssh/config` while preserving all comments, blank lines, and formatting—a non-negotiable requirement for config files users manually maintain.

Research shows the existing codebase already uses `kevinburke/ssh_config` for parsing (with comment preservation), but this library is **read-only** for practical purposes. Writing requires a different approach: either switch to `patrikkj/sshconf` (full read-write with formatting preservation) or implement custom Host block serialization. Forms will use Charm's `huh` library for a polished multi-field experience, with field-level validation triggering on blur (field exit). Atomic writes via `google/renameio` prevent corruption. An in-memory undo buffer enables recovery from accidental deletes within the session.

**Primary recommendation:** Switch to `patrikkj/sshconf` for bidirectional config operations (read + write with formatting preservation), build forms with `huh.Form` for validation and field management, and use `renameio.WriteFile` for atomic config writes.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Form interaction:**
- Full-screen form for add and edit (dedicated screen with labeled fields)
- Field navigation supports both Tab/Shift+Tab and j/k (Vim-style)
- Fields: Alias (required), Hostname (required), User (required), Port, IdentityFile, plus a free-text area for extra SSH config directives
- Free-text area allows any valid SSH config directive (ProxyJump, ForwardAgent, etc.)

**Validation & feedback:**
- Validation triggers on field exit (when user tabs/moves away from a field)
- Errors display inline below the invalid field, in red/warning color
- Required fields: Alias, Hostname, User
- Hostname performs DNS resolution check on save (catches typos early)

**Delete safety:**
- Delete triggered with 'd' key from the server list
- Confirmation requires typing the server alias to confirm (prevents accidental deletion)
- Session undo buffer: deleted entries stay in memory until session ends, 'u' key to undo last delete
- One server deleted at a time (no bulk delete)

**Config file handling:**
- Preserve all comments, blank lines, and indentation exactly as-is
- Only the modified Host block changes on edit
- New Host blocks appended at end of file
- Single backup before each write: ~/.ssh/config.bak (overwritten each time)
- Include directives: read-only (parse and display servers from included files, but only write to the main config file)

### Claude's Discretion

- Add/edit trigger keybinding (e.g. 'a' for add, 'e' for edit, or another pattern)
- Exact form layout and field spacing
- How free-text area renders and handles multi-line input
- Error message wording
- DNS check timeout and UX during the check (spinner, blocking, etc.)
- Undo buffer size and behavior details

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope

</user_constraints>

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `patrikkj/sshconf` | Latest | Parse & write SSH config with formatting preservation | Only Go library that preserves comments/whitespace on write operations; essential for user-maintained config files |
| `charmbracelet/huh` | v0.6+ | Terminal forms and prompts | Official Charm library for forms; handles multi-field validation, focus management, and blur events natively |
| `google/renameio` | v2.0+ | Atomic file writes | Industry-standard atomic write pattern; prevents corruption during config saves; POSIX-only (acceptable for SSH use case) |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `charmbracelet/bubbles/textarea` | v0.17+ | Multi-line text input | For free-text SSH config directives field (ProxyJump, ForwardAgent, etc.) |
| `net` (stdlib) | Go 1.21+ | DNS hostname validation | Built-in `net.LookupHost()` for hostname verification; no external deps needed |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `patrikkj/sshconf` | `kevinburke/ssh_config` | Current parser is read-only; would require custom Host block serialization and fragile comment/whitespace preservation logic |
| `huh.Form` | Custom Bubbletea form | Huh provides blur validation, focus management, and accessibility out-of-box; custom form = reinventing 500+ lines of code |
| `renameio` | Manual temp file + `os.Rename()` | Manual approach requires handling fsync, temp file cleanup, and cross-platform edge cases; renameio does this correctly |

**Installation:**
```bash
go get github.com/patrikkj/sshconf
go get github.com/charmbracelet/huh
go get github.com/google/renameio/v2
```

---

## Architecture Patterns

### Recommended Project Structure

```
internal/
├── sshconfig/
│   ├── parser.go         # Keep existing read logic (or migrate to sshconf)
│   ├── writer.go         # NEW: Write operations with sshconf
│   └── backup.go         # NEW: Backup/undo buffer management
├── tui/
│   ├── model.go          # Existing: root model
│   ├── form_add.go       # NEW: Add server form
│   ├── form_edit.go      # NEW: Edit server form
│   ├── confirm_delete.go # NEW: Delete confirmation screen
│   └── keys.go           # UPDATE: Add 'a', 'e', 'd', 'u' keybindings
└── domain/
    └── validation.go     # UPDATE: Add SSH-specific validators
```

### Pattern 1: Two-Library Strategy (Read vs Write)

**What:** Keep `kevinburke/ssh_config` for read operations (already integrated), add `patrikkj/sshconf` for write operations.

**When to use:** When migration risk is too high; allows incremental adoption.

**Example:**
```go
// Read: existing parser (kevinburke/ssh_config)
hosts, err := sshconfig.ParseSSHConfig("~/.ssh/config")

// Write: new writer (patrikkj/sshconf)
func UpdateHost(configPath string, hostName string, updates map[string]string) error {
    cfg, err := sshconf.ParseConfigFile(configPath)
    if err != nil {
        return err
    }

    // Build Host block with directives
    hostBlock := fmt.Sprintf("Host %s\n", hostName)
    for key, val := range updates {
        hostBlock += fmt.Sprintf("    %s %s\n", key, val)
    }

    // Patch preserves surrounding content
    err = cfg.Patch(fmt.Sprintf("Host %s", hostName), hostBlock)
    if err != nil {
        return err
    }

    return cfg.WriteFile(configPath)
}
```

**Alternative:** Migrate to `sshconf` for both read and write (cleaner, but requires updating parser integration).

### Pattern 2: Form with Blur Validation (Huh)

**What:** Use `huh.Form` with field-level validators that trigger on field exit (blur event).

**When to use:** Multi-field forms requiring inline error display.

**Example:**
```go
// Source: https://github.com/charmbracelet/huh (form validation examples)
import "github.com/charmbracelet/huh"

type ServerForm struct {
    Alias        string
    Hostname     string
    User         string
    Port         string
    IdentityFile string
    ExtraConfig  string
}

func NewAddServerForm() *huh.Form {
    form := &ServerForm{}

    return huh.NewForm(
        huh.NewGroup(
            huh.NewInput().
                Title("Alias").
                Description("Short name for this connection").
                Value(&form.Alias).
                Validate(func(s string) error {
                    if s == "" {
                        return errors.New("alias is required")
                    }
                    // Check for uniqueness against existing config
                    return nil
                }),

            huh.NewInput().
                Title("Hostname").
                Description("Server address or IP").
                Value(&form.Hostname).
                Validate(func(s string) error {
                    if s == "" {
                        return errors.New("hostname is required")
                    }
                    return nil
                }),

            huh.NewInput().
                Title("User").
                Description("SSH username").
                Value(&form.User).
                Validate(func(s string) error {
                    if s == "" {
                        return errors.New("user is required")
                    }
                    return nil
                }),

            huh.NewInput().
                Title("Port").
                Description("SSH port (default: 22)").
                Value(&form.Port).
                Placeholder("22").
                Validate(func(s string) error {
                    if s == "" {
                        return nil // Optional field
                    }
                    port, err := strconv.Atoi(s)
                    if err != nil || port < 1 || port > 65535 {
                        return errors.New("port must be 1-65535")
                    }
                    return nil
                }),

            huh.NewInput().
                Title("IdentityFile").
                Description("Path to SSH key (optional)").
                Value(&form.IdentityFile),

            huh.NewText().
                Title("Additional Config").
                Description("Extra SSH directives (ProxyJump, ForwardAgent, etc.)").
                CharLimit(1000).
                Value(&form.ExtraConfig),
        ),
    )
}
```

**Key insight:** Huh's `Validate()` runs on field blur (Tab/Shift+Tab navigation), matching the user requirement exactly. Errors render inline automatically.

### Pattern 3: Atomic Config Write with Backup

**What:** Before any write, create `.bak` backup, then use atomic rename to replace config.

**When to use:** Every config modification (add, edit, delete).

**Example:**
```go
// Source: https://pkg.go.dev/github.com/google/renameio/v2
import "github.com/google/renameio/v2"

func SaveConfigSafely(configPath string, content []byte) error {
    // 1. Create backup (overwrite previous .bak)
    backupPath := configPath + ".bak"
    original, err := os.ReadFile(configPath)
    if err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("read original: %w", err)
    }

    if len(original) > 0 {
        if err := os.WriteFile(backupPath, original, 0600); err != nil {
            return fmt.Errorf("create backup: %w", err)
        }
    }

    // 2. Atomic write to config
    if err := renameio.WriteFile(configPath, content, 0600); err != nil {
        return fmt.Errorf("atomic write: %w", err)
    }

    return nil
}
```

**Why atomic:** If write fails mid-operation (crash, disk full, SIGKILL), config file is never left in corrupt/partial state. Old version stays intact until rename succeeds.

### Pattern 4: Session Undo Buffer

**What:** Store deleted Host entries in memory (slice of deleted configs) until app exits.

**When to use:** User presses 'u' to undo last delete.

**Example:**
```go
type UndoBuffer struct {
    deletedHosts []DeletedHost
    maxSize      int // Claude's discretion: suggest 10
}

type DeletedHost struct {
    Name      string
    HostBlock string // Full Host block text from config
    Position  int    // Line number where it was deleted
}

func (u *UndoBuffer) RecordDelete(name, block string, pos int) {
    u.deletedHosts = append(u.deletedHosts, DeletedHost{
        Name:      name,
        HostBlock: block,
        Position:  pos,
    })

    // Trim to max size (FIFO)
    if len(u.deletedHosts) > u.maxSize {
        u.deletedHosts = u.deletedHosts[1:]
    }
}

func (u *UndoBuffer) UndoLast() (DeletedHost, bool) {
    if len(u.deletedHosts) == 0 {
        return DeletedHost{}, false
    }

    last := u.deletedHosts[len(u.deletedHosts)-1]
    u.deletedHosts = u.deletedHosts[:len(u.deletedHosts)-1]
    return last, true
}
```

**Restoration logic:** Re-parse config, insert deleted block at original position, write atomically.

### Anti-Patterns to Avoid

- **String manipulation for Host blocks:** Don't build config blocks with string concatenation; use library APIs or structured builders to avoid syntax errors and escaping bugs.
- **Forgetting fsync:** Manual atomic writes without `fsync()` can result in 0-byte files after crashes (renameio handles this).
- **Synchronous DNS lookups blocking UI:** Hostname validation on every keystroke = 100-500ms freeze per character; validate only on blur or submit.
- **Editing included files:** User requirement says Include directives are read-only; never write to files outside main `~/.ssh/config`.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Config parsing with comment preservation | Custom SSH config lexer/parser | `patrikkj/sshconf` or `kevinburke/ssh_config` | SSH config has 100+ directives, Match blocks, Include recursion, pattern wildcards, and multi-value keys (IdentityFile); regex-based parsers break on edge cases |
| Atomic file writes | `os.WriteFile()` + manual temp file cleanup | `google/renameio` | Must handle fsync, temp dir on same filesystem, cleanup on error, and Windows incompatibility; renameio solves all of this |
| Form validation and field focus | Custom Bubbletea form state machine | `charmbracelet/huh` | Blur validation, focus ring, keyboard navigation (Tab/Shift+Tab/j/k), and error styling require ~500 LOC; Huh does it correctly with accessibility support |
| DNS resolution with timeout | Goroutine + channel + time.After | `net.Resolver` with `context.WithTimeout` | Stdlib resolver supports context cancellation, respects DNS search paths, and handles IPv4/IPv6 correctly; manual timeout logic is race-prone |

**Key insight:** Config file manipulation is deceptively complex. A "simple" parser that works for 90% of configs will silently corrupt the other 10% (comments inside Host blocks, trailing whitespace, CRLF vs LF line endings). Libraries have battle-tested these edge cases.

---

## Common Pitfalls

### Pitfall 1: Overwriting Comments During Edit

**What goes wrong:** User edits a Host block that contains inline comments (e.g., `Port 2222 # corporate firewall`). After save, comments vanish.

**Why it happens:** Library writes a new Host block without preserving comment nodes from original parse tree.

**How to avoid:** Use `patrikkj/sshconf`'s `Patch()` method instead of full config replacement. Patch only modifies the targeted Host block and leaves surrounding content untouched.

**Warning signs:** Test with a config file containing comments between directives (not just above Host lines). If comments disappear after edit, patch logic is wrong.

---

### Pitfall 2: DNS Lookup Blocking UI Thread

**What goes wrong:** Hostname validation calls `net.LookupHost()` on every keystroke. UI freezes for 100-500ms per character during typing.

**Why it happens:** DNS resolution is synchronous and slow (network round-trip, timeouts for non-existent domains).

**How to avoid:**
1. **Only validate on field blur** (user tabs away from Hostname field).
2. Wrap validation in `context.WithTimeout(ctx, 2*time.Second)` to fail fast.
3. Show spinner during check: "Verifying hostname..."

**Warning signs:** Typing in Hostname field feels laggy, especially on slow networks or when entering invalid hostnames.

---

### Pitfall 3: Undo After Config Reload

**What goes wrong:** User deletes Host `prod-server`, then exits and reopens TUI. Presses 'u' expecting undo, but undo buffer is empty (session-scoped, not persisted).

**Why it happens:** Requirement says "session undo buffer" — deletions aren't stored on disk for recovery after app restart.

**How to avoid:**
1. Show clear messaging: "Undo available until you close ssherpa" or undo buffer indicator in status bar.
2. Consider persisting to temp file (`~/.cache/ssherpa/undo.json`) for cross-session recovery (Claude's discretion).

**Warning signs:** User bug reports like "undo doesn't work after I restart."

---

### Pitfall 4: Alias Conflicts on Add

**What goes wrong:** User adds `Host myserver`, but `~/.ssh/config` already has a `Host myserver` entry (possibly from an included file). SSH uses first match, so new entry may be ignored.

**Why it happens:** Validation doesn't check for duplicate Host patterns across main config + included files.

**How to avoid:**
1. Parse config before showing add form.
2. In alias field validation, check if `Host <alias>` already exists.
3. Show error: "Alias 'myserver' already defined in ~/.ssh/config:42" (include file path and line number if possible).

**Warning signs:** User adds a server, but can't connect to it (SSH picks the old/wrong entry).

---

### Pitfall 5: Port Field Accepts Non-Numeric Input

**What goes wrong:** User types "two thousand" in Port field. Validation doesn't catch it until form submit, then entire form fails.

**Why it happens:** Field validation only checks "is it a number 1-65535?" without filtering input characters.

**How to avoid:**
1. Use `huh.NewInput().CharLimit()` combined with numeric-only filter (reject non-digits during typing).
2. Blur validation shows error if value isn't a valid port.
3. Alternative: Accept empty string (defaults to 22), numeric string, or show dropdown with common ports (22, 2222, 8022).

**Warning signs:** Users can type arbitrary text into Port field.

---

### Pitfall 6: IdentityFile Path Expansion

**What goes wrong:** User enters `~/keys/prod.pem` in IdentityFile field. Config saves as-is, but SSH fails with "key not found: ~/keys/prod.pem" (tilde not expanded).

**Why it happens:** `~` expansion is a shell feature, not SSH config spec. SSH config supports `~` in values, but some SSH implementations are inconsistent.

**How to avoid:**
1. **Best:** Expand `~` to actual home directory path on save: `filepath.Join(os.Getenv("HOME"), "keys/prod.pem")`.
2. **Alternative:** Leave `~` as-is (SSH spec supports it), but validate file exists at expanded path during form validation.

**Warning signs:** User reports key file not found errors after adding server with `~` in IdentityFile.

---

### Pitfall 7: Delete Confirmation Typo Tolerance

**What goes wrong:** User wants to delete `prod-web-01` but confirmation prompt requires exact match. User types `prod web 01` (spaces instead of hyphens) and deletion fails silently or shows cryptic error.

**Why it happens:** String comparison is case-sensitive and whitespace-sensitive.

**How to avoid:**
1. **Show alias prominently:** "Type **prod-web-01** to confirm deletion:"
2. **Real-time feedback:** As user types, show red/green indicator of whether input matches alias.
3. **Case-insensitive match:** `strings.EqualFold(input, alias)` or normalize both to lowercase.

**Warning signs:** Users complain about needing to "type the name perfectly" for delete.

---

## Code Examples

Verified patterns from official sources:

### DNS Hostname Validation with Timeout

```go
// Source: https://pkg.go.dev/net (LookupHost with context)
import (
    "context"
    "net"
    "time"
)

func ValidateHostname(hostname string) error {
    if hostname == "" {
        return errors.New("hostname is required")
    }

    // Create context with 2-second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    // Use custom resolver with context
    resolver := &net.Resolver{}
    addrs, err := resolver.LookupHost(ctx, hostname)

    if err != nil {
        // Check if timeout
        if ctx.Err() == context.DeadlineExceeded {
            return errors.New("hostname verification timed out (check network)")
        }
        return fmt.Errorf("hostname not found: %w", err)
    }

    if len(addrs) == 0 {
        return errors.New("hostname resolves to no addresses")
    }

    return nil
}
```

### Huh Form Integration in Bubbletea

```go
// Source: https://github.com/charmbracelet/huh (Bubbletea integration)
import (
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/huh"
)

type FormModel struct {
    form     *huh.Form
    viewport viewport.Model
}

func (m FormModel) Init() tea.Cmd {
    return m.form.Init()
}

func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Check if form is complete
        if m.form.State == huh.StateCompleted {
            // User submitted form successfully
            return m, m.handleFormSubmit()
        }
    }

    // Pass message to form
    form, cmd := m.form.Update(msg)
    if f, ok := form.(*huh.Form); ok {
        m.form = f
    }

    return m, cmd
}

func (m FormModel) View() string {
    if m.form.State == huh.StateCompleted {
        return "Saving..."
    }

    return m.form.View()
}
```

### Delete Confirmation with Typed Input

```go
// Source: Inspired by GitHub's repository deletion pattern
import "github.com/charmbracelet/bubbles/textinput"

type ConfirmDeleteModel struct {
    serverAlias string
    input       textinput.Model
    confirmed   bool
    cancelled   bool
}

func NewConfirmDelete(alias string) ConfirmDeleteModel {
    ti := textinput.New()
    ti.Placeholder = alias
    ti.Focus()
    ti.CharLimit = 100

    return ConfirmDeleteModel{
        serverAlias: alias,
        input:       ti,
    }
}

func (m ConfirmDeleteModel) Update(msg tea.Msg) (ConfirmDeleteModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            // Check if input matches alias
            if m.input.Value() == m.serverAlias {
                m.confirmed = true
                return m, nil
            }
            // Mismatch — could show error or clear input
            m.input.SetValue("")
            return m, nil

        case "esc":
            m.cancelled = true
            return m, nil
        }
    }

    var cmd tea.Cmd
    m.input, cmd = m.input.Update(msg)
    return m, cmd
}

func (m ConfirmDeleteModel) View() string {
    // Real-time match indicator
    matchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Red
    if m.input.Value() == m.serverAlias {
        matchStyle = matchStyle.Foreground(lipgloss.Color("10")) // Green
    }

    return fmt.Sprintf(
        "Delete server '%s'?\n\n"+
        "Type the server alias to confirm:\n%s\n\n"+
        "%s\n\n"+
        "Esc: cancel",
        m.serverAlias,
        m.input.View(),
        matchStyle.Render(fmt.Sprintf("Match: %t", m.input.Value() == m.serverAlias)),
    )
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual SSH config editing | TUI-based config management | 2020+ | Tools like `assh`, `storm` popularized; ssherpa fits this trend |
| String-based config writes | AST-preserving parsers | 2018+ (kevinburke/ssh_config) | Comments no longer lost during programmatic edits |
| Sync validation on every keystroke | Blur validation (validate on field exit) | 2022+ (UX research) | Reduced interruption, better form completion rates |
| `ioutil.WriteFile` for configs | Atomic writes with `renameio` | 2019+ (renameio release) | Prevents corruption from crashes/kills during write |
| Custom TUI forms | Framework forms (Huh, tview) | 2023+ (Huh release) | Accessibility, consistency, and less boilerplate |

**Deprecated/outdated:**
- **`kevinburke/ssh_config` for writes:** No write support; community has moved to `sshconf` or custom serializers.
- **Blocking DNS on UI thread:** Modern TUIs use async validation with spinners; sync validation feels broken on slow networks.
- **`os.Rename()` without fsync:** Pre-renameio era; missing fsync caused silent corruption bugs after crashes.

---

## Open Questions

### 1. Should we migrate parser from kevinburke to patrikkj?

**What we know:**
- Current parser (`kevinburke/ssh_config`) is read-only and works well.
- `patrikkj/sshconf` does both read and write with formatting preservation.

**What's unclear:**
- Migration effort vs. maintaining two libraries.
- Performance differences between parsers.

**Recommendation:**
- **Low-risk path:** Keep `kevinburke` for reads, use `patrikkj` only for writes. Add integration tests to verify both parse the same files identically.
- **Clean path:** Migrate to `patrikkj` for all operations if parser tests pass (less surface area for bugs).

---

### 2. How to handle Include directives during write?

**What we know:**
- User requirement: "Include directives: read-only (parse and display servers from included files, but only write to the main config file)."
- `patrikkj/sshconf` supports Include parsing.

**What's unclear:**
- If user edits a Host from an included file, should we:
  1. **Block edit:** "This host is defined in included file X, cannot edit."
  2. **Copy to main config:** Duplicate Host to main `~/.ssh/config`, warn user about override precedence.
  3. **Edit in place:** Write to the included file (violates requirement).

**Recommendation:**
- **Option 1 (safest):** Block edits/deletes for hosts from included files. Show warning: "This host is defined in `~/.ssh/config.d/work` (read-only). To edit, modify that file manually."
- **Option 2 (power user):** Allow edit, but create override in main config with comment: `# Overrides Host from ~/.ssh/config.d/work`. User must understand SSH's first-match-wins precedence.

---

### 3. Undo buffer persistence across sessions?

**What we know:**
- User requirement: "Session undo buffer: deleted entries stay in memory until session ends."
- No mention of persistence after exit.

**What's unclear:**
- Would users benefit from undo after app restart? (e.g., accidental delete, close app, reopen and undo)

**Recommendation:**
- **Start with session-only undo** (simpler, matches requirement).
- **Future enhancement:** Persist to `~/.cache/ssherpa/undo.json` (max 10 deletes, 24-hour TTL). Requires additional UX for "recovering old deletes."

---

### 4. Extra config field: validate syntax or accept blindly?

**What we know:**
- Free-text area for directives like `ProxyJump`, `ForwardAgent`, etc.
- Users can enter any valid SSH config directive.

**What's unclear:**
- Should we validate syntax (parse as valid SSH directives) or accept any text?
- Invalid syntax won't break the app, but will break SSH connections.

**Recommendation:**
- **Basic validation:** Check that each line matches pattern `Directive Value` or `Directive=Value` (catches typos like `ProxyJum` or `=`).
- **Defer deep validation:** Don't validate directive names against full SSH spec (100+ options, version-dependent). SSH itself will error on invalid directives at connection time.
- **Show example:** Placeholder text in field: `ProxyJump bastion.example.com\nForwardAgent yes`

---

## Sources

### Primary (HIGH confidence)

- [kevinburke/ssh_config GitHub](https://github.com/kevinburke/ssh_config) - Current parser used in codebase
- [patrikkj/sshconf Go Packages](https://pkg.go.dev/github.com/patrikkj/sshconf) - Read-write SSH config library
- [charmbracelet/huh GitHub](https://github.com/charmbracelet/huh) - Terminal forms library
- [google/renameio Go Packages](https://pkg.go.dev/github.com/google/renameio/v2) - Atomic file writes
- [net package Go Packages](https://pkg.go.dev/net) - DNS resolution (LookupHost)
- [ssh_config man page](https://man7.org/linux/man-pages/man5/ssh_config.5.html) - SSH config format specification

### Secondary (MEDIUM confidence)

- [SSH Config Complete Guide 2026](https://devtoolbox.dedyn.io/blog/ssh-config-complete-guide) - Current best practices
- [Inline Validation UX - Smart Interface Design Patterns](https://smart-interface-design-patterns.com/articles/inline-validation-ux/) - Field-level validation patterns
- [Atomically writing files in Go - Michael Stapelberg](https://michael.stapelberg.ch/posts/2017-01-28-golang_atomically_writing/) - Atomic write patterns

### Tertiary (LOW confidence)

- [Bubbletea textarea discussions](https://github.com/charmbracelet/bubbletea/issues/800) - Multi-line input handling
- [Go validator libraries](https://github.com/go-playground/validator) - General validation patterns (not TUI-specific)

---

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** - Libraries are mature, widely used, and officially recommended
- Architecture: **HIGH** - Patterns verified from official docs and existing codebase
- Pitfalls: **MEDIUM** - Based on common SSH config bugs and TUI UX research; some are hypothetical
- DNS validation: **HIGH** - stdlib `net` package is well-documented and stable

**Research date:** 2026-02-14
**Valid until:** 30 days (stable domain; libraries unlikely to change significantly)
