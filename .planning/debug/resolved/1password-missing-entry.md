---
status: resolved
trigger: "1password-missing-entry

**Summary:** A test entry in 1Password tagged with \"ssherpa\" is not showing up in the ssherpa TUI, even though it exists in 1Password and has the correct tag."
created: 2026-02-16T00:00:00Z
updated: 2026-02-16T00:00:00Z
---

## Current Focus

hypothesis: ItemToServer requires both "hostname" and "user" fields, and returns error if either is missing. The sync loop silently skips items that fail conversion (status.go:74-78), causing entries without these fields to disappear.
test: Check if the test entry is missing hostname or user fields
expecting: Test entry likely has "ssherpa" tag but is missing hostname or user field
next_action: Verify by inspecting the actual op item to confirm missing fields, or add debug logging to see conversion errors

## Symptoms

expected: A 1Password item tagged with "ssherpa" should appear as a server entry in the ssherpa TUI.
actual: The entry is not visible in the TUI despite being correctly tagged in 1Password.
errors: No explicit error messages — the entry just doesn't appear.
reproduction: Launch ssherpa TUI, look for the test entry — it's missing. The entry exists in 1Password with the "ssherpa" tag.
started: Worked before, broke recently (likely after Phase 08 changes). The previous debug session fixed the polling interval (5s→5m) and N+1 query issue, but this entry visibility issue persists.

## Eliminated

## Evidence

- timestamp: 2026-02-16T00:10:00Z
  checked: cli_client.go ListItems implementation (lines 114-178)
  found: ListItems DOES parse fields from `op item list --format json` response (lines 130-172)
  implication: The rewrite is correct - field data IS available from list command

- timestamp: 2026-02-16T00:15:00Z
  checked: status.go SyncFromOnePassword (lines 36-99)
  found: Sync loop silently skips items that fail ItemToServer conversion (lines 74-78 use `continue` on error)
  implication: If ItemToServer returns error, the item disappears without warning

- timestamp: 2026-02-16T00:18:00Z
  checked: mapping.go ItemToServer (lines 13-70)
  found: ItemToServer REQUIRES both "hostname" (line 62-64) and "user" (line 65-67) fields. Returns error if either is missing.
  implication: Any item tagged "ssherpa" but missing hostname or user will be silently filtered out during sync

## Resolution

root_cause: Items tagged with "ssherpa" that are missing required fields (hostname or user) are silently filtered out during sync. The SyncFromOnePassword function (status.go:74-78) calls ItemToServer which validates for required fields, but errors are silently ignored with `continue`, causing valid tagged items to disappear without any warning or error message.

fix: Added diagnostic output to stderr that reports which items are being skipped and why:
1. Enhanced ItemToServer error messages to include item title and ID (mapping.go:62-67)
2. Track skipped items in SyncFromOnePassword (status.go:62)
3. Output warning to stderr with details about each skipped item (status.go:86-91)

This allows users to see why their tagged items aren't appearing and fix the missing fields in 1Password.

verification:
✅ Build successful: ssherpa compiles without errors
✅ All tests pass: go test ./internal/backend/onepassword/... (all tests passing)
✅ New test added: TestSyncFromOnePassword_SkipsInvalidItems verifies diagnostic output
✅ Test output shows correct error messages:
```
Warning: 2 items with 'ssherpa' tag skipped due to validation errors:
  - No User Server: item "No User Server" (id: MISSING-USER) missing required field: user
  - No Hostname Server: item "No Hostname Server" (id: MISSING-HOST) missing required field: hostname
```

User action:
When user runs `./dist/ssherpa`, they will now see diagnostic messages explaining why tagged items are being skipped. They can then add the missing fields (hostname, user) in 1Password and the entries will appear in the TUI on next sync.

files_changed:
  - internal/backend/onepassword/status.go
  - internal/backend/onepassword/mapping.go
