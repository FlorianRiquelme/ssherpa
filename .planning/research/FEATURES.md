# Feature Landscape

**Domain:** SSH Connection Management TUI
**Researched:** 2026-02-14
**Confidence:** MEDIUM

## Table Stakes

Features users expect. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Read/parse ~/.ssh/config | Every SSH TUI reads standard config files | Low | Non-destructive parsing that preserves comments/formatting |
| List available connections | Core UX - show what's available | Low | Filter, search, and navigate hosts |
| Quick connect to host | Primary purpose - launch SSH session | Low | Execute via system `ssh` binary |
| Search/filter hosts | Essential with >10 hosts | Low | Fuzzy search by name, IP, tags |
| Add new connection | Users need to add hosts | Medium | Interactive forms or wizard |
| Edit existing connection | Config changes are common | Medium | Non-destructive edits preserving formatting |
| Delete connection | Cleanup needed | Low | With confirmation prompt |
| SSH key management | Users have multiple keys | Medium | Key selection, path autocomplete |
| Port forwarding setup | Common DevOps workflow | Medium | Local (-L), Remote (-R), Dynamic/SOCKS (-D) |
| Connection status | Users want to know if host is up | Medium | Async ping/health checks without blocking UI |
| Keyboard navigation | TUI users expect vim-like shortcuts | Low | hjkl navigation, enter to connect |

## Differentiators

Features that set product apart. Not expected, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Git remote auto-detection** | Auto-discover project servers from git remote URLs | Medium | Parse git remotes, extract hosts, suggest configs - UNIQUE to ssherpa |
| **Project-based organization** | Group connections by project not flat list | Low | Auto-detect from git context or manual grouping |
| **1Password backend integration** | Secure credential storage + team sharing | Medium | Use `op` CLI for credential retrieval |
| **Pluggable credential backends** | Users can choose their vault (1Password, Vault, etc) | High | Abstract credential interface, multiple implementations |
| **Team credential sharing** | Share server configs across team via vault | Medium | Depends on backend (1Password vaults, etc) |
| **Connection history** | Track last connection time per host | Low | Simple timestamp tracking |
| **Tag-based categorization** | Organize by env (prod/staging/dev) or function | Low | Multi-tag support per host |
| **Port forwarding history** | Remember previous tunnel configs | Low | Reuse common forwarding setups |
| **ProxyJump/Bastion support** | Connect through intermediate hosts | Medium | SSH native feature, just expose in UI |
| **Real-time connectivity status** | Live indicators showing host availability | Medium | Async background pings with latency |
| **Batch operations** | Test/connect to multiple hosts | Medium | Parallel execution for status checks |
| **Configuration validation** | Prevent invalid SSH config | Low | Validate before writing to file |
| **Automatic backups** | Backup config before modifications | Low | Timestamped backups, max retention |
| **Include directive support** | Support SSH Include for organized configs | Low | Parse and respect Include statements |
| **Custom SSH options** | Any valid SSH config option | Low | Form-based or raw text entry |
| **Export/Import configs** | Share configs between machines | Low | JSON/YAML export of host definitions |

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Built-in terminal emulator | Complexity explosion, reinventing wheel | Use system `ssh` - let users choose their terminal |
| Custom SSH implementation | Security risk, maintenance burden | Always use system SSH binary |
| GUI version | Scope creep, different target audience | Stay focused on TUI - there are GUI tools already |
| Cloud sync service | Infrastructure cost, security liability | Use pluggable backends - let 1Password/Vault handle it |
| File transfer UI (SFTP/SCP) | Feature bloat, tools exist | Keep focus on connection management |
| Session recording/logging | Privacy concerns, scope expansion | Users can use `script` or terminal features |
| Built-in credential storage | Security risk without proper audit | Always delegate to vault backends |
| Multi-protocol support (RDP, VNC, etc) | Dilutes focus | SSH only - other tools exist for other protocols |
| Web UI | Different tech stack, deployment complexity | TUI only - single binary distribution |
| Chat/collaboration features | Way beyond scope | Credentials shared via vault is sufficient |

