# Technology Stack

**Project:** ssherpa
**Researched:** 2026-02-14
**Overall Confidence:** HIGH

## Recommended Stack

### Core Framework

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go | 1.23+ | Language runtime | Industry standard for CLI tools, excellent cross-compilation, single binary output, strong SSH ecosystem support |
| Bubbletea | v2.x | TUI framework | Battle-tested (Kubernetes, Docker, Hugo, GitHub CLI use it), elm-architecture pattern, excellent ecosystem, active development |
| Lipgloss | v2.x | Terminal styling | Official companion to Bubbletea, CSS-like declarative styling, composable styles, standard for production TUIs |
| Bubbles | v2.x | TUI components | Official component library, pre-built widgets (list, input, spinner, viewport), production-ready, maintained by Charm |

### SSH & Credentials

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| golang.org/x/crypto/ssh | v0.48.0+ | SSH client implementation | Official Go crypto library, comprehensive SSH support, known_hosts management, host key verification |
| golang.org/x/crypto/ssh/knownhosts | v0.48.0+ | known_hosts parsing | Official parser for OpenSSH known_hosts, standard compliance, battle-tested |
| 1Password SDK Go | v0.x (beta) | Credential backend | Official 1Password SDK, end-to-end encryption, production-ready despite v0, native Go integration, programmatic secret access |

**1Password Integration Note:** Use official SDK over `op` CLI wrappers. SDK provides:
- Native Go API (no shell-out overhead)
- End-to-end encryption in-process
- Secret references for secure loading
- Read/write/update capabilities
- Official support and maintenance

**Confidence:** HIGH - All official libraries from authoritative sources (Go team, 1Password, Charm)

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Cobra | v1.x | CLI argument parsing | Multi-command CLI structure, nested subcommands, flag management, auto-generated help |
| Viper | v1.x | Configuration management | YAML/TOML/JSON config files, environment variables, defaults, works seamlessly with Cobra |

**Alternative Considered:** koanf (lighter, fewer dependencies, better abstractions than Viper)
- **Use koanf if:** Configuration needs are simple, want minimal dependencies, need custom providers
- **Use Viper if:** Standard configuration patterns, tight Cobra integration needed, team familiarity

**For ssherpa:** Recommend **Viper** - Standard choice, Cobra integration, config file + env vars pattern matches SSH tool conventions

**Confidence:** HIGH for Cobra (industry standard), MEDIUM for Viper (koanf is viable alternative)

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| GoReleaser | Binary distribution | Automates multi-platform builds, creates GitHub releases, supports Homebrew/Scoop/AUR, single config file |
| Go standard testing | Unit tests | Built-in testing is sufficient for CLI tools, no extra framework needed |
| go:embed | Static file embedding | For default config templates, help text - keeps single binary goal |

**Testing Framework Decision:** **No external framework needed**
- Go's stdlib `testing` package is sufficient
- Testify/Ginkgo add complexity without clear benefit for CLI tools
- Keep it simple: table-driven tests with stdlib

**Confidence:** HIGH - Simple testing, proven tooling

## Installation

```bash
# Core dependencies
go get github.com/charmbracelet/bubbletea/v2
go get github.com/charmbracelet/lipgloss/v2
go get github.com/charmbracelet/bubbles/v2

# SSH and crypto
go get golang.org/x/crypto/ssh

# 1Password SDK
go get github.com/1password/onepassword-sdk-go

# CLI framework
go get github.com/spf13/cobra
go get github.com/spf13/viper
```

## Alternatives Considered

| Category | Recommended | Alternative | Why Not Alternative |
|----------|-------------|-------------|---------------------|
| TUI Framework | Bubbletea | tview, termui | Bubbletea has larger ecosystem, better architecture (elm-pattern), official component library |
| CLI Framework | Cobra | urfave/cli, ffcli | Cobra is standard for complex CLIs, better nesting, used by kubectl/hugo/docker |
| Config Management | Viper | koanf, godotenv | Viper is standard, tight Cobra integration (koanf is valid for simpler needs) |
| 1Password Integration | Official SDK | op CLI wrappers | SDK is production-ready, native Go, end-to-end encryption, official support |
| Testing | stdlib testing | Testify, Ginkgo | Stdlib sufficient for CLI, avoid dependency bloat for assertions/BDD features |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Old Bubbletea v1 | v2 is current major version, better API, active development | Bubbletea v2 |
| op CLI shell wrappers | Unofficial, shell-out overhead, no official support, security concerns | 1Password SDK Go |
| ssh.Dial() directly | No timeout control, hangs on auth failures | net.Dial() + ssh.NewClientConn() with SetDeadline() |
| packr/statik for embedding | Deprecated, go:embed is stdlib since Go 1.16 | go:embed directive |
| Complex mocking frameworks | Over-engineering for CLI tools, slow test suites | stdlib testing with table-driven tests |

## Stack Patterns by Feature

