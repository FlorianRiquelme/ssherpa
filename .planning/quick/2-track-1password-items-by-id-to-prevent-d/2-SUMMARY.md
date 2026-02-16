---
phase: quick-2
plan: 01
subsystem: backend/multi
tags: [bug-fix, deduplication, 1password-sync]
dependency_graph:
  requires: [backend, domain]
  provides: [ssherpa-generated-filtering]
  affects: [server-listing, multi-backend]
tech_stack:
  added: []
  patterns: [filtering-before-dedup]
key_files:
  created: []
  modified:
    - internal/backend/multi.go
    - internal/backend/multi_test.go
decisions:
  - Filter ssherpa-generated servers before dedup (not during)
  - Reuse existing pattern from sync/conflict.go for detection
  - Preserve all existing DisplayName dedup logic unchanged
metrics:
  duration_seconds: 120
  completed_date: 2026-02-16T06:17:40Z
  tasks_completed: 2
  files_modified: 2
  tests_added: 3
  commits: 2
---

# Quick Task 2: Fix duplicate servers on 1Password item rename

**One-liner:** Filter ssherpa-generated SSH config mirrors before deduplication to prevent phantom duplicates when 1Password items are renamed.

## What Was Built

Fixed a bug where renaming a 1Password item caused duplicate server entries in the TUI. The root cause was that `~/.ssh/ssherpa_config` (auto-generated SSH include file) contained stale mirrors of 1Password data. When an item was renamed, the old name in ssherpa_config didn't match the new name from 1Password, so DisplayName-based dedup failed to recognize them as the same server.

### Solution

Added filtering logic to `MultiBackend.ListServers()` that removes ssherpa-generated mirrors before deduplication:

1. **Detection pattern:** `Source == "ssh-config"` AND `Notes` contains `"ssherpa_config"`
2. **Filter placement:** After collecting from all backends, before DisplayName dedup
3. **Preserved entries:** User-authored SSH config entries (Notes doesn't contain ssherpa_config)

This approach mirrors the existing `isSshjesusGenerated` pattern in `sync/conflict.go`, maintaining consistency across the codebase.

## Implementation Details

### Changes to internal/backend/multi.go

**Added helper function:**
```go
func isSsherpaGenerated(server *domain.Server) bool {
    return server.Source == "ssh-config" && strings.Contains(server.Notes, "ssherpa_config")
}
```

**Modified ListServers() flow:**
1. Collect servers from all backends (unchanged)
2. **NEW:** Filter out ssherpa-generated servers
3. Deduplicate by DisplayName case-insensitive (unchanged)
4. Return deduplicated list

### Test Coverage

Added 3 comprehensive test cases to `multi_test.go`:

1. **TestMultiBackend_FiltersOutSsherpaGeneratedServers**
   - Verifies ssherpa_config mirrors are filtered out
   - Verifies user-authored SSH config entries are preserved
   - Verifies 1Password entries are kept
   - Result: 2 servers from mixed sources (not 3 with duplicate mirror)

2. **TestMultiBackend_RenamedOnePasswordItemNoDuplicate**
   - Tests the exact bug scenario described in the issue
   - Stale "old-name" in ssherpa_config + "new-name" in 1Password
   - Result: 1 server with new name (stale mirror filtered)

3. **TestMultiBackend_PureSshConfigServersNotFiltered**
   - Ensures user-authored SSH config entries never filtered
   - No 1Password backend present
   - Result: User entry preserved

All 14 backend tests pass (11 existing + 3 new).

## Verification

```bash
# All backend tests pass
go test ./internal/backend/ -v
# PASS: 14/14 tests

# Full project builds
go build ./...
# SUCCESS

# All project tests pass (no regressions)
go test ./...
# PASS: All packages
```

## Deviations from Plan

None - plan executed exactly as written.

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Filter before dedup (not during) | Cleaner separation of concerns: filtering removes noise, dedup handles conflicts |
| Reuse sync/conflict.go pattern | Consistent detection logic across codebase for ssherpa-generated entries |
| Preserve DisplayName dedup unchanged | Existing priority-based dedup works correctly for non-ssherpa conflicts |

## Files Modified

| File | Changes | LOC |
|------|---------|-----|
| internal/backend/multi.go | Added isSsherpaGenerated helper + filtering step in ListServers | +20 |
| internal/backend/multi_test.go | Added 3 test cases for filtering scenarios | +137 |

## Commits

| Hash | Message |
|------|---------|
| 77c72d3 | feat(quick-2): filter ssherpa-generated servers to prevent duplicates on rename |
| eed1526 | test(quick-2): add tests for ssherpa-generated server filtering |

## Impact

**User-facing:**
- Renamed 1Password items now appear exactly once in server list
- No more confusing duplicate entries with old/new names
- User-authored SSH config entries unaffected

**Developer-facing:**
- Clear separation between ssherpa-generated and user-authored SSH config
- Consistent detection pattern across backend and sync packages
- Comprehensive test coverage for edge cases

## Self-Check: PASSED

**Files verified:**
```bash
[ -f "internal/backend/multi.go" ] && echo "FOUND: internal/backend/multi.go"
# FOUND: internal/backend/multi.go

[ -f "internal/backend/multi_test.go" ] && echo "FOUND: internal/backend/multi_test.go"
# FOUND: internal/backend/multi_test.go
```

**Commits verified:**
```bash
git log --oneline --all | grep -q "77c72d3" && echo "FOUND: 77c72d3"
# FOUND: 77c72d3

git log --oneline --all | grep -q "eed1526" && echo "FOUND: eed1526"
# FOUND: eed1526
```

All claimed files and commits exist.
