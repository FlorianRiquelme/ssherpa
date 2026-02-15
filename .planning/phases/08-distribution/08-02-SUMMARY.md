---
phase: 08-distribution
plan: 02
subsystem: distribution
tags: [goreleaser, github-actions, release-automation, install-script]
dependency_graph:
  requires:
    - Phase 07 (SSH Key Selection)
  provides:
    - GoReleaser v2 configuration for multi-platform binary distribution
    - Automated GitHub Actions release workflow
    - Curl install script with checksum verification
  affects:
    - Release process: fully automated on git tag push
    - Distribution channels: GitHub Releases, Homebrew tap, curl install
tech_stack:
  added:
    - goreleaser/goreleaser-action@v6
    - GoReleaser v2 configuration
  patterns:
    - Multi-platform builds (3 OS x 2 arch = 6 binaries)
    - SHA256 checksum verification
    - Auto-update Homebrew tap on release
key_files:
  created:
    - .goreleaser.yaml: GoReleaser v2 config with multi-platform builds, Homebrew tap integration, and release notes template
    - .github/workflows/release.yml: GitHub Actions workflow triggered by version tags, runs GoReleaser with tap token
    - scripts/install.sh: Curl install script with OS/arch detection and checksum verification
  modified: []
decisions:
  - decision: "Use GoReleaser v2 for release automation"
    rationale: "Industry standard for Go binary releases, supports multi-platform builds, Homebrew tap automation, and checksum generation out of the box"
    alternatives: "Manual builds and uploads, or custom release scripts"
  - decision: "Auto-push Homebrew formula to separate tap repository"
    rationale: "Users get tap auto-updated on release without manual PR creation, TAP_GITHUB_TOKEN secret enables cross-repo push"
    alternatives: "Manual tap updates or tap-as-subdirectory in main repo"
  - decision: "CGO_ENABLED=0 for all builds"
    rationale: "Pure Go binaries with no C dependencies ensure clean cross-compilation and simpler distribution"
    alternatives: "Enable CGO for specific platforms if needed"
  - decision: "Separate archives for Windows (.zip) vs macOS/Linux (.tar.gz)"
    rationale: "Windows users expect .zip, Unix users expect .tar.gz — GoReleaser format_overrides handles this"
    alternatives: "Single archive format for all platforms"
  - decision: "Prerelease auto-detection by tag pattern"
    rationale: "Tags like v0.1.0-alpha.1 automatically marked as pre-release, no manual workflow changes needed"
    alternatives: "Manual prerelease flag or separate workflow for pre-releases"
  - decision: "Curl install script verifies SHA256 checksum"
    rationale: "Security best practice to prevent MITM attacks and corrupted downloads"
    alternatives: "Skip verification (insecure) or use GPG signatures (more complex)"
metrics:
  duration: 138
  completed_date: "2026-02-15"
---

# Phase 08 Plan 02: Release Automation Summary

**One-liner:** GoReleaser v2 configuration, GitHub Actions workflow, and curl install script for automated multi-platform binary distribution with Homebrew tap integration.

## What Was Built

Complete release automation pipeline:

1. **GoReleaser Configuration** (.goreleaser.yaml):
   - Multi-platform builds: macOS (amd64/arm64), Linux (amd64/arm64), Windows (amd64/arm64) = 6 binaries
   - Version injection via ldflags (Version, Commit, Date, GoVersion from internal/version package)
   - Archive naming: `ssherpa_0.1.0_darwin_arm64.tar.gz`
   - SHA256 checksums.txt generation
   - Homebrew tap auto-update on release (ssherpa/homebrew-tap)
   - Release notes template with install instructions

2. **GitHub Actions Workflow** (.github/workflows/release.yml):
   - Triggers on version tags (v*)
   - Runs GoReleaser action v6 with `--clean` flag
   - Uses TAP_GITHUB_TOKEN secret for cross-repo Homebrew push
   - Auto-injects GOVERSION env var from go.mod

3. **Install Script** (scripts/install.sh):
   - OS/arch detection (macOS/Linux, amd64/arm64)
   - Downloads latest release or VERSION-pinned version
   - Verifies SHA256 checksum (supports sha256sum and shasum)
   - Installs to /usr/local/bin with sudo-awareness
   - Temp directory cleanup via trap

## How It Works

