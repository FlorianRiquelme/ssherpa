# Rename ssherpa to ssherpa — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rename the tool from "ssherpa" to "ssherpa" across the entire codebase — Go module, source, tests, and docs.

**Architecture:** Mechanical find-and-replace in stages: Go module path first (breaks compilation), then fix all imports (restores compilation), then string literals, then docs. Each stage is verified before moving on.

**Tech Stack:** Go, sed, git mv

---

### Task 1: Rename Go module path in go.mod

**Files:**
- Modify: `go.mod:1`

**Step 1: Update module path**

In `go.mod`, change line 1:
```
module github.com/florianriquelme/ssherpa
```
to:
```
module github.com/florianriquelme/ssherpa
```

**Step 2: Commit**

```bash
git add go.mod
git commit --no-gpg-sign -m "refactor: rename Go module from ssherpa to ssherpa"
```

Note: The build will be broken until Task 2 completes (all imports still reference old module path).

---

### Task 2: Rename cmd directory and update all Go import paths

**Files:**
- Rename: `cmd/ssherpa/` → `cmd/ssherpa/`
- Modify: All 73 `.go` files — replace import path `github.com/florianriquelme/ssherpa/` with `github.com/florianriquelme/ssherpa/`

**Step 1: Rename the cmd directory**

```bash
git mv cmd/ssherpa cmd/ssherpa
```

**Step 2: Replace all import paths in every .go file**

```bash
find . -name '*.go' -exec sed -i '' 's|github.com/florianriquelme/ssherpa/|github.com/florianriquelme/ssherpa/|g' {} +
```

This updates import statements like:
```go
// Before
import "github.com/florianriquelme/ssherpa/internal/backend"

// After
import "github.com/florianriquelme/ssherpa/internal/backend"
```

**Step 3: Verify build compiles**

```bash
go build ./...
```

Expected: Success (all imports now resolve to the new module path).

**Step 4: Verify tests pass**

```bash
go test ./...
```

Expected: All tests pass (import paths updated, no string literals changed yet).

**Step 5: Commit**

```bash
git add -A
git commit --no-gpg-sign -m "refactor: rename cmd directory and update all import paths to ssherpa"
```

---

### Task 3: Update hardcoded string references in Go source

**Files to modify (all under `internal/` and `cmd/`):**
- `cmd/ssherpa/main.go` — path constants for history, cache, config
- `internal/config/config.go` — XDG config path `ssherpa/config.toml`
- `internal/history/history.go` — history filename `ssherpa_history.json`
- `internal/sync/ssh_include.go` — include file path `ssherpa_config`
- `internal/sync/conflict.go` — conflict detection for `ssherpa_config`
- `internal/backend/onepassword/mapping.go` — 1Password tag `ssherpa`
- `internal/backend/onepassword/backend.go` — cache filename
- `internal/tui/wizard.go` — wizard text references
- `internal/tui/migration.go` — migration text references
- `internal/tui/model.go` — UI text references
- `internal/tui/form.go` — form text references
- `internal/tui/delete.go` — delete text references
- `internal/tui/status_bar.go` — status bar text
- `internal/tui/messages.go` — message text
- `internal/tui/detail_view.go` — detail view text
- `internal/tui/list_view.go` — list view text
- `internal/tui/picker.go` — picker text
- `internal/tui/undo.go` — undo text
- `internal/tui/key_picker.go` — key picker text
- `internal/backend/onepassword/status.go` — status text
- `internal/backend/onepassword/poller.go` — poller references
- `internal/backend/onepassword/client.go` — client references
- `internal/backend/onepassword/cli_client.go` — CLI client references
- `internal/backend/backend.go` — backend text
- `internal/backend/multi.go` — multi-backend text
- `internal/backend/mock/mock.go` — mock backend text
- `internal/sshconfig/backend.go` — sshconfig backend text
- `internal/sshkey/discovery.go` — key discovery text
- `internal/errors/errors.go` — error messages
- `internal/sync/toml_cache.go` — cache references

**Step 1: Replace all remaining "ssherpa" string references in Go source files**

```bash
find . -name '*.go' -exec sed -i '' 's/ssherpa/ssherpa/g' {} +
```

