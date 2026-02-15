# Fix 1Password Sign-in Sync Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** After `op signin` completes, immediately sync from 1Password so the status banner clears and servers load.

**Architecture:** Add a `Syncer` interface to the backend package (following the existing Writer/Filterer pattern). Add a `syncBackendCmd` tea.Cmd in the TUI that type-asserts to Syncer, calls `SyncFromOnePassword`, and returns `OnePasswordStatusMsg`. The existing message handler already updates the status bar and triggers server loading.

**Tech Stack:** Go, Bubble Tea (TUI framework), 1Password CLI

---

### Task 1: Add Syncer interface to backend package

**Files:**
- Modify: `internal/backend/backend.go` (after the existing Writer/Filterer interfaces)

**Step 1: Add the Syncer interface**

Add after the existing optional capability interfaces:

```go
// Syncer is an optional interface for backends that support on-demand synchronization.
// Type-assert to Syncer to trigger a sync cycle (e.g., after user signs in).
type Syncer interface {
	SyncFromBackend(ctx context.Context) error
	GetStatus() BackendStatus
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: PASS (interface is unused so far, no breaking changes)

**Step 3: Commit**

```bash
git add internal/backend/backend.go
git commit -m "feat(backend): add Syncer interface for on-demand sync"
```

---

### Task 2: Add syncBackendCmd to TUI model

**Files:**
- Modify: `internal/tui/model.go:320` area (near `loadBackendServersCmd`)

**Step 1: Add the syncBackendCmd function**

Add near `loadBackendServersCmd` (around line 320):

```go
// syncBackendCmd triggers a backend sync and returns the new status.
// Used after sign-in to immediately refresh data and status.
func syncBackendCmd(b backend.Backend) tea.Cmd {
	return func() tea.Msg {
		if syncer, ok := b.(backend.Syncer); ok {
			ctx := context.Background()
			syncer.SyncFromBackend(ctx)
			// Error is OK - status is set appropriately by SyncFromBackend
			return OnePasswordStatusMsg{Status: syncer.GetStatus()}
		}
		// Backend doesn't support sync - no-op
		return nil
	}
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/tui/model.go
git commit -m "feat(tui): add syncBackendCmd for on-demand backend sync"
```

---

### Task 3: Wire up opSigninFinishedMsg handler

**Files:**
- Modify: `internal/tui/model.go:805-816` (the `opSigninFinishedMsg` case)

**Step 1: Replace loadBackendServersCmd with syncBackendCmd**

Change the `opSigninFinishedMsg` handler from:

```go
case opSigninFinishedMsg:
	// op signin process completed
	if msg.err != nil {
		m.statusMsg = "Sign-in failed"
	} else {
		m.statusMsg = "Signed in to 1Password!"
		// Trigger immediate server list refresh
		if m.appBackend != nil {
			return m, loadBackendServersCmd(m.appBackend)
		}
	}
	return m, nil
```

To:

```go
case opSigninFinishedMsg:
	// op signin process completed
	if msg.err != nil {
		m.statusMsg = "Sign-in failed"
	} else {
		m.statusMsg = "Signed in to 1Password!"
		// Trigger immediate sync to refresh status and servers
		if m.appBackend != nil {
			return m, syncBackendCmd(m.appBackend)
		}
	}
	return m, nil
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/tui/model.go
git commit -m "fix(tui): trigger sync after sign-in instead of reading empty cache"
```

---

### Task 4: Implement Syncer on onepassword.Backend

**Files:**
- Modify: `internal/backend/onepassword/status.go` (add SyncFromBackend method)

**Step 1: Write the failing test**

Add to `internal/backend/onepassword/status_test.go`:

```go
func TestBackend_ImplementsSyncer(t *testing.T) {
	mock := NewMockClient()
	b := New(mock)

	// Verify onepassword.Backend satisfies backend.Syncer
	var _ backendpkg.Syncer = b
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/backend/onepassword/ -run TestBackend_ImplementsSyncer -v`
Expected: FAIL (Backend doesn't have SyncFromBackend method yet)

**Step 3: Add SyncFromBackend method**

Add to `internal/backend/onepassword/status.go`:

```go
// SyncFromBackend implements backend.Syncer.
// It delegates to SyncFromOnePassword for the actual sync logic.
func (b *Backend) SyncFromBackend(ctx context.Context) error {
	return b.SyncFromOnePassword(ctx)
}
```

**Step 4: Add compile-time interface check**

Add to `internal/backend/onepassword/backend.go` in the existing `var` block:

```go
_ backendpkg.Syncer = (*Backend)(nil)
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/backend/onepassword/ -run TestBackend_ImplementsSyncer -v`
Expected: PASS

**Step 6: Run all tests**

Run: `go test ./...`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/backend/onepassword/status.go internal/backend/onepassword/status_test.go internal/backend/onepassword/backend.go
git commit -m "feat(1password): implement Syncer interface"
```

---

### Task 5: Final verification

**Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All PASS

**Step 2: Build the binary**

Run: `go build -o ssherpa ./cmd/ssherpa`
Expected: PASS

**Step 3: Manual smoke test (optional)**

1. Run `./ssherpa`
2. If 1Password CLI is not signed in, banner should show "Press 's' to sign in"
3. Press 's', complete sign-in
4. Banner should clear immediately and servers should appear
