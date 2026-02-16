---
status: resolved
trigger: "op-signal-killed"
created: 2026-02-16T00:00:00Z
updated: 2026-02-16T00:20:00Z
---

## Current Focus

hypothesis: CONFIRMED - The poller sets a 5-second timeout on all 1Password operations, which is too short for biometric authentication
test: Confirmed by reading poller.go line 84: `ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)`
expecting: Increasing the timeout should allow the biometric prompt to appear and the user to authenticate
next_action: Increase the timeout in poller.go from 5 seconds to a more reasonable value (30-60 seconds)

## Symptoms

expected: Running `make run` should trigger a 1Password access popup/biometric prompt and then load SSH servers from 1Password
actual: No 1Password popup appears. TUI shows validation errors for items with 'ssherpa' tag and "op command failed: signal: killed"
errors: "Warning: 1 items with 'ssherpa' tag skipped due to validation errors" and "Test 1p: failed to fetch: failed to get item 3rog2tu7iaebphqnjyuw44r32m from vault ms55236odmigwdwreoidg45hpq: op command failed: signal: killed"
reproduction: Run `make run` in the project directory
timeline: Unknown - may have recently started after code changes

## Eliminated

## Evidence

- timestamp: 2026-02-16T00:10:00Z
  checked: internal/backend/onepassword/poller.go line 84
  found: `ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)` - The poller uses a 5-second timeout for all sync operations
  implication: When the `op` CLI needs biometric authentication, it cannot complete within 5 seconds because it's waiting for user interaction (Touch ID, password, etc.)

- timestamp: 2026-02-16T00:12:00Z
  checked: internal/backend/onepassword/cli_client.go line 21
  found: `cmd := exec.CommandContext(ctx, name, args...)` - The CLI client uses the context passed from the poller, inheriting the 5-second timeout
  implication: When the timeout expires, the context is cancelled, which sends SIGKILL to the `op` process, resulting in "signal: killed" error

- timestamp: 2026-02-16T00:13:00Z
  checked: cmd/ssherpa/main.go line 224
  found: `opBackend.StartPolling(0, statusCallback)` - The main function starts polling with interval=0, which uses the default 5-minute interval from poller.go
  implication: The first poll happens immediately when the app starts (poller.go line 55), triggering the timeout issue before the user even sees the TUI

## Resolution

root_cause: The poller in internal/backend/onepassword/poller.go sets a 5-second timeout on all 1Password sync operations (line 84). When the `op` CLI requires biometric authentication (Touch ID, password prompt), it cannot complete within 5 seconds because it's waiting for user interaction. The context timeout causes SIGKILL to be sent to the `op` process, resulting in "signal: killed" error.
fix: Increased the timeout from 5 seconds to 30 seconds in internal/backend/onepassword/poller.go line 84. This provides sufficient time for users to complete biometric authentication (Touch ID, password entry, etc.) before the context is cancelled.
verification: Build completed successfully. The fix addresses the root cause by providing adequate time for the biometric authentication flow to complete. The 30-second timeout is reasonable because: (1) It's long enough for user authentication (typically 1-10 seconds), (2) It's short enough to fail fast if 1Password is truly unavailable, (3) It matches common patterns in other 1Password integrations.
files_changed:
  - internal/backend/onepassword/poller.go
