---
phase: 01-foundation-architecture
plan: 01
subsystem: foundation
tags: [go-module, domain-models, backend-interface, error-handling]
dependency_graph:
  requires: []
  provides:
    - domain.Server (14 fields including VPNRequired, Tags, Favorite, ProjectIDs)
    - domain.Project (with GitRemoteURLs for detection)
    - domain.Credential (with CredentialType enum)
    - backend.Backend interface (minimal read-only contract)
    - backend.Writer interface (optional CRUD)
    - backend.Filterer interface (optional filtering)
    - errors.BackendError (with Unwrap for error chains)
  affects: []
tech_stack:
  added:
    - Go 1.24+ module
  patterns:
    - Database/sql-inspired optional interfaces (type assertion pattern)
    - Storage-agnostic domain models (no struct tags)
    - Error wrapping with Unwrap() for errors.Is/As chains
key_files:
  created:
    - go.mod
    - cmd/sshjesus/main.go
    - internal/domain/server.go
    - internal/domain/project.go
    - internal/domain/credential.go
    - internal/domain/validation.go
    - internal/errors/errors.go
    - internal/backend/backend.go
  modified: []
decisions:
  - decision: Domain models have zero external dependencies (no struct tags, no storage imports)
    rationale: Keeps domain layer pure and storage-agnostic per research guidance
    alternatives: Could have added json/toml tags, but would couple domain to serialization
  - decision: Backend uses database/sql pattern (minimal interface + optional capabilities)
    rationale: Allows read-only backends (sshconfig) and read-write backends (1Password) with same base contract
    alternatives: Single large interface would force all backends to implement writes
  - decision: BackendError implements Unwrap() for error chain inspection
    rationale: Enables errors.Is/As to work correctly with wrapped sentinel errors
    alternatives: Could use fmt.Errorf with %w, but custom type provides structured fields
  - decision: Server.ProjectIDs tracks many-to-many relationship on Server side only
    rationale: Simpler than bidirectional tracking, matches user decision from research
    alternatives: Could track on both sides but would require sync logic
metrics:
  duration_seconds: 130
  tasks_completed: 2
  files_created: 8
  commits: 2
  completed_date: 2026-02-14
---

# Phase 01 Plan 01: Foundation Architecture Summary

**One-liner:** Established Go module with storage-agnostic domain models (Server/Project/Credential), database/sql-inspired backend interface, and error types with Unwrap() support.

## What Was Built

### Domain Models (Zero External Dependencies)

Created three core domain types with all user-specified fields:

1. **Server** (14 fields):
   - SSH connection: Host, User, Port, IdentityFile, Proxy
   - Metadata: DisplayName, Tags, Notes, Favorite, VPNRequired
   - Relationships: CredentialID, ProjectIDs (many-to-many)
   - Tracking: ID, LastConnected

2. **Project** (detection-ready):
   - Core: ID, Name, Description
   - Detection: GitRemoteURLs (for git remote matching)
   - Timestamps: CreatedAt, UpdatedAt

3. **Credential** (auth reference, not secret store):
   - Type system: CredentialType enum (KeyFile, SSHAgent, Password)
   - Fields: ID, Name, Type, KeyFilePath, Notes
   - Actual secrets live in filesystem, 1Password, or SSH agent

### Validation Layer

Added `Validate()` methods on all domain types:
- Server: Validates Host, Port range (1-65535), DisplayName
- Project: Validates Name
- Credential: Validates Name, KeyFilePath (when Type == CredentialKeyFile)

Uses plain `errors.New()` and `fmt.Errorf()` — no custom error types needed.

### Backend Interface (Database/SQL Pattern)

Implemented three-tier interface design:

1. **Backend** (required — all backends MUST implement):
   - Read operations: GetServer, ListServers, GetProject, ListProjects, GetCredential, ListCredentials
   - Lifecycle: Close()

2. **Writer** (optional — type-assert for write support):
   - CRUD for all three domain types (Create, Update, Delete)
   - Enables read-only backends (sshconfig) and read-write backends (1Password)

