---
phase: 04-project-detection
plan: 01
subsystem: project-detection
tags: [git, project-context, config, toml, color-generation]
dependency_graph:
  requires: [phase-01, phase-02]
  provides: [project-detection-foundation, git-url-parsing, project-colors, project-config-storage]
  affects: [tui, config-system]
tech_stack:
  added:
    - github.com/whilp/git-urls (v1.0.0)
  patterns:
    - FNV-1a hashing for deterministic color generation
    - HSL-to-RGB color conversion
    - TOML array-of-tables for project storage
key_files:
  created:
    - internal/project/detector.go
    - internal/project/detector_test.go
    - internal/project/colors.go
    - internal/project/colors_test.go
  modified:
    - internal/config/config.go
    - internal/config/config_test.go
    - go.mod
    - go.sum
decisions:
  - decision: "Only use origin remote for project detection"
    rationale: "Simplifies implementation, covers 99% of use cases"
    alternatives: "Check all remotes, prioritize by name"
  - decision: "Return empty string (not error) for non-git directories and missing origin"
    rationale: "Non-git context is valid use case, not an error condition"
    alternatives: "Return error, require git repo"
  - decision: "Use hex colors instead of ANSI 256 codes"
    rationale: "Better precision, lipgloss handles degradation automatically"
    alternatives: "ANSI 256 color codes"
  - decision: "Empty Color field means auto-generate"
    rationale: "Allows user override while defaulting to deterministic generation"
    alternatives: "Always auto-generate, separate override field"
metrics:
  duration_seconds: 266
  tasks_completed: 2
  tests_added: 18
  files_created: 4
  files_modified: 4
  completed_at: "2026-02-14T10:04:23Z"
---

# Phase 04 Plan 01: Project Detection Foundation Summary

**One-liner:** Git remote URL detection with org/repo extraction using git-urls library, FNV-1a based deterministic color generation, and TOML [[project]] config storage with multi-URL and server mapping support.

## Objective Achieved

Implemented the foundation for project detection features with git remote URL parsing, deterministic badge color generation, and TOML config persistence. All functionality delivered via TDD with comprehensive test coverage.

## Tasks Completed

### Task 1: Git remote detection and project color generation (TDD)
- **Commit:** 8913073
- **Duration:** ~180 seconds
- **Files:**
  - internal/project/detector.go
  - internal/project/detector_test.go
  - internal/project/colors.go
  - internal/project/colors_test.go
  - go.mod
  - go.sum

**Implementation:**
- Created `DetectCurrentProject()` to shell out to `git config --get remote.origin.url`
- Created `ExtractOrgRepo()` to parse SSH, HTTPS, and nested group URLs using git-urls library
- Handles non-git directories and missing origin gracefully (returns empty string, not error)
- Created `ProjectColor()` using FNV-1a hash for deterministic hue generation
- HSL-to-RGB conversion with fixed 65% saturation
- Different lightness for light terminals (40%) vs dark terminals (70%)
- Returns lipgloss.AdaptiveColor with hex color strings

**Test coverage:**
- SSH URL parsing: `git@github.com:acme/backend-api.git` → `acme/backend-api`
- HTTPS URL parsing: `https://github.com/acme/backend-api.git` → `acme/backend-api`
- HTTPS without .git suffix
- GitLab nested groups: `company/team/service`
- Empty URLs and malformed inputs handled gracefully
- Non-git directories return empty string
- Missing origin remote returns empty string
- Color determinism (same ID → same color)
- Color diversity (different IDs → different colors)
- Valid hex format output

### Task 2: TOML config storage for projects (TDD)
- **Commit:** cdafc24
- **Duration:** ~86 seconds
- **Files:**
  - internal/config/config.go
  - internal/config/config_test.go

**Implementation:**
- Added `ProjectConfig` struct with fields:
  - `ID` - Project identifier (org/repo)
  - `Name` - Human-readable name
  - `GitRemoteURLs` - Multiple git remotes per project
  - `Color` - Optional user override (empty = auto-generate)
  - `ServerNames` - Server-to-project mapping (many-to-many)
- Added `Projects []ProjectConfig` to main `Config` struct
- Uses TOML `[[project]]` array-of-tables syntax
- Full round-trip TOML persistence

**Test coverage:**
- Save and load with multiple projects
- All fields preserved (ID, Name, GitRemoteURLs, Color, ServerNames)
- Empty Projects slice handled correctly
- Multiple git remote URLs per project
- Multiple server names per project
- TOML format verification

## Deviations from Plan

None - plan executed exactly as written.

## Success Criteria Met

- ✅ `DetectCurrentProject()` returns `org/repo` from origin remote URL (SSH and HTTPS)
- ✅ Non-git directories and missing origin return empty string, not error
- ✅ `ProjectColor()` generates deterministic colors (same input = same output)
- ✅ Config TOML supports `[[project]]` array-of-tables with round-trip persistence
- ✅ All tests pass with `-race` flag
- ✅ `go build ./...` succeeds
- ✅ `go vet ./...` clean

## Technical Implementation Details

### Git URL Parsing
- Uses `github.com/whilp/git-urls` for robust URL parsing
- Handles SSH URLs: `git@github.com:org/repo.git`
- Handles HTTPS URLs: `https://github.com/org/repo.git`
- Handles nested groups: `git@gitlab.com:company/team/service.git`
- Graceful degradation for malformed URLs (no panics)

### Color Generation Algorithm
1. Hash project ID with FNV-1a (32-bit)
2. Convert hash to hue: `hash % 360` (degrees)
3. Fixed saturation: 65% (vibrant but professional)
4. Variable lightness:
   - Light terminals: 40% (darker for readability on white)
   - Dark terminals: 70% (lighter for readability on black)
5. Convert HSL to RGB using standard algorithm
6. Format as hex color string: `#RRGGBB`

### TOML Schema
```toml
version = 1
backend = "sshconfig"

[[project]]
  id = "acme/backend-api"
  name = "Backend API"
  git_remote_urls = ["git@github.com:acme/backend-api.git"]
  color = "#FF5733"
  server_names = ["api-prod", "api-staging"]

[[project]]
  id = "example/frontend"
  name = "Frontend App"
  git_remote_urls = ["https://github.com/example/frontend.git"]
  server_names = ["web-prod"]
```

## Next Steps

This foundation enables:
- **04-02:** Project-aware TUI features (grouping, filtering, badges)
- **Future phases:** Server-to-project mapping, multi-project workflows

The core detection and persistence mechanisms are now in place and tested.

## Self-Check: PASSED

✅ File verification:
- internal/project/detector.go exists
- internal/project/detector_test.go exists
- internal/project/colors.go exists
- internal/project/colors_test.go exists
- internal/config/config.go modified
- internal/config/config_test.go modified

✅ Commit verification:
- 8913073 exists (Task 1)
- cdafc24 exists (Task 2)

✅ Test verification:
- All project tests pass with race detector
- All config tests pass with race detector
- Full build succeeds
- No vet warnings
