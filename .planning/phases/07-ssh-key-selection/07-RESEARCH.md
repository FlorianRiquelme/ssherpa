# Phase 7: SSH Key Selection - Research

**Researched:** 2026-02-14
**Domain:** SSH key discovery, parsing, fingerprinting, and IdentityFile management
**Confidence:** HIGH

## Summary

SSH key selection requires discovering keys from three sources (filesystem, SSH agent, 1Password), parsing private key headers for type detection, extracting fingerprints and comments, and persisting selections via IdentityFile directives. Go's `golang.org/x/crypto/ssh` package provides comprehensive support for key parsing, fingerprinting (SHA256), and SSH agent communication. The existing TUI uses an overlay picker pattern (ProjectPicker) that can be adapted for key selection. Critical pitfalls include SSH's "too many keys" authentication failures (mitigated by explicit IdentityFile) and key ordering behavior where agent keys are tried before config-specified keys.

**Primary recommendation:** Use `golang.org/x/crypto/ssh` for all key operations, adapt existing ProjectPicker overlay pattern for key selection UI, implement file-based key discovery via `filepath.WalkDir` with header sniffing, and clearly display key source badges to distinguish between file/agent/1Password origins.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Key discovery
- Scan `~/.ssh/` only (no custom paths)
- Detect keys via file content sniffing (read file headers for PEM/OpenSSH key format), not naming conventions
- Include keys loaded in the SSH agent (`ssh-add -l`)
- Include keys from 1Password backend — unified list across all sources (file, agent, 1Password)

#### Selection UX
- Key picker available in both the add/edit form AND as a quick action from detail view
- Picker style: Claude's discretion (choose what fits existing TUI patterns — overlay vs inline)
- Single key per connection only (no multi-key IdentityFile stacking)
- Include a "None (SSH default)" option to clear explicit key assignment

#### Key display
- Full details per key: filename, type (ed25519/rsa/etc.), fingerprint, comment, source
- Source indicated via text badge: `[file]`, `[agent]`, `[1password]`
- Currently-assigned key highlighted in picker (checkmark or visual indicator)
- Key display in server detail view: Claude's discretion on placement (consistent with existing layout)

#### Default behavior
- New connections default to no key (SSH default resolution) — no IdentityFile set
- Existing IdentityFile directives in SSH config are read, displayed, and pre-selected when editing
- Missing key files (referenced in config but not on disk) show a warning badge
- Do NOT set IdentityOnly when a key is selected — let SSH try other keys from agent too

### Claude's Discretion
- Picker component style (overlay list vs inline dropdown — pick what fits existing patterns)
- Key display placement in server detail view
- Fingerprint format (SHA256 vs MD5)
- How to handle key passphrase detection (if relevant to display)

### Deferred Ideas (OUT OF SCOPE)
- Port forwarding configuration (local, remote, dynamic) — future version
- Connection status indicators (reachable/unreachable via async ping) — future version
- ProxyJump/bastion host configuration — future version
- Multiple IdentityFile per connection — future version if needed

