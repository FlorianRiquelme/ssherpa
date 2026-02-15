# Project Research Summary

**Project:** ssherpa
**Domain:** SSH Connection Management TUI with Pluggable Credential Backends
**Researched:** 2026-02-14
**Confidence:** HIGH

## Executive Summary

ssherpa is a terminal-based SSH connection manager that differentiates itself through git-based project organization and pluggable credential backends (1Password first). Research shows this domain is well-established with multiple competitors (Termius, sshm, lazyssh), but none offer automatic project detection from git remotes or first-class vault integration. The recommended approach uses Go with Bubbletea v2 for the TUI, the official 1Password SDK for credentials, and hexagonal architecture for pluggable backends.

The technical foundation is solid: Go's ecosystem for CLI tools is mature, Bubbletea v2 is production-ready despite being in beta, and the 1Password SDK provides official support. The main challenges aren't technical but operational: SSH key lifecycle management (preventing forgotten credentials), credential security in the plugin architecture (agent-blind design required), and Bubbletea's async patterns (avoiding blocking the event loop). These pitfalls have clear solutions and must be addressed from day one - retrofitting security and async patterns is expensive.

The critical path is bottom-up: backend interface first (enables testing without real 1Password), then business logic (git detection, SSH execution), then real adapters (1Password SDK), and finally the TUI layer. This sequence allows testing at each layer and minimizes rework. Skip the temptation to start with UI - the architecture patterns and security boundaries defined early determine long-term success.

## Key Findings

### Recommended Stack

Go with Bubbletea v2 provides the best foundation for a cross-platform TUI with single-binary distribution. The ecosystem is mature and battle-tested: Kubernetes, Docker, and GitHub CLI all use Bubbletea for their TUIs. The 1Password SDK (despite v0.x versioning) is production-ready per official documentation and provides native Go integration without shell-out overhead.

**Core technologies:**
- **Go 1.23+**: Industry standard for CLI tools, excellent cross-compilation, strong SSH ecosystem support
- **Bubbletea v2**: TUI framework with elm-architecture pattern, official component library (Bubbles), battle-tested in production tools
- **Lipgloss v2**: Terminal styling with CSS-like declarative syntax, handles cross-platform rendering automatically
- **1Password SDK Go**: Official credential backend, end-to-end encryption, native API (not CLI wrapper), read/write/update capabilities
- **golang.org/x/crypto/ssh**: Official SSH library with comprehensive support for known_hosts, host key verification, timeouts
- **Cobra + Viper**: Standard CLI framework for multi-command structure and configuration management

**Critical versions:**
- Use Bubbletea/Lipgloss/Bubbles v2 ecosystem (pin specific versions - currently alpha/beta, expect breaking changes)
- Use 1Password SDK v0.x (production-ready despite beta designation)
- Avoid Bubbletea v1 (v2 is current)
- Avoid `op` CLI shell wrappers (use official SDK instead)

### Expected Features

Research shows clear differentiation opportunity through git-based project organization and vault integration. Competitors focus on manual configuration or proprietary cloud sync - none auto-detect project context from git remotes.

**Must have (table stakes):**
- Read/parse ~/.ssh/config with format preservation
- List, search, filter connections with keyboard navigation
- Quick connect via system SSH binary
- Basic CRUD for connections with validation
- SSH key management and selection
- Port forwarding setup (local/remote/dynamic)
- Connection status indicators

**Should have (differentiators):**
- Git remote auto-detection for project organization (UNIQUE to ssherpa)
- 1Password backend integration for secure credentials
- Pluggable credential backend interface
- Team credential sharing via vault
- Project-based organization (not flat tags)
- Connection history and status tracking
- ProxyJump/bastion host support

**Defer (v2+):**
- Additional backends (HashiCorp Vault, AWS Secrets Manager)
- Batch operations for multiple hosts
- Export/import configurations
- Include directive support for organized configs
- Advanced SSH options UI

**Anti-features (never build):**
- Built-in terminal emulator (use system SSH)
- Custom SSH implementation (security risk)
- GUI version (scope creep)
- Cloud sync service (use vault backends)
- File transfer UI (feature bloat)

### Architecture Approach

