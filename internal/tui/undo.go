package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/florianriquelme/sshjesus/internal/sshconfig"
)

// UndoEntry represents a deleted Host block that can be restored.
type UndoEntry struct {
	Alias       string    // The host alias that was deleted
	ConfigPath  string    // Path to the SSH config file
	RawLines    []string  // The raw text lines that were removed
	DeletedAt   time.Time // When the delete occurred
}

// UndoBuffer manages a session-scoped stack of deleted entries.
type UndoBuffer struct {
	entries []UndoEntry
	maxSize int
}

// NewUndoBuffer creates an undo buffer with the specified maximum size.
func NewUndoBuffer(maxSize int) *UndoBuffer {
	return &UndoBuffer{
		entries: make([]UndoEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Push adds an entry to the buffer. If the buffer is at capacity,
// the oldest entry is evicted.
func (b *UndoBuffer) Push(entry UndoEntry) {
	b.entries = append(b.entries, entry)

	// Evict oldest if over capacity
	if len(b.entries) > b.maxSize {
		b.entries = b.entries[1:]
	}
}

// Pop removes and returns the most recent entry.
// Returns (zero-value, false) if the buffer is empty.
func (b *UndoBuffer) Pop() (UndoEntry, bool) {
	if len(b.entries) == 0 {
		return UndoEntry{}, false
	}

	lastIdx := len(b.entries) - 1
	entry := b.entries[lastIdx]
	b.entries = b.entries[:lastIdx]

	return entry, true
}

// Len returns the number of entries in the buffer.
func (b *UndoBuffer) Len() int {
	return len(b.entries)
}

// IsEmpty returns true if the buffer has no entries.
func (b *UndoBuffer) IsEmpty() bool {
	return len(b.entries) == 0
}

// RestoreHost restores a deleted Host block to the SSH config file.
// This function is co-located with undo logic to keep restore functionality
// separate from the writer.go add/edit operations.
func RestoreHost(configPath string, rawLines []string) error {
	// Create backup before restoring
	if err := sshconfig.CreateBackup(configPath); err != nil {
		return fmt.Errorf("create backup: %w", err)
	}

	// Read current file contents
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	// Get file permissions for atomic write
	stat, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("stat config: %w", err)
	}

	// Build restored content: original + blank line + restored block
	restoredBlock := strings.Join(rawLines, "\n")
	var newContent string

	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		newContent = string(content) + "\n\n" + restoredBlock
	} else if len(content) > 0 {
		newContent = string(content) + "\n" + restoredBlock
	} else {
		newContent = restoredBlock
	}

	// Write atomically
	if err := sshconfig.AtomicWrite(configPath, []byte(newContent), stat.Mode().Perm()); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
