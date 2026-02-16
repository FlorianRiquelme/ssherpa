---
phase: quick-1
plan: 01
subsystem: tui
tags: [help, documentation, 1password, reference]
dependency_graph:
  requires: []
  provides: [field-reference-cli, help-overlay]
  affects: [tui, cli]
tech_stack:
  added: []
  patterns: [overlay-ui, scrollable-viewport]
key_files:
  created:
    - internal/tui/help_overlay.go
  modified:
    - cmd/ssherpa/main.go
    - internal/tui/keys.go
    - internal/tui/model.go
    - internal/tui/styles.go
decisions: []
metrics:
  duration: 214s
  completed: 2026-02-16
---

# Quick 1 Plan 01: 1Password Field Reference Help Summary

**One-liner:** Added --fields CLI flag and ? help overlay showing complete 1Password field reference for user guidance

## Tasks Completed

| Task | Description | Commit | Files |
|------|-------------|--------|-------|
| 1 | Add --fields CLI flag and help overlay view | 9b13b20 | help_overlay.go, main.go, styles.go |
| 2 | Wire ? key to toggle help overlay in TUI | f45aec7 | keys.go, model.go |

## Implementation Details

### Task 1: CLI Flag and Help Content

Created `internal/tui/help_overlay.go` with:
- `RenderFieldReference()`: Styled version for TUI display with lipgloss formatting
- `RenderFieldReferencePlain()`: Plain-text version for CLI output (no ANSI codes)
- `HelpOverlay`: Scrollable viewport-based overlay component
- Complete 1Password field mapping documentation including:
  - Item properties (Title, Category, Tag)
  - 9 field definitions (hostname, user, port, identity_file, proxy_jump, project_tags, remote_project_path, forward_agent, extra_config)
  - Required vs optional indicators
  - Default values
  - Minimal example entry
  - Case-insensitivity note

Added `--fields` flag to `cmd/ssherpa/main.go`:
- Prints plain-text field reference to stdout
- Exits immediately (no TUI launch)
- Positioned after `--version` check, before backend initialization

Added styles to `internal/tui/styles.go`:
- `helpOverlayStyle`: Bordered overlay container (72 chars wide)
- `helpFooterStyle`: Footer text styling for navigation hints

### Task 2: TUI Integration

Extended `KeyMap` in `internal/tui/keys.go`:
- Added `Help` binding mapped to `?` key
- Included in `ShortHelp()` footer display
- Added to `FullHelp()` alongside Undo and SignIn

Updated `Model` in `internal/tui/model.go`:
- Added `showingHelp bool` and `helpOverlay *HelpOverlay` fields
- Key handling in `ViewList` mode:
  - Pressing `?` toggles help overlay (creates new overlay or closes existing)
  - When help showing: `?` or `Esc` closes overlay
  - Arrow keys route to help viewport for scrolling
- Window resize updates help overlay dimensions
- Overlay rendering: centered via `lipgloss.Place()`, rendered on top of base view

## Field Reference Content

The help overlay documents all 1Password fields recognized by ssherpa:

**Required fields:**
- `hostname`: Server hostname or IP address
- `user`: SSH username

**Optional fields:**
- `port` (default: 22): SSH port number
- `identity_file` (default: SSH default): Path to SSH private key file
- `proxy_jump`: Bastion/jump host for ProxyJump
- `project_tags`: Comma-separated project tags (e.g. "web,api")
- `remote_project_path`: Remote path to cd into on connect
- `forward_agent`: Enable SSH agent forwarding (noted, not yet mapped)
- `extra_config`: Additional SSH config directives (noted, not yet mapped)

**Item properties:**
- Title: Display name for the server (becomes the alias)
- Category: Must be "Server"
- Tag: Must include "ssherpa" (case-insensitive)

## User Experience

**CLI workflow:**
```bash
$ ssherpa --fields
1Password Field Reference
=========================

Item Properties:
  • Title: Display name for the server (becomes the alias)
  • Category: Must be "Server"
  • Tag: Must include "ssherpa" (case-insensitive)

Fields:
  Field                 Required  Default       Description
  ─────────────────────────────────────────────────────────────────────────
  hostname              yes       -             Server hostname or IP address
  user                  yes       -             SSH username
  [...]
```

**TUI workflow:**
1. Launch ssherpa normally
2. Press `?` in list view
3. Scrollable help overlay appears with complete field reference
4. Press `?` or `Esc` to dismiss and return to server list
5. Help key appears in footer: `? help`

## Testing

- `go build -o /dev/null ./cmd/ssherpa/` ✅ Compiles successfully
- `go vet ./...` ✅ No issues
- `go test ./...` ✅ All tests pass
- `ssherpa --fields` ✅ Prints complete field reference and exits

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check: PASSED

**Created files exist:**
```
FOUND: internal/tui/help_overlay.go
```

**Modified files exist:**
```
FOUND: cmd/ssherpa/main.go
FOUND: internal/tui/keys.go
FOUND: internal/tui/model.go
FOUND: internal/tui/styles.go
```

**Commits exist:**
```
FOUND: 9b13b20
FOUND: f45aec7
```
