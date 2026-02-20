package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectInstallMethod_Homebrew(t *testing.T) {
	assert.Equal(t, installHomebrew, detectInstallMethod("/opt/homebrew/Cellar/ssherpa/0.2.0/bin/ssherpa"))
	assert.Equal(t, installHomebrew, detectInstallMethod("/usr/local/Cellar/ssherpa/0.2.0/bin/ssherpa"))
}

func TestDetectInstallMethod_Binary(t *testing.T) {
	assert.Equal(t, installBinary, detectInstallMethod("/usr/local/bin/ssherpa"))
	assert.Equal(t, installBinary, detectInstallMethod("/home/user/bin/ssherpa"))
}

func TestArchiveURL(t *testing.T) {
	url := archiveURL("0.3.0", "darwin", "arm64")
	assert.Equal(t, "https://github.com/FlorianRiquelme/ssherpa/releases/download/v0.3.0/ssherpa_0.3.0_darwin_arm64.tar.gz", url)
}

func TestChecksumsURL(t *testing.T) {
	url := checksumsURL("0.3.0")
	assert.Equal(t, "https://github.com/FlorianRiquelme/ssherpa/releases/download/v0.3.0/checksums.txt", url)
}

func TestVerifyChecksum(t *testing.T) {
	// Known SHA256 of "hello\n"
	data := []byte("hello\n")
	expected := "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"
	assert.True(t, verifyChecksum(data, expected))
	assert.False(t, verifyChecksum(data, "0000000000000000000000000000000000000000000000000000000000000000"))
}

func TestFindChecksumInFile(t *testing.T) {
	checksums := `abc123  ssherpa_0.3.0_linux_amd64.tar.gz
def456  ssherpa_0.3.0_darwin_arm64.tar.gz
ghi789  ssherpa_0.3.0_windows_amd64.zip`

	hash, err := findChecksumForFile(checksums, "ssherpa_0.3.0_darwin_arm64.tar.gz")
	assert.NoError(t, err)
	assert.Equal(t, "def456", hash)

	_, err = findChecksumForFile(checksums, "nonexistent.tar.gz")
	assert.Error(t, err)
}
