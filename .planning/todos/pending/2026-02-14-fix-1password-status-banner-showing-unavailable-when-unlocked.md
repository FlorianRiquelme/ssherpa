---
created: 2026-02-14T21:00:22.089Z
title: Fix 1Password status banner showing unavailable when unlocked
area: general
files:
  - internal/onepassword/status.go
  - internal/tui/status_bar.go
---

## Problem

The TUI shows a constant banner indicating 1Password is not available, even though the 1Password desktop app is unlocked. The likely root cause is that the `op` CLI is not signed in / authenticated separately from the desktop app. The BackendStatus check may be using `op` CLI commands that require their own authentication session, independent of the desktop app's lock state.

This creates a confusing UX where the user sees "1Password unavailable" despite having 1Password open and unlocked.

## Solution

1. Investigate what the status check actually tests (op CLI session vs desktop app vs service account)
2. Determine if the check should use the service account token (OP_SERVICE_ACCOUNT_TOKEN) instead of relying on `op` CLI interactive sign-in
3. Fix the status detection to accurately reflect availability based on the configured authentication method
4. Consider adding a more specific banner message (e.g., "op CLI not signed in" vs "1Password unavailable") to help users diagnose the issue
