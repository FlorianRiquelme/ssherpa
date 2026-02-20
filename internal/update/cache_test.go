package update

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update.json")

	data := CacheData{
		LastCheck:        time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC),
		LatestVersion:    "0.3.0",
		ChangelogMD:      "# Changelog\n...",
		DismissedVersion: "",
	}

	err := writeCache(path, data)
	require.NoError(t, err)

	loaded, err := readCache(path)
	require.NoError(t, err)
	assert.Equal(t, data.LatestVersion, loaded.LatestVersion)
	assert.Equal(t, data.ChangelogMD, loaded.ChangelogMD)
	assert.True(t, data.LastCheck.Equal(loaded.LastCheck))
}

func TestReadCacheMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	data, err := readCache(path)
	require.NoError(t, err)
	assert.Equal(t, CacheData{}, data)
}

func TestCacheExpired(t *testing.T) {
	fresh := CacheData{LastCheck: time.Now()}
	stale := CacheData{LastCheck: time.Now().Add(-25 * time.Hour)}

	assert.False(t, cacheExpired(fresh))
	assert.True(t, cacheExpired(stale))
}

func TestDefaultCachePath(t *testing.T) {
	path := defaultCachePath()
	assert.Contains(t, path, "ssherpa")
	assert.Contains(t, path, "update.json")
}
