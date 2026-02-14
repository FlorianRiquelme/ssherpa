---
phase: 04-project-detection
plan: 02
subsystem: tui
tags: [bubbletea, project-badges, grouping, search-prioritization, lipgloss]
dependency_graph:
  requires: [04-01]
  provides: [project-aware-tui, colored-badges, project-grouping, prioritized-search]
  affects: [tui, user-experience]
tech_stack:
  added: []
  patterns:
    - Inline colored badges using lipgloss AdaptiveColor
    - Project-based server grouping with current-project-first ordering
    - Search result prioritization with separator between current and other projects
    - Host-to-project mapping via ProjectConfig.ServerNames
key_files:
  created:
    - internal/tui/badges.go
  modified:
    - internal/tui/model.go
    - internal/tui/list_view.go
    - internal/tui/messages.go
    - internal/tui/styles.go
    - cmd/sshjesus/main.go
decisions:
  - decision: "Inline badges (not section headers or collapsible sections)"
    rationale: "Simpler implementation, clearer visual hierarchy, works at any terminal width"
    alternatives: "Section headers with collapsible groups, split-panel view"
  - decision: "Project separator in search results between current and other projects"
    rationale: "Visual clarity for prioritization, matches user mental model"
    alternatives: "Color-coding only, no separator"
  - decision: "Unassigned servers at bottom (after project groups)"
    rationale: "Encourages project assignment, keeps project-related servers prominent"
    alternatives: "Unassigned at top, mixed with projects"
metrics:
  duration_seconds: 1905
  tasks_completed: 2
  files_created: 1
  files_modified: 5
  completed_at: "2026-02-14T11:10:45Z"
---

# Phase 04 Plan 02: Project-Aware TUI Summary

**One-liner:** Project-aware TUI with inline colored badges, current-project-first grouping, and prioritized fuzzy search separating current project matches from other results.

## Objective Achieved

Transformed the flat server list into a project-organized view with visual badges, intelligent grouping, and search prioritization. When launched from a git repository, servers from the current project automatically float to the top. This is the core UX differentiator of sshjesus.

## Tasks Completed

### Task 1: Project badges, grouped list, and project-aware search
- **Commit:** 494ac77
- **Duration:** ~30 minutes
- **Files:**
  - internal/tui/badges.go (created)
  - internal/tui/model.go
  - internal/tui/list_view.go
  - internal/tui/messages.go
  - internal/tui/styles.go
  - cmd/sshjesus/main.go

**Implementation:**

**Badge rendering (`internal/tui/badges.go`):**
- Created `RenderProjectBadge()` function for GitHub-label-style badges
- White bold text on colored background
- Uses lipgloss AdaptiveColor from project colors
- Padding(0, 1) for inline display

**Main entry point (`cmd/sshjesus/main.go`):**
- Calls `project.DetectCurrentProject()` at startup
- Loads `config.Projects` from TOML config
- Passes both `currentProjectID` and `projects` to `tui.New()`
- Handles non-git contexts gracefully (empty currentProjectID)

**TUI Model (`internal/tui/model.go`):**
- Updated `New()` signature to accept currentProjectID and projects
- Added fields: `currentProjectID`, `projects`, `projectMap`
- Built `projectMap` for fast host-to-project lookups
- Completely rewrote `rebuildListItems()` for project-aware grouping:
  - Build host-to-project mapping from ProjectConfig.ServerNames
  - Separate servers into: currentProject, otherProjects, unassigned
  - Sort: current project first, other projects alphabetical, unassigned last
  - Add project separator between groups when needed
  - Preserve wildcard handling at bottom
- Updated `filterHosts()` for prioritized search:
  - Split matches into current project vs others
  - Sort each group by fuzzy score
  - Insert separator between current and other results
  - Handle non-git context (no separation, just score-based)

**List view (`internal/tui/list_view.go`):**
- Modified `hostItem` struct to include `projectBadges []badgeData`
- Updated `Title()` to render badges inline after hostname
- Format: `[star] Name (hostname) [badge1] [badge2]`
- Preserves warning indicators and existing formatting

**Messages (`internal/tui/messages.go`):**
- Added `projectSeparatorItem` type for visual separation
- Implements `list.Item` with empty `FilterValue()`
- Label: "──── Other Projects ────"

**Styles (`internal/tui/styles.go`):**
- Added `badgeStyle` (white text, colored background, bold)
- Added `projectSeparatorStyle` (secondary color, italic)

