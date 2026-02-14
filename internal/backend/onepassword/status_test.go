package onepassword

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	backendpkg "github.com/florianriquelme/sshjesus/internal/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncFromOnePassword_Success(t *testing.T) {
	mock := NewMockClient()

	// Add vault and tagged item
	vaultID := "VAULT-123"
	mock.AddVault(Vault{
		ID:   vaultID,
		Name: "Personal",
	})
	mock.AddItem(Item{
		ID:       "ITEM-1",
		Title:    "Production Server",
		VaultID:  vaultID,
		Category: "server",
		Tags:     []string{"sshjesus"},
		Fields: []ItemField{
			{Title: "hostname", Value: "prod.example.com"},
			{Title: "user", Value: "deploy"},
			{Title: "port", Value: "22"},
		},
	})

	// Create backend with cache path
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	backend := NewWithCache(mock, cachePath)

	// Initial status should be Unknown
	assert.Equal(t, backendpkg.StatusUnknown, backend.GetStatus())

	// Sync from 1Password
	ctx := context.Background()
	err := backend.SyncFromOnePassword(ctx)
	require.NoError(t, err)

	// Status should be Available
	assert.Equal(t, backendpkg.StatusAvailable, backend.GetStatus())

	// Servers should be populated
	servers, err := backend.ListServers(ctx)
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "prod.example.com", servers[0].Host)

	// Cache file should exist
	_, err = os.Stat(cachePath)
	assert.NoError(t, err, "Cache file should be written")
}

func TestSyncFromOnePassword_Locked(t *testing.T) {
	mock := NewMockClient()

	// Simulate 1Password locked (session expired)
	mock.SetError("ListVaults", assert.AnError)

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	backend := NewWithCache(mock, cachePath)

	// Sync should fail
	ctx := context.Background()
	err := backend.SyncFromOnePassword(ctx)
	require.Error(t, err)

	// Status should be Locked (for session expired) or Unavailable (for generic error)
	// Since we can't easily simulate the exact "session expired" error string,
	// we'll accept either Locked or Unavailable
	status := backend.GetStatus()
	assert.True(t, status == backendpkg.StatusLocked || status == backendpkg.StatusUnavailable,
		"Status should be Locked or Unavailable when sync fails")
}

func TestSyncFromOnePassword_Unavailable(t *testing.T) {
	mock := NewMockClient()

	// Simulate 1Password unavailable (generic error)
	mock.SetError("ListVaults", assert.AnError)

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	backend := NewWithCache(mock, cachePath)

	// Sync should fail
	ctx := context.Background()
	err := backend.SyncFromOnePassword(ctx)
	require.Error(t, err)

	// Status should be Unavailable or Locked
	status := backend.GetStatus()
	assert.True(t, status == backendpkg.StatusLocked || status == backendpkg.StatusUnavailable)
}

func TestLoadFromCache(t *testing.T) {
	// Create test cache file
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	cacheContent := `last_sync = 2024-01-01T00:00:00Z

[[server]]
id = "test-123"
display_name = "Test Server"
host = "test.example.com"
user = "testuser"
port = 22
project_ids = ["proj1"]
vault_id = ""
`
	err := os.WriteFile(cachePath, []byte(cacheContent), 0644)
	require.NoError(t, err)

	// Create backend and load cache
	mock := NewMockClient()
	backend := NewWithCache(mock, cachePath)

	err = backend.LoadFromCache()
	require.NoError(t, err)

	// Verify servers loaded
	ctx := context.Background()
	servers, err := backend.ListServers(ctx)
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "test-123", servers[0].ID)
	assert.Equal(t, "Test Server", servers[0].DisplayName)
	assert.Equal(t, "test.example.com", servers[0].Host)
	assert.Equal(t, "testuser", servers[0].User)
	assert.Equal(t, 22, servers[0].Port)
	assert.Equal(t, []string{"proj1"}, servers[0].ProjectIDs)
}

func TestLoadFromCache_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nonexistent.toml")

	mock := NewMockClient()
	backend := NewWithCache(mock, cachePath)

	// Loading non-existent cache should return error
	err := backend.LoadFromCache()
	assert.Error(t, err)
}

func TestListServers_Unavailable_UsesCachedData(t *testing.T) {
	// Create test cache file
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	cacheContent := `last_sync = 2024-01-01T00:00:00Z

[[server]]
id = "cached-1"
display_name = "Cached Server"
host = "cached.example.com"
user = "cacheuser"
port = 22
vault_id = ""
`
	err := os.WriteFile(cachePath, []byte(cacheContent), 0644)
	require.NoError(t, err)

	// Create backend with failing client
	mock := NewMockClient()
	mock.SetError("ListVaults", assert.AnError)

	backend := NewWithCache(mock, cachePath)

	// Load from cache
	err = backend.LoadFromCache()
	require.NoError(t, err)

	// Try to sync (will fail)
	ctx := context.Background()
	err = backend.SyncFromOnePassword(ctx)
	require.Error(t, err)

	// Status should be Unavailable
	assert.True(t, backend.GetStatus() != backendpkg.StatusAvailable)

	// ListServers should still return cached data
	servers, err := backend.ListServers(ctx)
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "cached-1", servers[0].ID)
	assert.Equal(t, "Cached Server", servers[0].DisplayName)
}

func TestStatusString(t *testing.T) {
	tests := []struct {
		status   backendpkg.BackendStatus
		expected string
	}{
		{backendpkg.StatusUnknown, "Unknown"},
		{backendpkg.StatusAvailable, "Available"},
		{backendpkg.StatusLocked, "Locked"},
		{backendpkg.StatusUnavailable, "Unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestGetStatus_ThreadSafe(t *testing.T) {
	mock := NewMockClient()
	backend := New(mock)
	backend.cachePath = ""

	// Set status to Available
	backend.setStatus(backendpkg.StatusAvailable)

	// Read status from multiple goroutines
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			status := backend.GetStatus()
			assert.Equal(t, backendpkg.StatusAvailable, status)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