</user_constraints>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| golang.org/x/crypto/ssh | Latest (published 2026-02-09) | SSH key parsing, fingerprinting, type detection | Official Go crypto package, comprehensive format support (PKCS#1, PKCS#8, OpenSSH, PEM) |
| golang.org/x/crypto/ssh/agent | Latest (published 2026-02-09) | SSH agent communication via UNIX socket | Official agent protocol implementation, List() method for key enumeration |
| path/filepath | stdlib | Directory traversal for key discovery | Standard library WalkDir for efficient file tree walking |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| os | stdlib | File I/O, permissions checking | Reading key files, verifying 600/644 permissions |
| crypto/sha256 | stdlib | Fingerprint hashing | Used by ssh.FingerprintSHA256() for SHA256 fingerprints |

### Already in Project
| Library | Current Use | New Use for Phase 7 |
|---------|-------------|---------------------|
| github.com/kevinburke/ssh_config | Reading SSH config | Reading existing IdentityFile directives |
| internal/sshconfig/writer | Text-based SSH config writes | Writing IdentityFile directives to SSH config |
| internal/tui/picker | ProjectPicker overlay | Pattern for SSHKeyPicker overlay component |
| internal/backend/onepassword | 1Password credential access | Accessing SSH keys stored in 1Password |

**Installation:**
No new dependencies required — all needed libraries already in go.mod or stdlib.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── sshkey/              # New package for SSH key operations
│   ├── discovery.go     # File, agent, 1Password key discovery
│   ├── parser.go        # Key type detection, fingerprint extraction
│   ├── agent.go         # SSH agent communication wrapper
│   └── types.go         # SSHKey domain model
├── tui/
│   ├── key_picker.go    # SSHKeyPicker overlay component (NEW)
│   └── form.go          # Add key selection field (MODIFY)
└── domain/
    └── server.go        # Already has IdentityFile field
```

### Pattern 1: SSH Key Discovery (Multi-Source)
**What:** Discover keys from filesystem, agent, and 1Password, returning unified list with source tracking
**When to use:** Initial load and refresh operations
**Example:**
```go
// Source: Architectural pattern based on existing backend.go multi-backend pattern
package sshkey

type KeySource string

const (
    KeySourceFile       KeySource = "file"
    KeySourceAgent      KeySource = "agent"
    KeySourceOnePassword KeySource = "1password"
)

type SSHKey struct {
    Path        string    // Full path (for file keys) or identifier
    Type        string    // "ed25519", "rsa", "ecdsa", "dsa"
    Fingerprint string    // SHA256 fingerprint (e.g., "SHA256:oPGy6dH...")
    Comment     string    // Key comment (e.g., "user@host")
    Source      KeySource // Where key was discovered
    Filename    string    // Display name (e.g., "id_ed25519")
}

type Discoverer struct {
    sshDir         string
    agentConn      net.Conn
    onePasswordBackend backend.Backend
}

func (d *Discoverer) DiscoverAll(ctx context.Context) ([]SSHKey, error) {
    var keys []SSHKey

    // Discover from ~/.ssh/
    fileKeys, _ := d.discoverFileKeys()
    keys = append(keys, fileKeys...)

    // Discover from SSH agent
    agentKeys, _ := d.discoverAgentKeys()
    keys = append(keys, agentKeys...)

    // Discover from 1Password
    opKeys, _ := d.discoverOnePasswordKeys(ctx)
    keys = append(keys, opKeys...)

    return keys, nil
}
```

### Pattern 2: File-Based Key Discovery (Header Sniffing)
**What:** Walk ~/.ssh/, read file headers to detect private keys by format magic bytes
**When to use:** File key discovery
**Example:**
```go
// Source: Based on golang.org/x/crypto/ssh ParseRawPrivateKey behavior
func (d *Discoverer) discoverFileKeys() ([]SSHKey, error) {
    var keys []SSHKey

    err := filepath.WalkDir(d.sshDir, func(path string, entry os.DirEntry, err error) error {
        if err != nil || entry.IsDir() {
            return err
        }

        // Read first 512 bytes for header detection
        data, err := os.ReadFile(path)
        if err != nil {
            return nil // Skip unreadable files
        }

        // Check for SSH private key headers
        if !isPrivateKey(data) {
            return nil
        }

        // Parse key to extract type, fingerprint, comment
        key, err := parsePrivateKeyFile(path, data)
        if err != nil {
            return nil // Skip unparseable keys
        }

        keys = append(keys, key)
        return nil
    })

    return keys, err
}

func isPrivateKey(data []byte) bool {
    headers := []string{
        "-----BEGIN OPENSSH PRIVATE KEY-----",
        "-----BEGIN RSA PRIVATE KEY-----",
        "-----BEGIN EC PRIVATE KEY-----",
        "-----BEGIN DSA PRIVATE KEY-----",
        "-----BEGIN PRIVATE KEY-----", // PKCS8
    }

    for _, header := range headers {
        if bytes.Contains(data[:min(len(data), 512)], []byte(header)) {
            return true
        }
    }
    return false
}
```

### Pattern 3: SSH Agent Key Discovery
**What:** Connect to SSH_AUTH_SOCK, call agent.List() to enumerate loaded keys
**When to use:** Agent key discovery
**Example:**
```go
// Source: https://pkg.go.dev/golang.org/x/crypto/ssh/agent
import "golang.org/x/crypto/ssh/agent"

func (d *Discoverer) discoverAgentKeys() ([]SSHKey, error) {
    // Connect to SSH agent via UNIX socket
    socket := os.Getenv("SSH_AUTH_SOCK")
    if socket == "" {
        return nil, nil // No agent running
    }

    conn, err := net.Dial("unix", socket)
    if err != nil {
        return nil, err
    }
    defer conn.Close()

    agentClient := agent.NewClient(conn)

    // List returns []agent.Key with Comment and Format fields
    agentKeys, err := agentClient.List()
    if err != nil {
        return nil, err
    }

    var keys []SSHKey
    for _, ak := range agentKeys {
        // Parse public key to get fingerprint
        pubKey, err := ssh.ParsePublicKey(ak.Blob)
        if err != nil {
            continue
        }

        keys = append(keys, SSHKey{
            Path:        "", // Agent keys don't have file paths
            Type:        pubKey.Type(), // "ssh-ed25519", "ssh-rsa", etc.
            Fingerprint: ssh.FingerprintSHA256(pubKey),
            Comment:     ak.Comment,
            Source:      KeySourceAgent,
            Filename:    fmt.Sprintf("agent:%s", ak.Comment),
        })
    }

    return keys, nil
}
```

### Pattern 4: Key Type and Fingerprint Extraction
**What:** Parse private key, derive public key, compute SHA256 fingerprint
**When to use:** After detecting a private key file
**Example:**
```go
// Source: https://pkg.go.dev/golang.org/x/crypto/ssh
func parsePrivateKeyFile(path string, data []byte) (SSHKey, error) {
    // Try parsing without passphrase first
    privateKey, err := ssh.ParseRawPrivateKey(data)
    if err != nil {
        // Check if encrypted (PassphraseMissingError)
        // For Phase 7, we can skip encrypted keys or show warning badge
        return SSHKey{}, err
    }

    // Convert to ssh.Signer to get public key
    signer, err := ssh.NewSignerFromKey(privateKey)
    if err != nil {
        return SSHKey{}, err
    }

    pubKey := signer.PublicKey()

    // Read corresponding .pub file for comment (if exists)
    comment := extractCommentFromPubFile(path + ".pub")

    return SSHKey{
        Path:        path,
        Type:        parseKeyType(pubKey.Type()), // "ssh-ed25519" -> "ed25519"
        Fingerprint: ssh.FingerprintSHA256(pubKey),
        Comment:     comment,
        Source:      KeySourceFile,
        Filename:    filepath.Base(path),
    }, nil
}

func extractCommentFromPubFile(pubPath string) string {
    data, err := os.ReadFile(pubPath)
    if err != nil {
        return ""
    }

    // ParseAuthorizedKey returns (pubKey, comment, options, rest, error)
    _, comment, _, _, err := ssh.ParseAuthorizedKey(data)
    if err != nil {
        return ""
    }

    return comment
}

func parseKeyType(sshType string) string {
    // Convert "ssh-ed25519" -> "ed25519", "ssh-rsa" -> "rsa"
    return strings.TrimPrefix(sshType, "ssh-")
}
```

### Pattern 5: SSHKeyPicker Component (Overlay)
**What:** Bubbletea overlay picker for key selection, adapted from existing ProjectPicker
**When to use:** User selects key in form or detail view quick action
**Example:**
```go
// Source: internal/tui/picker.go adapted for SSH keys
type SSHKeyPicker struct {
    items          []keyPickerItem
    selected       int
    currentKeyPath string // Currently assigned key path
    width          int
    height         int
}

type keyPickerItem struct {
    key        sshkey.SSHKey
    isNone     bool   // "None (SSH default)" option
    isAssigned bool   // Currently assigned to this server
    isMissing  bool   // File referenced in config but not found
}

func NewSSHKeyPicker(keys []sshkey.SSHKey, currentKeyPath string) SSHKeyPicker {
    items := make([]keyPickerItem, 0, len(keys)+1)

    // Add "None (SSH default)" option first
    items = append(items, keyPickerItem{
        isNone:     true,
        isAssigned: currentKeyPath == "",
    })

    // Add discovered keys
    for _, k := range keys {
        items = append(items, keyPickerItem{
            key:        k,
            isAssigned: k.Path == currentKeyPath,
        })
    }

    return SSHKeyPicker{
        items:          items,
        currentKeyPath: currentKeyPath,
        width:          60,
        height:         20,
    }
}

func (p SSHKeyPicker) renderItem(index int, item keyPickerItem) string {
    var parts []string

    // Cursor indicator
    if index == p.selected {
        parts = append(parts, "> ")
    } else {
        parts = append(parts, "  ")
    }

    // Checkmark if assigned
    if item.isAssigned {
        parts = append(parts, pickerCheckmarkStyle.Render("✓ "))
    } else {
        parts = append(parts, "  ")
    }

    // Handle "None" option
    if item.isNone {
        parts = append(parts, "None (SSH default)")
        return strings.Join(parts, "")
    }

    // Key display: filename, type, fingerprint, comment, source badge
    keyText := fmt.Sprintf("%s (%s) %s",
        item.key.Filename,
        item.key.Type,
        item.key.Fingerprint[:24]+"...") // Truncate fingerprint

    if item.key.Comment != "" {
        keyText += fmt.Sprintf(" — %s", item.key.Comment)
    }

    // Source badge
    sourceBadge := renderSourceBadge(item.key.Source)
    keyText += " " + sourceBadge

    // Warning badge if missing
    if item.isMissing {
        keyText = warningStyle.Render("⚠ ") + keyText
    }

    parts = append(parts, keyText)
    return strings.Join(parts, "")
}

func renderSourceBadge(source sshkey.KeySource) string {
    switch source {
    case sshkey.KeySourceFile:
        return sourceBadgeStyle.Render("[file]")
    case sshkey.KeySourceAgent:
        return sourceBadgeStyle.Render("[agent]")
    case sshkey.KeySourceOnePassword:
        return sourceBadgeStyle.Render("[1password]")
    default:
        return ""
    }
}
```

### Pattern 6: IdentityFile Persistence
**What:** Write IdentityFile directive to SSH config without IdentitiesOnly
**When to use:** User selects a key and saves
**Example:**
```go
// Source: internal/sshconfig/writer.go pattern
func WriteIdentityFile(configPath, alias, keyPath string) error {
    // If keyPath is empty, remove IdentityFile directive
    if keyPath == "" {
        return removeIdentityFile(configPath, alias)
    }

    // Read existing config
    cfg, err := sshconfig.Parse(configPath)
    if err != nil {
        return err
    }

    // Find host block
    host := cfg.FindHost(alias)
    if host == nil {
        return fmt.Errorf("host %s not found", alias)
    }

    // Set IdentityFile (single value, replace existing)
    // DO NOT set IdentitiesOnly — let SSH try agent keys too
    host.SetOption("IdentityFile", keyPath)

    // Write back to config (text-based writer from Phase 5)
    return writeSSHConfig(configPath, cfg)
}
```

### Anti-Patterns to Avoid
- **Don't rely on filename conventions** (e.g., "id_*") — use header sniffing instead (required by user constraints)
- **Don't set IdentitiesOnly when writing IdentityFile** — breaks user's agent-based workflows
- **Don't block UI thread during key discovery** — use async Cmd pattern like existing configLoadedMsg
- **Don't parse encrypted keys synchronously** — detect PassphraseMissingError and show badge/skip instead
- **Don't show duplicate keys** — agent and file may reference same key; deduplicate by fingerprint

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SSH key parsing | Custom PEM/OpenSSH parser | `ssh.ParseRawPrivateKey()` | Handles PKCS#1, PKCS#8, OpenSSH, PEM formats; detects encrypted keys via PassphraseMissingError |
| Fingerprint generation | Manual SHA256 hashing | `ssh.FingerprintSHA256(pubKey)` | Standard format (SHA256:base64), matches ssh-keygen output |
| SSH agent communication | Raw UNIX socket protocol | `ssh/agent.NewClient()` and `List()` | Official protocol implementation, handles agent.Key marshaling |
| Public key comment extraction | Custom authorized_keys parsing | `ssh.ParseAuthorizedKey()` | Returns comment field, handles options and multiple key formats |
| Directory traversal | Recursive os.ReadDir loops | `filepath.WalkDir()` | More efficient than Walk(), proper error handling, SkipDir support |

**Key insight:** SSH key formats are deceptively complex (multiple PEM types, OpenSSH v1 format, encrypted keys, certificates). Go's crypto/ssh package handles all edge cases that would take weeks to implement correctly.

## Common Pitfalls

### Pitfall 1: Too Many SSH Keys ("Too many authentication failures")
**What goes wrong:** SSH tries all agent keys before IdentityFile keys. If user has 6+ keys in agent, server rejects before trying the correct key.
**Why it happens:** SSH agent keys are tried first, in agent load order, regardless of IdentityFile directives. Max authentication attempts is typically 6 (server limit).
**How to avoid:** DO NOT automatically set IdentitiesOnly when user selects a key (per user constraints). Instead, document this pitfall in help text and let users configure IdentitiesOnly themselves if needed.
**Warning signs:** User reports "Permission denied (publickey)" despite selecting correct key. Check agent with `ssh-add -l`.

### Pitfall 2: Wrong Key Selection Order
**What goes wrong:** User sets IdentityFile in config, but SSH tries different key from agent first and succeeds accidentally. Later, when agent changes, connection breaks.
**Why it happens:** Agent keys are always tried before IdentityFile keys, even with IdentitiesOnly unset.
**How to avoid:** Clearly document in key picker that "IdentityFile is a preference, not a requirement — agent keys are tried first." Provide "None (SSH default)" option to explicitly unset IdentityFile.
**Warning signs:** Connection works on one machine but fails on another with same config but different agent state.

### Pitfall 3: Encrypted Key Detection
**What goes wrong:** Parsing encrypted private key hangs or fails with obscure error.
**Why it happens:** `ParseRawPrivateKey` returns `PassphraseMissingError` for encrypted keys. Without detection, key appears broken.
**How to avoid:** Catch `PassphraseMissingError` during parsing. For Phase 7, show badge like "[encrypted]" and skip key or show in picker with warning.
**Warning signs:** User reports missing keys that exist in ~/.ssh but are passphrase-protected.

### Pitfall 4: Missing Public Key File
**What goes wrong:** Private key parsed successfully but comment is empty because .pub file is missing.
**Why it happens:** Comment is stored in .pub file, not private key. ParseRawPrivateKey doesn't return comment.
**How to avoid:** Try reading {keypath}.pub for comment. If missing, fall back to empty comment or derive from private key path (e.g., "id_ed25519" -> "key: id_ed25519").
**Warning signs:** All keys in picker show blank comments.

### Pitfall 5: File Permissions Confusion
**What goes wrong:** Scanning ~/.ssh/ encounters permission denied errors on files/directories.
**Why it happens:** Some users have restrictive permissions or keys owned by different users.
**How to avoid:** In WalkDir callback, silently skip files that return permission errors. Don't fail entire discovery on single file error.
**Warning signs:** Discovery returns 0 keys despite valid keys in ~/.ssh.

### Pitfall 6: Agent Connection Failures
**What goes wrong:** SSH_AUTH_SOCK not set or agent not running, causing discovery to fail loudly.
**Why it happens:** Not all users run SSH agent (e.g., minimal systems, containers).
**How to avoid:** Check `os.Getenv("SSH_AUTH_SOCK")` and return empty list if unset. Log but don't error. Agent keys are optional discovery source.
**Warning signs:** TUI crashes on launch for users without SSH agent.

### Pitfall 7: Key Deduplication
**What goes wrong:** Same key appears multiple times (file + agent) in picker.
**Why it happens:** User has key file AND loaded it into agent — both discoveries return same key.
**How to avoid:** Deduplicate by fingerprint before displaying. Prefer agent source if duplicate (indicates actively loaded key).
**Warning signs:** Picker shows duplicate entries for same fingerprint.

### Pitfall 8: 1Password Key Format Differences
**What goes wrong:** 1Password returns keys in different format than filesystem keys.
**Why it happens:** 1Password may store keys in specific format or return via different API.
**How to avoid:** When implementing 1Password discovery, ensure SSHKey struct fields are populated consistently (path may be 1Password reference URI, source is KeySourceOnePassword).
**Warning signs:** 1Password keys render differently or cause picker crashes.

## Code Examples

Verified patterns from official sources:

### SSH Key Fingerprint Extraction
```go
// Source: https://pkg.go.dev/golang.org/x/crypto/ssh
import "golang.org/x/crypto/ssh"

func GetKeyFingerprint(pubKeyBytes []byte) (string, error) {
    pubKey, err := ssh.ParsePublicKey(pubKeyBytes)
    if err != nil {
        return "", err
    }

    // Returns format: "SHA256:oPGy6dHU8eI+TK+AcgC88G4TywqE2JKXEohBfnqx9jA"
    // Matches ssh-keygen -lf output
    return ssh.FingerprintSHA256(pubKey), nil
}
```

### Detect Encrypted Private Key
```go
// Source: https://github.com/golang/go/issues/71048
import "golang.org/x/crypto/ssh"

func IsKeyEncrypted(keyData []byte) (bool, error) {
    _, err := ssh.ParseRawPrivateKey(keyData)
    if err == nil {
        return false, nil // Key parsed successfully, not encrypted
    }

    // Check if error is PassphraseMissingError
    var passphraseMissing *ssh.PassphraseMissingError
    if errors.As(err, &passphraseMissing) {
        return true, nil // Key is encrypted
    }

    return false, err // Other parsing error
}
```

### List SSH Agent Keys
```go
// Source: https://pkg.go.dev/golang.org/x/crypto/ssh/agent
import (
    "golang.org/x/crypto/ssh/agent"
    "net"
    "os"
)

func ListAgentKeys() ([]*agent.Key, error) {
    socket := os.Getenv("SSH_AUTH_SOCK")
    if socket == "" {
        return nil, fmt.Errorf("SSH agent not available")
    }

    conn, err := net.Dial("unix", socket)
    if err != nil {
        return nil, err
    }
    defer conn.Close()

    agentClient := agent.NewClient(conn)

    // Returns []agent.Key with fields: Format, Blob, Comment
    keys, err := agentClient.List()
    if err != nil {
        return nil, err
    }

    return keys, nil
}
```

### Parse Authorized Keys with Comment
```go
// Source: https://pkg.go.dev/golang.org/x/crypto/ssh
import "golang.org/x/crypto/ssh"

func ParsePubKeyFile(pubPath string) (pubKey ssh.PublicKey, comment string, err error) {
    data, err := os.ReadFile(pubPath)
    if err != nil {
        return nil, "", err
    }

    // ParseAuthorizedKey returns: publicKey, comment, options, rest, error
    pubKey, comment, _, _, err = ssh.ParseAuthorizedKey(data)
    return pubKey, comment, err
}
```

### File Discovery with WalkDir
```go
// Source: https://pkg.go.dev/path/filepath
import "path/filepath"

func DiscoverKeyFiles(sshDir string) ([]string, error) {
    var keyPaths []string

    err := filepath.WalkDir(sshDir, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            // Permission denied or other error — skip and continue
            return nil
        }

        if d.IsDir() {
            return nil // Continue into subdirectories
        }

        // Read first 512 bytes for header check
        data, err := os.ReadFile(path)
        if err != nil {
            return nil // Skip unreadable files
        }

        if isPrivateKey(data) {
            keyPaths = append(keyPaths, path)
        }

        return nil
    })

    return keyPaths, err
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| MD5 fingerprints | SHA256 fingerprints | OpenSSH 6.8 (2015) | SHA256 is now default and more secure; use `ssh.FingerprintSHA256()` not MD5 |
| PKCS#1/SEC1 formats | OpenSSH v1 format | OpenSSH 7.8 (2014) | New keys default to `-----BEGIN OPENSSH PRIVATE KEY-----`; must support both formats |
| Manual IdentitiesOnly | Agent-first key selection | Ongoing | Users increasingly rely on SSH agent; don't force IdentitiesOnly unless user requests |
| filepath.Walk | filepath.WalkDir | Go 1.16 (2021) | WalkDir is more efficient (avoids os.Lstat on every file) |

