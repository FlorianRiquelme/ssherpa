# sshjesus

## What This Is

An open-source TUI tool for managing and connecting to SSH servers, organized by project. It auto-detects which project you're working in (via git remote), shows available servers, and lets you connect instantly. Team sharing is powered by credential stores like 1Password — no custom backend needed.

## Core Value

Find and connect to the right SSH server instantly, from any repo, without remembering aliases or grepping config files.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Interactive TUI for browsing and searching SSH servers
- [ ] Servers organized by project/group
- [ ] Context-aware project detection via git remote matching
- [ ] Direct SSH connection from the TUI (opens ssh in current terminal)
- [ ] Pluggable backend interface for credential/config storage
- [ ] 1Password backend (read/write server configs via `op` CLI)
- [ ] Team sharing via 1Password shared vaults
- [ ] Per-person access control via vault permissions
- [ ] Single binary distribution (Go)

### Out of Scope

- Web UI — this is a terminal tool
- Built-in SSH terminal emulator — delegates to system ssh
- Custom backend/server — credential stores are the backend
- Mobile app — terminal-only
- SSH key generation or management — 1Password handles this natively

## Context

- Currently managing 30-100 SSH connections as .zshrc aliases
- Pain points: hard to discover available servers, no project organization, can't share with per-person access
- Team uses 1Password with SSH agent support already enabled
- 1Password CLI (`op`) provides programmatic access to vaults and items
- 1Password shared vaults provide natural team sharing + access control
- Git remote URLs can identify which project a developer is working in
- The pluggable backend design enables community contributions (Bitwarden, pass, local YAML, etc.)

## Constraints

- **Language**: Go — for single binary distribution and Bubbletea TUI ecosystem
- **TUI framework**: Bubbletea — proven library used by lazygit, lazydocker, etc.
- **First backend**: 1Password via `op` CLI — most common in the target team
- **Distribution**: Single binary via `go install`, Homebrew, and GitHub releases
- **License**: Open source (TBD specific license)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| TUI over desktop app | Stays in terminal workflow, faster to build, fits the audience | — Pending |
| 1Password as backend, not custom server | Eliminates need for custom infra, leverages existing team tooling | — Pending |
| Pluggable backend interface | Enables open-source community to add Bitwarden, pass, local file, etc. | — Pending |
| Git remote matching for project detection | Zero-config project awareness — no manual tagging needed | — Pending |
| Go + Bubbletea | Single binary, great TUI ecosystem, familiar to OSS community | — Pending |

---
*Last updated: 2026-02-14 after initialization*
