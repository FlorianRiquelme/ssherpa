package sshkey

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/florianriquelme/ssherpa/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSshDir creates a temporary ~/.ssh directory with various files
func setupSshDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create a real ed25519 key
	keyPath := filepath.Join(tmpDir, "id_ed25519")
	err := os.WriteFile(keyPath, []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACA5MqIvhHUstWLx18zonLdo0D1hXvDkFS23aVdjryfF2AAAAJg5Eif7ORIn
+wAAAAtzc2gtZWQyNTUxOQAAACA5MqIvhHUstWLx18zonLdo0D1hXvDkFS23aVdjryfF2A
AAAEANaDZs1W7DPDnWOKHSY+kk3aCQ9jcZjqSMnM1u5OtdSjkyoi+EdSy1YvHXzOict2jQ
PWFe8OQVLbdpV2OvJ8XYAAAAE2Zsb3JpYW5AZXhhbXBsZS5jb20BAg==
-----END OPENSSH PRIVATE KEY-----
`), 0600)
	require.NoError(t, err)

	pubPath := filepath.Join(tmpDir, "id_ed25519.pub")
	err = os.WriteFile(pubPath, []byte(`ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDkyoi+EdSy1YvHXzOict2jQPWFe8OQVLbdpV2OvJ8XY florian@example.com
`), 0644)
	require.NoError(t, err)

	// Create an RSA key
	rsaPath := filepath.Join(tmpDir, "id_rsa")
	err = os.WriteFile(rsaPath, []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAABFwAAAAdzc2gtcn
NhAAAAAwEAAQAAAQEAsdWUwJNwioQqJZZAirLxGS3DIc6I1gY7zSGdf8nDF78JSEqxDFnr
Y67e7IQo+Qaj71JcV2pW7DCRimj0eubq4hkd+v+y1nLMktf7voxTRpq32Go+0uPU+osG7X
f37Rht4iX0khRPnMgfnLjz+oxDMRcVWpB7e+SJu9B8F30QgNdIg/Ow8IF1M8RKztABqoUk
1D1bTBdvMKnvV00YBDDGYsjABYva3RBom14wcfQZL+dg0wvP8uas2JboaKcGUUh9lQguFs
JylK6iIX38x2FE/V/DU1MtSwtGTFP1ZaNAPBtimuQA0+rhXvA9lLTu0whCtzQqDY8ZTe7R
ywcr+xdgLQAAA8gN8bmFDfG5hQAAAAdzc2gtcnNhAAABAQCx1ZTAk3CKhCollkCKsvEZLc
MhzojWBjvNIZ1/ycMXvwlISrEMWetjrt7shCj5BqPvUlxXalbsMJGKaPR65uriGR36/7LW
csyS1/u+jFNGmrfYaj7S49T6iwbtd/ftGG3iJfSSFE+cyB+cuPP6jEMxFxVakHt75Im70H
wXfRCA10iD87DwgXUzxErO0AGqhSTUPVtMF28wqe9XTRgEMMZiyMAFi9rdEGibXjBx9Bkv
52DTC8/y5qzYluhopwZRSH2VCC4WwnKUrqIhffzHYUT9X8NTUy1LC0ZMU/Vlo0A8G2Ka5A
DT6uFe8D2UtO7TCEK3NCoNjxlN7tHLByv7F2AtAAAAAwEAAQAAAQAVPyVUlMj/Y6b9cqQn
bhWsInGL2ncyyu/eJEQC/oIWljZvsqzQgvXCpSPxMiELD6YKc9aggw37jhn1ZXDANlKdzM
5uLJqbUH/sk216aJ2Sc+2/J4J2A85wWKftO0Ydx6tpN4uu4EpauvY77UUJDDUC6nUcquJ1
/OoPzGrnC4QrQFphJJwwl6gQrmYEo5VZvShb34vjL83znaaTlYlDiflrYJfNfSJhMa3uJR
vHq0BtqaspoCD6oX7AjVoTiEZ4NaJ9IERtRKAIOJUL0FkYHvngddlsfeCfEwifHAFpneYv
eCBeEQGGYryICDVETPz0/aHhAUphITaYHjCmZzl/F9vBAAAAgEAJvea0ZrwX3bjqfilWHf
6RzooDlT9dPg3XV9uUWaCB0ohLZMrNj++eHTbvnkl8QYxO2KBtwHIkmg6ivvYivtXZ935Y
X8aAK5OSliXd0+8YX8x95Xy7nUeAso/fWx/UDoJrzeXgk00yMXu2Q/Y/N05ptYXsVzrGQU
1/utFOP/cYAAAAgQDcJRXmtr6biykcM59h81jQnK/Sp/cUypUaTy13vWLuyfi/qnKs+SEP
8nxaQgzzqHmYDl6+dFpAdINTVK6NU0w4x1OZv5uAhSl9aEwt2TdG5H5cNXf+49MSziuTBQ
BkiDfs/8wyAy1sySHfOgkLonpvy5jZ7zQsJ/SabMZ3JyQh5QAAAIEAzsxciWKidf5yjWMZ
R7kvAboji49G9gt26QAkb+snIggKKNRMxqErmUUxz33xvIyGq7FoyCr3GAlxdg3xedOLUX
bX54f2tgFgOOpz4y+NH4eYCh8fFaejtiLrOl8W3Z6gcMK9Unbi5MS7TDmCt2EegKO0ks1Z
0uI1TaWIDLN5AKkAAAATcnNhLWtleUBleGFtcGxlLmNvbQ==
-----END OPENSSH PRIVATE KEY-----
`), 0600)
	require.NoError(t, err)

	rsaPubPath := filepath.Join(tmpDir, "id_rsa.pub")
	err = os.WriteFile(rsaPubPath, []byte(`ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCx1ZTAk3CKhCollkCKsvEZLcMhzojWBjvNIZ1/ycMXvwlISrEMWetjrt7shCj5BqPvUlxXalbsMJGKaPR65uriGR36/7LWcsyS1/u+jFNGmrfYaj7S49T6iwbtd/ftGG3iJfSSFE+cyB+cuPP6jEMxFxVakHt75Im70HwXfRCA10iD87DwgXUzxErO0AGqhSTUPVtMF28wqe9XTRgEMMZiyMAFi9rdEGibXjBx9Bkv52DTC8/y5qzYluhopwZRSH2VCC4WwnKUrqIhffzHYUT9X8NTUy1LC0ZMU/Vlo0A8G2Ka5ADT6uFe8D2UtO7TCEK3NCoNjxlN7tHLByv7F2At rsa-key@example.com
`), 0644)
	require.NoError(t, err)

	// Create files that should be SKIPPED
	// .pub file (already have the private keys)
	// known_hosts
	knownHostsPath := filepath.Join(tmpDir, "known_hosts")
	err = os.WriteFile(knownHostsPath, []byte("github.com ssh-rsa AAAAB3...\n"), 0644)
	require.NoError(t, err)

	// config file
	configPath := filepath.Join(tmpDir, "config")
	err = os.WriteFile(configPath, []byte("Host example\n  User test\n"), 0644)
	require.NoError(t, err)

	// authorized_keys
	authKeysPath := filepath.Join(tmpDir, "authorized_keys")
	err = os.WriteFile(authKeysPath, []byte("ssh-ed25519 AAAAC3... user@host\n"), 0644)
	require.NoError(t, err)

	// A file with no SSH header (should be skipped)
	randomFile := filepath.Join(tmpDir, "notes.txt")
	err = os.WriteFile(randomFile, []byte("Some random notes\n"), 0644)
	require.NoError(t, err)

	return tmpDir
}

