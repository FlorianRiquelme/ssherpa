# Phase 3: Connection & Navigation - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can search/filter the server list and connect to a selected server via system SSH. Phase 2 delivers the browsable TUI with server list and detail view. This phase adds fuzzy search, SSH connection handoff, post-connection behavior, and full keyboard navigation. Project detection is Phase 4. Config CRUD is Phase 5.

</domain>

<decisions>
## Implementation Decisions

### Search & filtering
- Always-on filter bar visible at top or bottom of the TUI — typing immediately filters the list
- Fuzzy matching — typing "prd" matches "production-server" (characters match in order, not necessarily adjacent)
- Search matches against: Host name, Hostname, and User fields
- No matches: show "No matches" empty state message in the list area, search bar stays active
- Esc defocuses search bar and clears it, returning focus to the server list

### Connection flow
- Enter on a selected server connects immediately — no confirmation step
- Silent handoff — TUI disappears, SSH takes over the terminal immediately (feels like running `ssh` directly)
- Phase 2's Enter-for-details is reassigned: Tab (and a Vim alternative) opens the detail view instead
- On SSH connection failure: let SSH's native error output display in the terminal, then return to TUI on keypress

### Post-connection experience
- Default behavior: after SSH session ends, sshjesus exits entirely (user returns to their shell)
- This is configurable — option to return to TUI instead of exiting (stored in app config)
- On relaunch: preselect the last server connected from the current working directory path
- Last-connected indicator shown next to servers you've recently connected to (subtle marker or timestamp)
- Connection history stored in a separate history file (not in TOML app config) — tracks connections per path

### Keyboard navigation
- Full Vim navigation available: j/k, g/G, Ctrl+d/u for half-page scroll
- Arrow keys, Page Up/Down, Home/End also work — easy for non-Vim users
- q quits from list view (when search bar is not focused), Ctrl+C/Esc works everywhere as fallback
- Esc clears and defocuses the search bar, returning to list navigation mode
- Persistent footer bar showing key hints (e.g., "Enter: connect | Tab: details | /: search | q: quit")

### Claude's Discretion
- Filter bar placement (top vs bottom)
- Fuzzy matching algorithm/library choice
- Exact footer bar content and styling
- History file format and location
- Last-connected indicator visual design (icon, timestamp, or both)
- Vim alternative key for detail view (e.g., `i`, `l`, or `o`)
- How "return to TUI" config option is named and structured

</decisions>

<specifics>
## Specific Ideas

- Connection should feel like running `ssh` directly — zero friction, silent handoff, native error output
- The "last connected from this path" preselection enables a rapid reconnect workflow: cd into project, launch sshjesus, hit Enter
- Non-Vim users should never feel lost — arrow keys and persistent footer bar make it immediately usable

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-connection-navigation*
*Context gathered: 2026-02-14*
