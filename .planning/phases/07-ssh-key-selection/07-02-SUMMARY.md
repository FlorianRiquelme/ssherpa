---
phase: 07-ssh-key-selection
plan: 02
subsystem: tui
tags: [bubbletea, tui, overlay, ssh-key, identity-file, 1password, key-picker]

# Dependency graph
requires:
  - phase: 07-ssh-key-selection
    plan: 01
    provides: SSHKey domain model, DiscoverKeys multi-source discovery
  - phase: 05-config-management
    provides: SSH config writer (AddHost, EditHost), ServerForm component
  - phase: 04-project-detection
    provides: ProjectPicker overlay pattern adapted for key picker
provides:
  - SSH key picker overlay with source badges and checkmark
  - Form integration replacing IdentityFile text input with picker
  - Detail view key quick-select via K keybinding
  - IdentityFile persistence to SSH config on save
  - 1Password IdentityAgent socket discovery from SSH config
affects: [08-distribution]

# Tech tracking
tech-stack:
  added: []
  patterns: [overlay picker pattern reuse, IdentityAgent SSH config parsing, async multi-source discovery]

key-files:
  created:
    - internal/tui/key_picker.go
  modified:
    - internal/tui/model.go
    - internal/tui/form.go
    - internal/tui/detail_view.go
    - internal/tui/messages.go
    - internal/tui/styles.go
    - internal/tui/keys.go
    - internal/sshkey/agent.go
    - internal/sshkey/discovery.go

key-decisions:
  - "1Password IdentityAgent discovery: Parse ~/.ssh/config directly for IdentityAgent directives, connect to socket, tag as Source1Password"
  - "Default label reflects SSH config: Show 'Default (1Password agent)' when IdentityAgent is configured instead of 'None (SSH default)'"
  - "Checkmark only for non-empty paths: Agent/1Password keys with empty Path don't false-match empty currentKeyPath"
  - "DiscoverKeysFromSocket exported: Enables connecting to arbitrary agent sockets with configurable source tag"
  - "Re-discover keys after hosts load: Initial discovery runs with nil, re-triggers after configLoadedMsg with full host context"

patterns-established:
  - "IdentityAgent socket discovery: Parse SSH config for IdentityAgent, detect 1Password by path substring, connect and list keys"
  - "Overlay picker reuse: SSHKeyPicker follows same Init/Update/View pattern as ProjectPicker"
  - "Form field picker pattern: Read-only text input that opens overlay on Enter/Space"

# Metrics
duration: 900s
completed: 2026-02-14
---

# Phase 07 Plan 02: SSH Key Picker UI Summary

**Key picker overlay with 1Password IdentityAgent discovery, form/detail integration, and context-aware default labels**

## Performance

- **Duration:** ~15 min (agent) + ~10 min (manual fixes)
- **Started:** 2026-02-14T17:42:00+01:00
- **Completed:** 2026-02-14T18:15:00+01:00
- **Tasks:** 3 (2 auto + 1 checkpoint)
- **Files modified:** 9

## Accomplishments

- SSH Key Picker overlay component with source badges ([file] green, [agent] blue, [1password] purple)
- 1Password SSH keys discovered via IdentityAgent socket in ~/.ssh/config
- Form IdentityFile field opens picker instead of text input
- Detail view quick-select with K keybinding
- Context-aware default: "Default (1Password agent)" when IdentityAgent configured
- Checkmark on currently-assigned key, no false matches on agent keys
- Key selection persists as IdentityFile directive in SSH config

## Task Commits

1. **Task 1: SSHKeyPicker overlay and TUI wiring** - `8d818fe` (feat)
   - Created key_picker.go with Update/View methods
   - Added message types, styles, K keybinding
   - Async key discovery on Init
   - Detail view K handler opens picker

2. **Task 2: Form integration and IdentityFile persistence** - `5032660` (feat)
   - Form IdentityFile field opens picker on Enter/Space
   - Edit form pre-selects existing IdentityFile
   - Detail view key update via sshconfig.EditHost
   - Key selection persists to SSH config

3. **Task 3: Checkpoint fixes** - `9cac919` (fix)
   - 1Password IdentityAgent socket discovery from SSH config
   - DiscoverKeysFromSocket with configurable source tag
   - Fixed double checkmark on agent keys with empty paths
   - Context-aware default label reflecting 1Password config
   - Re-trigger key discovery after hosts load from backend