func TestDiscoverFileKeys(t *testing.T) {
	sshDir := setupSshDir(t)

	keys, err := DiscoverFileKeys(sshDir)
	require.NoError(t, err)

	// Should find exactly 2 keys (id_ed25519 and id_rsa)
	assert.Len(t, keys, 2, "should find 2 private keys")

	// Verify all keys have required fields
	for _, key := range keys {
		assert.Equal(t, SourceFile, key.Source)
		assert.NotEmpty(t, key.Path)
		assert.NotEmpty(t, key.Filename)
		assert.NotEmpty(t, key.Type)
		assert.NotEmpty(t, key.Fingerprint)
		assert.False(t, key.Missing)
	}

	// Check that specific keys were found
	var hasEd25519, hasRSA bool
	for _, key := range keys {
		if key.Type == "ed25519" {
			hasEd25519 = true
			assert.Equal(t, "id_ed25519", key.Filename)
		}
		if key.Type == "rsa" {
			hasRSA = true
			assert.Equal(t, "id_rsa", key.Filename)
		}
	}
	assert.True(t, hasEd25519, "should find ed25519 key")
	assert.True(t, hasRSA, "should find RSA key")
}

func TestDiscoverFileKeys_SkipsNonKeys(t *testing.T) {
	sshDir := setupSshDir(t)

	keys, err := DiscoverFileKeys(sshDir)
	require.NoError(t, err)

	// Verify that non-key files were skipped
	for _, key := range keys {
		assert.NotEqual(t, "known_hosts", key.Filename)
		assert.NotEqual(t, "config", key.Filename)
		assert.NotEqual(t, "authorized_keys", key.Filename)
		assert.NotEqual(t, "notes.txt", key.Filename)
		assert.NotContains(t, key.Filename, ".pub")
	}
}

