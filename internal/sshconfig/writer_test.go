package sshconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddHost_AppendsToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create existing config
	existingContent := `Host example
    HostName example.com
    User alice

Host another
    HostName another.com
    User bob
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Add new host
	entry := HostEntry{
		Alias:    "newhost",
		Hostname: "newhost.com",
		User:     "charlie",
	}
	err = AddHost(configPath, entry)
	require.NoError(t, err)

	// Read back and verify
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Verify existing content is unchanged
	assert.Contains(t, string(content), "Host example")
	assert.Contains(t, string(content), "Host another")

	// Verify new host is at the end
	assert.Contains(t, string(content), "Host newhost")
	assert.Contains(t, string(content), "HostName newhost.com")
	assert.Contains(t, string(content), "User charlie")
}

func TestAddHost_DuplicateAlias(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create existing config
	existingContent := `Host example
    HostName example.com
    User alice
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Try to add duplicate
	entry := HostEntry{
		Alias:    "example",
		Hostname: "different.com",
		User:     "bob",
	}
	err = AddHost(configPath, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAddHost_DuplicateAlias_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create existing config with lowercase alias
	existingContent := `Host example
    HostName example.com
    User alice
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Try to add duplicate with different case
	entry := HostEntry{
		Alias:    "EXAMPLE",
		Hostname: "different.com",
		User:     "bob",
	}
	err = AddHost(configPath, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAddHost_PreservesComments(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config with comments
	existingContent := `# This is a comment
Host example
    HostName example.com
    # Inline comment
    User alice

# Another comment
Host another
    HostName another.com
    User bob
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Add new host
	entry := HostEntry{
		Alias:    "newhost",
		Hostname: "newhost.com",
		User:     "charlie",
	}
	err = AddHost(configPath, entry)
	require.NoError(t, err)

	// Read back and verify comments are preserved
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "# This is a comment")
	assert.Contains(t, string(content), "# Inline comment")
	assert.Contains(t, string(content), "# Another comment")
}

func TestAddHost_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create empty config
	err := os.WriteFile(configPath, []byte(""), 0600)
	require.NoError(t, err)

	// Add host with all fields
	entry := HostEntry{
		Alias:        "fullhost",
		Hostname:     "full.example.com",
		User:         "admin",
		Port:         "2222",
		IdentityFile: "~/.ssh/custom_key",
		ExtraConfig:  "ForwardAgent yes\nProxyJump bastion",
	}
	err = AddHost(configPath, entry)
	require.NoError(t, err)

	// Read back and verify all fields
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "Host fullhost")
	assert.Contains(t, string(content), "HostName full.example.com")
	assert.Contains(t, string(content), "User admin")
	assert.Contains(t, string(content), "Port 2222")
	assert.Contains(t, string(content), "IdentityFile ~/.ssh/custom_key")
	assert.Contains(t, string(content), "ForwardAgent yes")
	assert.Contains(t, string(content), "ProxyJump bastion")
}

func TestAddHost_MinimalFields(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create empty config
	err := os.WriteFile(configPath, []byte(""), 0600)
	require.NoError(t, err)

	// Add host with minimal fields (no port, identity, or extra)
	entry := HostEntry{
		Alias:    "minimal",
		Hostname: "minimal.com",
		User:     "user",
	}
	err = AddHost(configPath, entry)
	require.NoError(t, err)

	// Read back and verify
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "Host minimal")
	assert.Contains(t, string(content), "HostName minimal.com")
	assert.Contains(t, string(content), "User user")
	assert.NotContains(t, string(content), "Port")
	assert.NotContains(t, string(content), "IdentityFile")
}

func TestAddHost_WithExtraConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create empty config
	err := os.WriteFile(configPath, []byte(""), 0600)
	require.NoError(t, err)

	// Add host with extra config
	entry := HostEntry{
		Alias:       "extrahost",
		Hostname:    "extra.com",
		User:        "user",
		ExtraConfig: "StrictHostKeyChecking no\nUserKnownHostsFile /dev/null",
	}
	err = AddHost(configPath, entry)
	require.NoError(t, err)

	// Read back and verify indentation
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	lines := strings.Split(string(content), "\n")
	var foundExtra bool
	for _, line := range lines {
		if strings.Contains(line, "StrictHostKeyChecking") {
			foundExtra = true
			// Verify it's indented
			assert.True(t, strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t"))
		}
	}
	assert.True(t, foundExtra, "ExtraConfig should be present")
}

func TestAddHost_CreatesBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	backupPath := configPath + ".bak"

	// Create existing config
	existingContent := `Host example
    HostName example.com
    User alice
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Add new host
	entry := HostEntry{
		Alias:    "newhost",
		Hostname: "newhost.com",
		User:     "charlie",
	}
	err = AddHost(configPath, entry)
	require.NoError(t, err)

	// Verify backup was created
	assert.FileExists(t, backupPath)

	// Verify backup has original content
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(backupContent))
}

func TestEditHost_ChangesOnlyTargetBlock(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config with multiple hosts
	existingContent := `Host first
    HostName first.com
    User alice

Host target
    HostName old.com
    User bob

