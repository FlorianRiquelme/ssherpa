package sshconfig

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// HostEntry represents an SSH config host entry for add/edit operations.
type HostEntry struct {
	Alias        string // Host alias (the name after "Host")
	Hostname     string // HostName directive value
	User         string // User directive value
	Port         string // Port directive value (empty = omit, use SSH default 22)
	IdentityFile string // IdentityFile path (empty = omit)
	ExtraConfig  string // Free-text extra SSH directives (multi-line, e.g. "ProxyJump bastion\nForwardAgent yes")
}

// AddHost adds a new Host block to the SSH config file.
// Creates a backup before writing. Returns an error if the alias already exists.
func AddHost(configPath string, entry HostEntry) error {
	// Create backup first
	if err := CreateBackup(configPath); err != nil {
		return fmt.Errorf("create backup: %w", err)
	}

	// Read existing file contents
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	// Check for duplicate alias
	if hostExists(string(content), entry.Alias) {
		return fmt.Errorf("host alias %q already exists", entry.Alias)
	}

	// Build new Host block
	block := buildHostBlock(entry)

	// Append to file with blank line separator
	var newContent string
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		newContent = string(content) + "\n\n" + block
	} else if len(content) > 0 {
		newContent = string(content) + "\n" + block
	} else {
		newContent = block
	}

	// Write atomically
	if err := AtomicWrite(configPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// EditHost modifies an existing Host block in the SSH config file.
// Creates a backup before writing. Returns an error if the host is not found
// or if renaming the alias would create a conflict.
func EditHost(configPath string, originalAlias string, entry HostEntry) error {
	// Create backup first
	if err := CreateBackup(configPath); err != nil {
		return fmt.Errorf("create backup: %w", err)
	}

	// Read existing file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Find the target block
	startIdx, endIdx, found := findHostBlock(lines, originalAlias)
	if !found {
		return fmt.Errorf("host %q not found", originalAlias)
	}

	// If alias changed, check for conflicts
	if !strings.EqualFold(originalAlias, entry.Alias) {
		// Build content without the original block for conflict checking
		var otherLines []string
		otherLines = append(otherLines, lines[:startIdx]...)
		otherLines = append(otherLines, lines[endIdx:]...)
		otherContent := strings.Join(otherLines, "\n")

		if hostExists(otherContent, entry.Alias) {
			return fmt.Errorf("host alias %q already exists", entry.Alias)
		}
	}

	// Build new block
	newBlock := buildHostBlock(entry)
	newBlockLines := strings.Split(newBlock, "\n")

	// Replace the block
	var newLines []string
	newLines = append(newLines, lines[:startIdx]...)
	newLines = append(newLines, newBlockLines...)
	newLines = append(newLines, lines[endIdx:]...)

	// Write atomically
	newContent := strings.Join(newLines, "\n")
	if err := AtomicWrite(configPath, []byte(newContent), 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// RemoveHost deletes a Host block from the SSH config file.
// Creates a backup before writing. Returns the removed block lines for undo.
func RemoveHost(configPath string, alias string) ([]string, error) {
	// Create backup first
	if err := CreateBackup(configPath); err != nil {
		return nil, fmt.Errorf("create backup: %w", err)
	}

	// Read existing file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Find the target block
	startIdx, endIdx, found := findHostBlock(lines, alias)
	if !found {
		return nil, fmt.Errorf("host %q not found", alias)
	}

	// Extract removed lines for undo
	removedLines := make([]string, endIdx-startIdx)
	copy(removedLines, lines[startIdx:endIdx])

	// Remove the block (also remove trailing blank line if present)
	var newLines []string
	newLines = append(newLines, lines[:startIdx]...)

	// Skip the block and one trailing blank line if it exists
	if endIdx < len(lines) && strings.TrimSpace(lines[endIdx]) == "" {
		endIdx++
	}

	newLines = append(newLines, lines[endIdx:]...)

	// Write atomically
	newContent := strings.Join(newLines, "\n")
	if err := AtomicWrite(configPath, []byte(newContent), 0600); err != nil {
		return nil, fmt.Errorf("write config: %w", err)
	}

	return removedLines, nil
}

// buildHostBlock creates a formatted Host block from a HostEntry.
func buildHostBlock(entry HostEntry) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("Host %s", entry.Alias))
	lines = append(lines, fmt.Sprintf("    HostName %s", entry.Hostname))
	lines = append(lines, fmt.Sprintf("    User %s", entry.User))

	if entry.Port != "" {
		lines = append(lines, fmt.Sprintf("    Port %s", entry.Port))
	}

	if entry.IdentityFile != "" {
		lines = append(lines, fmt.Sprintf("    IdentityFile %s", entry.IdentityFile))
	}

	// Add extra config lines (each line indented with 4 spaces)
	if entry.ExtraConfig != "" {
		extraLines := strings.Split(strings.TrimSpace(entry.ExtraConfig), "\n")
		for _, line := range extraLines {
			// If line is already indented, use as-is; otherwise add indent
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
					lines = append(lines, line)
				} else {
					lines = append(lines, "    "+trimmed)
				}
			}
		}
	}

	return strings.Join(lines, "\n")
}

// hostExists checks if a host alias already exists in the config content (case-insensitive).
func hostExists(content string, alias string) bool {
	scanner := bufio.NewScanner(strings.NewReader(content))
	aliasLower := strings.ToLower(alias)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Match "Host <alias>" lines (case-insensitive)
		if strings.HasPrefix(strings.ToLower(line), "host ") {
			// Extract the alias part
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				existingAlias := strings.ToLower(parts[1])
				if existingAlias == aliasLower {
					return true
				}
			}
		}
	}

	return false
}

// findHostBlock locates the start and end indices of a Host block for the given alias.
// Returns (startIdx, endIdx, found) where the block is lines[startIdx:endIdx].
func findHostBlock(lines []string, alias string) (int, int, bool) {
	aliasLower := strings.ToLower(alias)
	startIdx := -1

	// Find the start of the block
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "host ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 && strings.ToLower(parts[1]) == aliasLower {
				startIdx = i
				break
			}
		}
	}

	if startIdx == -1 {
		return 0, 0, false
	}

	// Find the end of the block (next Host line or EOF)
	endIdx := len(lines)
	for i := startIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		// End block at next Host line
		if strings.HasPrefix(strings.ToLower(trimmed), "host ") {
			endIdx = i
			break
		}
	}

	// Backtrack to exclude blank lines and comments between blocks
	// A block's content ends at the last non-blank, non-comment line that's indented
	for endIdx > startIdx+1 {
		line := lines[endIdx-1]
		trimmed := strings.TrimSpace(line)
		// If it's a blank line or comment (starts with #), it belongs between blocks
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			endIdx--
		} else {
			// Found actual content, this is the end
			break
		}
	}

	return startIdx, endIdx, true
}
