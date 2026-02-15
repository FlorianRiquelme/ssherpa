---
phase: 08-distribution
plan: 01
subsystem: cli-ux
tags: [version-info, onboarding, cli-flags, user-experience]

dependency_graph:
  requires: []
  provides:
    - version-package
    - cli-flags
    - onboarding-flow
  affects:
    - cmd/ssherpa/main.go
    - internal/config/config.go

tech_stack:
  added:
    - internal/version package with ldflags injection
  patterns:
    - Build-time variable injection via ldflags
    - First-run onboarding with skippable steps
    - CLI flag parsing before backend initialization

key_files:
  created:
    - internal/version/version.go
  modified:
    - cmd/ssherpa/main.go
    - internal/config/config.go

decisions:
  - title: "ldflags injection for version info"
    rationale: "Standard Go practice for embedding build metadata without hardcoding"
    alternatives: ["Environment variables", "Hardcoded constants"]
  - title: "OnboardingDone field in config"
    rationale: "Simple boolean flag prevents re-showing onboarding on every launch"
    alternatives: ["Separate state file", "First-launch detection via config existence"]
  - title: "SSH config host counting via kevinburke/ssh_config"
    rationale: "Reuses existing dependency, reliable parsing of SSH config format"
    alternatives: ["Manual text parsing", "exec 'grep Host'"]

metrics:
  duration: 182
  tasks_completed: 2
  files_created: 1
  files_modified: 2
  commits: 2
  completed_at: "2026-02-15T12:02:23Z"
---

# Phase 08 Plan 01: CLI Flags & Onboarding Summary

**One-liner:** Version information embedding via ldflags and first-run onboarding flow with SSH config detection and optional 1Password setup.

## Tasks Completed

### Task 1: Create version package with ldflags injection
**Commit:** 7999e95
**Files:** internal/version/version.go

Created version package with:
- Build-time variables (Version, Commit, Date, GoVersion) for ldflags injection
- Short() - returns version string only
- Full() - returns version with short commit hash
- Detailed() - multi-line output with all metadata
- Platform() - returns OS/arch string

The package supports the following ldflags pattern:
```bash
-X github.com/florianriquelme/ssherpa/internal/version.Version={{.Version}}
-X github.com/florianriquelme/ssherpa/internal/version.Commit={{.FullCommit}}
-X github.com/florianriquelme/ssherpa/internal/version.Date={{.Date}}
-X github.com/florianriquelme/ssherpa/internal/version.GoVersion={{.Env.GOVERSION}}
```

### Task 2: Add CLI flags and first-run onboarding
**Commit:** 7aa4770
**Files:** cmd/ssherpa/main.go, internal/config/config.go

Implemented:
- `--version` flag: prints detailed version info (commit, build date, Go version, platform) and exits
- `--setup` flag: re-triggers onboarding flow regardless of OnboardingDone state
- OnboardingDone field in Config struct for state persistence

**Onboarding flow (runOnboarding function):**
1. **Welcome + SSH config detection:** Prints welcome message, counts non-wildcard hosts in ~/.ssh/config using kevinburke/ssh_config parser
2. **Optional 1Password setup:** Prompts user with [y/N], launches existing setup wizard if yes, saves sshconfig backend if no
3. **Mark done:** Sets OnboardingDone=true and saves config

**Behavioral requirements:**
- Onboarding shows on first run (cfg==nil || !cfg.OnboardingDone)
- Skippable at every step (N at 1Password prompt saves sshconfig backend)
- `--setup` re-triggers regardless of OnboardingDone state
- After onboarding, TUI launches normally

## Verification Results

All verification checks passed:
- `go build ./...` - all packages compile
- `go vet ./...` - no issues
- `go test ./internal/config/...` - all 11 tests pass
- `./ssherpa --version` - prints version detail and exits
- `./ssherpa --setup` - triggers onboarding (verified via code inspection, interactive flow not run)

## Deviations from Plan

None - plan executed exactly as written.

## Implementation Notes

**Version package design:**
- Default values ("dev", "none", "unknown") ensure binary works without ldflags
- Commit short hash handles edge case where Commit is <7 chars (uses full string)
- Detailed() format matches common CLI tool conventions

**Onboarding UX:**
- Non-wildcard host filtering excludes `Host *` patterns from count
- Empty SSH config shows "No SSH config found" message (graceful handling)
- Wizard integration reuses existing tui.NewSetupWizard (no duplication)
- Config reload after wizard ensures main() has wizard-saved backend choice

**Flag ordering:**
- Flags parsed BEFORE config loading (allows --version without valid config)
- --version exits immediately (no backend initialization overhead)
- Onboarding runs after config load but before backend initialization

## Self-Check

Verified all artifacts exist:

**Files created:**
```bash
[ -f "internal/version/version.go" ] && echo "FOUND: internal/version/version.go" || echo "MISSING: internal/version/version.go"
# Result: FOUND: internal/version/version.go
```

**Commits exist:**
```bash
git log --oneline --all | grep -q "7999e95" && echo "FOUND: 7999e95" || echo "MISSING: 7999e95"
# Result: FOUND: 7999e95

git log --oneline --all | grep -q "7aa4770" && echo "FOUND: 7aa4770" || echo "MISSING: 7aa4770"
# Result: FOUND: 7aa4770
```

**Self-Check: PASSED**