## Feature Dependencies

```
SSH Config Parsing
    └──requires──> List Connections
                       └──requires──> Search/Filter
                       └──requires──> Quick Connect
                       └──enables──> Edit Connection
                       └──enables──> Delete Connection

Git Remote Detection
    └──requires──> Git installation
    └──enhances──> Add Connection (auto-populate)
    └──enables──> Project Organization

Pluggable Backend Interface
    └──requires──> Credential abstraction layer
    └──enables──> 1Password Backend
    └──enables──> Team Sharing (via backend)

Connection Status
    └──requires──> Async background workers
    └──enhances──> List Connections (show status)

Port Forwarding
    └──requires──> SSH Config Writing
    └──enhances──> Quick Connect (with tunnels)
```

## MVP Recommendation

### Launch With (v0.1)

Prioritize:
1. **SSH Config parsing** - Read standard config, preserve format
2. **List & search connections** - Core navigation UX
3. **Quick connect** - Execute system ssh to selected host
4. **Basic CRUD** - Add, edit, delete hosts with validation
5. **Git remote detection** - Auto-discover project hosts (differentiator)
6. **Project grouping** - Organize by project context
7. **1Password backend** - First credential backend via `op` CLI

**Rationale:** This validates core value prop (git-based project organization + 1Password backend) with minimal viable feature set.

### Add After Validation (v0.2-0.5)

- **Connection status** - Once core UX is stable (adds async complexity)
- **Port forwarding UI** - Common request, but not critical for launch
- **Tag system** - Nice organization but project grouping may suffice
- **Connection history** - Low value until users have usage patterns
- **ProxyJump support** - Important for enterprise users
- **Configuration backups** - Safety feature before wider adoption
- **Additional backends** - HashiCorp Vault, AWS Secrets Manager, etc.

### Future Consideration (v1.0+)

- **Batch operations** - Niche use case, add if requested
- **Export/Import** - Can be manual JSON editing for now
- **Custom SSH options** - Power user feature, defer
- **Port forwarding history** - Nice-to-have optimization
- **Include directive support** - Edge case, most users don't use

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| SSH Config parsing | HIGH | MEDIUM | P1 |
| List connections | HIGH | LOW | P1 |
| Quick connect | HIGH | LOW | P1 |
| Search/filter | HIGH | LOW | P1 |
| Git remote detection | HIGH | MEDIUM | P1 |
| Project organization | HIGH | LOW | P1 |
| 1Password backend | HIGH | MEDIUM | P1 |
| Add/Edit/Delete hosts | HIGH | MEDIUM | P1 |
| Connection status | MEDIUM | MEDIUM | P2 |
| Port forwarding | MEDIUM | MEDIUM | P2 |
| ProxyJump support | MEDIUM | LOW | P2 |
| Tag system | MEDIUM | LOW | P2 |
| Config backups | MEDIUM | LOW | P2 |
| Key management UI | MEDIUM | MEDIUM | P2 |
| Connection history | LOW | LOW | P3 |
| Batch operations | LOW | HIGH | P3 |
| Export/Import | LOW | LOW | P3 |
| Additional backends | MEDIUM | HIGH | P3 |
| Include support | LOW | MEDIUM | P3 |

**Priority key:**
- P1: Must have for launch - validates core concept
- P2: Should have - add once MVP is validated
- P3: Nice to have - defer until product-market fit

## Competitor Feature Analysis

