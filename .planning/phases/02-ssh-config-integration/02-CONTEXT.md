# Phase 2: SSH Config Integration - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Parse `~/.ssh/config` (and Include'd files) and display all SSH connections in a working TUI with basic keyboard navigation. Users can browse servers and view details. Connecting to servers (Enter → SSH) is Phase 3. Search/filter is Phase 3. CRUD operations are Phase 5.

</domain>

<decisions>
## Implementation Decisions

### Server list layout
- Two lines per server entry — name + hostname on first line, user/port on second
- Show: Name, Hostname, User, Port (key path reserved for detail view)
- Sorted alphabetically by Host name
- Wildcard entries (Host *) displayed in a separate section at the bottom of the list

### Config parsing scope
- Follow Include directives recursively (e.g., `~/.ssh/config.d/*.conf`)
- Ignore Match blocks entirely — only parse Host blocks
- Use an existing Go SSH config parser library (e.g., kevinburke/ssh_config)
- Malformed or unreadable entries shown in the list with a warning indicator (not silently skipped)

### Detail view behavior
- Enter key opens detail view (does NOT connect — that's Phase 3)
- Detail view shows ALL SSH config options set for the host (IdentityFile, ProxyJump, ForwardAgent, etc.)
- Detail view includes which config file the entry was defined in (source tracking)
- Detail view layout (right panel, bottom panel, or inline expansion): Claude's discretion

### Initial launch experience
- Missing or empty `~/.ssh/config`: show friendly empty state message with guidance on creating one
- Show loading indicator (spinner or status text) while parsing config files
- Use accent colors to distinguish structural elements (hostnames, users, ports) — not monochrome, not overwhelming
- Read sshjesus TOML app config to determine which backend to use (integrate with Phase 1 config system)

### Claude's Discretion
- Detail view layout style (right panel vs bottom panel vs inline expansion)
- Exact color palette and accent color choices
- Loading spinner implementation details
- Keyboard shortcut assignments beyond arrow keys and Enter
- How to display the warning indicator for malformed entries

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-ssh-config-integration*
*Context gathered: 2026-02-14*
