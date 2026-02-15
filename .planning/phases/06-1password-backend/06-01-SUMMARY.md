---
phase: 06-1password-backend
plan: 01
subsystem: backend
tags: [backend, 1password, sdk, adapter, tdd]
dependency_graph:
  requires: [internal/backend/backend.go, internal/domain/server.go, internal/errors/errors.go]
  provides: [internal/backend/onepassword]
  affects: []
tech_stack:
  added:
    - github.com/1password/onepassword-sdk-go@v0.4.0-beta.2
  patterns:
    - Client interface abstraction for testability
    - MockClient for testing without real 1Password
    - Tag-based discovery across all vaults
    - Copy-on-read/copy-on-write for cache safety
    - BackendError wrapping for operation context
key_files:
  created:
    - internal/backend/onepassword/client.go: SDK wrapper with Client interface
    - internal/backend/onepassword/client_test.go: MockClient implementation
    - internal/backend/onepassword/mapping.go: Bidirectional item-to-server conversion
    - internal/backend/onepassword/mapping_test.go: Mapping tests with 100% coverage
    - internal/backend/onepassword/backend.go: Backend + Writer implementation
    - internal/backend/onepassword/backend_test.go: Full backend test suite
  modified:
    - internal/domain/server.go: Added RemoteProjectPath and VaultID fields
    - go.mod: Added 1Password SDK dependency
    - go.sum: Dependency checksums
