# Domain Pitfalls

**Domain:** SSH Connection Management TUI with Pluggable Credential Backends
**Researched:** 2026-02-14
**Confidence:** MEDIUM-HIGH

## Critical Pitfalls

### Pitfall 1: SSH Key Management Blind Spots

**What goes wrong:**
Tools often fail to track SSH keys properly, leading to forgotten keys from ex-employees or old devices remaining active indefinitely. Every forgotten key becomes a potential security breach since people lose devices and accounts get compromised.

**Why it happens:**
Developers focus on making connections work but don't implement lifecycle management for credentials. Manual key rotation and audit processes are error-prone and forgotten over time.

**How to avoid:**
- Add key metadata (creation date, purpose, last used timestamp) to stored connections
- Implement expiration warnings for keys older than 90 days
- Provide audit views showing all active keys with usage patterns
- Support bulk revocation workflows
- Auto-flag keys that haven't been used in 6+ months

**Warning signs:**
- No timestamp tracking on stored credentials
- Missing "last used" indicators in the UI
- No bulk operations for credential management
- Users can't answer "which servers have this key?"

**Phase to address:**
Phase 1 (Core data model) - Build audit fields from the start; Phase 3 (Security hardening) - Implement audit views and lifecycle management

---

### Pitfall 2: Credential Backend Plugin Trust Model

**What goes wrong:**
If any plugin can register a credential backend without verification, malicious plugins could intercept or exfiltrate SSH credentials. Credentials flowing through untrusted plugins creates a massive attack surface.

**Why it happens:**
Plugin architectures prioritize flexibility over security by default. Developers underestimate the sensitivity of SSH credentials compared to other secrets.

**How to avoid:**
- Design plugin system to never expose raw credentials to plugins
- Plugins should only provide connection metadata; `op` CLI fetches actual secrets
- Verify plugin signatures using checksums before loading
- Run plugins as separate processes with IPC isolation (like HashiCorp Vault)
- Document the trust boundary explicitly: plugins provide pointers, not secrets
- Consider "agent-blind" architecture where TUI never sees plaintext credentials

**Warning signs:**
- Plugins receive complete connection objects with credentials
- No signature verification before plugin loading
- Plugins run in same process as main TUI
- Plugin API includes credential retrieval methods

**Phase to address:**
Phase 1 (Architecture) - Design secure plugin boundary from day one; Phase 4 (Plugin system) - Implement verification and isolation

---

### Pitfall 3: 1Password CLI Session Security Assumptions

**What goes wrong:**
Developers assume biometric prompts provide complete security, but critical limitations exist: root/admin users can bypass security measures if 1Password app is unlocked, and macOS accessibility permissions can circumvent authorization prompts. Tools that spawn sub-shells face different authorization scopes on Windows vs Unix.

**Why it happens:**
Documentation buries security limitations deep in reference docs. Developers test the happy path (biometric prompt works) without understanding the threat model boundaries.

**How to avoid:**
- Document that `op` CLI security requires locked 1Password app for full protection
- Warn users if running with elevated privileges (detect sudo/admin context)
- On macOS, detect and warn about accessibility permission risks
- Handle Windows sub-shell auth differently (requires re-auth per process)
- Implement session timeout tracking to warn users before 12-hour hard limit
- Test what happens when 1Password locks mid-session

**Warning signs:**
- No detection of elevated privilege context
- Same code path for Windows and Unix session handling
- Missing session expiry warnings
- No graceful degradation when `op` CLI becomes unavailable

**Phase to address:**
Phase 2 (1Password integration) - Handle all edge cases and security boundaries correctly

---

### Pitfall 4: SSH Multiplexing Session Limit Chaos

**What goes wrong:**
OpenSSH limits multiplexed sessions to 10 per TCP connection by default. When exceeded, connections fail with cryptic "Session open refused by peer" errors. Restarting sshd doesn't kill existing sessions, and `ControlPersist yes` keeps master connections alive indefinitely, making stale sockets a persistent problem.

**Why it happens:**
Tools assume unlimited connections per host. Control socket cleanup isn't automatic. Users enable `ControlPersist` for convenience without understanding persistence implications.

