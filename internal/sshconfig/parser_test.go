package sshconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSSHConfig_ValidHosts(t *testing.T) {
	// Create temp config with 3 host blocks
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
`

	tmpFile := createTempConfig(t, content)
	defer func() { _ = os.Remove(tmpFile) }()

	hosts, err := ParseSSHConfig(tmpFile)
	require.NoError(t, err)
	require.Len(t, hosts, 3)

	// Verify server1
	assert.Equal(t, "server1", hosts[0].Name)
	assert.Equal(t, "example.com", hosts[0].Hostname)
	assert.Equal(t, "alice", hosts[0].User)
	assert.Equal(t, "22", hosts[0].Port)
	assert.False(t, hosts[0].IsWildcard)
	assert.Nil(t, hosts[0].ParseError)
	assert.NotEmpty(t, hosts[0].AllOptions)
	assert.Contains(t, hosts[0].AllOptions, "HostName")
	assert.Equal(t, []string{"example.com"}, hosts[0].AllOptions["HostName"])

	// Verify server2
	assert.Equal(t, "server2", hosts[1].Name)
	assert.Equal(t, "192.168.1.100", hosts[1].Hostname)
	assert.Equal(t, "bob", hosts[1].User)
	assert.Equal(t, "2222", hosts[1].Port)
	assert.Len(t, hosts[1].IdentityFile, 1)
	assert.Equal(t, "~/.ssh/id_rsa", hosts[1].IdentityFile[0])
	assert.False(t, hosts[1].IsWildcard)

	// Verify server3
	assert.Equal(t, "server3", hosts[2].Name)
	assert.Equal(t, "prod.example.com", hosts[2].Hostname)
	assert.Equal(t, "charlie", hosts[2].User)
	assert.Empty(t, hosts[2].Port) // No port specified
	assert.Len(t, hosts[2].IdentityFile, 1)
	assert.False(t, hosts[2].IsWildcard)

	// All should have SourceFile set
	for _, host := range hosts {
		absPath, _ := filepath.Abs(tmpFile)
		assert.Equal(t, absPath, host.SourceFile)
	}
}

func TestParseSSHConfig_WildcardDetection(t *testing.T) {
	content := `
Host *
    User default

Host *.example.com
    Port 2222

Host server1
    HostName example.com

Host prod-*
    User admin
`

	tmpFile := createTempConfig(t, content)
	defer func() { _ = os.Remove(tmpFile) }()

	hosts, err := ParseSSHConfig(tmpFile)
	require.NoError(t, err)
	require.Len(t, hosts, 4)

	// "Host *" should be wildcard
	assert.Equal(t, "*", hosts[0].Name)
	assert.True(t, hosts[0].IsWildcard)

	// "Host *.example.com" should be wildcard
	assert.Equal(t, "*.example.com", hosts[1].Name)
	assert.True(t, hosts[1].IsWildcard)

	// "Host server1" should NOT be wildcard
	assert.Equal(t, "server1", hosts[2].Name)
	assert.False(t, hosts[2].IsWildcard)

	// "Host prod-*" should be wildcard
	assert.Equal(t, "prod-*", hosts[3].Name)
	assert.True(t, hosts[3].IsWildcard)
}

func TestParseSSHConfig_MissingFile(t *testing.T) {
	_, err := ParseSSHConfig("/nonexistent/path/to/config")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open SSH config")
}

func TestParseSSHConfig_EmptyFile(t *testing.T) {
	tmpFile := createTempConfig(t, "")
	defer func() { _ = os.Remove(tmpFile) }()

	hosts, err := ParseSSHConfig(tmpFile)
	require.NoError(t, err)
	assert.Empty(t, hosts)
}

func TestParseSSHConfig_MalformedFile(t *testing.T) {
	// Match blocks are not supported by kevinburke/ssh_config
	content := `
Match host prod-*
    User admin
`

	tmpFile := createTempConfig(t, content)
	defer func() { _ = os.Remove(tmpFile) }()

	hosts, err := ParseSSHConfig(tmpFile)
	require.NoError(t, err) // Should NOT return fatal error
	require.Len(t, hosts, 1)

	// Should return a single SSHHost with ParseError set
	assert.NotNil(t, hosts[0].ParseError)
	assert.Contains(t, hosts[0].ParseError.Error(), "Match")
	assert.NotEmpty(t, hosts[0].SourceFile)
}

func TestParseSSHConfig_MultiValueKeys(t *testing.T) {
	content := `
Host server1
    HostName example.com
    IdentityFile ~/.ssh/id_rsa
    IdentityFile ~/.ssh/id_ed25519
    IdentityFile ~/.ssh/backup_key
    LocalForward 8080 localhost:8080
    LocalForward 9090 localhost:9090
`

	tmpFile := createTempConfig(t, content)
	defer func() { _ = os.Remove(tmpFile) }()

	hosts, err := ParseSSHConfig(tmpFile)
	require.NoError(t, err)
	require.Len(t, hosts, 1)

	host := hosts[0]

	// Verify all IdentityFile values captured
	assert.Len(t, host.IdentityFile, 3)
	assert.Equal(t, "~/.ssh/id_rsa", host.IdentityFile[0])
	assert.Equal(t, "~/.ssh/id_ed25519", host.IdentityFile[1])
	assert.Equal(t, "~/.ssh/backup_key", host.IdentityFile[2])

	// Verify AllOptions contains all values
	assert.Len(t, host.AllOptions["IdentityFile"], 3)
	assert.Equal(t, []string{
		"~/.ssh/id_rsa",
		"~/.ssh/id_ed25519",
		"~/.ssh/backup_key",
	}, host.AllOptions["IdentityFile"])

	// Verify LocalForward multi-values
	assert.Len(t, host.AllOptions["LocalForward"], 2)
	assert.Equal(t, []string{
		"8080 localhost:8080",
		"9090 localhost:9090",
	}, host.AllOptions["LocalForward"])
}

func TestOrganizeHosts(t *testing.T) {
	hosts := []SSHHost{
		{Name: "zebra", IsWildcard: false},
		{Name: "*", IsWildcard: true},
		{Name: "alpha", IsWildcard: false},
		{Name: "*.example.com", IsWildcard: true},
		{Name: "beta", IsWildcard: false},
		{Name: "prod-*", IsWildcard: true},
	}

	regular, wildcards := OrganizeHosts(hosts)

	// Verify separation
	require.Len(t, regular, 3)
	require.Len(t, wildcards, 3)

	// Verify regular hosts are sorted alphabetically
	assert.Equal(t, "alpha", regular[0].Name)
	assert.Equal(t, "beta", regular[1].Name)
	assert.Equal(t, "zebra", regular[2].Name)

	// Verify wildcard hosts are sorted alphabetically
	assert.Equal(t, "*", wildcards[0].Name)
	assert.Equal(t, "*.example.com", wildcards[1].Name)
	assert.Equal(t, "prod-*", wildcards[2].Name)

	// Verify all regular hosts have IsWildcard=false
	for _, h := range regular {
		assert.False(t, h.IsWildcard)
	}

	// Verify all wildcard hosts have IsWildcard=true
	for _, h := range wildcards {
		assert.True(t, h.IsWildcard)
	}
}

func TestOrganizeHosts_EmptyInput(t *testing.T) {
	regular, wildcards := OrganizeHosts(nil)
	assert.Nil(t, regular)
	assert.Nil(t, wildcards)

	regular, wildcards = OrganizeHosts([]SSHHost{})
	assert.Nil(t, regular)
	assert.Nil(t, wildcards)
}

// Helper: Create temporary SSH config file
func createTempConfig(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "ssh_config")

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	require.NoError(t, err)

	return tmpFile
}
