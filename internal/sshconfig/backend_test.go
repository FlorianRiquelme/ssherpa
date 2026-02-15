package sshconfig

import (
	"context"
	"os"
	"testing"

	"github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendNew(t *testing.T) {
	content := `
Host server1
    HostName example.com
    User alice
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)
	require.NotNil(t, b)
	assert.Len(t, b.hosts, 1)
}

func TestBackendNew_InvalidPath(t *testing.T) {
	_, err := New("/nonexistent/config")
	require.Error(t, err)

	var backendErr *errors.BackendError
	require.True(t, errors.As(err, &backendErr))
	assert.Equal(t, "New", backendErr.Op)
	assert.Equal(t, "sshconfig", backendErr.Backend)
}

func TestBackendListServers(t *testing.T) {
	content := `
Host server1
    HostName example.com
    User alice
    Port 22

Host server2
    HostName 192.168.1.100
    User bob
    Port 2222
    IdentityFile ~/.ssh/id_rsa

Host server3
    HostName prod.example.com
    User charlie
    IdentityFile ~/.ssh/prod_key
    ProxyJump bastion
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	servers, err := b.ListServers(ctx)
	require.NoError(t, err)
	require.Len(t, servers, 3)

	// Verify server1
	assert.Equal(t, "server1", servers[0].ID)
	assert.Equal(t, "server1", servers[0].DisplayName)
	assert.Equal(t, "example.com", servers[0].Host)
	assert.Equal(t, "alice", servers[0].User)
	assert.Equal(t, 22, servers[0].Port)
	assert.Empty(t, servers[0].IdentityFile)
	assert.Empty(t, servers[0].Proxy)
	assert.Empty(t, servers[0].Tags)
	assert.Contains(t, servers[0].Notes, "Source:")

	// Verify server2
	assert.Equal(t, "server2", servers[1].ID)
	assert.Equal(t, "192.168.1.100", servers[1].Host)
	assert.Equal(t, "bob", servers[1].User)
	assert.Equal(t, 2222, servers[1].Port)
	assert.Equal(t, "~/.ssh/id_rsa", servers[1].IdentityFile)
	assert.Empty(t, servers[1].Proxy)

	// Verify server3
	assert.Equal(t, "server3", servers[2].ID)
	assert.Equal(t, "prod.example.com", servers[2].Host)
	assert.Equal(t, "charlie", servers[2].User)
	assert.Equal(t, 22, servers[2].Port) // Default port
	assert.Equal(t, "~/.ssh/prod_key", servers[2].IdentityFile)
	assert.Equal(t, "bastion", servers[2].Proxy)
}

func TestBackendGetServer(t *testing.T) {
	content := `
Host server1
    HostName example.com
    User alice
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	server, err := b.GetServer(ctx, "server1")
	require.NoError(t, err)
	require.NotNil(t, server)

	assert.Equal(t, "server1", server.ID)
	assert.Equal(t, "example.com", server.Host)
	assert.Equal(t, "alice", server.User)
}

func TestBackendGetServer_NotFound(t *testing.T) {
	content := `
Host server1
    HostName example.com
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = b.GetServer(ctx, "nonexistent")
	require.Error(t, err)

	// Verify it's ErrServerNotFound
	assert.True(t, errors.Is(err, errors.ErrServerNotFound))

	var backendErr *errors.BackendError
	require.True(t, errors.As(err, &backendErr))
	assert.Equal(t, "GetServer", backendErr.Op)
	assert.Equal(t, "sshconfig", backendErr.Backend)
}

func TestBackendListProjects_Empty(t *testing.T) {
	content := `
Host server1
    HostName example.com
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	projects, err := b.ListProjects(ctx)
	require.NoError(t, err)
	assert.Empty(t, projects)
	assert.NotNil(t, projects) // Should be empty slice, not nil
}

func TestBackendGetProject_NotFound(t *testing.T) {
	content := `
Host server1
    HostName example.com
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = b.GetProject(ctx, "anyproject")
	require.Error(t, err)

	assert.True(t, errors.Is(err, errors.ErrProjectNotFound))

	var backendErr *errors.BackendError
	require.True(t, errors.As(err, &backendErr))
	assert.Equal(t, "GetProject", backendErr.Op)
}

