# Requirements: sshjesus

**Defined:** 2026-02-14
**Core Value:** Find and connect to the right SSH server instantly, from any repo, without remembering aliases or grepping config files.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Connection Management

- [ ] **CONN-01**: User can view all SSH connections parsed from ~/.ssh/config with formatting preserved
- [ ] **CONN-02**: User can search and filter connections with fuzzy matching
- [ ] **CONN-03**: User can select a server and SSH opens in the current terminal
- [ ] **CONN-04**: User can see real-time connection status (reachable/unreachable) via async ping

### Config Operations

- [ ] **CONF-01**: User can add a new SSH connection via interactive form
- [ ] **CONF-02**: User can edit an existing SSH connection's details
- [ ] **CONF-03**: User can delete an SSH connection with confirmation prompt

### Project Organization

- [ ] **PROJ-01**: Tool auto-detects current project from git remote URL
- [ ] **PROJ-02**: Servers are grouped by project in the TUI
- [ ] **PROJ-03**: User can manually assign servers to projects

### Credential Backend

- [ ] **BACK-01**: Pluggable Go interface for credential/config backends
- [ ] **BACK-02**: 1Password backend reads/writes server configs via 1Password SDK

### SSH Features

- [ ] **SSH-01**: User can select which SSH key to use per connection
- [ ] **SSH-02**: User can set up port forwarding (local, remote, dynamic) per connection

### Distribution

- [ ] **DIST-01**: Tool is distributed as a single Go binary via Homebrew tap

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Team Sharing

- **TEAM-01**: Team members can share server configs via 1Password shared vaults
- **TEAM-02**: Per-person access control via vault permissions

### Additional Backends

- **ADDL-01**: Local file backend (YAML/JSON) for users without 1Password
- **ADDL-02**: HashiCorp Vault backend
- **ADDL-03**: AWS Secrets Manager backend

### Advanced SSH

- **ADVS-01**: ProxyJump/bastion host support
- **ADVS-02**: Tag-based categorization (prod/staging/dev)
- **ADVS-03**: Connection history with last-used timestamps

### Distribution Expansion

- **DSTX-01**: Cross-platform support (Linux, Windows)
- **DSTX-02**: Scoop manifest for Windows

### Security Hardening

- **SECR-01**: SSH key lifecycle tracking (created, last used, expiry warnings)
- **SECR-02**: Audit view showing all active keys and usage
- **SECR-03**: Automatic config backups before modifications

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Built-in terminal emulator | Use system SSH — let users choose their terminal |
| Custom SSH implementation | Security risk, maintenance burden — always use system SSH binary |
| GUI version | Scope creep, different audience — TUI tools exist for this niche |
| Cloud sync service | Infrastructure cost, security liability — use vault backends |
| File transfer UI (SFTP/SCP) | Feature bloat — tools already exist for this |
| Session recording/logging | Privacy concerns, scope expansion — use `script` or terminal features |
| Built-in credential storage | Security risk — always delegate to vault backends |
| Multi-protocol (RDP, VNC) | Dilutes focus — SSH only |
| Web UI | Different tech stack — single binary TUI only |
| Mobile app | Terminal-only tool |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CONN-01 | Phase 2 | Pending |
| CONN-02 | Phase 3 | Pending |
| CONN-03 | Phase 3 | Pending |
| CONN-04 | Phase 7 | Pending |
| CONF-01 | Phase 5 | Pending |
| CONF-02 | Phase 5 | Pending |
| CONF-03 | Phase 5 | Pending |
| PROJ-01 | Phase 4 | Pending |
| PROJ-02 | Phase 4 | Pending |
| PROJ-03 | Phase 4 | Pending |
| BACK-01 | Phase 1 | Pending |
| BACK-02 | Phase 6 | Pending |
| SSH-01 | Phase 7 | Pending |
| SSH-02 | Phase 7 | Pending |
| DIST-01 | Phase 8 | Pending |

**Coverage:**
- v1 requirements: 15 total
- Mapped to phases: 15
- Unmapped: 0

---
*Requirements defined: 2026-02-14*
*Last updated: 2026-02-14 after roadmap creation*