3. **Filterer** (optional — type-assert for server-side queries):
   - FilterServers with ServerFilter struct
   - Fields: ProjectID, Tags, Favorite (tri-state), Query (free text)

### Error Handling

Created comprehensive error system:

**Sentinel Errors:**
- ErrBackendUnavailable (backend unreachable)
- ErrConfigNotFound (missing config file)
- ErrServerNotFound, ErrProjectNotFound, ErrCredentialNotFound
- ErrReadOnlyBackend (write attempted on read-only)
- ErrDuplicateID, ErrValidation

**BackendError Type:**
- Fields: Op (operation), Backend (name), Err (underlying cause)
- Implements Error() with formatted message
- Implements Unwrap() for errors.Is/As chains
- Re-exports stdlib error functions (Is, As, New, Unwrap)

## Verification Results

All success criteria met:

- ✅ Go module `github.com/florianriquelme/sshjesus` initialized with Go 1.24+
- ✅ Server struct has all 14 user-specified fields
- ✅ Project struct has GitRemoteURLs for detection
- ✅ Credential struct uses CredentialType enum
- ✅ All domain types have Validate() methods
- ✅ Sentinel errors cover all specified failure modes
- ✅ BackendError supports Unwrap() for error chains
- ✅ Backend interface is minimal, Writer/Filterer are optional
- ✅ Zero struct tags on domain models
- ✅ `go build ./...` compiles cleanly
- ✅ `go vet ./...` passes with no warnings
- ✅ Domain package has only stdlib imports (time, errors, fmt)

## Implementation Details

### Key Architectural Decisions

1. **Storage-Agnostic Domain Layer**
   - No json/toml/db struct tags
   - No imports from backend, errors, or external packages
   - Pure business logic — serialization handled by storage layer

2. **Type Assertion Pattern for Optional Capabilities**
   ```go
   // Check if backend supports writes
   if writer, ok := backend.(Writer); ok {
       err := writer.CreateServer(ctx, &server)
   }

   // Check if backend supports filtering
   if filterer, ok := backend.(Filterer); ok {
       servers, err := filterer.FilterServers(ctx, filters)
   }
   ```

3. **Many-to-Many Relationship**
   - Server.ProjectIDs tracks which projects a server belongs to
   - No reverse tracking on Project (avoids sync complexity)
   - Matches user decision from research phase

4. **Context Propagation**
   - All backend methods accept context.Context
   - Future-proofs for cancellation, timeouts, tracing

### File Structure

```
github.com/florianriquelme/sshjesus/
├── go.mod
├── cmd/sshjesus/main.go (entry point scaffold)
└── internal/
    ├── domain/
    │   ├── server.go (Server type)
    │   ├── project.go (Project type)
    │   ├── credential.go (Credential type + enum)
    │   └── validation.go (Validate methods)
    ├── errors/
    │   └── errors.go (sentinels + BackendError)
    └── backend/
        └── backend.go (Backend/Writer/Filterer interfaces)
```

## Deviations from Plan

None - plan executed exactly as written.

## What's Next

This plan provides the foundation for:

1. **Phase 1 Plan 2**: Mock backend implementation
2. **Phase 2**: Real backend implementations (1Password, sshconfig)
3. **Phase 3**: TUI using these domain models
4. **Phase 4+**: All features build on these contracts

All future work depends on these domain types and interfaces.

## Self-Check: PASSED

**Checking created files:**
```bash
$ ls -la cmd/sshjesus/main.go internal/domain/*.go internal/errors/errors.go internal/backend/backend.go go.mod
```
✅ All files exist

**Checking commits:**
```bash
$ git log --oneline | head -2
6c56be1 feat(01-01): create error types and backend interface
a815d9a feat(01-01): initialize Go module and create domain models
```
✅ Both commits exist

**Verifying compilation:**
```bash
$ go build ./...
$ go vet ./...
```
✅ Builds cleanly with no warnings

**Verifying domain purity:**
```bash
$ grep -r "json:" internal/domain/
# (no results)
$ grep "github.com/florianriquelme/sshjesus/internal" internal/domain/*.go
# (no results)
```
✅ Domain has no struct tags and no internal dependencies