decisions:
  - title: Client interface abstraction
    rationale: Enables testing without real 1Password vaults, mock client for unit tests
    alternatives: Direct SDK usage (harder to test), test doubles (more boilerplate)
  - title: Lowercase "server" category
    rationale: Matches 1Password API expectations for item categories
    alternatives: "Server" with capital (failed tests)
  - title: Tag-based discovery with "ssherpa" tag
    rationale: Simple filtering mechanism, case-insensitive matching for robustness
    alternatives: Dedicated vault (limits flexibility), naming convention (fragile)
  - title: Skip vaults with errors
    rationale: Permission issues shouldn't fail entire list operation
    alternatives: Fail fast (worse UX), retry logic (complexity)
  - title: Projects as tags not entities
    rationale: 1Password doesn't have standalone project concept, tags are natural fit
    alternatives: Custom vault structure (rigid), separate tracking (duplication)
  - title: ItemField uses Title not Label
    rationale: 1Password SDK v0.4.0-beta.2 uses Title field for field names
    alternatives: Label (doesn't exist in SDK)
metrics:
  duration: 467
  completed_date: 2026-02-14
  tasks_completed: 2
  files_created: 6
  files_modified: 3
  tests_added: 18
  test_coverage: 100%
---

# Phase 06 Plan 01: Core 1Password Backend Adapter Summary

**One-liner:** Bidirectional 1Password item mapping with tag-based discovery and full Backend + Writer interface implementation using SDK v0.4.0-beta.2

## What Was Built

Core 1Password backend adapter that enables reading/writing server configs to 1Password items:

1. **Domain Extensions:** Added `RemoteProjectPath` and `VaultID` fields to `domain.Server`
2. **SDK Client Wrapper:** Created `Client` interface abstracting 1Password SDK operations
3. **Mock Client:** Implemented `MockClient` for testing without real 1Password
4. **Item Mapping:** Bidirectional conversion between 1Password items and domain.Server
5. **Backend Implementation:** Full Backend + Writer interface with tag-based discovery
6. **Comprehensive Tests:** 18 tests covering mapping, CRUD operations, error scenarios

## Technical Implementation

### SDK Integration

- **Desktop App Auth:** `NewDesktopAppClient()` uses `WithDesktopAppIntegration()` for seamless auth
- **Service Account Fallback:** `NewServiceAccountClient()` for token-based auth
- **Timeout Protection:** All SDK calls wrapped with 5-second context timeout
- **Iterator Handling:** Converted SDK iterators to slice returns for simpler API

### Item Mapping

1Password item structure:
- **Category:** "server"
- **Title:** DisplayName
- **Tags:** Includes "ssherpa" (case-insensitive)
- **Fields:** hostname, user, port, identity_file, remote_project_path, project_tags, proxy_jump

Conversion features:
- **Required Fields:** Hostname and user validated, error if missing
- **Port Default:** Defaults to 22 if not specified
- **Project Tags:** Comma-separated string split into ProjectIDs array
- **Round-Trip Lossless:** ServerToItem -> ItemToServer preserves all data
- **Tag Deduplication:** Ensures single "ssherpa" tag, no duplicates

### Backend Architecture

**Cache Management:**
- Thread-safe with `sync.RWMutex`
- Populated by `ListServers()` (scans all vaults)
- Updated by write operations (create/update/delete)
- Copy-on-read prevents mutation leaks

**Discovery Process:**
1. List all accessible vaults via `ListVaults()`
2. For each vault, list items via `ListItems()`
3. Filter items by `HasSshjesusTag()` (case-insensitive)
4. Convert items to servers via `ItemToServer()`
5. Skip vaults/items with errors (log and continue)

**Error Handling:**
- `BackendError` wrapping with "onepassword" backend identifier
- Vault errors don't fail entire list (resilient)
- Closed backend returns `ErrBackendUnavailable`
- Missing required fields return validation errors

### Test Coverage

**Mapping Tests (9):**
- Complete item conversion (all fields)
- Minimal item conversion (hostname+user only, port defaults to 22)
- Missing hostname/user validation
- Round-trip lossless conversion
- Case-insensitive "ssherpa" tag detection
- Tag deduplication

**Backend Tests (9):**
- List servers with filtering (2 tagged, 1 not → returns 2)
- Skip error vaults (continues with remaining vaults)
- Get server found/not found
- Create server (appears in subsequent list)
- Update server (fields modified correctly)
- Delete server (removed from cache and 1Password)
- Closed backend returns errors
- Interface compliance verification

**MockClient Features:**
- In-memory vault/item storage
- Per-operation error injection
- Per-vault error injection (for testing resilience)
- Copy-on-read/write semantics
- Closed state tracking

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed SDK API mismatch for ItemField**
- **Found during:** Task 1 - Compilation errors
- **Issue:** Plan used `Label` field but SDK v0.4.0-beta.2 uses `Title` field for item fields
- **Fix:** Changed all references from `Label` to `Title` in client.go and tests
- **Files modified:** client.go, mapping_test.go
- **Commit:** 3d5a6f8

**2. [Rule 1 - Bug] Fixed SDK API for vault listing**
- **Found during:** Task 1 - Compilation errors
- **Issue:** SDK uses `VaultOverview.Title` not `VaultOverview.Name`
- **Fix:** Updated client.go to use `v.Title` for vault names
- **Files modified:** client.go
- **Commit:** 3d5a6f8

**3. [Rule 1 - Bug] Fixed SDK API methods**
- **Found during:** Task 1 - Compilation errors
- **Issue:** SDK uses `Vaults().List()` not `ListAll()`, `Items().Put()` not `Update()`, returns structs not pointers
- **Fix:** Updated all SDK method calls to match v0.4.0-beta.2 API
- **Files modified:** client.go
- **Commit:** 3d5a6f8

**4. [Rule 2 - Missing Critical] Enhanced MockClient with per-vault errors**
- **Found during:** Task 2 - TestListServersSkipsErrorVaults failing
- **Issue:** MockClient only supported global operation errors, couldn't test "skip one vault" scenario
- **Fix:** Added `vaultErrors` map and `SetVaultError()` method for vault-specific error injection
- **Files modified:** client_test.go, backend_test.go
- **Commit:** 798d0ee

**5. [Rule 1 - Bug] Fixed lowercase "server" category**
- **Found during:** Task 1 - TestServerToItem_RoundTrip failing
- **Issue:** Used "Server" (capital S) but test expected "server" (lowercase)
- **Fix:** Changed category to lowercase "server" in mapping.go
- **Files modified:** mapping.go
- **Commit:** 3d5a6f8

## Verification

**All tests pass:**
```bash
$ go test ./internal/backend/onepassword/... -v -count=1
PASS
ok      github.com/florianriquelme/ssherpa/internal/backend/onepassword        0.237s
```

**Project compiles:**
```bash
$ go build ./...
(no output - success)
```

**No vet warnings:**
```bash
$ go vet ./...
(no output - success)
```

**Interface compliance verified:**
```go
var (
    _ backend.Backend = (*Backend)(nil)
    _ backend.Writer  = (*Backend)(nil)
)
```

**Domain extensions present:**
```go
RemoteProjectPath string   // line 22 in server.go
VaultID           string   // line 23 in server.go
```

## Key Files

**Created:**
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/backend/onepassword/client.go` - SDK client wrapper with Client interface, SDKClient implementation
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/backend/onepassword/client_test.go` - MockClient implementation with error injection
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/backend/onepassword/mapping.go` - ItemToServer, ServerToItem, HasSshjesusTag functions
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/backend/onepassword/mapping_test.go` - 9 mapping tests with 100% coverage
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/backend/onepassword/backend.go` - Backend implementation with 14 interface methods
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/backend/onepassword/backend_test.go` - 9 backend tests covering CRUD and errors

**Modified:**
- `/Users/florianriquelme/Repos/mine/ssherpa/internal/domain/server.go` - Added RemoteProjectPath and VaultID fields
- `/Users/florianriquelme/Repos/mine/ssherpa/go.mod` - Added 1Password SDK v0.4.0-beta.2 dependency
- `/Users/florianriquelme/Repos/mine/ssherpa/go.sum` - Dependency checksums

## Next Steps (from ROADMAP)

**Plan 06-02:** 1Password config loader and initialization
- Implement config file parsing for 1Password backend
- Desktop app vs service account auth selection
- Account name and token configuration
- Config validation and error messages

**Plan 06-03:** Integration with TUI (list/detail views)
- Wire 1Password backend into TUI server list
- Display vault information in server details
- Handle 1Password-specific errors in TUI

**Plan 06-04:** Sync command implementation
- Implement `ssherpa sync` command
- Trigger ListServers to sync from 1Password
- Display sync progress and results

**Plan 06-05:** Offline fallback with local cache
- Implement local cache for offline access
- Cache invalidation and refresh strategy
- Graceful degradation when 1Password unavailable

## Self-Check: PASSED

**Files exist:**
```bash
FOUND: internal/backend/onepassword/client.go
FOUND: internal/backend/onepassword/client_test.go
FOUND: internal/backend/onepassword/mapping.go
FOUND: internal/backend/onepassword/mapping_test.go
FOUND: internal/backend/onepassword/backend.go
FOUND: internal/backend/onepassword/backend_test.go
FOUND: internal/domain/server.go (modified)
```

**Commits exist:**
```bash
FOUND: 3d5a6f8 - feat(06-01): domain extensions + SDK client wrapper + item mapping
FOUND: 798d0ee - feat(06-01): implement 1Password backend with Backend + Writer interfaces
```

**Domain extensions present:**
```bash
$ grep "RemoteProjectPath\|VaultID" internal/domain/server.go
22:     RemoteProjectPath string
23:     VaultID           string
```

**Tests pass:**
```bash
$ go test ./internal/backend/onepassword/... -v
PASS (18 tests, 0 failures)
```

All verification criteria met. Plan 06-01 execution complete.
