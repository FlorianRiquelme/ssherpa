---
phase: 02-ssh-config-integration
plan: 01
subsystem: sshconfig
tags: [backend, parser, ssh-config, kevinburke]
dependency_graph:
  requires:
    - internal/backend/backend.go (Backend interface)
    - internal/domain/server.go (domain models)
    - internal/errors/errors.go (error handling)
  provides:
    - internal/sshconfig/parser.go (SSH config parsing)
    - internal/sshconfig/backend.go (backend adapter)
  affects: []
tech_stack:
  added:
    - github.com/kevinburke/ssh_config@v1.4.0
  patterns:
    - "Backend interface implementation (database/sql pattern)"
    - "Copy-on-read for data safety"
    - "Graceful error handling for malformed configs"
    - "Wildcard detection and organization"
key_files:
  created:
    - internal/sshconfig/parser.go (175 lines)
    - internal/sshconfig/parser_test.go (249 lines)
    - internal/sshconfig/backend.go (245 lines)
    - internal/sshconfig/backend_test.go (378 lines)
  modified:
    - go.mod (added kevinburke/ssh_config dependency)
    - go.sum (dependency checksums)
decisions: []
metrics:
  duration: 293s
  tasks_completed: 2
  files_created: 4
  files_modified: 2
  test_count: 23
  coverage: 97.5%
  completed_date: 2026-02-14
---

# Phase 02 Plan 01: SSH Config Parser & Backend Summary

**One-liner:** SSH config parser wrapper using kevinburke/ssh_config with backend adapter converting parsed hosts to domain.Server (97.5% test coverage, 23 tests).

## What Was Built

Created the `internal/sshconfig` package with two main components:

1. **Parser wrapper** (`parser.go`):
   - SSHHost struct capturing all SSH config data (Host, User, Port, IdentityFile, AllOptions, etc.)
   - ParseSSHConfig function wrapping kevinburke/ssh_config library
   - Automatic Include directive handling (via library's recursive expansion)
   - Wildcard detection for patterns containing `*` or `?`
   - Graceful malformed entry handling (ParseError set, not fatal)
   - OrganizeHosts utility to separate and sort regular vs wildcard hosts

2. **Backend adapter** (`backend.go`):
   - Backend struct implementing backend.Backend interface (compile-time verified)
   - Read-only backend (no Writer interface implementation)
   - SSHHost to domain.Server conversion with intelligent field mapping
   - Port defaulting to 22 when empty or invalid
   - ProxyJump extraction from AllOptions
   - Hostname fallback to Name when empty (SSH behavior)
   - Thread-safe operations with RWMutex
   - Copy-on-read pattern preventing mutation leaks
   - Empty results for projects/credentials (not supported by SSH config)

## Implementation Highlights

**Library integration:**
- Uses kevinburke/ssh_config v1.4.0 for parsing
- Library handles Include recursion automatically (up to depth 5)
- Library creates implicit "Host *" entries with no options — filtered out in parser
- No position tracking available from library (SourceLine set to 0)

**Error handling:**
- Malformed configs (e.g., Match blocks) produce SSHHost with ParseError set
- Match directive errors get clear error messages
- All backend operations wrap errors with BackendError{Op, Backend, Err}
- Closed backend returns ErrBackendUnavailable for all operations

**Data model decisions:**
- Port stored as string in SSHHost (preserves raw config), converted to int in domain.Server
- AllOptions map captures every SSH config option (preserves multi-values)
- IdentityFile slice preserves all values, backend takes first for domain.Server.IdentityFile
- Notes field includes source file info or parse error details

**Test coverage:**
- Parser: 8 tests covering valid/malformed/wildcard/multi-value scenarios
- Backend: 15 tests covering interface compliance, field mapping, error handling, lifecycle
- Total: 23 tests, 97.5% coverage

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking Issue] Library does not expose position information**
- **Found during:** Task 1 implementation
- **Issue:** kevinburke/ssh_config Host struct has no Pos() method to get line numbers
- **Fix:** Removed Pos() calls, set SourceLine to 0, added comment documenting limitation
- **Files modified:** internal/sshconfig/parser.go
- **Commit:** c43e38f

