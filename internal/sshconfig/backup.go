package sshconfig

import (
	"fmt"
	"os"

	"github.com/google/renameio/v2/maybe"
)

// CreateBackup creates a backup of the specified config file.
// Copies configPath to configPath + ".bak", overwriting any existing backup.
// Uses the same file permissions as the original file.
// Returns an error if the source file doesn't exist.
func CreateBackup(configPath string) error {
	// Check if source file exists and get its permissions
	info, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("stat source file: %w", err)
	}

	// Read the source file contents
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	// Write backup file with same permissions
	backupPath := configPath + ".bak"
	if err := os.WriteFile(backupPath, data, info.Mode()); err != nil {
		return fmt.Errorf("write backup file: %w", err)
	}

	return nil
}

// AtomicWrite writes data to the specified path atomically.
// Uses renameio to write to a temp file in the same directory,
// then renames it to the target path. This prevents partial writes
// from corrupting the file.
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	// Use renameio to write atomically
	if err := maybe.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("atomic write: %w", err)
	}

	return nil
}
