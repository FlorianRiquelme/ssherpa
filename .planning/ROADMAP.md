# Roadmap: sshjesus

## Overview

This roadmap delivers an SSH connection manager TUI that differentiates through git-based project detection and pluggable credential backends. The journey starts with architectural foundations (backend interface, domain models), progresses through core functionality (config parsing, connection flow, project detection), adds 1Password integration for team credential sharing, and culminates in advanced SSH features and multi-platform distribution. Each phase delivers verifiable user capabilities, building toward an MVP that solves the "30-100 SSH aliases in .zshrc" problem with zero-config project awareness.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation & Architecture** - Backend interface and domain models
- [x] **Phase 2: SSH Config Integration** - Parse and display ~/.ssh/config
- [x] **Phase 3: Connection & Navigation** - Search, filter, and connect to servers
- [x] **Phase 4: Project Detection** - Git remote-based project organization
- [x] **Phase 5: Config Management** - CRUD operations for SSH connections
- [x] **Phase 6: 1Password Backend** - Credential storage via 1Password SDK
- [ ] **Phase 7: SSH Key Selection** - Select which SSH key to use per connection
- [ ] **Phase 8: Distribution** - Single binary packaging and release

## Phase Details

### Phase 1: Foundation & Architecture
**Goal**: Establish pluggable backend architecture with domain models and mock implementation for testing
**Depends on**: Nothing (first phase)
**Requirements**: BACK-01
**Success Criteria** (what must be TRUE):
  1. Backend interface defines clear contract for credential/config operations (read, write, update, delete)
  2. Domain types (Project, Server, Credential) exist with proper Go idioms
  3. Mock backend implementation allows testing without external dependencies
  4. Core architecture supports future 1Password, local file, and vault backends
**Plans:** 2 plans

Plans:
- [x] 01-01-PLAN.md — Go module, domain models, error types, and backend interface contracts
- [x] 01-02-PLAN.md — Mock backend implementation, config management, and comprehensive TDD tests

### Phase 2: SSH Config Integration
**Goal**: Users can view all SSH connections from ~/.ssh/config in a working TUI
**Depends on**: Phase 1
**Requirements**: CONN-01
**Success Criteria** (what must be TRUE):
  1. User can launch sshjesus and see all connections from ~/.ssh/config
  2. Config parser preserves formatting, comments, and Include directives
  3. TUI renders server list with basic keyboard navigation (arrow keys, Enter)
  4. Connection details display hostname, user, port, and key path
**Plans:** 2 plans

Plans:
- [x] 02-01-PLAN.md — SSH config parser wrapper and sshconfig backend adapter
- [x] 02-02-PLAN.md — Bubbletea TUI with list view, detail view, and main.go wiring

### Phase 3: Connection & Navigation
**Goal**: Users can search servers and connect via system SSH
**Depends on**: Phase 2
**Requirements**: CONN-02, CONN-03
**Success Criteria** (what must be TRUE):
  1. User can fuzzy-search servers by name, hostname, or user
  2. Search updates in real-time as user types
  3. User can select a server and press Enter to initiate SSH connection
  4. SSH opens in current terminal session (tea.ExecProcess handoff)
  5. TUI resumes correctly after SSH session ends
**Plans:** 2 plans

Plans:
- [x] 03-01-PLAN.md — Connection history package and SSH connection helper
- [x] 03-02-PLAN.md — TUI overhaul with fuzzy search, SSH handoff, keybindings, and help footer

### Phase 4: Project Detection
**Goal**: Servers organize automatically by project based on git remote URL matching
**Depends on**: Phase 3
**Requirements**: PROJ-01, PROJ-02, PROJ-03
**Success Criteria** (what must be TRUE):
  1. Tool detects current project from git remote URL when launched in a repo
  2. Servers tagged with project identifiers display grouped by project
  3. User sees their current project's servers highlighted or filtered by default
  4. User can manually assign servers to projects via TUI
  5. Git detection handles SSH/HTTPS URLs and multiple remotes gracefully
**Plans:** 3 plans

Plans:
- [x] 04-01-PLAN.md — Git remote detection, project color generation, and TOML project storage (TDD)
- [x] 04-02-PLAN.md — TUI overhaul with project badges, grouped list, and project-aware search
- [x] 04-03-PLAN.md — Project picker overlay with hostname matcher and persistent assignment

