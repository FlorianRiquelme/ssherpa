# Phase 6: 1Password Backend - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

1Password backend adapter that stores and syncs SSH server configurations via 1Password SDK. Enables team credential sharing through shared vaults. Servers sync to local storage (SSH config include file + ssherpa TOML) for offline availability. CRUD operations for connections are Phase 5; advanced SSH features are Phase 7.

</domain>

<decisions>
## Implementation Decisions

### Setup & onboarding
- First launch triggers an interactive wizard prompting which backend to use (ssh-config or 1Password for v1)
- Wizard walks user through 1Password setup step-by-step (detect app, pick vaults, confirm)
- Multiple backends can be active simultaneously — servers from all backends merge into one unified list
- No visual distinction between backends in the TUI — source is an implementation detail

### What gets stored
- Each 1Password item = one full server config (hostname, user, port, key path, project tags, remote project path)
- Remote project path stored per server — enables `ssh user@host -t 'cd /path && $SHELL'` to land in the right directory
- Connection history stays local (personal to each machine, not synced)
- When 1Password is the backend, project-to-server assignments are stored in the 1Password item (not local TOML) — team sees same project groupings
- When a server exists in both ssh-config AND 1Password, 1Password wins

### Team sharing model
- ssherpa scans ALL accessible vaults (not configured to specific vaults)
- Items are discoverable via a specific tag (e.g., `ssherpa`) — that's how ssherpa identifies its managed items
- Existing vault-per-customer organization stays as-is — ssherpa works with whatever vault structure exists
- Migration wizard offered to convert existing unstructured SSH items to ssherpa format with proper tags
- Real-time sync — changes by team members appear immediately
- Personal vault items are supported — users can tag items in their Private vault for personal-only servers

### Sync to local storage
- 1Password is the source of truth; synced down to local for offline/fallback
- Sync targets: both `~/.ssh/ssherpa_config` (include file) AND ssherpa local TOML config
- SSH config sync uses a separate include file (`~/.ssh/ssherpa_config`) with an `Include` directive added to `~/.ssh/config` — fully isolated, never touches user's existing SSH entries
- Local TOML gets the extra ssherpa-specific fields (project path, project tags, custom metadata)
- Sync triggers: on launch + on every change (add/edit/remove in 1Password)
- Conflict detection: if a server exists in both 1Password and user's original ssh-config (not the synced include file), show a warning and let the user decide

### Fallback behavior
- When 1Password is unavailable (not running, locked): immediately show ssh-config servers + persistent banner prompting to unlock 1Password
- Auto-detect when 1Password becomes available mid-session and automatically load its servers
- Clear warning bar when auth fails (expired token, revoked access) — keep working with available backends
- When no backend is configured at all: show empty TUI with setup prompt (call-to-action to configure)

### Claude's Discretion
- 1Password SDK authentication approach (desktop app integration, service accounts)
- Item field mapping and custom field naming in 1Password
- Tag naming convention for ssherpa-managed items
- Sync conflict resolution UI details
- Include directive placement strategy in ssh-config
- Polling interval for auto-detect when 1Password becomes available

</decisions>

<specifics>
## Specific Ideas

- "We already have existing vaults per customer, each SSH connection is stored there — but not with a uniform structure. ssherpa needs to dictate how to store this."
- Remote project path per server — so users can SSH directly into the correct folder without remembering where it is
- Tag-based discovery across all vaults rather than vault-specific configuration
- Migration wizard to bring existing unstructured SSH items into ssherpa format

</specifics>

<deferred>
## Deferred Ideas

- None — discussion stayed within phase scope

</deferred>

---

*Phase: 06-1password-backend*
*Context gathered: 2026-02-14*