This catches:
- `ssherpa_config` → `ssherpa_config`
- `ssherpa_history.json` → `ssherpa_history.json`
- `ssherpa_1password_cache.toml` → `ssherpa_1password_cache.toml`
- `ssherpa/config.toml` → `ssherpa/config.toml` (XDG path)
- `"ssherpa"` tag → `"ssherpa"` tag (1Password)
- Any remaining comments or strings

**Step 2: Verify build still compiles**

```bash
go build ./...
```

**Step 3: Verify all tests pass**

```bash
go test ./...
```

**Step 4: Manually verify key files have correct replacements**

Spot-check these critical paths:
- `internal/config/config.go` — should have `ssherpa/config.toml`
- `internal/sync/ssh_include.go` — should have `ssherpa_config`
- `internal/backend/onepassword/mapping.go` — should have `ssherpa` tag
- `cmd/ssherpa/main.go` — should have `ssherpa_history.json` and `ssherpa_1password_cache.toml`

**Step 5: Commit**

```bash
git add -A
git commit --no-gpg-sign -m "refactor: update all hardcoded ssherpa references to ssherpa in Go source"
```

---

### Task 4: Update test files

**Files:** All `*_test.go` files (should already be updated by Task 3's sed, but verify)

**Step 1: Verify no remaining "ssherpa" references in test files**

```bash
grep -r "ssherpa" --include="*_test.go" .
```

Expected: No matches. If any remain, update them manually.

**Step 2: Run full test suite**

```bash
go test ./... -v
```

Expected: All tests pass with the new paths and strings.

**Step 3: Commit (if any manual fixes were needed)**

```bash
git add -A
git commit --no-gpg-sign -m "test: fix remaining ssherpa references in test files"
```

---

### Task 5: Update planning and documentation files

**Files:** All `.planning/**/*.md` files (69 files)

**Step 1: Replace all "ssherpa" references in planning files**

```bash
find .planning -name '*.md' -exec sed -i '' 's/ssherpa/ssherpa/g' {} +
```

**Step 2: Update design docs**

```bash
find docs -name '*.md' -exec sed -i '' 's/ssherpa/ssherpa/g' {} +
```

**Step 3: Verify no remaining references**

```bash
grep -r "ssherpa" .planning/ docs/ 2>/dev/null
```

Expected: No matches (except possibly the rename todo itself describing the old name, which is fine).

**Step 4: Commit**

```bash
git add -A
git commit --no-gpg-sign -m "docs: rename ssherpa to ssherpa across all planning and documentation"
```

---

### Task 6: Cleanup and final verification

**Step 1: Delete the old compiled binary**

```bash
rm -f ssherpa
```

**Step 2: Build the new binary**

```bash
go build -o ssherpa ./cmd/ssherpa/
```

Expected: Clean build producing `ssherpa` binary.

**Step 3: Run full test suite one final time**

```bash
go test ./...
```

Expected: All tests pass.

**Step 4: Verify no remaining "ssherpa" anywhere in tracked Go/config files**

```bash
grep -r "ssherpa" --include="*.go" --include="*.toml" --include="*.mod" .
```

Expected: No matches.

**Step 5: Add ssherpa binary to .gitignore if not already there**

Check `.gitignore` and ensure both old and new binary names are ignored.

**Step 6: Commit**

```bash
git add -A
git commit --no-gpg-sign -m "chore: cleanup old binary and verify complete rename to ssherpa"
```

**Step 7: Move todo to done**

```bash
mv .planning/todos/pending/2026-02-14-rename-tool-from-ssherpa-to-ssherpa.md .planning/todos/done/ 2>/dev/null || true
```

---

## Verification Checklist

After all tasks complete, these must be true:
- [ ] `go build ./cmd/ssherpa/` succeeds
- [ ] `go test ./...` passes all tests
- [ ] `grep -r "ssherpa" --include="*.go" .` returns nothing
- [ ] `go.mod` says `module github.com/florianriquelme/ssherpa`
- [ ] `cmd/ssherpa/main.go` exists (not cmd/ssherpa)
- [ ] Config path is `ssherpa/config.toml`
- [ ] SSH paths use `ssherpa_config`, `ssherpa_history.json`, `ssherpa_1password_cache.toml`
- [ ] 1Password tag is `ssherpa`
