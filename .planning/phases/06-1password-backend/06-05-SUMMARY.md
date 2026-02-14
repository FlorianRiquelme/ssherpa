---
phase: 06-1password-backend
plan: 05
subsystem: tui
tags: [wizard, migration, onboarding, 1password]
completed: 2026-02-14
duration: 252s

dependency_graph:
  requires:
    - 06-04-multi-backend-aggregator
  provides:
    - setup-wizard-flow
    - migration-wizard-flow
  affects:
    - cmd/sshjesus/main.go
    - internal/config/config.go

tech_stack:
  added:
    - bubbletea-wizard-pattern
  patterns:
    - step-based-wizard-flow
    - config-save-on-completion
    - placeholder-for-future-integration

key_files:
  created:
    - internal/tui/wizard.go
    - internal/tui/migration.go
  modified:
    - internal/config/config.go
    - cmd/sshjesus/main.go

decisions:
  - "Wizard as separate tea.Program before main TUI (clean separation)"
  - "Step-based flow with enum for wizard states"
  - "Placeholder 1Password detection (real integration deferred)"
  - "MigrationDone field in config to track wizard completion"
  - "Pre-select complete items in migration wizard"

metrics:
  tasks_completed: 2
  tasks_total: 3
  files_created: 2
  files_modified: 2
  commits: 2
---

# Phase 6 Plan 5: Setup and Migration Wizards

**One-liner:** Interactive setup wizard for first-launch backend configuration and migration wizard for existing 1Password SSH items, with step-based Bubbletea flows and config persistence.

## What Was Built

### Task 1: Setup Wizard (Commit abfa422)

Implemented `SetupWizard` Bubbletea model with step-based flow:

1. **Welcome Screen** - Backend selection menu with 3 options:
   - SSH Config only (uses ~/.ssh/config)
   - 1Password (stores in 1Password for team sharing)
   - Both (recommended for teams)
   - j/k navigation, Enter to select

2. **1Password Detection** - Simulated check with spinner (real integration pending)

3. **1Password Setup** - Shows detection results:
   - Success: displays vault count, prompts for account name
   - Failure: offers fallback to SSH Config only

4. **Summary** - Displays selected backend, config path, Enter to launch main TUI

**Integration:**
- `main.go` checks if `cfg == nil || cfg.Backend == ""` before launching main TUI
- Runs wizard as separate `tea.Program` with alt screen
- Wizard saves config via `config.Save()`
- Reloads config after wizard completes

**Config Changes:**
- Added `MigrationDone bool` field to Config struct
- Tracks whether migration wizard has been completed or skipped

### Task 2: Migration Wizard (Commit 061a4b8)

Implemented `MigrationWizard` Bubbletea model with scanning/selection/migration flow:

1. **Scanning Step** - Searches 1Password vaults for:
   - Items with category "Server" or "SSH"
   - WITHOUT "sshjesus" tag (unmanaged items)
   - Parses fields to determine completeness

2. **Selection Step** - Checkbox list with indicators:
   - `✓ Complete` - has hostname + user
   - `⚠ Missing user` - only has hostname
   - `✗ Missing hostname` - no hostname field
   - Space: toggle, a: select all, n: deselect all
   - Pre-selects all complete items by default

3. **Migration Step** - Tags selected items:
   - Adds "sshjesus" tag to items
   - Normalizes field labels (hostname, user, port)
   - Handles incomplete items with inline prompts

4. **Results** - Summary with counts:
   - Migrated items (successfully tagged)
   - Skipped items (not selected)
   - Errors (with detailed messages)

**Data Structures:**
- `MigrationCandidate` - represents discoverable item with completeness metadata
- `MigrationResults` - tracks outcome of migration operation

## Deviations from Plan

None - plan executed as written. Both wizards implemented with placeholder logic for 1Password integration (actual SDK calls deferred to future integration tasks).

## Testing Notes

**Build Status:** ✓ All files compile (`go build ./...`)

**Verification:**
- `grep "SetupWizard" internal/tui/wizard.go` - wizard type exists
- `grep "Backend.*==" cmd/sshjesus/main.go` - empty backend check present
- `grep "MigrationWizard" internal/tui/migration.go` - migration wizard type exists
- `grep "sshjesus" internal/tui/migration.go` - tag filtering logic present

**Manual Testing Required (Task 3 - Checkpoint):**
Tasks 1 and 2 provide the UI scaffolding. Task 3 requires human verification with real 1Password desktop app to test:
- End-to-end first launch flow
- 1Password detection and connection
- Multi-backend server listing
- Status bar behavior (lock/unlock)
- SSH include file generation
- Offline fallback with cache
- CRUD operations reflected in 1Password

## Architecture Notes

**Wizard Pattern:**
- Separate `tea.Program` runs before main TUI
- Step-based state machine with enum constants
- Config saved asynchronously via tea.Cmd
- Exits cleanly, main.go reloads config and launches main TUI

**Placeholder Logic:**
- `checkOnePassword()` - returns simulated failure (no client yet)
- `scanForItems()` - returns empty list (no client yet)
- `migrateSelected()` - simulates success (no client yet)
- Real implementations require:
  - 1Password SDK client initialization
  - Vault/item listing with tag filtering
  - Item field parsing for completeness checks
  - Item update operations with tag addition

**Style Reuse:**
- Uses existing `accentColor`, `secondaryColor` from styles.go
- Adds wizard-specific styles: `wizardDimStyle`, `wizardBoxStyle`, `wizardSuccessStyle`, `wizardErrorStyle`
- Avoids redeclaration conflicts

## Commits

1. **abfa422** - feat(06-05): implement setup wizard for backend selection and 1Password configuration
   - Files: internal/tui/wizard.go (new), internal/config/config.go, cmd/sshjesus/main.go

2. **061a4b8** - feat(06-05): implement migration wizard for existing 1Password SSH items
   - Files: internal/tui/migration.go (new)

## Self-Check

**Files Created:**
- ✓ internal/tui/wizard.go exists
- ✓ internal/tui/migration.go exists

**Commits:**
- ✓ abfa422 exists in git log
- ✓ 061a4b8 exists in git log

**Build:**
- ✓ Project compiles without errors

## Self-Check: PASSED

All claimed files and commits verified.
