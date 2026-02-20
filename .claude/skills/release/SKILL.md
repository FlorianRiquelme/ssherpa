---
name: release
description: |
  Create a new release with changelog generation from git history.
  Use when releasing a new version.
  Usage: /release <version> (e.g., /release 0.2.0)
---

# Release Skill

Creates a new release by updating CHANGELOG.md from the `[Unreleased]` section and git commits, then creating a tagged release commit.

## Input

- `version`: Semantic version number (e.g., `1.0.0`, `0.2.0`)

## Process

### 1. Validate Version

- Must be valid semver format (MAJOR.MINOR.PATCH)
- Must not already exist as a git tag (`v` prefix: `vX.Y.Z`)
- Must be greater than the latest tag

### 2. Sync Missing Tags to CHANGELOG.md

Before creating the new release, ensure all existing git tags are represented in the changelog:

```bash
# Get all tags sorted by version
git tag -l --sort=v:refname

# Get versions already in CHANGELOG.md
grep -oP '(?<=\[)[0-9]+\.[0-9]+\.[0-9]+(?=\])' CHANGELOG.md
```

For each missing tag, generate entries from commits between that tag and the previous one.

### 3. Categorize Commits

Parse commit messages and categorize by prefix/keyword:

| Prefix/Keyword | Category |
|----------------|----------|
| `Add`, `feat`, `Implement`, `Create`, `Initial` | Added |
| `Fix`, `Repair`, `Resolve`, `Correct` | Fixed |
| `Change`, `Update`, `Modify`, `Refactor`, `Improve` | Changed |
| `Remove`, `Delete`, `Drop` | Removed |
| `Deprecate` | Deprecated |
| `Security`, `CVE`, `Vulnerability` | Security |

Use the first word of the commit message (after any `scope:` prefix) to determine category. Default to "Changed" if unclear.

### 4. Generate Changelog Entry

Format for each version:

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- Entry description

### Changed
- Entry description

### Fixed
- Entry description
```

- Use today's date for the release date
- Clean up commit messages: remove conventional commit prefixes, capitalize first letter
- Merge with any existing entries in `[Unreleased]` (Claude-written entries take priority over auto-generated ones)
- Group by category, omit empty categories
- Place new version after `## [Unreleased]`

### 5. Update CHANGELOG.md

- Keep the `## [Unreleased]` section (cleared of moved items)
- Insert new version section in descending order

### 6. Create Release Commit and Tag

```bash
# Stage changelog
git add CHANGELOG.md

# Commit
git commit -m "Release vX.Y.Z"

# Create annotated tag (v-prefixed for GoReleaser)
git tag -a vX.Y.Z -m "Release vX.Y.Z"
```

### 7. Final Output

Display:
- Summary of changes added to changelog
- The git commands to push (do NOT auto-push):

```
Release vX.Y.Z created locally.

To publish (triggers GoReleaser build + Homebrew update):
  git push && git push --tags
```

## Edge Cases

- **First release (no prior tags)**: Use all commits from initial commit
- **No commits since last tag and no Unreleased entries**: Warn and abort
- **Merge commits**: Skip merge commits (those starting with "Merge")
- **Version conflicts**: Abort if tag already exists
- **Unreleased has entries but no new commits**: Use only the Unreleased entries
