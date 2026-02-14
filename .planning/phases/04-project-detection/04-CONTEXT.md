# Phase 4: Project Detection - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Servers organize automatically by project based on git remote URL matching. Users see their current project's servers prioritized in the list, can manually assign servers to projects via an inline picker, and all servers display with project badges. Creating or editing SSH connections is a separate phase (Phase 5).

</domain>

<decisions>
## Implementation Decisions

### Default view behavior
- When launched inside a git repo: show all servers, but current project's servers float to the top
- When launched outside any repo: show all servers grouped by project (project clusters, current project concept doesn't apply)
- Always grouped by project — no toggle between flat/grouped views
- Fuzzy search: current project matches appear first, other matches below a separator

### Project grouping layout
- Inline labels, not section headers or collapsible sections — each server row shows a colored project badge
- Servers sorted by project name, so same-project servers cluster together naturally; current project's cluster goes first
- Unassigned servers (no project) appear at the bottom of the list, after all project groups
- Each project gets a distinct color for its badge — auto-assigned from a palette, user can change it

### Server assignment
- Inline shortcut: press a key on a highlighted server to open a quick project picker overlay
- A server can belong to multiple projects (many-to-many, consistent with Phase 1 domain model)
- Project picker shows known projects (from git detection or manually created) plus an option to create a new project on the spot
- Auto-suggest project assignment based on server hostname patterns (e.g., servers with similar hostnames to existing project members)

### Project naming & identity
- Auto-detected projects default to `org/repo` format from the git remote URL (e.g., `acme/backend-api`)
- Users can rename projects to a custom display name; original remote identifier kept internally for matching
- Only the `origin` remote is used for project detection — other remotes are ignored
- Project colors: auto-assigned by default, but user can edit the color

### Claude's Discretion
- Color palette selection and assignment algorithm
- Project picker overlay design and keybinding choice
- Hostname pattern matching algorithm for auto-suggestions
- How to handle edge cases: bare repos, missing origin remote, malformed URLs
- Separator design between current project results and other results in search

</decisions>

<specifics>
## Specific Ideas

- Project badges should feel like GitHub labels — small, colored, inline with the server name
- The inline project picker should be fast and lightweight — not a full-screen form, more like a popup menu
- Search behavior mirrors how Slack shows results from current channel first, then other channels

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-project-detection*
*Context gathered: 2026-02-14*
