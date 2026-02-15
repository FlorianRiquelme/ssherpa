# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-14)

**Core value:** Find and connect to the right SSH server instantly, from any repo, without remembering aliases or grepping config files.
**Current focus:** Phase 7 complete — SSH Key Selection fully implemented. Ready for Phase 8 (Distribution).

## Current Position

Phase: 8 of 8 (Distribution) — IN PROGRESS
Plan: 2 of 3 complete
Status: Phase 8 Plans 01-02 Complete — CLI flags, onboarding, and release automation configured
Last activity: 2026-02-15 — Completed Phase 08 Plan 01: Version package with ldflags injection, CLI flags (--version, --setup), and first-run onboarding flow with SSH config detection and optional 1Password setup.

Progress: [████████████████████████████████████████████████████████████████████████████████████████████████] 98%

## Performance Metrics

**Velocity:**
- Total plans completed: 19 (plus 06-05 partial: 2/3 tasks)
- Average duration: 597.4 seconds
- Total execution time: 3.18 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01    | 2     | 429s  | 214.5s   |
| 02    | 2     | 319s  | 159.5s   |
| 03    | 2     | 2823s | 1411.5s  |
| 04    | 3     | 4118s | 1372.7s  |
| 05    | 3     | 525s  | 175.0s   |
| 06    | 4     | 1538s | 384.5s   |
| 07    | 2     | 1554s | 777.0s   |
| 08    | 2     | 320s  | 160.0s   |

**Recent Plans:**

| Phase | Plan | Duration | Tasks | Files |
|-------|------|----------|-------|-------|
| 08    | 01   | 182s     | 2     | 3     |
| 08    | 02   | 138s     | 2     | 3     |
| 07    | 02   | 900s     | 3     | 9     |
| 07    | 01   | 654s     | 1     | 8     |
| 06    | 05   | 252s     | 2/3   | 4     |

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
- [Phase 06-01]: Tag-based discovery with case-insensitive "ssherpa" tag filtering across all vaults
- [Phase 06-01]: Skip vaults with errors (permission issues don't fail entire list operation)
- [Phase 06-01]: Projects as tags not entities (1Password doesn't have standalone project concept)
- [Phase 06-01]: Lowercase "server" category matches 1Password API expectations
- [Phase 06-02]: Prepend Include directive to ~/.ssh/config (not append) for first-match-wins SSH semantics
- [Phase 06-02]: ForwardAgent detected via tags or notes convention (1Password doesn't have native field)
- [Phase 06-02]: Exclude ssherpa_config entries from conflict detection (auto-detect by SourceFile path)
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
- [Phase 07-01]: Header sniffing over filename conventions (ParseKeyFile checks PEM/OpenSSH headers, not filenames)
- [Phase 07-01]: Agent wins deduplication over file (same fingerprint: agent version kept for richer metadata)
- [Phase 07-01]: File wins deduplication over 1Password (local filesystem is authoritative source)
- [Phase 07-01]: Graceful agent unavailability (DiscoverAgentKeys returns empty slice when agent down, not error)
- [Phase 07-01]: Encrypted key handling via .pub fallback (detect passphrase error, read .pub for metadata)
- [Phase 07-01]: Missing .pub file graceful (keys without .pub valid, comment field empty)
- [Phase 07-02]: 1Password IdentityAgent discovery from SSH config (parse ~/.ssh/config for IdentityAgent directives, detect 1Password by path)
- [Phase 07-02]: Context-aware default label ("Default (1Password agent)" when IdentityAgent configured)
- [Phase 07-02]: Checkmark only for non-empty paths (agent keys with empty Path don't false-match)
- [Phase 07-02]: Always parse SSH config for IdentityAgent (independent of backend mode)
- [Phase 07-02]: Re-discover keys after hosts load (initial nil, re-trigger with full host context)
- [Phase 08-01]: ldflags injection for version info (standard Go practice for embedding build metadata)
- [Phase 08-01]: OnboardingDone field in config (simple boolean flag prevents re-showing onboarding)
- [Phase 08-01]: SSH config host counting via kevinburke/ssh_config (reuses existing dependency, reliable parsing)
- [Phase 08-02]: GoReleaser v2 for release automation (multi-platform builds, Homebrew tap, checksums)
- [Phase 08-02]: Auto-push Homebrew formula to separate tap repository (TAP_GITHUB_TOKEN enables cross-repo push)
- [Phase 08-02]: CGO_ENABLED=0 for all builds (pure Go binaries with no C dependencies)
- [Phase 08-02]: Curl install script verifies SHA256 checksum (security best practice for download integrity)

### Pending Todos

- ~~**Wizard SDK wiring (06-05):**~~ DONE (commit 745339d). Uses service account token via OP_SERVICE_ACCOUNT_TOKEN env var or manual paste.
- **Re-run E2E checkpoint (06-05 Task 3):** Re-test all 6 scenarios from the checkpoint with real 1Password service account.
- **Rename tool to ssherpa:** Rename from "ssherpa" to "ssherpa" across codebase, Go module, configs, docs, and Phase 8 distribution plan.
- ~~**Fix selected server highlight readability:**~~ DONE (commit dd4eea6). AdaptiveColor for selectedStyle/pickerSelectedStyle — dark indigo bg on dark terminals, light indigo bg on light terminals.
- **Fix 1Password status banner showing unavailable when unlocked:** Banner says 1Password unavailable even when desktop app is unlocked — likely `op` CLI not signed in vs service account token issue.

### Blockers/Concerns

**Phase 6 considerations:**
- ~~1Password SDK error scenarios need discovery during implementation (network failures, corrupted vaults)~~ — RESOLVED: MockClient supports error injection, backend skips error vaults
- ~~1Password service account authentication and token management~~ — RESOLVED: Service account token via env var, wizard auto-detects or prompts for manual paste
- Vault/item browsing performance with large vaults — NOTE: ListItems fetches full items (not just overviews), may need optimization for large vaults
- **Architecture decision:** Switched from desktop app integration (beta, unreliable) to service account tokens (stable SDK). Desktop app integration (`WithDesktopAppIntegration`) requires beta SDK and failed to connect. Service accounts work on all plan types including Family.

**Cross-platform considerations:**
- Terminal compatibility matrix needs empirical testing (older terminals, SSH-forwarded, screen/tmux combinations)

## Session Continuity

Last session: 2026-02-15 (Phase 08 in progress)
Stopped at: Completed 08-01-PLAN.md — CLI flags (--version, --setup) and first-run onboarding flow with SSH config detection. Plans 01-02 complete. Plan 03 remaining (Documentation).
Resume file: None
Resume command: `/gsd:execute-phase 08` for Plan 03