Host third
    HostName third.com
    User charlie
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Edit the target host
	entry := HostEntry{
		Alias:    "target",
		Hostname: "new.com",
		User:     "bobby",
		Port:     "2222",
	}
	err = EditHost(configPath, "target", entry)
	require.NoError(t, err)

	// Read back and verify
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Verify first and third hosts are unchanged
	assert.Contains(t, string(content), "Host first")
	assert.Contains(t, string(content), "HostName first.com")
	assert.Contains(t, string(content), "User alice")
	assert.Contains(t, string(content), "Host third")
	assert.Contains(t, string(content), "HostName third.com")
	assert.Contains(t, string(content), "User charlie")

	// Verify target host is updated
	assert.Contains(t, string(content), "Host target")
	assert.Contains(t, string(content), "HostName new.com")
	assert.Contains(t, string(content), "User bobby")
	assert.Contains(t, string(content), "Port 2222")

	// Verify old values are gone
	assert.NotContains(t, string(content), "HostName old.com")
	// Check that "User bob" as a complete value is gone (not just substring)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		assert.NotEqual(t, "User bob", trimmed, "Old user value should be replaced")
	}
}

func TestEditHost_PreservesComments(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config with comments
	existingContent := `# Header comment
Host first
    HostName first.com
    User alice

# Comment before target
Host target
    HostName old.com
    User bob

# Comment after target
Host third
    HostName third.com
    User charlie
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Edit target host
	entry := HostEntry{
		Alias:    "target",
		Hostname: "new.com",
		User:     "bobby",
	}
	err = EditHost(configPath, "target", entry)
	require.NoError(t, err)

	// Read back and verify comments are preserved
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "# Header comment")
	assert.Contains(t, string(content), "# Comment before target")
	assert.Contains(t, string(content), "# Comment after target")
}

func TestEditHost_AliasChange(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config
	existingContent := `Host oldname
    HostName example.com
    User alice
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Edit with new alias
	entry := HostEntry{
		Alias:    "newname",
		Hostname: "example.com",
		User:     "alice",
	}
	err = EditHost(configPath, "oldname", entry)
	require.NoError(t, err)

	// Read back and verify
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "Host newname")
	assert.NotContains(t, string(content), "Host oldname")
}

func TestEditHost_AliasConflict(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config with two hosts
	existingContent := `Host first
    HostName first.com
    User alice

Host second
    HostName second.com
    User bob
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Try to rename second to first (conflict)
	entry := HostEntry{
		Alias:    "first",
		Hostname: "second.com",
		User:     "bob",
	}
	err = EditHost(configPath, "second", entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestEditHost_HostNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config
	existingContent := `Host example
    HostName example.com
    User alice
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Try to edit non-existent host
	entry := HostEntry{
		Alias:    "nonexistent",
		Hostname: "new.com",
		User:     "bob",
	}
	err = EditHost(configPath, "nonexistent", entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoveHost_RemovesBlock(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config with multiple hosts
	existingContent := `Host first
    HostName first.com
    User alice

Host target
    HostName target.com
    User bob

Host third
    HostName third.com
    User charlie
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Remove target host
	removedLines, err := RemoveHost(configPath, "target")
	require.NoError(t, err)
	assert.NotEmpty(t, removedLines)

	// Read back and verify
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Verify first and third hosts are preserved
	assert.Contains(t, string(content), "Host first")
	assert.Contains(t, string(content), "Host third")

	// Verify target is gone
	assert.NotContains(t, string(content), "Host target")
	assert.NotContains(t, string(content), "HostName target.com")
}

func TestRemoveHost_ReturnsLines(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config
	existingContent := `Host target
    HostName target.com
    User bob
    Port 2222
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Remove host
	removedLines, err := RemoveHost(configPath, "target")
	require.NoError(t, err)

	// Verify returned lines contain the block
	removedContent := strings.Join(removedLines, "\n")
	assert.Contains(t, removedContent, "Host target")
	assert.Contains(t, removedContent, "HostName target.com")
	assert.Contains(t, removedContent, "User bob")
	assert.Contains(t, removedContent, "Port 2222")
}

func TestRemoveHost_LastBlock(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config with single host
	existingContent := `Host onlyhost
    HostName only.com
    User alice
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Remove the only host
	removedLines, err := RemoveHost(configPath, "onlyhost")
	require.NoError(t, err)
	assert.NotEmpty(t, removedLines)

	// Read back and verify file is mostly empty (may have trailing newlines)
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Should not contain the host
	assert.NotContains(t, string(content), "Host onlyhost")
	assert.NotContains(t, string(content), "HostName only.com")
}

func TestRemoveHost_HostNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config
	existingContent := `Host example
    HostName example.com
    User alice
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Try to remove non-existent host
	removedLines, err := RemoveHost(configPath, "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, removedLines)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoveHost_PreservesOtherBlocks(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create config with comments and blank lines
	existingContent := `# Header comment

Host first
    HostName first.com
    User alice

# Middle comment
Host target
    HostName target.com
    User bob

Host third
    HostName third.com
    # Inline comment
    User charlie
`
	err := os.WriteFile(configPath, []byte(existingContent), 0600)
	require.NoError(t, err)

	// Remove target
	_, err = RemoveHost(configPath, "target")
	require.NoError(t, err)

	// Read back and verify
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	// Verify other blocks and comments are preserved
	assert.Contains(t, string(content), "# Header comment")
	assert.Contains(t, string(content), "Host first")
	assert.Contains(t, string(content), "Host third")
	assert.Contains(t, string(content), "# Inline comment")

	// Verify target is gone
	assert.NotContains(t, string(content), "Host target")
}