**How to avoid:**
- Don't rely on SSH multiplexing for connection management
- If implementing connection pooling, track active sessions per host
- Implement control socket cleanup on app startup
- Detect "Session open refused" errors and auto-retry with fresh connection
- Expose connection status in UI (reusing vs. new connection)
- Consider shorter `ControlPersist` timeouts (60s) vs. indefinite

**Warning signs:**
- No tracking of active connections per host
- Missing error handling for multiplexing limit errors
- Control sockets created but never cleaned up
- Users report intermittent "Session open refused" failures

**Phase to address:**
Phase 2 (SSH integration) - Design connection model that doesn't depend on multiplexing; Phase 3 (Reliability) - Add robust error handling

---

### Pitfall 5: Bubbletea Event Loop Blocking

**What goes wrong:**
Running expensive operations (SSH connection attempts, `op` CLI calls, git remote parsing) in `Update()` or `View()` blocks the event loop, freezing the UI. Users see unresponsive interface and assume the app crashed.

**Why it happens:**
Bubbletea's Elm Architecture isn't obvious about async boundaries. Developers coming from imperative programming put blocking calls directly in Update() instead of delegating to tea.Cmd.

**How to avoid:**
- **Rule: Update() and View() must be fast (< 10ms)**
- All I/O operations (SSH, `op` CLI, filesystem, network) must run as tea.Cmd
- Use tea.Sequence() when operations must run sequentially
- Show loading states immediately in Update() before async work begins
- Use spinner components for long-running operations
- Profile Update()/View() timing in testing

**Warning signs:**
- Direct SSH connection calls in Update()
- `exec.Command()` or `os.ReadFile()` in event handlers
- No spinner/loading states for I/O operations
- Laggy keyboard input or screen updates

**Phase to address:**
Phase 1 (Core TUI) - Establish async patterns from the beginning; Phase 2 (Integration) - All external calls use tea.Cmd

---

### Pitfall 6: Terminal Compatibility Assumptions

**What goes wrong:**
TUIs render incorrectly across different terminal emulators due to inconsistent ANSI escape sequence support, color handling, and timing issues. True color support isn't reliably detectable, and some terminals (like Emacs shell mode) aren't fully functional for TUI apps.

**Why it happens:**
Developers test in one terminal (usually iTerm2 or Alacritty) and assume universal compatibility. Terminal capability detection is unreliable - COLORTERM isn't forwarded through SSH, terminfo entries are inconsistent.

**How to avoid:**
- Test in multiple terminals: iTerm2, Alacritty, Windows Terminal, standard terminal app, gnome-terminal, tmux, screen
- Use lipgloss for layout (handles cross-platform rendering)
- Provide fallback rendering for limited color support
- Detect and warn about incompatible environments (Emacs shell mode, basic terminal)
- Add `--simple` flag for degraded terminals
- Document minimum terminal requirements clearly

**Warning signs:**
- Only testing in one terminal emulator
- Hard-coded ANSI escape sequences instead of lipgloss
- No fallback for limited color support
- UI layout depends on exact character dimensions

**Phase to address:**
Phase 1 (TUI rendering) - Use lipgloss from start; Phase 3 (Polish) - Test matrix and fallbacks

---

### Pitfall 7: Git Remote Parsing Fragility

**What goes wrong:**
Project detection via git remote fails due to: SSH vs HTTPS URL formats, missing `.git` suffix, submodules, case sensitivity on different hosts, monorepo subdirectories, and git worktrees. Users in these edge cases see "No project detected" despite being in a valid repo.

**Why it happens:**
Developers test with simple GitHub HTTPS URLs and miss the variety of real-world git configurations. Remote parsing logic doesn't handle all URL formats robustly.

**How to avoid:**
- Support multiple URL formats: `git@github.com:user/repo.git`, `https://github.com/user/repo`, `ssh://git@github.com/user/repo.git`
- Strip `.git` suffix if present
- Handle case-insensitive hosts (GitHub) vs case-sensitive (self-hosted)
- Test with: GitLab, GitHub, Bitbucket, self-hosted, SSH URLs, HTTPS URLs, submodules, worktrees
- Fall back to directory name if git remote parsing fails
- Allow manual project override in config
- Use `git config --get remote.origin.url` instead of parsing `.git/config` manually

**Warning signs:**
- Only handles one remote URL format
- No fallback when git remote missing
- Doesn't test with non-GitHub remotes
- Case-sensitive URL matching

