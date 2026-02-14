# Phase 7: SSH Key Selection - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can select which SSH key to use for each connection. Keys are discovered from local files, SSH agent, and 1Password backend, presented in a unified picker, and persisted via the IdentityFile directive in SSH config. Port forwarding, connection status, and ProxyJump/bastion support are deferred to a future version.

</domain>

<decisions>
## Implementation Decisions

### Key discovery
- Scan `~/.ssh/` only (no custom paths)
- Detect keys via file content sniffing (read file headers for PEM/OpenSSH key format), not naming conventions
- Include keys loaded in the SSH agent (`ssh-add -l`)
- Include keys from 1Password backend — unified list across all sources (file, agent, 1Password)

### Selection UX
- Key picker available in both the add/edit form AND as a quick action from detail view
- Picker style: Claude's discretion (choose what fits existing TUI patterns — overlay vs inline)
- Single key per connection only (no multi-key IdentityFile stacking)
- Include a "None (SSH default)" option to clear explicit key assignment

### Key display
- Full details per key: filename, type (ed25519/rsa/etc.), fingerprint, comment, source
- Source indicated via text badge: `[file]`, `[agent]`, `[1password]`
- Currently-assigned key highlighted in picker (checkmark or visual indicator)
- Key display in server detail view: Claude's discretion on placement (consistent with existing layout)

### Default behavior
- New connections default to no key (SSH default resolution) — no IdentityFile set
- Existing IdentityFile directives in SSH config are read, displayed, and pre-selected when editing
- Missing key files (referenced in config but not on disk) show a warning badge
- Do NOT set IdentityOnly when a key is selected — let SSH try other keys from agent too

### Claude's Discretion
- Picker component style (overlay list vs inline dropdown — pick what fits existing patterns)
- Key display placement in server detail view
- Fingerprint format (SHA256 vs MD5)
- How to handle key passphrase detection (if relevant to display)

</decisions>

<specifics>
## Specific Ideas

- Unified key list: all sources (file, agent, 1Password) appear in one flat list with source badges — no grouping by source
- Warning badge for missing keys gives users a clear signal something needs attention without hiding info
- "None" option respects SSH's default key resolution for users who don't want explicit key pinning

</specifics>

<deferred>
## Deferred Ideas

- Port forwarding configuration (local, remote, dynamic) — future version
- Connection status indicators (reachable/unreachable via async ping) — future version
- ProxyJump/bastion host configuration — future version
- Multiple IdentityFile per connection — future version if needed

</deferred>

---

*Phase: 07-ssh-key-selection*
*Context gathered: 2026-02-14*