**Preserved functionality:**
- Fuzzy search filters in real-time
- Enter connects to SSH
- Tab/i shows detail view
- Vim navigation (j/k)
- History indicators (star)
- Wildcard entries at bottom

### Task 2: Verify project-aware TUI display and interaction
- **Status:** APPROVED by user
- **Verification:** Human checkpoint completed

**What was verified:**
1. ✅ Without projects configured - TUI identical to Phase 3 behavior
2. ✅ With projects configured - colored badges display inline
3. ✅ Servers grouped by project with current project first
4. ✅ Git detection works - current project's servers float to top
5. ✅ Search prioritization - current project matches above separator
6. ✅ Badge colors are visibly different per project
7. ✅ All Phase 3 features preserved (search, connect, navigation, history)

## Deviations from Plan

None - plan executed exactly as written.

## Success Criteria Met

- ✅ Every server row shows inline colored project badge(s) when assigned to projects
- ✅ Server list grouped by project name (current project first, unassigned last)
- ✅ Fuzzy search floats current project matches above separator
- ✅ All Phase 3 features preserved (search, connect, navigation, history, wildcards)
- ✅ Graceful degradation when no projects configured (identical to Phase 3 behavior)
- ✅ Human verified visual appearance and interaction

## Technical Implementation Details

### Badge Rendering
- Uses `lipgloss.NewStyle().Background(color).Foreground("#FFFFFF").Bold(true).Padding(0, 1)`
- Colors from `project.ProjectColor()` (deterministic FNV-1a based)
- Inline rendering after hostname: `server-name (hostname) [badge]`

### Project Grouping Algorithm
1. Build host-to-project map from `ProjectConfig.ServerNames`
2. Separate hosts into buckets:
   - Current project servers (if currentProjectID matches)
   - Other project servers (grouped by project name)
   - Unassigned servers (not in any project)
3. Sort within each bucket by hostname
4. Concatenate: current → separator → others → unassigned → wildcards

### Search Prioritization
1. Run fuzzy match on all hosts
2. Split results by project membership:
   - Current project matches
   - Other project matches
3. Sort each group by fuzzy score (descending)
4. Insert `projectSeparatorItem` between groups
5. Special case: no currentProjectID → single list sorted by score

### Graceful Degradation
- No projects configured → behaves exactly like Phase 3
- No git repo (empty currentProjectID) → groups by project, no "current"
- Missing ProjectConfig fields → safe defaults, no crashes

## Files Created/Modified

**Created:**
- `internal/tui/badges.go` - Badge rendering with colored labels

**Modified:**
- `internal/tui/model.go` - Project-aware grouping and search (+304 lines, core logic overhaul)
- `internal/tui/list_view.go` - Project badge display in Title()
- `internal/tui/messages.go` - Project separator item
- `internal/tui/styles.go` - Badge and separator styles
- `cmd/sshjesus/main.go` - Project detection at startup

## Decisions Made

Per plan specifications:

1. **Inline badges over section headers**: Simpler implementation, works at any terminal width, clearer visual hierarchy. Badges appear next to server name rather than as collapsible sections.

2. **Project separator in search results**: Visual clarity between current project matches and other matches. Uses `projectSeparatorItem` with label "──── Other Projects ────".

3. **Unassigned servers at bottom**: Encourages project assignment, keeps project-related servers prominent in the list.

## Issues Encountered

None - implementation proceeded smoothly with all tests passing and visual verification approved.

## Next Phase Readiness

Phase 4 (Project Detection) is now complete. The TUI successfully:
- Detects current git project context
- Displays project assignments visually with colored badges
- Groups and prioritizes servers based on project membership
- Provides intelligent search that surfaces relevant servers first

Ready for:
- **Phase 5:** 1Password integration (backend implementation)
- **Phase 6:** Server management commands (add, edit, delete)
- **Phase 7:** Setup wizard for first-run experience

The project detection foundation (Phase 4) provides the context-awareness that will enhance all future features.

## Self-Check: PASSED

✅ File verification:
- internal/tui/badges.go exists
- internal/tui/model.go modified (304 lines added)
- internal/tui/list_view.go modified
- internal/tui/messages.go modified
- internal/tui/styles.go modified
- cmd/sshjesus/main.go modified

✅ Commit verification:
- 494ac77 exists (Task 1 implementation)

✅ Functionality verification:
- Build succeeds: `go build ./...`
- No vet warnings: `go vet ./...`
- Human verification APPROVED

---
*Phase: 04-project-detection*
*Completed: 2026-02-14*
