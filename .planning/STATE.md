# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-14)

**Core value:** Find and connect to the right SSH server instantly, from any repo, without remembering aliases or grepping config files.
**Current focus:** Phase 3 complete — Core search-and-connect workflow functional. Ready for Phase 4 (project context detection)

## Current Position

Phase: 4 of 8 (Project Detection)
Plan: 1 of 2 (complete)
Status: In Progress
Last activity: 2026-02-14 — Completed Plan 04-01: Git remote detection, project colors, and TOML config storage

Progress: [███████████████████████████████████████████████████████████████] 65%

## Performance Metrics

**Velocity:**
- Total plans completed: 7
- Average duration: 587.6 seconds
- Total execution time: 1.14 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01    | 2     | 429s  | 214.5s   |
| 02    | 2     | 319s  | 159.5s   |
| 03    | 2     | 2823s | 1411.5s  |
| 04    | 1     | 266s  | 266.0s   |

**Recent Plans:**

| Phase | Plan | Duration | Tasks | Files |
|-------|------|----------|-------|-------|
| 04    | 01   | 266s     | 2     | 8     |
| 03    | 02   | 2236s    | 2     | 8     |
| 03    | 01   | 587s     | 2     | 3     |
| 02    | 02   | 26s      | 3     | 8     |
| 02    | 01   | 293s     | 2     | 6     |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- **TUI over desktop app**: Stays in terminal workflow, faster to build, fits the audience
- **1Password as backend, not custom server**: Eliminates need for custom infra, leverages existing team tooling
- **Pluggable backend interface**: Enables open-source community to add Bitwarden, pass, local file, etc.
- **Git remote matching for project detection**: Zero-config project awareness — no manual tagging needed
- **Go + Bubbletea**: Single binary, great TUI ecosystem, familiar to OSS community
- **Storage-agnostic domain models (01-01)**: Domain types have zero external dependencies (no struct tags, no storage imports)
- **Database/sql pattern for backends (01-01)**: Minimal required interface, optional capabilities via type assertions
- **Many-to-many on Server side only (01-01)**: Server.ProjectIDs tracks relationships, no reverse tracking on Project
- **Copy-on-read and copy-on-write (01-02)**: Prevents mutation leaks between caller and backend state
- **TOML config format (01-02)**: Eliminates YAML indentation bugs, provides explicit types
- **Empty Backend config signals setup needed (01-02)**: No default backend selection, setup wizard deferred to Phase 2+
- **Testify for test assertions (01-02)**: Better error messages and cleaner test code than stdlib alone
- **Full-screen detail view over split-panel (02-02)**: Simpler implementation, works at any terminal width, clearer focus
- **AdaptiveColor for TUI theming (02-02)**: Supports both light and dark terminals without manual detection
- **JSON lines for history (03-01)**: Newline-delimited JSON enables append-only writes, graceful corruption recovery
- **bufio.Scanner for parsing (03-01)**: Line-by-line unmarshaling skips malformed entries without decoder hang
- **SSH config alias-only (03-01)**: Pass alias to ssh command, leverage existing config (ProxyJump, IdentityFile, etc.)
- **Always-on search bar (03-02)**: Filter bar always visible at top, matches browser UX pattern for zero-friction search
- **Enter connects, Tab/i for details (03-02)**: Primary action (connect) gets most natural key, details are secondary
- **Exit after SSH by default (03-02)**: ReturnToTUI=false matches native ssh UX, keeps user in terminal flow
- **Only origin remote for project detection (04-01)**: Simplifies implementation, covers 99% of use cases
- **Empty string for non-git contexts (04-01)**: Non-git directories are valid use case, not error condition
- **Hex colors over ANSI 256 (04-01)**: Better precision, lipgloss handles degradation automatically
- **Empty Color field means auto-generate (04-01)**: User override support while defaulting to deterministic generation

### Pending Todos

None yet.

### Blockers/Concerns

**Phase 4 considerations:**
- 1Password SDK error scenarios need discovery during implementation (network failures, corrupted vaults)
- Git remote parsing must handle monorepos, submodules, enterprise hosting edge cases
- Project-to-server mapping needs flexible matching (exact, prefix, regex patterns)

**Cross-platform considerations:**
- Terminal compatibility matrix needs empirical testing (older terminals, SSH-forwarded, screen/tmux combinations)

## Session Continuity

Last session: 2026-02-14 (plan execution)
Stopped at: Completed 04-01-PLAN.md — Git remote detection, project colors, and TOML config storage implemented. Foundation ready for project-aware TUI features in 04-02.
Resume file: None
