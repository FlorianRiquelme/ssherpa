package sshkey

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverAgentKeys_NoSocket(t *testing.T) {
	// Save original env var
	originalSocket := os.Getenv("SSH_AUTH_SOCK")
	defer func() { _ = os.Setenv("SSH_AUTH_SOCK", originalSocket) }()

	// Clear SSH_AUTH_SOCK
	_ = os.Unsetenv("SSH_AUTH_SOCK")

	keys, err := DiscoverAgentKeys()
	require.NoError(t, err, "should not error when agent unavailable")
	assert.Empty(t, keys, "should return empty slice when no agent")
}

func TestDiscoverAgentKeys_InvalidSocket(t *testing.T) {
	// Save original env var
	originalSocket := os.Getenv("SSH_AUTH_SOCK")
	defer func() { _ = os.Setenv("SSH_AUTH_SOCK", originalSocket) }()

	// Set invalid socket path
	_ = os.Setenv("SSH_AUTH_SOCK", "/nonexistent/socket/path")

	keys, err := DiscoverAgentKeys()
	require.NoError(t, err, "should not error when agent unreachable")
	assert.Empty(t, keys, "should return empty slice when agent unreachable")
}

func TestDiscoverAgentKeys_RealAgent(t *testing.T) {
	// This test only runs if SSH_AUTH_SOCK is set (integration test)
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		t.Skip("Skipping integration test: SSH_AUTH_SOCK not set")
	}

	keys, err := DiscoverAgentKeys()
	require.NoError(t, err)

	// If agent is available, we should get some keys (or empty list if no keys loaded)
	// We can't assert specific keys, but we can verify the structure
	for _, key := range keys {
		assert.Equal(t, SourceAgent, key.Source, "all keys should have SourceAgent")
		assert.NotEmpty(t, key.Type, "key type should be populated")
		assert.NotEmpty(t, key.Fingerprint, "fingerprint should be populated")
		// Comment may be empty for some keys
		// Path and Filename should be empty for agent keys
		assert.Empty(t, key.Path, "agent keys should not have path")
		assert.Empty(t, key.Filename, "agent keys should not have filename")
	}

	t.Logf("Found %d keys in SSH agent", len(keys))
}