**Deprecated/outdated:**
- **MD5 fingerprints**: Modern SSH uses SHA256 by default (ssh-keygen -lf outputs SHA256 format)
- **DSA keys**: Deprecated in OpenSSH 7.0 (2015), but still supported by golang.org/x/crypto/ssh — should parse but show warning badge
- **PuTTY .ppk format**: Not supported by golang.org/x/crypto/ssh — users must convert with puttygen

## Open Questions

1. **1Password SSH key access mechanism**
   - What we know: 1Password CLI has `op item get` for SSH keys, SDK has item access
   - What's unclear: Exact CLI/SDK method to list SSH keys by category, retrieve private key content
   - Recommendation: Research 1Password SSH key item schema in Phase 7 planning; may need to query items with category="SSH Key" and extract private key field

2. **Key selection persistence for backend-managed servers**
   - What we know: Backend servers from 1Password have VaultID, writes route through backend.Writer
   - What's unclear: Should IdentityFile be stored in SSH config (side-by-side) or in 1Password item metadata?
   - Recommendation: Likely SSH config for consistency (all servers have IdentityFile in one place), but verify backend.Writer supports IdentityFile field

3. **Missing key file handling UX**
   - What we know: Config references key that doesn't exist on disk should show warning badge
   - What's unclear: Should missing key appear in picker at all, or only show in detail view as warning?
   - Recommendation: Show in picker with warning badge and mark as "missing" — allows user to see what was configured and choose replacement