**2. [Rule 1 - Bug] Library adds implicit "Host *" entries**
- **Found during:** Task 1 testing
- **Issue:** ssh_config.Decode creates default "Host *" entry with no options, breaking test assertions
- **Fix:** Added filter to skip "Host *" entries with empty AllOptions map
- **Files modified:** internal/sshconfig/parser.go
- **Commit:** c43e38f

**3. [Rule 1 - Bug] Unused import in backend_test.go**
- **Found during:** Task 2 compilation
- **Issue:** path/filepath imported but not used in backend_test.go
- **Fix:** Removed unused import
- **Files modified:** internal/sshconfig/backend_test.go
- **Commit:** c8f5f51

## Verification Results

All success criteria met:

- ✅ SSHHost struct captures all SSH config data including AllOptions, SourceFile, IsWildcard, ParseError
- ✅ ParseSSHConfig handles valid configs, empty files, malformed files, and wildcard detection
- ✅ OrganizeHosts separates wildcards from regular hosts and sorts alphabetically
- ✅ sshconfig.Backend implements backend.Backend interface (compile-time verified via `var _ backend.Backend = (*Backend)(nil)`)
- ✅ SSHHost -> domain.Server conversion maps all relevant fields correctly
- ✅ Port defaults to 22 when empty or invalid
- ✅ Closed backend returns ErrBackendUnavailable for all operations
- ✅ All tests pass with `go test -race ./internal/sshconfig/` (23 tests)
- ✅ `go build ./...` compiles cleanly
- ✅ Coverage: 97.5% exceeds 70% requirement

## Files Changed

**Created:**
- `/Users/florianriquelme/Repos/mine/sshjesus/internal/sshconfig/parser.go`
- `/Users/florianriquelme/Repos/mine/sshjesus/internal/sshconfig/parser_test.go`
- `/Users/florianriquelme/Repos/mine/sshjesus/internal/sshconfig/backend.go`
- `/Users/florianriquelme/Repos/mine/sshjesus/internal/sshconfig/backend_test.go`

**Modified:**
- `/Users/florianriquelme/Repos/mine/sshjesus/go.mod`
- `/Users/florianriquelme/Repos/mine/sshjesus/go.sum`

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | SSH config parser wrapper | c43e38f | parser.go, parser_test.go, go.mod, go.sum |
| 2 | sshconfig backend adapter | c8f5f51 | backend.go, backend_test.go |

## What's Next

**Ready for Plan 02-02:** SSH config TUI implementation can now consume this backend via the Backend interface. The parser wrapper isolates SSH config complexity, and the backend adapter provides clean domain.Server objects for the TUI to display.

**Integration notes:**
- Backend is read-only (check for Writer interface support via type assertion)
- Malformed entries included in ListServers with ParseError info in Notes field
- Use OrganizeHosts if TUI needs to separate wildcard entries visually
- SourceLine is always 0 due to library limitation (use SourceFile for tracking)

## Self-Check: PASSED

**Files created:**
- ✅ FOUND: /Users/florianriquelme/Repos/mine/sshjesus/internal/sshconfig/parser.go
- ✅ FOUND: /Users/florianriquelme/Repos/mine/sshjesus/internal/sshconfig/parser_test.go
- ✅ FOUND: /Users/florianriquelme/Repos/mine/sshjesus/internal/sshconfig/backend.go
- ✅ FOUND: /Users/florianriquelme/Repos/mine/sshjesus/internal/sshconfig/backend_test.go

**Commits exist:**
- ✅ FOUND: c43e38f (Task 1 - SSH config parser wrapper)
- ✅ FOUND: c8f5f51 (Task 2 - sshconfig backend adapter)

**Tests pass:**
- ✅ go test -race ./internal/sshconfig/ — 23 tests pass
- ✅ go vet ./internal/sshconfig/ — no warnings
- ✅ go build ./... — compiles cleanly
- ✅ Coverage 97.5% > 70% requirement
