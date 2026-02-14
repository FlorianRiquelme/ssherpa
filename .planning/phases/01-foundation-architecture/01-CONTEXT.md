# Phase 1: Foundation & Architecture - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Establish the pluggable backend architecture with domain models (Server, Project, Credential) and a mock backend implementation for testing. This phase delivers the internal contracts and types that all subsequent phases build on. No TUI, no SSH config parsing, no user-facing features yet.

</domain>

<decisions>
## Implementation Decisions

### Domain model scope
- **Server** = SSH config fields (host, user, port, identity file, proxy) + metadata (tags, notes, last connected timestamp, favorite flag, display name, VPN requirement flag)
- VPN requirement flag lets the TUI warn users before connecting to a server that needs VPN
- **Project** = named group of servers (e.g., "payments-api"). Git remote URL is one detection method, but servers can also be manually assigned to projects
- A server can belong to **multiple projects** (shared infra spanning teams)
- **Credential** = auth reference, not a secret store. Points to a key file path, SSH agent, or marks "password auth". Actual secrets live in the filesystem, 1Password, or agent — not in sshjesus

### Backend capabilities
- Backends handle **storage only** (CRUD for servers, projects, credentials). Operational tasks (connectivity checks, import, sync) live outside the backend interface
- Backends can be **read-only** — the interface has optional write methods. SSH config backend may be read-only; 1Password supports full CRUD
- **Querying/filtering is optional** in the backend interface. Backends that support it can filter server-side; others return everything and the app layer filters in-memory
- **Request/response only** — no change notifications, no file watchers, no push events

### Multi-backend strategy
- **One backend active at a time** — user picks ssh config OR 1Password, not both simultaneously
- Backend selection via **config file** (~/.config/sshjesus or similar)
- **First run with no config → interactive setup wizard** prompts user to pick a backend and creates the config file
- **Switching backends via TUI settings screen** — discoverable, not just config file editing

### Error handling
- **Backend unavailable at startup → error and exit** with clear message explaining what's wrong
- **Mid-use operation failure → show error, keep user data** so they can retry without re-entering
- **Errors are technical** — target audience is developers, surface actual error messages (SDK errors, file system errors, etc.) directly. No consumer-friendly abstraction layer
- **Malformed config file → show error, offer to reset** by re-running the setup wizard

### Claude's Discretion
- Go package structure and module layout
- Exact interface method signatures and return types
- Mock backend implementation details
- Error type hierarchy design
- Config file format choice (TOML, YAML, JSON)

</decisions>

<specifics>
## Specific Ideas

- VPN awareness: "we sometimes need VPNs to access some servers — would be great if I would know when trying to ssh"
- The tool is developer-only — no need for consumer-friendly UX patterns. Technical output is expected and preferred

</specifics>

<deferred>
## Deferred Ideas

- TUI settings screen for switching backends — Phase 2+ (needs TUI first)
- Interactive setup wizard — Phase 2+ (needs TUI or at minimum a CLI prompt flow)

</deferred>

---

*Phase: 01-foundation-architecture*
*Context gathered: 2026-02-14*