## Files Created/Modified

### Created
- `internal/tui/key_picker.go` — SSHKeyPicker overlay component with source badges, checkmarks, fingerprints

### Modified
- `internal/tui/model.go` — Key picker routing, async discovery with IdentityAgent, expandTilde, has1PasswordKeys
- `internal/tui/form.go` — IdentityFile field opens picker, selectedKey tracking
- `internal/tui/detail_view.go` — Key info display in detail view
- `internal/tui/messages.go` — keyPickerClosedMsg, keySelectedMsg, keysDiscoveredMsg, formRequestKeyPickerMsg
- `internal/tui/styles.go` — Source badge styles (file green, agent blue, 1password purple)
- `internal/tui/keys.go` — SelectKey binding (K)
- `internal/sshkey/agent.go` — DiscoverKeysFromSocket exported, refactored shared socket logic
- `internal/sshkey/discovery.go` — IdentityAgentSource type, DiscoverKeys accepts variadic agents

## Decisions Made

1. **1Password IdentityAgent discovery from SSH config**: Rather than relying on SSH_AUTH_SOCK (not set in all contexts), parse ~/.ssh/config directly for IdentityAgent directives. Detect 1Password by path substring. This works even when the backend loads servers from 1Password API.

2. **Context-aware default label**: When 1Password IdentityAgent is configured, show "Default (1Password agent)" instead of "None (SSH default)". Reflects actual SSH behavior — without explicit IdentityFile, SSH uses the configured agent.

3. **Checkmark only for non-empty paths**: Agent and 1Password keys discovered from sockets have empty Path field. Only match checkmark when both key.Path and currentKeyPath are non-empty, preventing false double-checkmarks.

4. **Always parse SSH config for IdentityAgent**: discoverKeysCmd parses ~/.ssh/config regardless of backend mode, since IdentityAgent is an SSH config concept that exists outside the backend server list.

5. **Re-discover keys after hosts load**: Initial discovery fires at Init with nil hosts, second discovery triggers after configLoadedMsg with full host context including IdentityFile references.

## Deviations from Plan

### Post-Checkpoint Fixes

**1. 1Password IdentityAgent discovery not implemented in plan**
- **Found during:** Checkpoint verification
- **Issue:** Plan assumed keys would be in ~/.ssh/ or SSH_AUTH_SOCK agent. User's keys only in 1Password, served via IdentityAgent socket.
- **Fix:** Added IdentityAgent parsing from SSH config, DiscoverKeysFromSocket, source tagging
- **Files modified:** internal/sshkey/agent.go, internal/sshkey/discovery.go, internal/tui/model.go

**2. Double checkmark on "None" and 1Password key**
- **Found during:** Checkpoint verification
- **Issue:** Agent keys have Path="" which matched currentKeyPath="" (no IdentityFile set)
- **Fix:** Only show checkmark when key.Path is non-empty
- **Files modified:** internal/tui/key_picker.go

**3. Misleading "None (SSH default)" label**
- **Found during:** Checkpoint verification
- **Issue:** When 1Password IdentityAgent is configured, "None" is misleading — the default IS 1Password
- **Fix:** Dynamic label based on discovered 1Password keys
- **Files modified:** internal/tui/key_picker.go, internal/tui/model.go

---

**Total deviations:** 3 post-checkpoint fixes (all UX correctness)
**Impact on plan:** Essential fixes for 1Password users. No scope creep — all directly related to key selection feature.

## Issues Encountered

- **SSH_AUTH_SOCK empty in sandbox**: Claude Code sandbox doesn't inherit SSH_AUTH_SOCK, so DiscoverAgentKeys returned empty. 1Password uses IdentityAgent in SSH config instead. Solved by parsing SSH config directly for IdentityAgent directives.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

**Phase 7 complete. Ready for Phase 8 (Distribution):**
- All SSH key selection functionality working
- 1Password integration seamless via IdentityAgent
- All existing tests pass, project builds cleanly

**No blockers or concerns.**

## Self-Check: PASSED

- All commits verified in git history
- go build ./... passes
- go test ./... passes (all packages)
- go vet ./... clean
- Human verification approved

---
*Phase: 07-ssh-key-selection*
*Completed: 2026-02-14*
