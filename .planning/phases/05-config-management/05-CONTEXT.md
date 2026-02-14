# Phase 5: Config Management - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can add, edit, and delete SSH connections with validation. Changes write back to ~/.ssh/config with formatting preservation. Config modifications include automatic backup and undo support. Manual server-to-project assignment (from Phase 4 success criteria) is also in scope.

</domain>

<decisions>
## Implementation Decisions

### Form interaction
- Full-screen form for add and edit (dedicated screen with labeled fields)
- Field navigation supports both Tab/Shift+Tab and j/k (Vim-style)
- Fields: Alias (required), Hostname (required), User (required), Port, IdentityFile, plus a free-text area for extra SSH config directives
- Free-text area allows any valid SSH config directive (ProxyJump, ForwardAgent, etc.)

### Validation & feedback
- Validation triggers on field exit (when user tabs/moves away from a field)
- Errors display inline below the invalid field, in red/warning color
- Required fields: Alias, Hostname, User
- Hostname performs DNS resolution check on save (catches typos early)

### Delete safety
- Delete triggered with 'd' key from the server list
- Confirmation requires typing the server alias to confirm (prevents accidental deletion)
- Session undo buffer: deleted entries stay in memory until session ends, 'u' key to undo last delete
- One server deleted at a time (no bulk delete)

### Config file handling
- Preserve all comments, blank lines, and indentation exactly as-is
- Only the modified Host block changes on edit
- New Host blocks appended at end of file
- Single backup before each write: ~/.ssh/config.bak (overwritten each time)
- Include directives: read-only (parse and display servers from included files, but only write to the main config file)

### Claude's Discretion
- Add/edit trigger keybinding (e.g. 'a' for add, 'e' for edit, or another pattern)
- Exact form layout and field spacing
- How free-text area renders and handles multi-line input
- Error message wording
- DNS check timeout and UX during the check (spinner, blocking, etc.)
- Undo buffer size and behavior details

</decisions>

<specifics>
## Specific Ideas

- Form should feel like the existing full-screen detail view pattern already in the TUI
- Delete confirmation (type alias) mirrors the GitHub "type repo name to delete" pattern
- Undo with 'u' is consistent with Vim conventions already used in the TUI

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 05-config-management*
*Context gathered: 2026-02-14*