func TestDiscoverFileKeys_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	keys, err := DiscoverFileKeys(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, keys, "should return empty slice for directory with no keys")
}

func TestDiscoverFileKeys_NonexistentDir(t *testing.T) {
	keys, err := DiscoverFileKeys("/nonexistent/directory/path")
	// Should not error, just return empty
	assert.NoError(t, err)
	assert.Empty(t, keys)
}

func TestCreateMissingKeyEntry(t *testing.T) {
	path := "/home/user/.ssh/missing_key"

	key := CreateMissingKeyEntry(path)

	assert.True(t, key.Missing)
	assert.Equal(t, path, key.MissingPath)
	assert.Equal(t, "missing_key", key.Filename)
	assert.Equal(t, SourceFile, key.Source)
	assert.Empty(t, key.Type)
	assert.Empty(t, key.Fingerprint)
}

func TestDiscover1PasswordKeys(t *testing.T) {
	// Create temporary directory with a test key
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "1p_key")
	err := os.WriteFile(keyPath, []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACA5MqIvhHUstWLx18zonLdo0D1hXvDkFS23aVdjryfF2AAAAJg5Eif7ORIn
+wAAAAtzc2gtZWQyNTUxOQAAACA5MqIvhHUstWLx18zonLdo0D1hXvDkFS23aVdjryfF2A
AAAEANaDZs1W7DPDnWOKHSY+kk3aCQ9jcZjqSMnM1u5OtdSjkyoi+EdSy1YvHXzOict2jQ
PWFe8OQVLbdpV2OvJ8XYAAAAE2Zsb3JpYW5AZXhhbXBsZS5jb20BAg==
-----END OPENSSH PRIVATE KEY-----
`), 0600)
	require.NoError(t, err)

	servers := []*domain.Server{
		{
			ID:           "server1",
			DisplayName:  "Production Server",
			IdentityFile: keyPath,
		},
		{
			ID:           "server2",
			DisplayName:  "Staging Server",
			IdentityFile: keyPath, // Same key referenced twice
		},
		{
			ID:           "server3",
			DisplayName:  "Dev Server",
			IdentityFile: "/nonexistent/key",
		},
		{
			ID:          "server4",
			DisplayName: "No Key Server",
			// No IdentityFile set
		},
	}

	keys := Discover1PasswordKeys(servers)

	// Should find 2 unique keys (keyPath and /nonexistent/key)
	// Server with no IdentityFile is skipped
	// Duplicate keyPath is deduplicated
	assert.Len(t, keys, 2, "should find 2 unique identity files")

	// Check that the existing key was parsed
	var hasRealKey, hasMissingKey bool
	for _, key := range keys {
		assert.Equal(t, Source1Password, key.Source)

		if key.Path == keyPath {
			hasRealKey = true
			assert.Equal(t, "ed25519", key.Type)
			assert.NotEmpty(t, key.Fingerprint)
			assert.False(t, key.Missing)
		}

		if key.MissingPath == "/nonexistent/key" {
			hasMissingKey = true
			assert.True(t, key.Missing)
			assert.Empty(t, key.Type)
		}
	}
	assert.True(t, hasRealKey, "should parse existing key")
	assert.True(t, hasMissingKey, "should mark missing key")
}

func TestDiscoverKeys_Deduplication(t *testing.T) {
	// This test verifies that keys with the same fingerprint are deduplicated
	// For now, we'll create a simple test that just verifies the function runs
	// Full deduplication testing would require mocking the agent

	sshDir := setupSshDir(t)

	// No servers for this test
	keys, err := DiscoverKeys(sshDir, nil)
	require.NoError(t, err)

	// Should find the file keys at minimum
	assert.GreaterOrEqual(t, len(keys), 2, "should find at least file keys")

	// Verify all keys have unique combinations of source+path or source+fingerprint
	seen := make(map[string]bool)
	for _, key := range keys {
		// Create a unique identifier
		var id string
		if key.Source == SourceAgent {
			id = "agent:" + key.Fingerprint
		} else {
			id = key.Source.String() + ":" + key.Path
		}

		assert.False(t, seen[id], "duplicate key found: %s", id)
		seen[id] = true
	}
}

func TestDiscoverKeys_Sorting(t *testing.T) {
	sshDir := setupSshDir(t)

	keys, err := DiscoverKeys(sshDir, nil)
	require.NoError(t, err)

	// Verify that file keys come first
	var lastFileIndex int
	for i, key := range keys {
		if key.Source == SourceFile {
			lastFileIndex = i
		}
	}

	// All file keys should be before any non-file keys
	for i := 0; i < lastFileIndex; i++ {
		assert.Equal(t, SourceFile, keys[i].Source, "file keys should come first")
	}
}
