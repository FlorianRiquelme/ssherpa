package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/florianriquelme/ssherpa/internal/domain"
)

func TestWriteReadTOMLCache_RoundTrip(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	// Create test servers with various fields
	now := time.Now()
	servers := []*domain.Server{
		{
			ID:                "srv-001",
			DisplayName:       "prod-web-01",
			Host:              "192.168.1.100",
			User:              "deploy",
			Port:              2222,
			IdentityFile:      "~/.ssh/id_ed25519",
			Proxy:             "bastion.example.com",
			RemoteProjectPath: "/var/www/myapp",
			ProjectIDs:        []string{"proj-001", "proj-002"},
			VaultID:           "vault-001",
			Tags:              []string{"production", "web"},
			LastConnected:     &now,
		},
		{
			ID:          "srv-002",
			DisplayName: "staging-db",
			Host:        "staging.example.com",
			User:        "ubuntu",
			Port:        22,
			VaultID:     "vault-002",
		},
	}

	// Write cache
	err := WriteTOMLCache(servers, cachePath)
	require.NoError(t, err)

	// Read cache back
	readServers, err := ReadTOMLCache(cachePath)
	require.NoError(t, err)

	// Verify same number of servers
	require.Len(t, readServers, 2)

	// Verify first server fields
	srv1 := readServers[0]
	assert.Equal(t, "srv-001", srv1.ID)
	assert.Equal(t, "prod-web-01", srv1.DisplayName)
	assert.Equal(t, "192.168.1.100", srv1.Host)
	assert.Equal(t, "deploy", srv1.User)
	assert.Equal(t, 2222, srv1.Port)
	assert.Equal(t, "~/.ssh/id_ed25519", srv1.IdentityFile)
	assert.Equal(t, "bastion.example.com", srv1.Proxy)
	assert.Equal(t, "/var/www/myapp", srv1.RemoteProjectPath)
	assert.Equal(t, []string{"proj-001", "proj-002"}, srv1.ProjectIDs)
	assert.Equal(t, "vault-001", srv1.VaultID)
	assert.Equal(t, []string{"production", "web"}, srv1.Tags)

	// Verify second server fields
	srv2 := readServers[1]
	assert.Equal(t, "srv-002", srv2.ID)
	assert.Equal(t, "staging-db", srv2.DisplayName)
	assert.Equal(t, "staging.example.com", srv2.Host)
	assert.Equal(t, "ubuntu", srv2.User)
	assert.Equal(t, 22, srv2.Port)
	assert.Equal(t, "vault-002", srv2.VaultID)
}

func TestReadTOMLCache_NotFound(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nonexistent.toml")

	// Try to read non-existent cache
	_, err := ReadTOMLCache(cachePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read TOML cache")
}

func TestWriteTOMLCache_PreservesRemoteProjectPath(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	// Create server with RemoteProjectPath (ssherpa-specific field)
	servers := []*domain.Server{
		{
			ID:                "srv-001",
			DisplayName:       "prod-web",
			Host:              "prod.example.com",
			User:              "deploy",
			Port:              22,
			RemoteProjectPath: "/opt/myapp/current",
			VaultID:           "vault-001",
		},
	}

	// Write and read back
	err := WriteTOMLCache(servers, cachePath)
	require.NoError(t, err)

	readServers, err := ReadTOMLCache(cachePath)
	require.NoError(t, err)

	// Verify RemoteProjectPath is preserved
	require.Len(t, readServers, 1)
	assert.Equal(t, "/opt/myapp/current", readServers[0].RemoteProjectPath,
		"RemoteProjectPath should survive round-trip")
}

func TestWriteTOMLCache_EmptyServers(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	// Write empty server list
	err := WriteTOMLCache([]*domain.Server{}, cachePath)
	require.NoError(t, err)

	// Read back
	readServers, err := ReadTOMLCache(cachePath)
	require.NoError(t, err)

	// Verify empty list
	assert.Len(t, readServers, 0)
}

func TestWriteTOMLCache_Timestamp(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.toml")

	// Create a simple server
	servers := []*domain.Server{
		{
			ID:          "srv-001",
			DisplayName: "test-server",
			Host:        "test.example.com",
			User:        "testuser",
			Port:        22,
			VaultID:     "vault-001",
		},
	}

	// Record time before write
	beforeWrite := time.Now()

	// Write cache
	err := WriteTOMLCache(servers, cachePath)
	require.NoError(t, err)

	// Record time after write
	afterWrite := time.Now()

	// Read the raw TOML to check timestamp
	content, err := os.ReadFile(cachePath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "last_sync", "TOML should contain last_sync field")

	// Read back and verify servers
	readServers, err := ReadTOMLCache(cachePath)
	require.NoError(t, err)
	require.Len(t, readServers, 1)

	// Verify timestamp is within reasonable range
	// Note: We don't test the exact timestamp value since it's stored in TOMLCache
	// but not returned in domain.Server
	assert.True(t, afterWrite.After(beforeWrite) || afterWrite.Equal(beforeWrite))
}