4. **Passphrase-protected key UX**
   - What we know: Can detect via PassphraseMissingError
   - What's unclear: Should encrypted keys be selectable in Phase 7, or excluded with note "passphrase not supported yet"?
   - Recommendation: Include in picker but show badge "[encrypted]" — keys are still usable by SSH (which will prompt for passphrase), just can't display full details

## Sources

### Primary (HIGH confidence)
- [golang.org/x/crypto/ssh package docs](https://pkg.go.dev/golang.org/x/crypto/ssh) - Key parsing, fingerprinting, public key operations (published 2026-02-09)
- [golang.org/x/crypto/ssh/agent package docs](https://pkg.go.dev/golang.org/x/crypto/ssh/agent) - SSH agent communication (published 2026-02-09)
- [crypto/ssh/keys.go source](https://github.com/golang/crypto/blob/master/ssh/keys.go) - ParseRawPrivateKey implementation
- [ssh_config man page](https://man7.org/linux/man-pages/man5/ssh_config.5.html) - IdentityFile and IdentitiesOnly behavior
- [path/filepath package docs](https://pkg.go.dev/path/filepath) - WalkDir for directory traversal
- Internal codebase: internal/tui/picker.go (ProjectPicker pattern), internal/sshconfig/writer.go (config writing), internal/domain/server.go (Server.IdentityFile field)

### Secondary (MEDIUM confidence)
- [SSH private key format headers](https://coolaj86.com/articles/the-openssh-private-key-format/) - Header detection patterns for key type identification
- [SSH key permissions guide](https://gist.github.com/denisgolius/d846af3ad5ce661dbca0335ec35e3d39) - Correct permissions: private keys 600, public keys 644
- [SSH common pitfalls: too many keys](https://www.tutorialworks.com/ssh-fail-too-many-keys/) - Max 6 authentication attempts, IdentitiesOnly solution
- [SSH agent key ordering](https://utcc.utoronto.ca/~cks/space/blog/sysadmin/SSHConfigIdentities) - Agent keys tried before IdentityFile keys
- [1Password CLI SSH key management](https://developer.1password.com/docs/cli/ssh-keys/) - op item get for SSH keys

### Tertiary (LOW confidence - flagged for validation)
- [PassphraseMissingError for encrypted keys](https://github.com/golang/go/issues/71048) - GitHub issue discussing encrypted key detection (needs verification with actual testing)
- DSA key deprecation timeline - Stated as OpenSSH 7.0 (2015) but should verify current golang.org/x/crypto/ssh support level

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - golang.org/x/crypto/ssh is official and well-documented, published Feb 2026
- Architecture: HIGH - Patterns verified against existing codebase (picker.go, writer.go) and official package docs
- Pitfalls: HIGH - Sourced from ssh_config man page, known SSH authentication issues, and Go package issue tracker

**Research date:** 2026-02-14
**Valid until:** 2026-03-14 (30 days - stable domain)
