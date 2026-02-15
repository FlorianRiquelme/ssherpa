# Hide Wildcard Connections Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove wildcard SSH config entries from TUI display while keeping parser detection logic intact

**Architecture:** UI-only change - remove wildcard rendering code from two list builder functions in model.go and clean up unused separator type in list_view.go. No parser or backend changes.

**Tech Stack:** Go, Bubbletea TUI framework

**Design Reference:** `docs/plans/2026-02-15-hide-wildcard-connections-design.md`

---

## Task 1: Remove Wildcard Display from rebuildListItemsGrouped()

**Files:**
- Modify: `internal/tui/model.go:640-650`

**Context:**
The `rebuildListItemsGrouped()` function builds the list when projects are configured and not in search mode. Lines 640-650 add a separator and wildcard hosts to the bottom of the list.

**Step 1: Locate the wildcard display code**

Navigate to `internal/tui/model.go` and find the `rebuildListItemsGrouped()` function around line 640.

Expected code:
```go
// 4. Wildcards at bottom
if len(wildcards) > 0 {
    items = append(items, separatorItem{})
    for _, host := range wildcards {
        _, isRecent := m.recentHosts[host.Name]
        items = append(items, hostItem{
            host:          host,
            lastConnected: isRecent,
            projectBadges: nil,
        })
    }
}
```

**Step 2: Remove the wildcard display block**

Delete lines 640-650 (the entire `if len(wildcards) > 0` block).

After removal, the function should end with:
```go
// 3. Unassigned hosts
for _, hwp := range unassignedHosts {
    items = append(items, m.createHostItem(hwp.host, hwp.projects, hostProjectMap))
}

m.list.SetItems(items)
m.preselectLastConnectedHost(items)
```

**Step 3: Verify the change**

Run: `git diff internal/tui/model.go`

Expected: Shows deletion of ~11 lines containing the wildcard display logic

**Step 4: Build to verify no syntax errors**

Run: `go build ./cmd/ssherpa`

Expected: Clean build with no errors

**Step 5: Commit**

```bash
git add internal/tui/model.go
git commit -m "refactor(tui): remove wildcard display from grouped list view

Remove wildcard hosts and separator from rebuildListItemsGrouped().
Wildcards are still parsed but no longer rendered in the TUI when
projects are configured."
```

---

## Task 2: Remove Wildcard Display from rebuildListItemsSimple()

**Files:**
- Modify: `internal/tui/model.go:704-717`

**Context:**
The `rebuildListItemsSimple()` function builds the list when in search mode or no projects configured. Lines 704-717 add a separator and wildcard hosts.

**Step 1: Locate the wildcard display code**

In `internal/tui/model.go`, find the `rebuildListItemsSimple()` function around line 704.

Expected code:
```go
// Add separator if there are wildcards
if len(wildcards) > 0 {
    items = append(items, separatorItem{})

    // Add wildcard hosts
    for _, host := range wildcards {
        _, isRecent := m.recentHosts[host.Name]
        items = append(items, hostItem{
            host:          host,
            lastConnected: isRecent,
            projectBadges: nil,
        })
    }
}
```

**Step 2: Remove the wildcard display block**

Delete lines 704-717 (the entire `if len(wildcards) > 0` block).

After removal, the function should end with:
```go
} else {
    // No search or no current project: just add all regular hosts
    for _, host := range regular {
        items = append(items, m.createHostItem(host, hostProjectMap[host.Name], hostProjectMap))
    }
}

m.list.SetItems(items)
m.preselectLastConnectedHost(items)
```

**Step 3: Verify the change**

Run: `git diff internal/tui/model.go`

Expected: Shows deletion of ~14 additional lines

**Step 4: Build to verify no syntax errors**

Run: `go build ./cmd/ssherpa`

Expected: Clean build with no errors

**Step 5: Commit**

```bash
git add internal/tui/model.go
git commit -m "refactor(tui): remove wildcard display from simple list view

Remove wildcard hosts and separator from rebuildListItemsSimple().
Wildcards no longer appear in search mode or when no projects configured."
```

---

## Task 3: Remove Unused separatorItem Type

**Files:**
- Modify: `internal/tui/list_view.go:78-95`

**Context:**
The `separatorItem` type was used to display "--- Wildcard Entries ---" separator. With wildcard display removed, this type is now unused.

**Step 1: Verify separatorItem is unused**

Run: `grep -n "separatorItem" internal/tui/*.go`

Expected: Only shows the type definition in list_view.go (no other references)

**Step 2: Remove the separatorItem type**