Hexagonal architecture (ports and adapters) provides the foundation for pluggable backends while maintaining clean separation between business logic and external integrations. The core defines abstract interfaces for credential backends, while adapters implement these interfaces for specific services (1Password, local config, future vaults).

**Major components:**
1. **TUI Layer (Bubbletea)** — Root model routes messages to child models (server list, detail view, help), handles layout composition and global keys
2. **Core Business Logic** — Project service auto-detects from git remotes, backend interface defines abstract contract, SSH executor wraps system SSH binary
3. **Backend Adapters** — 1Password adapter implements backend interface via SDK, local config adapter for file-based storage, future adapters pluggable
4. **External Integrations** — go-git for repository operations, 1Password SDK for credentials, system SSH via tea.ExecProcess

**Critical patterns:**
- **Async-first TUI**: All I/O operations (SSH, SDK calls, git) must run as tea.Cmd to avoid blocking the event loop
- **Model composition**: Nested Bubbletea models with message routing from root, each model self-contained
- **Interface extension**: Base backend interface for required methods, optional interfaces for features not all backends support
- **Agent-blind credentials**: Plugins provide metadata only, 1Password SDK fetches actual secrets, TUI never sees plaintext

### Critical Pitfalls

1. **Bubbletea event loop blocking** — Running SSH connections, SDK calls, or git operations in Update() freezes the UI. Solution: All I/O must run as tea.Cmd with immediate loading states. Rule: Update() and View() must complete in <10ms.

2. **Credential backend plugin trust model** — Malicious plugins could intercept credentials if design exposes them. Solution: Agent-blind architecture where plugins only provide connection metadata, 1Password SDK fetches actual secrets. Plugins never receive plaintext credentials.

3. **SSH key lifecycle blind spots** — Forgotten keys from ex-employees or old devices remain active indefinitely. Solution: Track metadata (creation date, last used) from day one, implement expiration warnings (>90 days), provide audit views showing all active keys.

4. **1Password SDK security assumptions** — Root users can bypass security if desktop app is unlocked, macOS accessibility permissions can circumvent prompts. Solution: Detect elevated privileges and warn, handle Windows sub-shell auth differently than Unix, track session timeouts.

5. **Terminal compatibility assumptions** — TUIs render incorrectly across different terminals, true color support unreliable. Solution: Use lipgloss for all layout (handles cross-platform), test in multiple terminals (iTerm2, Alacritty, Windows Terminal, tmux), provide fallback for limited color support.

## Implications for Roadmap

Based on research, suggested phase structure follows bottom-up dependency order: interfaces before implementations, business logic before UI, security boundaries from day one.

### Phase 1: Foundation & Interfaces
**Rationale:** Backend interface must exist before any implementation work. This enables testing with mock backends and ensures the security boundary (agent-blind credentials) is designed correctly from the start. Git detection provides the unique differentiator, so validate this early.

**Delivers:** Backend interface definition, domain types (Project, Server, Credential), mock backend for testing, project service with git remote detection, basic SSH executor

**Addresses:** Core architecture from ARCHITECTURE.md (hexagonal pattern), git remote parsing from FEATURES.md (differentiator)

**Avoids:** Credential backend plugin trust model pitfall (design agent-blind API from start), git remote parsing fragility (handle SSH/HTTPS/multiple hosts early)

**Research flags:** STANDARD PATTERNS - Backend interface and domain modeling are well-documented Go patterns

### Phase 2: 1Password Integration
**Rationale:** Real credential backend validates the interface design and proves the unique value proposition. 1Password SDK integration has security nuances (session handling, privilege detection, platform differences) that need careful implementation.

**Delivers:** 1Password SDK adapter implementing backend interface, privilege escalation detection, session timeout tracking, platform-specific auth handling (Windows vs Unix)

**Uses:** 1Password SDK Go v0.x from STACK.md, backend interface from Phase 1

**Implements:** Backend adapter component from ARCHITECTURE.md

**Avoids:** 1Password security assumptions pitfall (detect root, handle timeouts, Windows sub-shell auth)

**Research flags:** NEEDS RESEARCH - 1Password SDK error scenarios, session edge cases, cross-platform auth differences need deeper investigation during phase planning

