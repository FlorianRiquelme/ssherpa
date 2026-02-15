# Phase 8: Distribution - Context

**Gathered:** 2026-02-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Package and ship ssherpa as a single binary via Homebrew and GitHub releases. Includes GoReleaser setup, Homebrew tap, GitHub release automation, install script, README, and first-run onboarding. Terminal compatibility testing across iTerm2, Alacritty, Windows Terminal, and tmux.

</domain>

<decisions>
## Implementation Decisions

### Versioning strategy
- First public release ships as v0.1.0 (signals early/beta, sets expectations for breaking changes)
- Use alpha/beta pre-release tags (v0.1.0-alpha.1, v0.1.0-beta.1) for formal pre-release cycle
- Binary embeds version + commit hash (e.g., v0.1.0 (abc1234)) for bug reports
- `ssherpa version` shows multi-line detail: version, commit, build date, Go version, OS/arch

### Homebrew tap identity
- Dedicated GitHub org (e.g., ssherpa/homebrew-tap) — feels more official, supports collaborators
- Aim for homebrew-core inclusion later once stable — start with own tap for speed and control
- Also provide GitHub releases + curl install script for Linux/macOS one-liner installs

### Release presentation
- Release notes: narrative intro paragraph highlighting what's new, then auto-generated changelog from conventional commits
- Release assets: platform binaries (.tar.gz/.zip) + SHA256 checksums file (no shell completions — TUI doesn't benefit)
- README: standard OSS structure — GIF + description + features list + install + usage + config + contributing
- README style inspired by lazygit/lazydocker — minimal but polished, GIF-forward
- 1Password listed as one of several features (not a prominent callout) — balanced presentation
- Demo GIF showing the TUI in action

### First-run experience
- Broader onboarding flow on first run (not just straight to TUI)
- Step 1: Detect SSH config, show count of hosts found
- Step 2: Offer to set up 1Password backend (leverages existing Phase 6 wizard)
- Skippable at any point, only shows once (stored in config)
- Re-triggerable via `ssherpa --setup` flag

### Claude's Discretion
- Homebrew tap naming convention (user deferred the exact tap/formula naming)
- Tagline/one-liner for the project (user asked Claude to write it)
- GoReleaser configuration details
- CI/CD pipeline setup (GitHub Actions)
- Curl install script implementation
- Cross-compilation strategy (CGo implications from 1Password SDK)
- Terminal compatibility testing approach
- Demo GIF creation tooling and content

</decisions>

<specifics>
## Specific Ideas

- README style should feel like lazygit/lazydocker — GIF-forward, clean feature list, quick install instructions
- Shell completions don't make sense for a fullscreen TUI app
- The existing 1Password setup wizard (Phase 6) should be wrapped in the broader onboarding, not replaced

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-distribution*
*Context gathered: 2026-02-15*
