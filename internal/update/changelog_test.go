package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testChangelog = `# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.3.0] - 2026-03-01

### Added

- Auto-update notifications
- Changelog overlay

### Fixed

- Status bar flicker

## [0.2.0] - 2026-02-20

### Added

- Context-aware shortcut footer

### Changed

- Footer uses shared style

## [0.1.0] - 2026-02-16

### Added

- Initial release
`

func TestParseChangelog_AllVersions(t *testing.T) {
	changes, err := ParseChangelog(testChangelog, "0.1.0", "0.3.0")
	require.NoError(t, err)
	require.Len(t, changes, 2) // 0.3.0 and 0.2.0 (not 0.1.0)

	// Newest first
	assert.Equal(t, "0.3.0", changes[0].Version)
	assert.Equal(t, "2026-03-01", changes[0].Date)
	require.Len(t, changes[0].Sections, 2)
	assert.Equal(t, "Added", changes[0].Sections[0].Category)
	assert.Len(t, changes[0].Sections[0].Entries, 2)
	assert.Equal(t, "Fixed", changes[0].Sections[1].Category)

	assert.Equal(t, "0.2.0", changes[1].Version)
	assert.Equal(t, "2026-02-20", changes[1].Date)
}

func TestParseChangelog_SingleVersion(t *testing.T) {
	changes, err := ParseChangelog(testChangelog, "0.2.0", "0.3.0")
	require.NoError(t, err)
	require.Len(t, changes, 1)
	assert.Equal(t, "0.3.0", changes[0].Version)
}

func TestParseChangelog_SameVersion(t *testing.T) {
	changes, err := ParseChangelog(testChangelog, "0.3.0", "0.3.0")
	require.NoError(t, err)
	assert.Len(t, changes, 0) // Nothing new
}

func TestParseChangelog_Empty(t *testing.T) {
	changes, err := ParseChangelog("", "0.1.0", "0.2.0")
	require.NoError(t, err)
	assert.Len(t, changes, 0)
}