### Phase 3: Core TUI Shell
**Rationale:** With working backend, build minimal TUI to validate async patterns and user interaction. This phase establishes the event loop discipline (no blocking in Update/View) that all future features depend on.

**Delivers:** Root model with message routing, server list view using Bubbles components, basic keyboard navigation, loading states and spinners, Lipgloss styling for cross-platform rendering

**Addresses:** TUI components from ARCHITECTURE.md, keyboard navigation from FEATURES.md (table stakes)

**Uses:** Bubbletea v2, Lipgloss v2, Bubbles v2 from STACK.md

**Avoids:** Event loop blocking pitfall (all I/O as tea.Cmd), terminal compatibility assumptions (lipgloss from start)

**Research flags:** STANDARD PATTERNS - Bubbletea async patterns well-documented, but validate with spike testing

### Phase 4: SSH Connection Flow
**Rationale:** Connecting to servers is the core user workflow. SSH execution via tea.ExecProcess has specific patterns for handing over terminal control and resuming. Integration with backend credentials completes the end-to-end flow.

**Delivers:** SSH command builder with credential injection, tea.ExecProcess integration with resume callback, connection status tracking, error handling for auth failures and timeouts

**Addresses:** Quick connect from FEATURES.md (table stakes), SSH executor from ARCHITECTURE.md

**Uses:** golang.org/x/crypto/ssh from STACK.md, system SSH binary via tea.ExecProcess