### Phase 5: Config Management
**Goal**: Users can add, edit, and delete SSH connections with validation
**Depends on**: Phase 4
**Requirements**: CONF-01, CONF-02, CONF-03
**Success Criteria** (what must be TRUE):
  1. User can add new SSH connection via interactive form with field validation
  2. User can edit existing connection's hostname, user, port, and key path
  3. User can delete connection with confirmation prompt (prevents accidental loss)
  4. Config modifications preserve existing formatting and comments
  5. Automatic backup created before any destructive operation
**Plans:** 3 plans

Plans:
- [x] 05-01-PLAN.md — SSH config writer with formatting-preserving add/edit/delete, backup, and atomic writes
- [x] 05-02-PLAN.md — Full-screen add/edit form with field validation and DNS checking
- [x] 05-03-PLAN.md — Delete confirmation, session undo buffer, and human verification of complete CRUD

### Phase 6: 1Password Backend
**Goal**: Credentials store securely in 1Password with team sharing via shared vaults
**Depends on**: Phase 5
**Requirements**: BACK-02
**Success Criteria** (what must be TRUE):
  1. 1Password backend adapter implements backend interface from Phase 1
  2. Server configs read from and write to 1Password items via SDK
  3. Tool detects 1Password Desktop app session and handles auth gracefully
  4. Privilege escalation detection warns users when running as root
  5. Shared vault items enable team access to same server configs
**Plans:** 5 plans

Plans:
- [x] 06-01-PLAN.md — 1Password SDK client wrapper, item mapping, and backend adapter (TDD)
- [x] 06-02-PLAN.md — Sync engine: SSH include file, TOML cache, and conflict detection (TDD)
- [x] 06-03-PLAN.md — Offline fallback: status tracking, background poller, auto-recovery (TDD)
- [x] 06-04-PLAN.md — Multi-backend aggregator, TUI status bar, and main.go wiring
- [x] 06-05-PLAN.md — Setup wizard, migration wizard, and end-to-end verification

### Phase 7: SSH Key Selection
**Goal**: Users can select which SSH key to use for each connection
**Depends on**: Phase 6
**Requirements**: SSH-01
**Success Criteria** (what must be TRUE):
  1. User can select which SSH key to use for each connection
  2. Available SSH keys are discovered and presented for selection
  3. Key selection persists in SSH config (IdentityFile directive)
  4. Key selection renders correctly without blocking TUI event loop
**Plans:** 2 plans

Plans:
- [ ] 07-01-PLAN.md — SSH key discovery package with file/agent/1Password sources and TDD tests
- [ ] 07-02-PLAN.md — Key picker overlay, form integration, detail view display, and end-to-end verification

**Deferred to future version:** Port forwarding configuration (SSH-02), connection status indicators (CONN-04), ProxyJump/bastion host support

### Phase 8: Distribution
**Goal**: Tool ships as single binary via Homebrew and GitHub releases
**Depends on**: Phase 7
**Requirements**: DIST-01
**Success Criteria** (what must be TRUE):
  1. GoReleaser produces single binaries for macOS, Linux, and Windows
  2. Homebrew tap allows `brew install sshjesus`
  3. GitHub releases include checksums and installation instructions
  4. Binary runs on all supported platforms without additional dependencies
  5. Terminal compatibility tested across iTerm2, Alacritty, Windows Terminal, and tmux
**Plans**: TBD

Plans:
- [ ] 08-01: TBD during planning

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation & Architecture | 2/2 | ✓ Complete | 2026-02-14 |
| 2. SSH Config Integration | 2/2 | ✓ Complete | 2026-02-14 |
| 3. Connection & Navigation | 2/2 | ✓ Complete | 2026-02-14 |
| 4. Project Detection | 3/3 | ✓ Complete | 2026-02-14 |
| 5. Config Management | 3/3 | ✓ Complete | 2026-02-14 |
| 6. 1Password Backend | 5/5 | ✓ Complete | 2026-02-14 |
| 7. SSH Key Selection | 0/2 | In Progress | - |
| 8. Distribution | 0/TBD | Not started | - |
