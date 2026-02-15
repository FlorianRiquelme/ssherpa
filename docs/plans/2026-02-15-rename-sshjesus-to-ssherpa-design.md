# Design: Rename ssherpa to ssherpa

**Date:** 2026-02-15
**Status:** Approved

## Context

The tool is currently named "ssherpa" throughout the codebase. It needs to be renamed to "ssherpa" before Phase 8 (Distribution). This is pre-release, so no migration logic is needed.

## Scope

- **637 occurrences** across **100 files**
- 41 Go source files, 12 test files, 69 planning/docs files

## Changes

### 1. Go Module & Imports

- `go.mod`: `github.com/florianriquelme/ssherpa` -> `github.com/florianriquelme/ssherpa`
- All internal Go files: update import paths
- Rename directory `cmd/ssherpa/` -> `cmd/ssherpa/`

### 2. Hardcoded Paths & Strings

- `~/.ssh/ssherpa_config` -> `~/.ssh/ssherpa_config`
- `~/.ssh/ssherpa_history.json` -> `~/.ssh/ssherpa_history.json`
- `~/.ssh/ssherpa_1password_cache.toml` -> `~/.ssh/ssherpa_1password_cache.toml`
- XDG config: `ssherpa/config.toml` -> `ssherpa/config.toml`
- 1Password tag: `ssherpa` -> `ssherpa`

### 3. Tests

- Update all import paths and expected path strings in test files

### 4. Planning & Docs

- Update all `.planning/` markdown files
- Update README if present

### 5. Cleanup

- Delete compiled `ssherpa` binary
- Rebuild as `ssherpa`
- Run `go build` and `go test` to verify

## Out of Scope

- GitHub repo rename (done separately on GitHub)
- No backward compatibility / migration logic (pre-release)