In `internal/tui/list_view.go`, delete lines 78-95:

```go
// separatorItem is a non-interactive list item that displays a separator.
// Used to separate wildcard entries from regular hosts.
type separatorItem struct{}

// FilterValue returns empty string (excluded from search).
func (s separatorItem) FilterValue() string {
    return ""
}

// Title returns the separator text.
func (s separatorItem) Title() string {
    return separatorStyle.Render("--- Wildcard Entries ---")
}

// Description returns empty string (no second line for separator).
func (s separatorItem) Description() string {
    return ""
}
```

**Step 3: Verify no references remain**

Run: `grep -r "separatorItem" internal/tui/`

Expected: No results (type completely removed)

**Step 4: Build to verify no compilation errors**

Run: `go build ./cmd/ssherpa`

Expected: Clean build with no errors

**Step 5: Commit**

```bash
git add internal/tui/list_view.go
git commit -m "refactor(tui): remove unused separatorItem type

Remove separatorItem type that was used for wildcard section separator.
No longer needed after removing wildcard display from list views."
```

---

## Task 4: Manual Testing

**Files:**
- No file changes

**Context:**
Verify the changes work correctly across different scenarios.

**Step 1: Create test SSH config with wildcards**

Create or modify `~/.ssh/config` to include:
```
Host test-regular
    HostName regular.example.com
    User testuser

Host *
    User defaultuser

Host *.wildcard.com
    User wildcarduser

Host dev-*
    Port 2222
```

**Step 2: Test with wildcards present**

Run: `./ssherpa`

**Verify:**
- ✅ No wildcard entries shown in the list
- ✅ No "--- Wildcard Entries ---" separator displayed
- ✅ Regular host "test-regular" displays correctly
- ✅ List navigation works normally

**Step 3: Test search/filter mode**

In the TUI, press `/` to enter search mode and type "wildcard"

**Verify:**
- ✅ No wildcard entries appear in filtered results
- ✅ Search for "test-regular" still works

**Step 4: Test with project grouping (if configured)**

If projects are configured in your setup:

**Verify:**
- ✅ Project groups display correctly
- ✅ No wildcard section at bottom

**Step 5: Test without wildcards**

Remove wildcard entries from `~/.ssh/config`, leaving only regular hosts.

Run: `./ssherpa`

**Verify:**
- ✅ List displays normally
- ✅ No visual changes or regressions

**Step 6: Document test results**

If all tests pass, proceed to final commit. If any issues found, debug and fix before proceeding.

---

## Task 5: Run Existing Tests and Final Commit

**Files:**
- No file changes

**Step 1: Run Go tests**

Run: `go test ./internal/tui/... -v`

Expected: All tests pass (no TUI tests should be affected by display-only changes)

**Step 2: Run parser tests to verify wildcard detection still works**

Run: `go test ./internal/sshconfig/... -v`

Expected: All tests pass, including wildcard detection tests

**Step 3: Verify build**

Run: `go build ./cmd/ssherpa`

Expected: Clean build

**Step 4: Check git status**

Run: `git status`

Expected: Clean working tree (all changes committed)

**Step 5: Review commit history**

Run: `git log --oneline -5`

Expected: Shows 3 new commits:
1. "refactor(tui): remove wildcard display from grouped list view"
2. "refactor(tui): remove wildcard display from simple list view"
3. "refactor(tui): remove unused separatorItem type"

---

## Success Criteria

All criteria from design document:

- ✅ No wildcard entries visible in TUI
- ✅ No "--- Wildcard Entries ---" separator displayed
- ✅ Regular hosts display unchanged
- ✅ Search/filter works correctly without wildcards
- ✅ Project grouping unaffected
- ✅ No regressions in existing functionality
- ✅ All existing tests still pass
- ✅ Clean builds with no errors

---

## Notes

**Wildcard Detection Preserved:**
The parser's wildcard detection logic (`IsWildcard` field, `containsWildcard()` function, `OrganizeHosts()`) remains intact. This may be useful for future features like analytics or user preferences.

**No-Op Code:**
After these changes, the wildcard collection logic in both list builders becomes a no-op (collects wildcards but never uses them). This is intentional and harmless - it keeps the code structure clean and makes re-enabling wildcards trivial if needed.

**Performance:**
Minimal impact - wildcards are still parsed and filtered, just not rendered. For typical SSH configs (< 100 hosts), this is negligible.

---

*Plan created: 2026-02-15*
*Based on: docs/plans/2026-02-15-hide-wildcard-connections-design.md*
