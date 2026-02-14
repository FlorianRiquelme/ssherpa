# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-14)

**Core value:** Find and connect to the right SSH server instantly, from any repo, without remembering aliases or grepping config files.
**Current focus:** Phase 3 complete — Core search-and-connect workflow functional. Ready for Phase 4 (project context detection)

## Current Position

Phase: 3 of 8 (Connection & Navigation)
Plan: 2 of 2 (complete)
Status: Complete
Last activity: 2026-02-14 — Completed Phase 3: Search-and-connect TUI with fuzzy search, SSH handoff, Vim navigation

Progress: [███████████████████████████████████████████████████████████] 60%

## Performance Metrics

**Velocity:**
- Total plans completed: 6
- Average duration: 643.5 seconds
- Total execution time: 1.07 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01    | 2     | 429s  | 214.5s   |
| 02    | 2     | 319s  | 159.5s   |
| 03    | 2     | 2823s | 1411.5s  |

**Recent Plans:**

| Phase | Plan | Duration | Tasks | Files |
|-------|------|----------|-------|-------|
| 03    | 02   | 2236s    | 2     | 8     |
| 03    | 01   | 587s     | 2     | 3     |
| 02    | 02   | 26s      | 3     | 8     |
| 02    | 01   | 293s     | 2     | 6     |
| 01    | 02   | 299s     | 2     | 7     |

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
Stopped at: Completed Phase 3 (Connection & Navigation) — Fuzzy search, SSH handoff, Vim navigation, and history tracking fully functional. Core search-and-connect workflow complete. Next: Phase 4 for project context detection.
Resume file: None