### SSH Connection Flow
```go
// Pattern: Manual net.Dial + ssh.NewClientConn (NOT ssh.Dial)
conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
conn.SetDeadline(time.Now().Add(30*time.Second)) // Auth timeout
client, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
```

**Why:** ssh.Dial() doesn't support timeouts, can hang indefinitely on auth

### Host Key Verification
```go
// Pattern: Use knownhosts.New() + custom callback for new hosts
hostKeyCallback, err := knownhosts.New(knownHostsPath)
// Wrap with custom handler for new/changed hosts
config.HostKeyCallback = customVerifyHost(hostKeyCallback)
```

**Why:** Strict verification by default, user-friendly prompts for new hosts

### 1Password Backend
```go
// Pattern: SDK client with secret references
client, err := onepassword.NewClient(ctx, ...)
secret, err := client.Secrets.Resolve(ctx, "op://vault/item/field")
```

**Why:** Secret references keep actual values out of code, end-to-end encryption

### Bubbletea + Bubbles
```go
// Pattern: Embed Bubbles components in model
type model struct {
    list list.Model    // Bubbles list component
    input textinput.Model
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Delegate to component updates
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}
```

**Why:** Component composition, idiomatic Bubbletea, reusable UI pieces

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| Bubbletea v2 | Lipgloss v2, Bubbles v2 | v2 ecosystem - use matching versions |
| Cobra v1.x | Viper v1.x | Designed to work together, tight integration |
| golang.org/x/crypto/ssh | Go 1.23+ | Pre-v1 but stable, use latest patch version |
| 1Password SDK v0.x | Go 1.23+ | Beta but production-ready per 1Password, expect breaking changes |

**Critical Note:** Charm ecosystem (Bubbletea/Lipgloss/Bubbles) is currently in v2 alpha/beta. For production:
- Track v2 releases closely
- Pin to specific versions in go.mod
- Monitor breaking changes
- v2 is recommended over v1 for new projects

## Project Structure

```
ssherpa/
├── cmd/
│   └── ssherpa/
│       └── main.go          # Minimal main, wire up deps
├── internal/
│   ├── tui/                 # Bubbletea models/views
│   ├── ssh/                 # SSH connection logic
│   ├── credentials/         # Backend interface + implementations
│   │   ├── backend.go       # Interface definition
│   │   └── onepassword/     # 1Password SDK integration
│   ├── config/              # Viper config management
│   └── project/             # Git remote detection
├── configs/                 # Embedded default configs (go:embed)
├── go.mod
└── .goreleaser.yml
```

**Why this structure:**
- `internal/` prevents external imports, free to refactor
- `cmd/` follows Go convention for binaries
- Pluggable backends via interface in `credentials/`
- Embedded configs keep single binary goal

## Build & Distribution

**GoReleaser Configuration:**
```yaml
builds:
  - main: ./cmd/ssherpa
    binary: ssherpa
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

brews:
  - repository:
      owner: your-org
      name: homebrew-tap
```

**Single Binary Features:**
- Cross-compilation: GoReleaser handles all platforms
- Static linking: Go produces self-contained binaries
- Embedded assets: Config templates via go:embed
- No dependencies: Pure Go, no external libs needed at runtime

## Sources

**High Confidence (Official Documentation & Context7):**
- [Bubbletea GitHub](https://github.com/charmbracelet/bubbletea) — v2 version confirmation, ecosystem overview
- [golang.org/x/crypto/ssh package](https://pkg.go.dev/golang.org/x/crypto/ssh) — v0.48.0 version, API documentation
- [1Password SDK Go](https://developer.1password.com/docs/sdks/) — Official SDK documentation, production readiness
- [GoReleaser documentation](https://goreleaser.com/) — Build automation, distribution patterns
- [Go embed package](https://pkg.go.dev/embed) — Stdlib embedding patterns

**Medium Confidence (WebSearch - Recent 2026 Articles):**
- [Bubbletea best practices](https://leg100.github.io/en/posts/building-bubbletea-programs/) — Development workflow, concurrency patterns
- [Go project structure 2026](https://oneuptime.com/blog/post/2026-01-07-go-project-structure/view) — cmd/internal layout recommendations
- [Go SSH timeout handling](https://utcc.utoronto.ca/~cks/space/blog/programming/GoSSHHostKeyCheckingNotes) — ssh.Dial vs net.Dial + NewClientConn pattern

**Medium Confidence (Comparison Research):**
- [Cobra vs alternatives](https://mt165.co.uk/blog/golang-cli-library/) — CLI framework comparison
- [Viper vs koanf](https://itnext.io/golang-configuration-management-library-viper-vs-koanf-eea60a652a22) — Config management trade-offs
- [Go testing frameworks 2026](https://reliasoftware.com/blog/golang-testing-framework) — Testing framework landscape

---
*Stack research for: Go-based SSH connection management TUI*
*Researched: 2026-02-14*
*Overall confidence: HIGH for core stack, MEDIUM for Charm v2 ecosystem timing*
