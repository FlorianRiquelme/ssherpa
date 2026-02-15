# Design: Hide Wildcard Connections from TUI

**Date:** 2026-02-15
**Status:** Approved
**Approach:** Remove wildcard display code

## Overview

Remove wildcard SSH config entries (e.g., `Host *`, `Host *.example.com`, `Host dev-*`) from the TUI display while keeping the parser's wildcard detection logic intact for potential future use.

## User Requirement

Wildcard connections should be completely hidden from all TUI views - no display, no search results, no visibility.

## Architecture

**Scope:** UI layer only - modify TUI list building logic

**Components affected:**
- `internal/tui/model.go` - Remove wildcard display code from both list builders
- `internal/tui/list_view.go` - Remove unused `separatorItem` type (cleanup)

**Components unchanged:**
- `internal/sshconfig/parser.go` - Keep wildcard detection logic (the `IsWildcard` field and `containsWildcard()` function remain for potential future use)
- All other backend, config, and sync logic - No changes

**Why keep parser logic?**
The wildcard detection in the parser is clean, well-tested, and may be useful for future features (e.g., analytics, debugging, or user preferences). Removing it saves minimal processing time and increases risk.

## Implementation Details

**Two functions need modification:**

1. **`rebuildListItemsGrouped()`** (used when projects are configured, not in search mode)
   - **Current:** Lines 640-650 add separator and wildcard hosts
   - **Change:** Delete lines 640-650 entirely
   - **Effect:** Items array ends with unassigned hosts, wildcards never added

2. **`rebuildListItemsSimple()`** (used in search mode or no projects)
   - **Current:** Lines 704-717 add separator and wildcard hosts
   - **Change:** Delete lines 704-717 entirely
   - **Effect:** Items array ends with regular hosts, wildcards never added

**Cleanup:**

3. **Remove `separatorItem` type** from `list_view.go` (lines 78-95)
   - Type becomes unused after removing wildcard display
   - Clean removal, no other dependencies

**What stays:**
- Wildcard collection logic (lines 565, 571-573 in `rebuildListItemsGrouped()`)
- Wildcard separation logic (lines 659, 662-663 in `rebuildListItemsSimple()`)
- These become no-ops but are harmless and keep the code structure clean

**Result:** Wildcards are parsed from SSH config but never rendered in the TUI.

## Testing Strategy

**Manual testing needed:**

1. **With wildcards in SSH config:**
   - Add test entries: `Host *`, `Host *.example.com`, `Host dev-*`
   - Launch ssherpa TUI
   - **Verify:** No wildcard entries shown, no separator displayed
   - **Verify:** Regular hosts still display correctly

2. **Without wildcards:**
   - Use SSH config with only regular hosts
   - **Verify:** No visual changes, list works as before

3. **Search/filter mode:**
   - Enter search mode with wildcards in config
   - **Verify:** Wildcards don't appear in filtered results
   - **Verify:** Search still works for regular hosts

4. **Project grouping:**
   - Test with projects configured
   - **Verify:** Project groups still display correctly
   - **Verify:** No wildcard section at bottom

**Existing tests:**
- Parser tests in `internal/sshconfig/parser_test.go` already cover wildcard detection
- No test changes needed (we're only modifying display logic, not parsing)

## Edge Cases

**Handled automatically:**

1. **All hosts are wildcards:** List will be empty (correct behavior)
2. **Mixed regular and wildcard:** Only regular hosts shown (desired)
3. **Search matches wildcard:** Wildcard won't appear (correct)
4. **Recent connection to wildcard:** Won't show with star indicator (acceptable)

**No special handling needed** - removing the display code naturally handles all cases.

## Alternatives Considered

### Approach 2: Filter Wildcards Early
Filter wildcards before they enter the filtered list, preventing processing entirely.
- **Rejected:** More complex, higher risk, minimal performance benefit

### Approach 3: Add Configuration Flag
Add `hideWildcards` config option for flexibility.
- **Rejected:** Over-engineered for this requirement, unnecessary complexity

## Success Criteria

- ✅ No wildcard entries visible in TUI
- ✅ No "--- Wildcard Entries ---" separator displayed
- ✅ Regular hosts display unchanged
- ✅ Search/filter works correctly without wildcards
- ✅ Project grouping unaffected
- ✅ No regressions in existing functionality

---

*Design approved: 2026-02-15*