**Release Flow:**
```
Developer pushes tag (git tag v0.1.0 && git push origin v0.1.0)
  ↓
GitHub Actions detects tag and triggers release.yml
  ↓
GoReleaser runs with .goreleaser.yaml config:
  - Runs `go mod tidy` (before hook)
  - Builds 6 platform binaries with version ldflags
  - Creates archives (.tar.gz for Unix, .zip for Windows)
  - Generates checksums.txt with SHA256 hashes
  - Pushes Homebrew formula to ssherpa/homebrew-tap
  - Creates GitHub Release with release notes
  ↓
Users install via:
  - Homebrew: brew tap ssherpa/tap && brew install ssherpa
  - Curl: curl -fsSL https://raw.githubusercontent.com/florianriquelme/ssherpa/main/scripts/install.sh | sh
  - Direct download from GitHub Releases with checksum verification
```

**Version Embedding:**
GoReleaser injects version metadata into internal/version package via ldflags:
- `Version`: Git tag (e.g., "0.1.0")
- `Commit`: Full commit hash
- `Date`: Build date
- `GoVersion`: Go compiler version

This enables `ssherpa version` to show accurate build information.

**Homebrew Tap Integration:**
- TAP_GITHUB_TOKEN secret (Personal Access Token with repo scope) allows GoReleaser to push formula to ssherpa/homebrew-tap
- Formula auto-generated from GoReleaser metadata
- Formula test command: `ssherpa version`
- Commit message: "chore: update formula for ssherpa version v0.1.0"

**Install Script Security:**
- Downloads archive + checksums.txt
- Verifies SHA256 checksum before extraction
- Fails fast on checksum mismatch
- Graceful fallback if sha256sum/shasum unavailable (prints warning)

## Configuration Details

**GoReleaser Build Matrix:**
```
GOOS       GOARCH    Output Binary
--------------------------------------
linux      amd64     ssherpa_0.1.0_linux_amd64.tar.gz
linux      arm64     ssherpa_0.1.0_linux_arm64.tar.gz
darwin     amd64     ssherpa_0.1.0_darwin_amd64.tar.gz
darwin     arm64     ssherpa_0.1.0_darwin_arm64.tar.gz
windows    amd64     ssherpa_0.1.0_windows_amd64.zip
windows    arm64     ssherpa_0.1.0_windows_arm64.zip
```

**Changelog Filtering:**
Excludes commit types from release notes:
- `docs:` commits
- `test:` commits
- `chore:` commits
- Commits with "typo" in message

**Release Notes Template:**
Includes header with tag and pre-release indicator, followed by installation instructions (Homebrew, curl, direct download), and footer with full changelog link.

## Deviations from Plan

None — plan executed exactly as written.

## Testing

**Verification Performed:**
1. ✓ GoReleaser config validated (version: 2 present)
2. ✓ GitHub Actions workflow triggers on v* tags
3. ✓ Install script is executable (755 permissions)
4. ✓ Install script has valid shell syntax (sh -n passed)
5. ✓ GoReleaser references correct build path (./cmd/ssherpa)
6. ✓ Version ldflags reference internal/version package
7. ✓ Release notes include Homebrew and curl install instructions

**Not Tested (requires actual release):**
- GoReleaser execution on real tag push
- Binary builds for all 6 platforms
- Homebrew tap push to ssherpa/homebrew-tap
- Install script with real release assets
- Checksum verification end-to-end

**Next Steps for First Release:**
1. Create internal/version package with Version, Commit, Date, GoVersion vars
2. Add TAP_GITHUB_TOKEN secret to GitHub repository settings
3. Create ssherpa/homebrew-tap repository
4. Push first version tag to trigger release workflow
5. Test install script with real release assets

## Files Changed

**Created (3 files):**
- `.goreleaser.yaml` (95 lines): GoReleaser v2 configuration
- `.github/workflows/release.yml` (37 lines): GitHub Actions release workflow
- `scripts/install.sh` (98 lines): Curl install script with checksum verification

**Modified:** None

## Commits

- `e887559`: feat(08-02): create GoReleaser configuration
- `7532422`: feat(08-02): add release workflow and install script

## Self-Check: PASSED

**Files exist:**
```
✓ .goreleaser.yaml exists
✓ .github/workflows/release.yml exists
✓ scripts/install.sh exists
```

**Commits exist:**
```
✓ e887559 found in git log
✓ 7532422 found in git log
```

**Content verification:**
```
✓ GoReleaser config has version: 2
✓ Workflow triggers on v* tags
✓ Build path is ./cmd/ssherpa
✓ Version ldflags reference internal/version
✓ Release notes include install instructions
✓ Install script is executable
✓ Install script has valid syntax
```

All claimed files and commits verified successfully.
