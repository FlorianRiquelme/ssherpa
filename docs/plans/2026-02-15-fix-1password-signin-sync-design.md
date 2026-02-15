# Fix 1Password sign-in not triggering sync

**Date:** 2026-02-15
**Status:** Approved

## Problem

After pressing 's' to sign in via `op signin`, the TUI shows "Signed in to 1Password!" but:
1. The "1Password CLI not signed in" banner persists
2. Servers from 1Password are never loaded

Root cause: the `opSigninFinishedMsg` handler calls `loadBackendServersCmd`, which only reads the in-memory cache (empty at this point). It never calls `SyncFromOnePassword` to fetch data from 1Password or update the backend status.

## Solution

Add a `syncBackendCmd` tea.Cmd that calls `SyncFromOnePassword` and returns `OnePasswordStatusMsg` with the new status. Replace the `loadBackendServersCmd` call in the `opSigninFinishedMsg` handler with `syncBackendCmd`.

The existing `OnePasswordStatusMsg` handler already:
- Updates `m.opStatus` and re-renders the status bar
- Toggles the sign-in keybinding on/off
- Triggers `loadBackendServersCmd` when status becomes Available

## Flow after fix

```
op signin -> success -> syncBackendCmd -> SyncFromOnePassword
  -> OnePasswordStatusMsg(Available) -> banner clears, keybinding disabled
  -> loadBackendServersCmd -> reads populated cache -> servers appear
```

## Files changed

- `internal/tui/model.go`: Add `syncBackendCmd`; update `opSigninFinishedMsg` handler
