---
phase: 02-ssh-config-integration
plan: 02
subsystem: tui
tags: [tui, bubbletea, ssh-config, user-interface]
dependency_graph:
  requires:
    - internal/sshconfig/parser.go (SSH config parsing)
    - internal/sshconfig/backend.go (backend adapter)
    - internal/config/config.go (config loading)
  provides:
    - internal/tui/model.go (Bubbletea state machine)
    - internal/tui/list_view.go (two-line list items)
    - internal/tui/detail_view.go (full SSH config detail view)
    - internal/tui/styles.go (adaptive color scheme)
    - cmd/ssherpa/main.go (TUI entry point)
  affects: []
tech_stack:
  added:
    - github.com/charmbracelet/bubbletea@v2.0.0-alpha.2
    - github.com/charmbracelet/bubbles@v0.20.0
    - github.com/charmbracelet/lipgloss@v1.0.0
  patterns:
    - "Bubbletea Elm architecture (Model, Update, View)"
    - "ViewMode state machine (list/detail views)"
    - "Async config loading with custom messages"
    - "AdaptiveColor for light/dark terminal support"
    - "Two-line list items with custom delegate"
    - "Full-screen detail view with viewport scrolling"
key_files:
  created:
    - internal/tui/model.go (268 lines)
    - internal/tui/list_view.go (75 lines)
    - internal/tui/detail_view.go (110 lines)
    - internal/tui/styles.go (91 lines)
    - internal/tui/messages.go (14 lines)
  modified:
    - cmd/ssherpa/main.go (52 lines - TUI wiring)
    - go.mod (added Bubbletea dependencies)
    - go.sum (dependency checksums)
decisions:
  - what: "Full-screen detail view instead of split-panel"
    why: "Simpler to implement, works at any terminal width, clearer focus"
    context: "Task 1 implementation"
  - what: "Wildcard entries in separate section at bottom"
    why: "Matches plan requirement and improves UX by separating structural config from connection targets"
    context: "Task 1 list organization"
  - what: "Alt screen for TUI rendering"
    why: "Prevents TUI output from polluting terminal history on exit"
    context: "Task 2 main.go wiring"
metrics:
  duration: 26s
  tasks_completed: 3
  files_created: 5
  files_modified: 3
  test_count: 0
  coverage: N/A
  completed_date: 2026-02-14
---

# Phase 02 Plan 02: SSH Config TUI Summary

**One-liner:** Working Bubbletea TUI displaying parsed SSH connections in navigable two-line list with full-screen detail view, adaptive colors, and async config loading.

## What Was Built

Created a complete TUI application that reads `~/.ssh/config` and displays all SSH connections in a navigable interface. This is the first user-facing screen of ssherpa.

**Key Components:**

1. **TUI Model** (`internal/tui/model.go` - 268 lines):
   - Bubbletea Elm architecture with ViewMode state machine (ViewList/ViewDetail)
   - Async config loading via `loadConfigCmd` and `configLoadedMsg`
   - Loading spinner during config parse
   - List component for connection navigation
   - Viewport component for scrollable detail view
   - Window size tracking and responsive layout

2. **List View** (`internal/tui/list_view.go` - 75 lines):
   - `hostItem` implementing Bubbles `list.Item` interface
   - Two-line layout: name+hostname on first line, user+port on second
   - Warning indicators for malformed entries (! prefix with warningStyle)
   - `separatorItem` for "--- Wildcard Entries ---" section divider

3. **Detail View** (`internal/tui/detail_view.go` - 110 lines):
   - Full-screen detail view replacing list on Enter
   - Shows all SSH config options from AllOptions map
   - Source file tracking ("Defined in: {SourceFile}:{SourceLine}")
   - Viewport scrolling for configs exceeding terminal height
   - Standard fields section (Hostname, User, Port, IdentityFile)
   - All Options section (sorted alphabetically)
   - Parse error display for malformed entries

4. **Adaptive Styles** (`internal/tui/styles.go` - 91 lines):
   - AdaptiveColor palette supporting light AND dark terminals
   - Accent color (indigo): hostnames and structural elements
   - Secondary color (slate): user/port info
   - Warning color (amber): malformed entry indicators
   - Border color (slate): separators and panels
   - 11 reusable Lipgloss styles for consistent theming

5. **Main Entry Point** (`cmd/ssherpa/main.go` - 52 lines):
   - Config loading with fallback to sshconfig backend
   - SSH config path resolution (~/.ssh/config)
   - Alt screen TUI launch (no terminal history pollution)
   - Error handling for missing home directory

## Implementation Highlights