**Avoids:** SSH multiplexing chaos (don't depend on multiplexing, clean up control sockets)

**Research flags:** STANDARD PATTERNS - SSH execution well-documented, but test timeout and error scenarios

### Phase 5: SSH Config Management
**Rationale:** With connection flow working, add full CRUD operations. Non-destructive parsing preserves user comments and formatting. This completes the table stakes feature set for MVP.

**Delivers:** ~/.ssh/config parser with format preservation, add/edit/delete operations with validation, configuration backups before modifications, Include directive support

**Addresses:** Config management from FEATURES.md (table stakes), configuration validation

**Avoids:** Data loss from destructive edits (preserve formatting and comments)

**Research flags:** NEEDS RESEARCH - SSH config parsing libraries, format preservation techniques need investigation

### Phase 6: Key Lifecycle & Audit
**Rationale:** Security hardening addresses the critical pitfall around forgotten credentials. Audit views and lifecycle tracking prevent the "ex-employee key active for 2 years" scenario.

**Delivers:** Key metadata tracking (created, last used, purpose), expiration warnings for old keys, audit views showing all keys and usage, bulk revocation workflows

**Addresses:** SSH key management from FEATURES.md, key lifecycle blind spots pitfall

**Avoids:** Security audit nightmare from accumulated stale credentials

**Research flags:** STANDARD PATTERNS - Timestamp tracking straightforward, UI design for audit views

### Phase 7: Advanced Features
**Rationale:** Polish and competitive features once core value is proven. Port forwarding, ProxyJump, and connection history add depth without changing the fundamental architecture.

**Delivers:** Port forwarding UI (local/remote/dynamic), ProxyJump/bastion support, connection history, tag system for organization

**Addresses:** Port forwarding and ProxyJump from FEATURES.md (should-have)

**Research flags:** STANDARD PATTERNS - SSH features well-documented, UI patterns from other tools

### Phase 8: Multi-Platform & Distribution
**Rationale:** Final phase ensures broad compatibility and easy installation. GoReleaser automates the complex multi-platform build process.

**Delivers:** GoReleaser configuration for cross-compilation, Homebrew tap for macOS, Scoop manifest for Windows, terminal compatibility testing matrix, degraded mode for limited terminals

**Uses:** GoReleaser from STACK.md, go:embed for static files

**Avoids:** Terminal compatibility pitfall (test matrix across terminals)

**Research flags:** STANDARD PATTERNS - GoReleaser well-documented, terminal testing matrix established

### Phase Ordering Rationale

- **Bottom-up dependencies**: Backend interface → real adapter → TUI → features prevents architectural rework
- **Validate uniqueness early**: Git detection in Phase 1 proves differentiator before heavy TUI investment
- **Security from day one**: Agent-blind API design and privilege detection can't be retrofitted
- **Async patterns established early**: Phase 3 TUI shell enforces tea.Cmd discipline before complexity increases
- **MVP at Phase 5**: Phases 1-5 deliver full CRUD + connection flow + 1Password integration for early users
- **Security hardening before wide release**: Phase 6 addresses audit requirements before marketing
- **Polish last**: Phases 7-8 add depth after core value proven

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (1Password Integration):** Complex SDK edge cases, session management, platform-specific auth differences need API research
- **Phase 5 (Config Management):** SSH config parsing and format preservation techniques need library evaluation

Phases with standard patterns (skip research-phase):
- **Phase 1 (Foundation):** Backend interfaces and domain modeling well-documented Go patterns
- **Phase 3 (TUI Shell):** Bubbletea async patterns documented in official guides and real-world examples
- **Phase 4 (SSH Connection):** SSH execution and tea.ExecProcess patterns established
- **Phase 6 (Key Lifecycle):** Timestamp tracking and audit UI are straightforward implementations
- **Phase 7 (Advanced Features):** SSH features and UI patterns documented extensively
- **Phase 8 (Distribution):** GoReleaser and cross-compilation well-documented

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Official sources (Go team, 1Password, Charm), production-ready libraries, mature ecosystem |
| Features | MEDIUM | Good competitor analysis and user expectations clear, but git-based organization is novel (unproven demand) |
| Architecture | HIGH | Hexagonal pattern well-established for Go, Bubbletea patterns documented with real-world examples |
| Pitfalls | MEDIUM-HIGH | Security concerns verified with official docs, event loop and terminal issues documented, some edge cases inferred |

**Overall confidence:** HIGH

### Gaps to Address

- **1Password SDK error scenarios:** Documentation shows happy path but edge cases (network failures, SDK errors, corrupted vaults) need discovery during Phase 2 implementation. Plan for extended testing period.

- **Git remote parsing completeness:** Research covered common cases (GitHub, GitLab, SSH vs HTTPS) but monorepos, submodules, and enterprise git hosting may have undocumented quirks. Build robust fallbacks and allow manual project override.

- **Bubbletea v2 stability:** Charm ecosystem in alpha/beta means potential breaking changes. Pin to specific versions and monitor releases closely. Budget for upgrade work if breaking changes occur.

- **Cross-platform terminal compatibility:** Research identified major terminals but full matrix (older terminals, SSH-forwarded terminals, screen/tmux combinations) needs empirical testing. Build degraded mode early to handle unknowns gracefully.

- **Plugin system scope:** Research focused on security boundaries but didn't address plugin discovery, versioning, or lifecycle management. If expanding beyond 1Password/local, design plugin system carefully in future phases.

## Sources

### Primary (HIGH confidence)
- Bubbletea GitHub & pkg.go.dev — v2 version confirmation, async patterns, tea.ExecProcess usage
- golang.org/x/crypto/ssh package docs — SSH client API, timeout handling, known_hosts management
- 1Password SDK Go documentation — Official SDK capabilities, production readiness, secret references
- GoReleaser documentation — Build automation, multi-platform distribution
- Context7: bubbletea, golang.org/x/crypto/ssh, 1password-sdk-go — Library API details

### Secondary (MEDIUM confidence)
- Tips for Building Bubble Tea Programs (leg100.github.io) — Real-world architecture patterns, concurrency best practices
- SSHM GitHub — Real-world Bubbletea TUI for SSH management, architecture reference
- Go Clean Architecture guides — Hexagonal pattern implementation in Go
- SSH best practices articles (DevToolbox, Hoop.dev, Linuxize) — Config management, key lifecycle
- 1Password CLI security documentation — App integration security boundaries, privilege concerns

### Tertiary (LOW confidence, needs validation)
- Competitor feature comparison — Inferred from tool descriptions and screenshots, not hands-on testing
- Terminal compatibility issues — Described in articles but not exhaustively tested across all platforms
- SSH multiplexing session limits — Specific 10-connection limit and behavior needs verification
- Git remote URL format coverage — Focused on major platforms, may miss self-hosted edge cases

---
*Research completed: 2026-02-14*
*Ready for roadmap: yes*
