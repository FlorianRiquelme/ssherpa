---
created: 2026-02-14T20:50:41.880Z
title: Rename tool from sshjesus to ssherpa
area: general
files:
  - .planning/STATE.md
  - .planning/PROJECT.md
  - .planning/phases/08-distribution/ROADMAP.md
---

## Problem

The tool is currently named "sshjesus" throughout the codebase, documentation, and planning files. It needs to be renamed to "ssherpa" before Phase 8 (Distribution) begins. This affects:

- Go module path and package references
- Binary name and CLI commands
- All planning documents and roadmap references
- README and user-facing documentation
- 1Password tag conventions (currently "sshjesus" tag for server discovery)
- SSH config paths (e.g., `~/.ssh/sshjesus_config`)
- Repository name and git remote URLs

## Solution

Before starting Phase 8 (Distribution):
1. Rename Go module path (`go.mod`, all imports)
2. Update all internal string references (binary name, config paths, 1Password tags)
3. Update planning docs (PROJECT.md, STATE.md, ROADMAP.md)
4. Update README and user-facing copy
5. Rename repository on GitHub
6. Ensure Phase 8 distribution plan uses "ssherpa" everywhere (Homebrew formula, goreleaser config, etc.)