**TUI Architecture:**
- State machine: ViewList (browsing connections) ↔ ViewDetail (inspecting config)
- Async loading: Init() fires loadConfigCmd → ParseSSHConfig runs in goroutine → configLoadedMsg updates model
- Navigation: arrow keys (list), Enter (detail), Esc (back to list), q (quit)
- Responsive: handles WindowSizeMsg for terminal resize

**List Features:**
- Two-line item layout per plan requirement
- Wildcard hosts (Host * or Host *.example.com) separated into bottom section
- Alphabetical sorting within each section
- Warning indicators for entries with ParseError
- Custom delegate with ShowDescription=true for two-line rendering

**Detail View Features:**
- Shows ALL SSH config options (AllOptions map iteration)
- Source tracking shows which config file defined the host
- Alphabetically sorted options for consistency
- Viewport scrolling for long configs
- Parse error display at top with warning styling

**Empty State:**
- Friendly guidance when ~/.ssh/config is missing or empty
- Shows example SSH config syntax
- Preserves quit key hint

**Color Scheme:**
- AdaptiveColor ensures readability in both light and dark terminals
- Avoids research anti-pattern of hardcoded colors
- Indigo accent for branding (hostnames, headers)
- Slate secondary for metadata (user, port)
- Amber warnings for parse errors

## Deviations from Plan

None. Plan executed exactly as written.

## Verification Results

All success criteria met (verified by human at checkpoint):

- ✅ `go run ./cmd/ssherpa/` launches TUI showing SSH connections from ~/.ssh/config
- ✅ List shows two-line entries: name+hostname / user+port
- ✅ Wildcard entries appear in separate section at bottom
- ✅ Arrow keys navigate, Enter opens detail view, Esc returns to list
- ✅ Detail view shows ALL SSH config options and source file location
- ✅ Missing/empty config shows friendly empty state with guidance
- ✅ Loading spinner displays during config parse
- ✅ Accent colors distinguish structural elements (hostname, user, port)
- ✅ Malformed entries show warning indicator (! prefix)
- ✅ `go build ./...` compiles cleanly

**Human verification (checkpoint approved):**
User confirmed TUI displays correctly, navigation works, colors render properly, and empty state is helpful.

## Files Changed

**Created:**
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/tui/model.go`
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/tui/list_view.go`
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/tui/detail_view.go`
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/tui/styles.go`
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/tui/messages.go`

**Modified:**
- `/Users/florianriquelme/Repos/mine/ssherpa/cmd/ssherpa/main.go`
- `/Users/florianriquelme/Repos/mine/ssherpa/go.mod`
- `/Users/florianriquelme/Repos/mine/ssherpa/go.sum`

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | TUI model, views, styles, and messages | e7f9ab3 | model.go, list_view.go, detail_view.go, styles.go, messages.go, go.mod, go.sum |
| 2 | Wire main.go to launch TUI | 6065dc2 | main.go |
| 3 | Human verification checkpoint | APPROVED | (verification only) |

## What's Next

**Phase 2 Complete:** This plan completes Phase 2 (SSH Config Integration). Users can now run `ssherpa` and see all their SSH connections from `~/.ssh/config` in a navigable TUI.

**Ready for Phase 3:** TUI foundation is ready for connection execution (Phase 3). The list view will need minimal changes to support Enter-to-connect (currently Enter opens detail, will need key remapping to d=detail, Enter=connect).

**Integration notes:**
- Model.hosts stores parsed SSHHost structs for detail view access
- ViewMode state machine makes adding new views straightforward
- Styles.go provides consistent color palette for future UI elements
- Empty state pattern can be reused for other empty views (no projects, no connections in current project, etc.)

## Self-Check: PASSED

**Files created:**
- ✅ FOUND: /Users/florianriquelme/Repos/mine/ssherpa/internal/tui/model.go
- ✅ FOUND: /Users/florianriquelme/Repos/mine/ssherpa/internal/tui/list_view.go
- ✅ FOUND: /Users/florianriquelme/Repos/mine/ssherpa/internal/tui/detail_view.go
- ✅ FOUND: /Users/florianriquelme/Repos/mine/ssherpa/internal/tui/styles.go
- ✅ FOUND: /Users/florianriquelme/Repos/mine/ssherpa/internal/tui/messages.go
- ✅ FOUND: /Users/florianriquelme/Repos/mine/ssherpa/cmd/ssherpa/main.go (modified)

**Commits exist:**
- ✅ FOUND: e7f9ab3 (Task 1 - TUI model, views, styles, messages)
- ✅ FOUND: 6065dc2 (Task 2 - Wire main.go to launch TUI)

**Build verification:**
- ✅ go build ./... — compiles cleanly
- ✅ go vet ./... — no warnings
- ✅ go run ./cmd/ssherpa/ — launches TUI successfully
