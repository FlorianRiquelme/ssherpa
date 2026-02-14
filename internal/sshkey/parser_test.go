package sshkey

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestKeys creates temporary SSH keys for testing
func setupTestKeys(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Create unencrypted ed25519 key (real key generated with ssh-keygen)
	ed25519Key := filepath.Join(tmpDir, "id_ed25519")
	err := os.WriteFile(ed25519Key, []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACA5MqIvhHUstWLx18zonLdo0D1hXvDkFS23aVdjryfF2AAAAJg5Eif7ORIn
+wAAAAtzc2gtZWQyNTUxOQAAACA5MqIvhHUstWLx18zonLdo0D1hXvDkFS23aVdjryfF2A
AAAEANaDZs1W7DPDnWOKHSY+kk3aCQ9jcZjqSMnM1u5OtdSjkyoi+EdSy1YvHXzOict2jQ
PWFe8OQVLbdpV2OvJ8XYAAAAE2Zsb3JpYW5AZXhhbXBsZS5jb20BAg==
-----END OPENSSH PRIVATE KEY-----
`), 0600)
	require.NoError(t, err)

	// Create companion .pub file with comment
	ed25519Pub := filepath.Join(tmpDir, "id_ed25519.pub")
	err = os.WriteFile(ed25519Pub, []byte(`ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDkyoi+EdSy1YvHXzOict2jQPWFe8OQVLbdpV2OvJ8XY florian@example.com
`), 0644)
	require.NoError(t, err)

	// Create encrypted ed25519 key (passphrase: "testpass")
	encryptedKey := filepath.Join(tmpDir, "id_encrypted")
	err = os.WriteFile(encryptedKey, []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAACmFlczI1Ni1jdHIAAAAGYmNyeXB0AAAAGAAAABBuvRvr+0
90sNB2diB7RqQYAAAAGAAAAAEAAAAzAAAAC3NzaC1lZDI1NTE5AAAAIOv6zy8f62PPTAui
XNb+Sw54/+NfXexAN6iy/F5U1HyUAAAAoLkBSOTkHO+MniJGVsvkbw1vK7zvyKftphvDmc
V7r2mkt3qLHqYWMoSXf9u6TuDb33hSmHJjykxW06Cid9SxU2BbDzo7Qhx9d5qVymikDSHA
slBSmluwuFFXZNra55G7dvftym3MjjtJA7EYYyelD9kvThyo6MBWqzS3Bz4odt3fPHNERX
F76dEp8x7I94+CkX48feVAAluu2hzSTmAvBnE=
-----END OPENSSH PRIVATE KEY-----
`), 0600)
	require.NoError(t, err)

	// Create .pub for encrypted key
	encryptedPub := filepath.Join(tmpDir, "id_encrypted.pub")
	err = os.WriteFile(encryptedPub, []byte(`ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOv6zy8f62PPTAuiXNb+Sw54/+NfXexAN6iy/F5U1HyU encrypted-key@example.com
`), 0644)
	require.NoError(t, err)

	// Create RSA key (real 2048-bit RSA key)
	rsaKey := filepath.Join(tmpDir, "id_rsa")
	err = os.WriteFile(rsaKey, []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
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

	// Create RSA .pub file
	rsaPub := filepath.Join(tmpDir, "id_rsa.pub")
	err = os.WriteFile(rsaPub, []byte(`ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCx1ZTAk3CKhCollkCKsvEZLcMhzojWBjvNIZ1/ycMXvwlISrEMWetjrt7shCj5BqPvUlxXalbsMJGKaPR65uriGR36/7LWcsyS1/u+jFNGmrfYaj7S49T6iwbtd/ftGG3iJfSSFE+cyB+cuPP6jEMxFxVakHt75Im70HwXfRCA10iD87DwgXUzxErO0AGqhSTUPVtMF28wqe9XTRgEMMZiyMAFi9rdEGibXjBx9Bkv52DTC8/y5qzYluhopwZRSH2VCC4WwnKUrqIhffzHYUT9X8NTUy1LC0ZMU/Vlo0A8G2Ka5ADT6uFe8D2UtO7TCEK3NCoNjxlN7tHLByv7F2At rsa-key@example.com
`), 0644)
	require.NoError(t, err)

	// Create a non-key file (should be skipped)
	nonKeyFile := filepath.Join(tmpDir, "config")
	err = os.WriteFile(nonKeyFile, []byte("Host example.com\n  User test\n"), 0644)
	require.NoError(t, err)

	return tmpDir
}

