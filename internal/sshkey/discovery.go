package sshkey

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/florianriquelme/sshjesus/internal/domain"
)

// DiscoverFileKeys discovers SSH private keys in the specified directory (usually ~/.ssh/).
// Uses header sniffing to identify private keys, not filename conventions.
// Skips .pub files, known_hosts, config, authorized_keys, and other non-key files.
func DiscoverFileKeys(sshDir string) ([]SSHKey, error) {
	var keys []SSHKey

	err := filepath.WalkDir(sshDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip paths we can't read
			return nil
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		filename := d.Name()

		// Skip known non-key files
		if shouldSkipFile(filename) {
			return nil
		}

		// Try to parse as SSH key
		key, err := ParseKeyFile(path)
		if err != nil {
			// Not a valid key file, skip silently
			return nil
		}

		keys = append(keys, *key)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk ssh dir: %w", err)
	}

	// Sort file keys alphabetically by filename
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Filename < keys[j].Filename
	})

	return keys, nil
}

// shouldSkipFile returns true if the file should be skipped during discovery
func shouldSkipFile(filename string) bool {
	// Skip .pub files (we read them separately when parsing private keys)
	if strings.HasSuffix(filename, ".pub") {
		return true
	}

	// Skip common SSH files that are not private keys
	skipFiles := []string{
		"known_hosts",
		"known_hosts.old",
		"config",
		"authorized_keys",
		"authorized_keys2",
		"environment",
	}

	for _, skip := range skipFiles {
		if filename == skip {
			return true
		}
	}

	return false
}

// Discover1PasswordKeys discovers SSH keys referenced in 1Password servers.
// Attempts to resolve the IdentityFile path on disk to get type/fingerprint.
// If the file doesn't exist, creates a missing key entry.
// Deduplicates by path (same IdentityFile referenced by multiple servers).
func Discover1PasswordKeys(servers []*domain.Server) []SSHKey {
	seen := make(map[string]bool)
	var keys []SSHKey

	for _, server := range servers {
		// Skip servers without IdentityFile
		if server.IdentityFile == "" {
			continue
		}

		// Deduplicate by path
		if seen[server.IdentityFile] {
			continue
		}
		seen[server.IdentityFile] = true

		// Try to parse the key file
		key, err := ParseKeyFile(server.IdentityFile)
		if err != nil {
			// File doesn't exist or can't be parsed - create missing entry
			missingKey := CreateMissingKeyEntry(server.IdentityFile)
			missingKey.Source = Source1Password
			keys = append(keys, missingKey)
			continue
		}

		// Update source to 1Password
		key.Source = Source1Password
		keys = append(keys, *key)
	}

	return keys
}

// DiscoverKeys performs unified multi-source SSH key discovery.
// Discovers keys from:
// - File system (~/.ssh/ directory via header sniffing)
// - SSH agent (via SSH_AUTH_SOCK)
// - 1Password servers (IdentityFile references)
//
// Deduplicates by SHA256 fingerprint with precedence:
// - Agent version wins over file version (richer metadata from agent comment)
// - File version wins over 1Password version (local is authoritative)
//
// Returns a sorted, deduplicated flat list:
// - File keys first (alphabetical by filename)
// - Agent keys second
// - 1Password keys third
func DiscoverKeys(sshDir string, servers []*domain.Server) ([]SSHKey, error) {
	// Discover from all sources
	fileKeys, err := DiscoverFileKeys(sshDir)
	if err != nil {
		return nil, fmt.Errorf("discover file keys: %w", err)
	}

	agentKeys, err := DiscoverAgentKeys()
	if err != nil {
		return nil, fmt.Errorf("discover agent keys: %w", err)
	}

	onePasswordKeys := Discover1PasswordKeys(servers)

	// Deduplicate by fingerprint
	// Build a map: fingerprint -> key, with precedence rules
	keyMap := make(map[string]SSHKey)

	// Add file keys first (baseline)
	for _, key := range fileKeys {
		if key.Fingerprint != "" {
			keyMap[key.Fingerprint] = key
		}
	}

	// Add agent keys (override file keys with same fingerprint)
	for _, key := range agentKeys {
		if key.Fingerprint != "" {
			keyMap[key.Fingerprint] = key
		}
	}

	// Add 1Password keys only if not already present (file/agent wins)
	for _, key := range onePasswordKeys {
		if key.Fingerprint != "" {
			if _, exists := keyMap[key.Fingerprint]; !exists {
				keyMap[key.Fingerprint] = key
			}
		} else {
			// Missing keys don't have fingerprints, add them directly
			// Use a synthetic key based on path
			syntheticKey := "missing:" + key.MissingPath
			keyMap[syntheticKey] = key
		}
	}

	// Convert map back to slice
	var result []SSHKey
	for _, key := range keyMap {
		result = append(result, key)
	}

	// Sort: file keys first (alphabetical), then agent, then 1Password
	sort.Slice(result, func(i, j int) bool {
		ki, kj := result[i], result[j]

		// Sort by source first
		if ki.Source != kj.Source {
			// File (0) < Agent (1) < 1Password (2)
			return ki.Source < kj.Source
		}

		// Within same source, sort by filename/comment
		if ki.Source == SourceFile {
			return ki.Filename < kj.Filename
		}
		if ki.Source == SourceAgent {
			return ki.Comment < kj.Comment
		}
		if ki.Source == Source1Password {
			return ki.Filename < kj.Filename
		}

		return false
	})

	return result, nil
}

// CreateMissingKeyEntry creates an SSHKey entry for a key file that doesn't exist.
// Used when an IdentityFile is referenced in SSH config or 1Password but the file is not on disk.
func CreateMissingKeyEntry(path string) SSHKey {
	return SSHKey{
		Filename:    filepath.Base(path),
		MissingPath: path,
		Missing:     true,
		Source:      SourceFile,
	}
}

// String returns a string representation of KeySource for debugging
func (s KeySource) String() string {
	switch s {
	case SourceFile:
		return "file"
	case SourceAgent:
		return "agent"
	case Source1Password:
		return "1password"
	default:
		return "unknown"
	}
}
