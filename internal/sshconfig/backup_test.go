package sshconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateBackup_CreatesBackupFile(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	backupPath := configPath + ".bak"

	// Create original file with known content
	originalContent := []byte("Host example\n    HostName example.com\n")
	err := os.WriteFile(configPath, originalContent, 0600)
	require.NoError(t, err)

	// Create backup
	err = CreateBackup(configPath)
	require.NoError(t, err)

	// Verify backup file exists
	assert.FileExists(t, backupPath)

	// Verify backup has same content
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, backupContent)
}

func TestCreateBackup_OverwritesExisting(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	backupPath := configPath + ".bak"

	// Create original file
	originalContent := []byte("Host example\n    HostName example.com\n")
	err := os.WriteFile(configPath, originalContent, 0600)
	require.NoError(t, err)

	// Create old backup with different content
	oldBackupContent := []byte("old backup content")
	err = os.WriteFile(backupPath, oldBackupContent, 0600)
	require.NoError(t, err)

	// Create new backup
	err = CreateBackup(configPath)
	require.NoError(t, err)

	// Verify backup was replaced with new content
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalContent, backupContent)
	assert.NotEqual(t, oldBackupContent, backupContent)
}

func TestCreateBackup_SourceNotFound(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent")

	// Try to create backup of non-existent file
	err := CreateBackup(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stat source file")
}

func TestCreateBackup_PreservesPermissions(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")
	backupPath := configPath + ".bak"

	// Create original file with specific permissions
	originalContent := []byte("Host example\n")
	err := os.WriteFile(configPath, originalContent, 0644)
	require.NoError(t, err)

	// Create backup
	err = CreateBackup(configPath)
	require.NoError(t, err)

	// Verify backup has same permissions
	originalInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	backupInfo, err := os.Stat(backupPath)
	require.NoError(t, err)
	assert.Equal(t, originalInfo.Mode(), backupInfo.Mode())
}

func TestAtomicWrite_WritesFile(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile")

	// Write data atomically
	data := []byte("test content\n")
	err := AtomicWrite(filePath, data, 0644)
	require.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, filePath)

	// Verify content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, data, content)

	// Verify permissions
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}

func TestAtomicWrite_OverwritesExisting(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "testfile")

	// Write initial content
	initialData := []byte("initial content\n")
	err := AtomicWrite(filePath, initialData, 0644)
	require.NoError(t, err)

	// Overwrite with new content
	newData := []byte("new content\n")
	err = AtomicWrite(filePath, newData, 0644)
	require.NoError(t, err)

	// Verify new content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, newData, content)
}

func TestAtomicWrite_AtomicOnError(t *testing.T) {
	// This test verifies that if AtomicWrite encounters an error,
	// it doesn't leave a corrupt file behind. We test this by
	// writing to a directory that doesn't exist.
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "nonexistent", "testfile")

	// Attempt to write to invalid path
	data := []byte("test content\n")
	err := AtomicWrite(invalidPath, data, 0644)
	assert.Error(t, err)

	// Verify no partial file was created
	_, err = os.Stat(invalidPath)
	assert.True(t, os.IsNotExist(err), "partial file should not exist")
}

func TestAtomicWrite_CreatesParentDirectory(t *testing.T) {
	// renameio should handle writing to existing directories,
	// but not create parent directories. This test verifies
	// the expected behavior when parent exists.
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	filePath := filepath.Join(subDir, "testfile")
	data := []byte("test content\n")
	err = AtomicWrite(filePath, data, 0644)
	require.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, filePath)

	// Verify content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, data, content)
}