| Feature | Termius (GUI) | sshm (TUI) | lazyssh (TUI) | sshs (TUI) | storm (CLI) | Our Approach |
|---------|---------------|------------|---------------|------------|-------------|--------------|
| Config management | Custom DB | ~/.ssh/config | ~/.ssh/config | ~/.ssh/config | ~/.ssh/config | ~/.ssh/config (standard) |
| Organization | Folders/Groups | Tags | Tags | None | Groups | **Projects (git-based)** |
| Credential storage | Encrypted cloud | System | System | System | ~/.ssh/config | **Pluggable (1Password first)** |
| Team sharing | Yes (paid) | No | No | No | No | **Yes (via vault backend)** |
| Port forwarding | GUI forms | TUI forms | TUI forms | No | CLI args | TUI forms |
| Status checking | Yes | Yes (async) | Yes (ping) | No | No | Yes (async) |
| Cross-platform | All platforms | All platforms | All platforms | All platforms | All platforms | All platforms (Go) |
| Backend integration | Proprietary | None | None | None | None | **Vault backends (unique)** |
| Auto-discovery | Manual | Manual | Manual | Manual | Manual | **Git remote (unique)** |

**Key differentiators:**
- **Git remote auto-detection** - No competitor has this
- **Project-based organization** - More intuitive than tags for dev workflows
- **Pluggable credential backends** - Only ssherpa treats credentials as pluggable
- **1Password-first** - Leverages existing tool devs already use

## TUI-Specific Considerations

### Must-Haves for TUI
- **Keyboard-first navigation** - Mouse support is bonus, not required
- **Responsive layout** - Handle terminal resize gracefully
- **Clear visual hierarchy** - Even with limited colors
- **Helpful shortcuts** - Displayed in footer/help screen
- **Minimal dependencies** - Single binary, zero config
- **Fast startup** - TUI users expect instant launch

### Nice-to-Haves for TUI
- **Mouse support** - Clickable lists for those who want it
- **Vim-like keybindings** - hjkl navigation
- **Color-coding** - Status indicators (green=up, red=down)
- **Live updates** - Status changes without refresh
- **Fuzzy search** - Like fzf/telescope in vim

### Anti-Patterns for TUI
- **Complex forms** - Keep input simple
- **Nested menus** - Prefer flat navigation
- **Modal dialogs** - Minimize interruptions
- **Excessive animations** - TUI isn't about animations
- **Hidden features** - Make capabilities discoverable

## Sources

**SSH Manager Feature Research:**
- [SSHM - Beautiful TUI SSH Manager](https://github.com/Gu1llaum-3/sshm)
- [Lazyssh - Terminal SSH Manager](https://github.com/Adembc/lazyssh)
- [sshs - Rust SSH TUI](https://github.com/quantumsheep/sshs)
- [Storm - Python SSH Manager](https://github.com/emre/storm)
- [SSH Manager TUI - LinuxLinks](https://www.linuxlinks.com/ssh-manager-tui-terminal-based-ssh-connection-manager/)
- [Best SSH Clients 2026 - Comparitech](https://www.comparitech.com/net-admin/best-ssh-client-and-connection-managers/)

**SSH Configuration Best Practices:**
- [SSH Config Complete Guide 2026](https://devtoolbox.dedyn.io/blog/ssh-config-complete-guide)
- [Efficiently Manage Multiple SSH Configurations](https://hoop.dev/blog/efficiently-manage-multiple-ssh-configurations-with-these-8-time-saving-routines/)
- [Using the SSH Config File - Linuxize](https://linuxize.com/post/using-the-ssh-config-file/)

**Credential Management:**
- [1Password SSH Agent](https://developer.1password.com/docs/ssh/agent/)
- [1Password for SSH & Git](https://developer.1password.com/docs/ssh/)
- [HashiCorp Vault SSH Secrets](https://developer.hashicorp.com/vault/docs/secrets/ssh)
- [Termius Secure Credentials Sharing](https://termius.com/documentation/secure-credentials-sharing)
- [Managing SSH Access with Vault](https://www.hashicorp.com/en/blog/managing-ssh-access-at-scale-with-hashicorp-vault)

**General SSH Tools:**
- [15 Best SSH Clients 2026](https://www.websentra.com/best-ssh-client-connection-managers/)
- [Best SSH Clients for Linux](https://www.linuxjournal.com/content/8-best-ssh-clients-linux)

---
*Feature research for: SSH Connection Management TUI*
*Researched: 2026-02-14*
