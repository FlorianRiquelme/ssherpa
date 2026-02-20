# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-02-20

### Added

- Context-aware shortcut footer that shows relevant keybindings per view, replacing the Bubbles help component
- Auto-discovery of 1Password vaults in setup wizard, replacing manual vault selection

### Changed

- Shortcut footer now uses a shared style across all views

### Fixed

- Shortcut footer renders correctly in detail view
- Vault scanning includes per-vault timeouts to prevent hanging
- Homebrew tap repository uses correct `homebrew-` prefix
- Resolved staticcheck QF1012 warnings
- Pinned golangci-lint to v2.10.1 for reproducible CI

## [0.1.0] - 2026-02-16

### Added

- Terminal UI (TUI) SSH connection manager built with Bubble Tea
- SSH config parser that reads `~/.ssh/config` and displays servers in a navigable list
- Fuzzy search across all servers with two-phase Esc behavior (blur then clear)
- Project detection via git remote URLs with automatic server grouping
- Project picker overlay with persistent project-to-host assignment
- Connection history tracking with last-connected timestamps
- Add, edit, and delete servers through full-screen TUI forms with field validation
- Async DNS checker in the add/edit form for hostname verification
- Delete confirmation dialog with undo buffer
- SSH config writer that preserves formatting and manages Include directives
- 1Password backend integration via `op` CLI with item-to-server mapping
- 1Password setup and migration wizards for onboarding
- Background 1Password availability poller with auto-recovery and cache fallback
- Multi-backend aggregator with priority-based deduplication
- TUI status bar showing 1Password connection state
- Sync engine with conflict detection and TOML cache
- SSH key discovery from files, SSH agent, and 1Password IdentityAgent socket
- SSH key picker overlay integrated into the form and detail views
- `--fields` CLI flag and `?` key help overlay showing 1Password field reference
- `--version` CLI flag with build metadata (version, commit, date, platform)
- First-run onboarding wizard for initial configuration
- GoReleaser configuration for cross-platform builds (Linux, macOS, Windows; amd64, arm64)
- Homebrew formula published to `FlorianRiquelme/homebrew-ssherpa`
- Install script for macOS and Linux (`scripts/install.sh`)
- CI pipeline with lint, test, build, tidy, and format checks
- MIT license

### Fixed

- 1Password sync timeout increased to allow biometric authentication
- 1Password items tracked by ID to prevent duplicates on rename
- TUI host merge replaced with full replacement to prevent stale duplicates
- Backend server wildcard detection to hide wildcard SSH entries from TUI
- Search mode no longer swallows action and navigation shortcut keys