func TestParseKeyFile_Ed25519(t *testing.T) {
	tmpDir := setupTestKeys(t)
	keyPath := filepath.Join(tmpDir, "id_ed25519")

	key, err := ParseKeyFile(keyPath)
	require.NoError(t, err)
	require.NotNil(t, key)

	assert.Equal(t, keyPath, key.Path)
	assert.Equal(t, "id_ed25519", key.Filename)
	assert.Equal(t, "ed25519", key.Type)
	assert.NotEmpty(t, key.Fingerprint)
	assert.True(t, len(key.Fingerprint) > 10, "fingerprint should be populated")
	assert.Equal(t, "florian@example.com", key.Comment)
	assert.Equal(t, SourceFile, key.Source)
	assert.Equal(t, 256, key.Bits) // ed25519 is always 256 bits
	assert.False(t, key.Encrypted)
	assert.False(t, key.Missing)
}

func TestParseKeyFile_RSA(t *testing.T) {
	tmpDir := setupTestKeys(t)
	keyPath := filepath.Join(tmpDir, "id_rsa")

	key, err := ParseKeyFile(keyPath)
	require.NoError(t, err)
	require.NotNil(t, key)

	assert.Equal(t, "rsa", key.Type)
	assert.NotEmpty(t, key.Fingerprint)
	assert.Equal(t, "rsa-key@example.com", key.Comment)
	assert.Greater(t, key.Bits, 0, "RSA key should have bits > 0")
	assert.False(t, key.Encrypted)
}

func TestParseKeyFile_Encrypted(t *testing.T) {
	tmpDir := setupTestKeys(t)
	keyPath := filepath.Join(tmpDir, "id_encrypted")

	key, err := ParseKeyFile(keyPath)
	// Should still succeed - we read from .pub file instead
	require.NoError(t, err)
	require.NotNil(t, key)

	assert.Equal(t, "ed25519", key.Type)
	assert.NotEmpty(t, key.Fingerprint)
	assert.Equal(t, "encrypted-key@example.com", key.Comment)
	assert.True(t, key.Encrypted, "key should be marked as encrypted")
}

func TestParseKeyFile_NonKeyFile(t *testing.T) {
	tmpDir := setupTestKeys(t)
	configPath := filepath.Join(tmpDir, "config")

	key, err := ParseKeyFile(configPath)
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestParseKeyFile_MissingFile(t *testing.T) {
	key, err := ParseKeyFile("/nonexistent/path/to/key")
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestParseKeyFile_WithoutPubFile(t *testing.T) {
	tmpDir := setupTestKeys(t)
	keyPath := filepath.Join(tmpDir, "id_no_pub")

	// Create a key without .pub file (real key)
	err := os.WriteFile(keyPath, []byte(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACA5MqIvhHUstWLx18zonLdo0D1hXvDkFS23aVdjryfF2AAAAJg5Eif7ORIn
+wAAAAtzc2gtZWQyNTUxOQAAACA5MqIvhHUstWLx18zonLdo0D1hXvDkFS23aVdjryfF2A
AAAEANaDZs1W7DPDnWOKHSY+kk3aCQ9jcZjqSMnM1u5OtdSjkyoi+EdSy1YvHXzOict2jQ
PWFe8OQVLbdpV2OvJ8XYAAAAE2Zsb3JpYW5AZXhhbXBsZS5jb20BAg==
-----END OPENSSH PRIVATE KEY-----
`), 0600)
	require.NoError(t, err)

	key, err := ParseKeyFile(keyPath)
	require.NoError(t, err)
	require.NotNil(t, key)

	assert.Equal(t, "ed25519", key.Type)
	assert.NotEmpty(t, key.Fingerprint)
	assert.Empty(t, key.Comment, "comment should be empty when .pub file missing")
}

func TestReadPubKeyComment(t *testing.T) {
	tmpDir := setupTestKeys(t)

	tests := []struct {
		name     string
		pubFile  string
		expected string
	}{
		{
			name:     "ed25519 pub file",
			pubFile:  filepath.Join(tmpDir, "id_ed25519.pub"),
			expected: "florian@example.com",
		},
		{
			name:     "rsa pub file",
			pubFile:  filepath.Join(tmpDir, "id_rsa.pub"),
			expected: "rsa-key@example.com",
		},
		{
			name:     "nonexistent pub file",
			pubFile:  filepath.Join(tmpDir, "nonexistent.pub"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := ReadPubKeyComment(tt.pubFile)
			assert.Equal(t, tt.expected, comment)
		})
	}
}

func TestReadPubKeyComment_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	invalidPub := filepath.Join(tmpDir, "invalid.pub")
	err := os.WriteFile(invalidPub, []byte("not a valid public key format\n"), 0644)
	require.NoError(t, err)

	comment := ReadPubKeyComment(invalidPub)
	assert.Empty(t, comment, "should return empty string for invalid pub file")
}