func TestBackendListCredentials_Empty(t *testing.T) {
	content := `
Host server1
    HostName example.com
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	creds, err := b.ListCredentials(ctx)
	require.NoError(t, err)
	assert.Empty(t, creds)
	assert.NotNil(t, creds)
}

func TestBackendGetCredential_NotFound(t *testing.T) {
	content := `
Host server1
    HostName example.com
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = b.GetCredential(ctx, "anycred")
	require.Error(t, err)

	assert.True(t, errors.Is(err, errors.ErrCredentialNotFound))
}

func TestBackendClosed(t *testing.T) {
	content := `
Host server1
    HostName example.com
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	// Close the backend
	err = b.Close()
	require.NoError(t, err)

	ctx := context.Background()

	// All operations should return ErrBackendUnavailable
	_, err = b.GetServer(ctx, "server1")
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))

	_, err = b.ListServers(ctx)
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))

	_, err = b.GetProject(ctx, "proj")
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))

	_, err = b.ListProjects(ctx)
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))

	_, err = b.GetCredential(ctx, "cred")
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))

	_, err = b.ListCredentials(ctx)
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))
}

func TestBackendPortParsing(t *testing.T) {
	content := `
Host server1
    HostName example.com
    Port 2222

Host server2
    HostName example.com

Host server3
    HostName example.com
    Port invalid

Host server4
    HostName example.com
    Port 0

Host server5
    HostName example.com
    Port 99999
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	servers, err := b.ListServers(ctx)
	require.NoError(t, err)
	require.Len(t, servers, 5)

	// server1: explicit port 2222
	assert.Equal(t, 2222, servers[0].Port)

	// server2: no port specified, defaults to 22
	assert.Equal(t, 22, servers[1].Port)

	// server3: invalid port, defaults to 22
	assert.Equal(t, 22, servers[2].Port)

	// server4: port 0 (invalid), defaults to 22
	assert.Equal(t, 22, servers[3].Port)

	// server5: port > 65535 (invalid), defaults to 22
	assert.Equal(t, 22, servers[4].Port)
}

func TestBackendInterfaceCompliance(t *testing.T) {
	// Verify that Backend implements backend.Backend interface
	var _ backend.Backend = (*Backend)(nil)
}

func TestBackendHostnameFallback(t *testing.T) {
	// Test that if Hostname is empty, Host falls back to Name
	content := `
Host myserver.com
    User alice
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	server, err := b.GetServer(ctx, "myserver.com")
	require.NoError(t, err)

	// Hostname was empty, so Host should fall back to Name
	assert.Equal(t, "myserver.com", server.Host)
	assert.Equal(t, "myserver.com", server.ID)
}

func TestBackendParseErrorHandling(t *testing.T) {
	// Test that hosts with ParseError are included in listings
	content := `
Match host prod-*
    User admin
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	servers, err := b.ListServers(ctx)
	require.NoError(t, err)
	require.Len(t, servers, 1)

	// Should have a server with parse error in Notes
	assert.Contains(t, servers[0].Notes, "Parse error")
	assert.Contains(t, servers[0].Notes, "Match")
}

func TestBackendCopyOnRead(t *testing.T) {
	// Verify copy-on-read pattern: modifying returned server doesn't affect backend state
	content := `
Host server1
    HostName example.com
    User alice
`

	tmpFile := createTempConfig(t, content)
	defer os.Remove(tmpFile)

	b, err := New(tmpFile)
	require.NoError(t, err)

	ctx := context.Background()
	server1, err := b.GetServer(ctx, "server1")
	require.NoError(t, err)

	// Modify the returned server
	originalUser := server1.User
	server1.User = "modified"

	// Get the server again
	server2, err := b.GetServer(ctx, "server1")
	require.NoError(t, err)

	// Should still have original value (copy-on-read)
	assert.Equal(t, originalUser, server2.User)
	assert.NotEqual(t, "modified", server2.User)
}