**Phase to address:**
Phase 1 (Project detection) - Robust parsing with fallbacks; Phase 3 (Edge cases) - Test with all git hosting platforms

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Skip plugin signature verification | Faster initial plugin system | Massive security risk, hard to add later | Never - build verification from day 1 |
| Store credentials in TUI state | Simpler data flow | Security nightmare, audit failure | Never - credentials must stay in `op` CLI |
| Block on SSH connections in Update() | Simpler code structure | Unresponsive UI, user frustration | Never - all I/O must be async |
| Hard-code ANSI escape sequences | Faster initial rendering | Terminal compatibility issues | Never - use lipgloss from start |
| Single terminal testing | Faster development cycle | Broken UX on other terminals | Only for prototype, not v0.1 |
| Assume 10-minute `op` session is infinite | Don't track expiry | Mysterious auth failures | MVP only, add tracking by v0.2 |
| No SSH key lifecycle tracking | Faster MVP | Security audit nightmare | MVP only if documented as limitation |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| 1Password CLI | Assuming biometric auth = complete security | Document root/admin bypass risk; detect elevated privileges |
| 1Password CLI | Not handling desktop app lock mid-session | Gracefully handle auth failures; re-prompt automatically |
| 1Password CLI | Same session logic on Windows and Unix | Windows requires re-auth for sub-shells; branch logic per platform |
| System SSH | Assuming multiplexing is reliable | Don't depend on it; clean up stale control sockets; handle limit errors |
| Git remote | Only parsing GitHub HTTPS URLs | Support SSH, HTTPS, all major hosts; provide fallback |
| Bubbletea logging | Using stdout for debug logs | Use `tea.LogToFile()` - stdout is TUI output |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Blocking I/O in Update() | Frozen UI, laggy input | All I/O via tea.Cmd | First SSH connection attempt |
| N+1 `op` CLI calls | Slow list rendering | Batch credential fetches | >10 connections in project |
| Layout arithmetic without lipgloss | Broken layout, text overflow | Use `lipgloss.Height()` / `Width()` | Window resize, different terminals |
| No message ordering assumptions | Race conditions, flashing UI | Use `tea.Sequence()` for dependent commands | Concurrent async operations |
| Large connection lists in single view | Memory spike, slow rendering | Paginate/virtualize lists | >100 connections |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Passing credentials through plugin API | Credential theft by malicious plugins | Agent-blind architecture: plugins provide metadata only |
| Storing SSH keys in app state | Keys leak via logs, crashes, debug dumps | Keep keys in `op` CLI; reference by secret URIs |
| No privilege escalation detection | Users run as root, bypassing `op` security | Detect sudo/admin context; warn explicitly |
| Same key reused across all servers | Single compromise = full breach | Warn when same key used >5 times; suggest rotation |
| No credential expiry tracking | Stale credentials accumulate | Track creation date, last used; prompt for review |
| Trusting accessibility permissions | macOS accessibility apps bypass biometric prompts | Document risk; detect and warn if app has accessibility access |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No loading state during SSH connection | Users think app froze | Show spinner immediately in Update(), run connection as tea.Cmd |
| "Session open refused by peer" without explanation | Users blame tool, not SSH multiplexing | Detect error; show friendly message about session limits; auto-retry |
| Project detection fails silently | Users confused why no servers show | Clear "No project detected" message with troubleshooting steps |
| `op` session expires without warning | Mysterious biometric prompt mid-workflow | Track session age; warn at 9 minutes; graceful re-auth |
| Terminal incompatibility crashes app | Users see garbled output or crash | Detect incompatible terminals; offer `--simple` mode fallback |
| No indication which backend is active | Users don't know where credentials are stored | Show active backend in header/footer |

## "Looks Done But Isn't" Checklist

