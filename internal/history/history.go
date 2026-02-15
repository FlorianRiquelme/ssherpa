package history

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// HistoryEntry represents a single SSH connection record
type HistoryEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	WorkingDir string    `json:"working_dir"`
	HostName   string    `json:"host_name"`
	Hostname   string    `json:"hostname"`
	User       string    `json:"user"`
}

// DefaultHistoryPath returns the default path for the history file
func DefaultHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "ssherpa_history.json")
}

// RecordConnection appends a connection record to the history file
func RecordConnection(path, hostName, hostname, user string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = ""
	}

	entry := HistoryEntry{
		Timestamp:  time.Now(),
		WorkingDir: workingDir,
		HostName:   hostName,
		Hostname:   hostname,
		User:       user,
	}

	if err := json.NewEncoder(f).Encode(entry); err != nil {
		return err
	}

	return nil
}

// GetLastConnectedForPath returns the most recent connection for a given working directory
func GetLastConnectedForPath(historyPath, workingDir string) (*HistoryEntry, error) {
	f, err := os.Open(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		var entry HistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Iterate backwards to find most recent match
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].WorkingDir == workingDir {
			return &entries[i], nil
		}
	}

	return nil, nil
}

// GetRecentHosts returns a map of hostname to most recent timestamp for the last N unique hosts
func GetRecentHosts(historyPath string, limit int) (map[string]time.Time, error) {
	f, err := os.Open(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]time.Time{}, nil
		}
		return nil, err
	}
	defer f.Close()

	hostMap := make(map[string]time.Time)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		var entry HistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines
			continue
		}

		// Update if we haven't seen this host or if this timestamp is newer
		if existing, exists := hostMap[entry.HostName]; !exists || entry.Timestamp.After(existing) {
			hostMap[entry.HostName] = entry.Timestamp
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// If we have more hosts than limit, we'd need to sort and trim
	// For now, return all hosts (TUI will handle limiting display)
	// This simplifies the implementation and is more flexible
	return hostMap, nil
}
