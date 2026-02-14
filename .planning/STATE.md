# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-14)

**Core value:** Find and connect to the right SSH server instantly, from any repo, without remembering aliases or grepping config files.
**Current focus:** Phase 6 in progress — 1Password backend adapter with SDK integration and tag-based discovery

## Current Position

Phase: 6 of 8 (1Password Backend)
Plan: 4 of 5 (complete)
Status: In Progress
Last activity: 2026-02-14 — Completed Plan 06-04: Multi-backend aggregator with priority-based deduplication, shared BackendStatus enum, TUI status bar for 1Password availability indication, and complete end-to-end wiring from backend to TUI.

Progress: [████████████████████████████████████████████████████████████████████████████████████████] 89%

## Performance Metrics

**Velocity:**
- Total plans completed: 15
- Average duration: 619.1 seconds
- Total execution time: 2.58 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01    | 2     | 429s  | 214.5s   |
| 02    | 2     | 319s  | 159.5s   |
| 03    | 2     | 2823s | 1411.5s  |
| 04    | 3     | 4118s | 1372.7s  |
| 05    | 3     | 525s  | 175.0s   |
| 06    | 3     | 1286s | 428.7s   |

**Recent Plans:**

| Phase | Plan | Duration | Tasks | Files |
|-------|------|----------|-------|-------|
| 06    | 04   | 570s     | 2     | 14    |
| 06    | 03   | 468s     | 2     | 7     |
| 06    | 02   | 351s     | 2     | 5     |
| 06    | 01   | 467s     | 2     | 9     |
| 05    | 03   | N/A      | 2     | 6     |

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
- **Inline badges over section headers (04-02)**: Simpler implementation, works at any terminal width, clearer visual hierarchy
- **Project separator in search results (04-02)**: Visual clarity between current project matches and other matches
- **Unassigned servers at bottom (04-02)**: Encourages project assignment, keeps project-related servers prominent
- **Levenshtein distance for hostname matching (04-03)**: Robust fuzzy matching handles typos and variations better than exact substring matching
- **70% similarity threshold for suggestions (04-03)**: Balances useful suggestions vs noise, empirically tested
- **Simple overlay without external library (04-03)**: Bubbletea v2 alpha compatibility concerns, fewer dependencies
- **App config path passed explicitly to TUI (04-03)**: Avoids confusion between SSH config and app config paths, enables proper persistence
- **Text-based SSH config manipulation (05-01)**: ssh_config library doesn't support writes, text-based approach preserves formatting byte-for-byte
- **4-space indentation standard (05-01)**: Consistent formatting for all generated Host blocks
- **Backup before every write (05-01)**: CreateBackup called before add/edit/delete creates .bak with same permissions
- **Case-insensitive alias matching (05-01)**: Duplicate detection matches SSH config behavior
- **Block boundary backtracking (05-01)**: Blank lines and comments between blocks excluded from block content
- [Phase 05-02]: Hand-built form component instead of charmbracelet/huh for Bubbletea v2 alpha compatibility
- [Phase 05-02]: Blur validation on field exit (Tab/j/k) with inline error messages
- [Phase 05-02]: DNS check is async and non-blocking (warning only, save proceeds)
- [Phase 05-03]: Type-to-confirm pattern for delete (GitHub-style UX for dangerous operations)
- [Phase 05-03]: Session-scoped undo buffer (max 10 entries, cleared on app exit)
- [Phase 05-03]: RestoreHost function in undo.go (avoids modifying Plan 01 writer files)
- [Phase 05-03]: Status flash messages for delete/undo feedback
- [Phase 06-01]: Client interface abstraction for testability (MockClient enables testing without real 1Password)
- [Phase 06-01]: Tag-based discovery with case-insensitive "sshjesus" tag filtering across all vaults
- [Phase 06-01]: Skip vaults with errors (permission issues don't fail entire list operation)
- [Phase 06-01]: Projects as tags not entities (1Password doesn't have standalone project concept)
- [Phase 06-01]: Lowercase "server" category matches 1Password API expectations
- [Phase 06-02]: Prepend Include directive to ~/.ssh/config (not append) for first-match-wins SSH semantics
- [Phase 06-02]: ForwardAgent detected via tags or notes convention (1Password doesn't have native field)
- [Phase 06-02]: Exclude sshjesus_config entries from conflict detection (auto-detect by SourceFile path)
- [Phase 06-02]: 1Password always wins conflicts per requirement (Winner="onepassword")
- [Phase 06-03]: Separate sync from list operations (ListServers returns cached data, SyncFromOnePassword handles fetching)
- [Phase 06-03]: BackendStatus enum with 4 states (Unknown/Available/Locked/Unavailable) for precise status reporting
- [Phase 06-03]: TOML cache with ServerCache wrapper type (domain model remains storage-agnostic)
- [Phase 06-03]: 5-second default poll interval, configurable via SSHJESUS_1PASSWORD_POLL_INTERVAL env var
- [Phase 06-03]: 10-second write debounce window to prevent sync loops after Create/Update/Delete operations
- [Phase 06-03]: Graceful poller shutdown in Close() (unlock mutex before Stop() to avoid deadlock)
- [Phase 06-04]: Shared BackendStatus at backend level (avoids import cycles when TUI references status)
- [Phase 06-04]: Multi-backend priority order (later backends win conflicts for deduplication)
- [Phase 06-04]: Case-insensitive DisplayName deduplication (SSH config is case-sensitive, but user-facing dedup should be case-insensitive)
- [Phase 06-04]: Writer delegation to first Writer-capable backend (simple rule, works for current use case)
- [Phase 06-04]: Status bar only shown when not Available (clean UI when everything works, banner only for issues)
- [Phase 06-04]: TUI New() signature change adds opStatus parameter (explicit dependency injection)

### Pending Todos

None yet.

### Blockers/Concerns

**Phase 6 considerations:**
- ~~1Password SDK error scenarios need discovery during implementation (network failures, corrupted vaults)~~ — RESOLVED: MockClient supports error injection, backend skips error vaults
- 1Password service account authentication and token management — IN PROGRESS: NewServiceAccountClient implemented, config loader needed (06-02)
- Vault/item browsing performance with large vaults — NOTE: ListItems fetches full items (not just overviews), may need optimization for large vaults

**Cross-platform considerations:**
- Terminal compatibility matrix needs empirical testing (older terminals, SSH-forwarded, screen/tmux combinations)

## Session Continuity

Last session: 2026-02-14 (plan execution)
Stopped at: Completed 06-04-PLAN.md — Multi-backend aggregator with priority-based deduplication (later backends win), shared BackendStatus enum at backend level (avoids import cycles), TUI status bar for 1Password availability indication (only shown when locked/unavailable), complete end-to-end wiring from backend to TUI. Config supports "onepassword" and "both" backend modes. 11 new multi-backend tests, all pass. Full project builds and passes vet.
Resume file: None