- [ ] **SSH Connections:** Async tea.Cmd pattern implemented - verify Update() doesn't block
- [ ] **Plugin System:** Signature verification working - test with unsigned plugin
- [ ] **Credential Access:** Plugins never receive raw credentials - audit plugin API surface
- [ ] **1Password Integration:** Root privilege detection working - test with `sudo`
- [ ] **Session Tracking:** 12-hour hard limit warning implemented - test long-running session
- [ ] **Terminal Compatibility:** Tested in 5+ different terminals - not just iTerm2
- [ ] **Git Remote Parsing:** Works with SSH URLs, HTTPS, no `.git` suffix - test all formats
- [ ] **Error Recovery:** Terminal resets after panics - test with forced crash
- [ ] **Layout Rendering:** Uses lipgloss for dimensions - no hard-coded widths
- [ ] **Connection Limits:** SSH multiplexing errors handled gracefully - test >10 connections

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Credentials in plugin API | HIGH | Major refactor of plugin system; breaking API change |
| Blocking I/O in Update() | MEDIUM | Refactor to tea.Cmd; may need state restructuring |
| No SSH key lifecycle tracking | MEDIUM | Add metadata schema; database migration; backfill data |
| Hard-coded ANSI sequences | MEDIUM | Replace with lipgloss; extensive re-testing |
| Missing `op` session tracking | LOW | Add expiry detection; graceful re-auth flow |
| SSH multiplexing dependency | MEDIUM | Redesign connection pooling; remove multiplexing assumptions |
| Git remote parsing too simple | LOW | Add URL parsing library; test suite for edge cases |
| Root privilege bypass undocumented | LOW | Add detection + warning; update docs |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Event loop blocking | Phase 1 (Core TUI) | Profile Update()/View() timing; all I/O async |
| Plugin credential access | Phase 1 (Architecture) | Audit plugin API - no raw credential methods |
| SSH key lifecycle blindness | Phase 1 (Data model) | Metadata fields present; audit views working |
| Terminal incompatibility | Phase 1 (Rendering) | Test suite covers 5+ terminals |
| Git remote parsing fragility | Phase 1 (Project detection) | Test suite covers SSH/HTTPS/GitLab/Bitbucket |
| 1Password security assumptions | Phase 2 (Integration) | Root detection works; Windows sub-shell auth correct |
| SSH multiplexing chaos | Phase 2 (SSH integration) | Control socket cleanup; session limit handling |
| Message ordering race conditions | Phase 2 (Async patterns) | tea.Sequence() used for dependent operations |
| No credential expiry tracking | Phase 3 (Security hardening) | Warnings for old keys; bulk audit views |
| Missing session timeout warnings | Phase 3 (Polish) | 9-minute warning implemented; graceful re-auth |

## Sources

**SSH Connection Management:**
- [Avoid These 11 Common Mistakes in SSH Connectivity](https://hoop.dev/blog/avoid-these-11-common-mistakes-in-ssh-connectivity-ensure-a-reliable-and-secure-connection/)
- [10 Common Mistakes Developers Make with SSH APIs](https://jadaptive.com/10-common-mistakes-developers-make-with-ssh-apis-and-how-to-avoid-them/)
- [SSH Key Management: Best Practices 2025-2026](https://www.brandonchecketts.com/archives/ssh-ed25519-key-best-practices-for-2025)
- [SSH Multiplexing Gotchas](https://thomasbroadley.com/blog/ssh-multiplexing-gotchas/)

**Go TUI Development:**
- [Tips for Building Bubble Tea Programs](https://leg100.github.io/en/posts/building-bubbletea-programs/)
- [Terminal UI: BubbleTea (Go) vs Ratatui (Rust)](https://www.glukhov.org/post/2026/02/tui-frameworks-bubbletea-go-vs-ratatui-rust/)
- [Terminal Compatibility and Rendering Issues](https://p.janouch.name/article-tui.html)

**1Password CLI Security:**
- [1Password CLI App Integration Security](https://developer.1password.com/docs/cli/app-integration-security/)
- [Security Concern with Terminal Access to 1Password](https://www.1password.community/discussions/developers/security-concern-with-allowing-terminal-complete-access-to-my-1p-account-via-op-/28388)

**Plugin Architecture Security:**
- [Agent-Blind Credential Architecture RFC](https://github.com/openclaw/openclaw/discussions/9676)
- [HashiCorp Vault Plugin Architecture](https://deepwiki.com/hashicorp/vault/5-plugin-architecture)

**Git and Cross-Platform:**
- [Common Git Mistakes and Fixes](https://about.gitlab.com/blog/git-happens/)
- [Go Cross-Platform Binary Issues](https://github.com/golang/go/issues/48540)

---
*Pitfalls research for: ssherpa - SSH Connection Management TUI*
*Researched: 2026-02-14*
*Confidence: MEDIUM-HIGH (WebSearch verified with official sources; some LOW confidence items flagged)*
