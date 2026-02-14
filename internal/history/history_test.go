package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordConnection_CreatesFileAndWritesEntry(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.json")

	err := RecordConnection(historyPath, "server1", "10.0.1.5", "user1")
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(historyPath)
	require.NoError(t, err)

	// Verify content
	data, err := os.ReadFile(historyPath)
	require.NoError(t, err)

	var entry HistoryEntry
	err = json.Unmarshal(data, &entry)
	require.NoError(t, err)

	assert.Equal(t, "server1", entry.HostName)
	assert.Equal(t, "10.0.1.5", entry.Hostname)
	assert.Equal(t, "user1", entry.User)
	assert.NotEmpty(t, entry.WorkingDir)
	assert.WithinDuration(t, time.Now(), entry.Timestamp, 2*time.Second)
}

func TestRecordConnection_AppendsToExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.json")

	// Record first connection
	err := RecordConnection(historyPath, "server1", "10.0.1.5", "user1")
	require.NoError(t, err)

	// Record second connection
	err = RecordConnection(historyPath, "server2", "10.0.1.6", "user2")
	require.NoError(t, err)

	// Verify both entries exist
	f, err := os.Open(historyPath)
	require.NoError(t, err)
	defer f.Close()

	decoder := json.NewDecoder(f)
	var entries []HistoryEntry
	for {
		var entry HistoryEntry
		err := decoder.Decode(&entry)
		if err != nil {
			break
		}
		entries = append(entries, entry)
	}

	require.Len(t, entries, 2)
	assert.Equal(t, "server1", entries[0].HostName)
	assert.Equal(t, "server2", entries[1].HostName)
}

func TestRecordConnection_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.json")

	err := RecordConnection(historyPath, "server1", "10.0.1.5", "user1")
	require.NoError(t, err)

	info, err := os.Stat(historyPath)
	require.NoError(t, err)

	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestGetLastConnectedForPath_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "nonexistent.json")

	entry, err := GetLastConnectedForPath(historyPath, "/some/path")
	require.NoError(t, err)
	assert.Nil(t, entry)
}

func TestGetLastConnectedForPath_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.json")

	// Create file with entry for different path
	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)
	entry := HistoryEntry{
		Timestamp:  time.Now(),
		WorkingDir: "/different/path",
		HostName:   "server1",
		Hostname:   "10.0.1.5",
		User:       "user1",
	}
	err = json.NewEncoder(f).Encode(entry)
	require.NoError(t, err)
	f.Close()

	result, err := GetLastConnectedForPath(historyPath, "/some/path")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetLastConnectedForPath_FindsMatch(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.json")

	targetPath := "/target/path"
	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)

	entry := HistoryEntry{
		Timestamp:  time.Now(),
		WorkingDir: targetPath,
		HostName:   "server1",
		Hostname:   "10.0.1.5",
		User:       "user1",
	}
	err = json.NewEncoder(f).Encode(entry)
	require.NoError(t, err)
	f.Close()

	result, err := GetLastConnectedForPath(historyPath, targetPath)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "server1", result.HostName)
	assert.Equal(t, targetPath, result.WorkingDir)
}

func TestGetLastConnectedForPath_FindsMostRecent(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.json")

	targetPath := "/target/path"
	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)

	// Write older entry
	oldEntry := HistoryEntry{
		Timestamp:  time.Now().Add(-2 * time.Hour),
		WorkingDir: targetPath,
		HostName:   "server1",
		Hostname:   "10.0.1.5",
		User:       "user1",
	}
	err = json.NewEncoder(f).Encode(oldEntry)
	require.NoError(t, err)

	// Write newer entry
	newEntry := HistoryEntry{
		Timestamp:  time.Now().Add(-1 * time.Hour),
		WorkingDir: targetPath,
		HostName:   "server2",
		Hostname:   "10.0.1.6",
		User:       "user2",
	}
	err = json.NewEncoder(f).Encode(newEntry)
	require.NoError(t, err)
	f.Close()

	result, err := GetLastConnectedForPath(historyPath, targetPath)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "server2", result.HostName, "should return most recent entry")
}

func TestGetLastConnectedForPath_SkipsMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.json")

	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)

	// Write malformed line
	_, err = f.WriteString("{invalid json}\n")
	require.NoError(t, err)

	// Write valid entry
	targetPath := "/target/path"
	entry := HistoryEntry{
		Timestamp:  time.Now(),
		WorkingDir: targetPath,
		HostName:   "server1",
		Hostname:   "10.0.1.5",
		User:       "user1",
	}
	err = json.NewEncoder(f).Encode(entry)
	require.NoError(t, err)
	f.Close()

	result, err := GetLastConnectedForPath(historyPath, targetPath)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "server1", result.HostName)
}

func TestGetRecentHosts_ReturnsUniqueHostsWithLatestTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.json")

	f, err := os.OpenFile(historyPath, os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)

	// Write entries for same host at different times
	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)

	oldEntry := HistoryEntry{
		Timestamp:  oldTime,
		WorkingDir: "/path1",
		HostName:   "server1",
		Hostname:   "10.0.1.5",
		User:       "user1",
	}
	err = json.NewEncoder(f).Encode(oldEntry)
	require.NoError(t, err)

	newEntry := HistoryEntry{
		Timestamp:  newTime,
		WorkingDir: "/path2",
		HostName:   "server1",
		Hostname:   "10.0.1.5",
		User:       "user1",
	}
	err = json.NewEncoder(f).Encode(newEntry)
	require.NoError(t, err)

	// Different host
	otherEntry := HistoryEntry{
		Timestamp:  time.Now(),
		WorkingDir: "/path3",
		HostName:   "server2",
		Hostname:   "10.0.1.6",
		User:       "user2",
	}
	err = json.NewEncoder(f).Encode(otherEntry)
	require.NoError(t, err)
	f.Close()

	result, err := GetRecentHosts(historyPath, 10)
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Should have latest timestamp for server1
	server1Time := result["server1"]
	assert.WithinDuration(t, newTime, server1Time, 1*time.Second)
}

func TestGetRecentHosts_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "nonexistent.json")

	result, err := GetRecentHosts(historyPath, 10)
	require.NoError(t, err)
	assert.Empty(t, result)
}
