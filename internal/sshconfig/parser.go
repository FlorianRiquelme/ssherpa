package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kevinburke/ssh_config"
)

// SSHHost represents a parsed SSH config host entry.
// Domain-independent model for SSH config data.
type SSHHost struct {
	Name         string              // Host pattern from config (e.g., "myserver", "*.example.com")
	Hostname     string              // HostName directive value
	User         string              // User directive value
	Port         string              // Port directive value (string, not int — preserve raw config value)
	IdentityFile []string            // all IdentityFile values (multi-value key)
	AllOptions   map[string][]string // every SSH config option set for this host, preserving multi-values
	SourceFile   string              // absolute path to the config file that defined this host
	SourceLine   int                 // line number in SourceFile where Host directive appears
	IsWildcard   bool                // true if any pattern contains `*` or `?`
	ParseError   error               // non-nil if this entry had issues (malformed, unreadable, etc.)
}

// ParseSSHConfig parses an SSH config file and returns structured host data.
// Handles Include directives (via library's automatic recursion), malformed files,
// and wildcard detection.
//
// If the file doesn't exist, returns an error.
// If the file is malformed (e.g., contains Match blocks), returns a single SSHHost
// with ParseError set, not a fatal error.
func ParseSSHConfig(path string) ([]SSHHost, error) {
	// Convert to absolute path for consistent SourceFile tracking
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}

	// Open the config file
	f, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("open SSH config: %w", err)
	}
	defer f.Close()

	// Parse the config using kevinburke/ssh_config
	cfg, err := ssh_config.Decode(f)
	if err != nil {
		// Decode failure (e.g., Match blocks) — don't fail fatally
		// Return a single SSHHost with ParseError set
		parseErr := err
		if strings.Contains(err.Error(), "Match") {
			parseErr = fmt.Errorf("SSH config contains unsupported Match directives: %w", err)
		}

		return []SSHHost{
			{
				Name:       filepath.Base(absPath),
				SourceFile: absPath,
				ParseError: parseErr,
			},
		}, nil
	}

	// Extract hosts from parsed config
	var hosts []SSHHost
	for _, host := range cfg.Hosts {
		// Skip empty pattern lists (comments-only blocks)
		if len(host.Patterns) == 0 {
			continue
		}

		// Build SSHHost entry
		sshHost := SSHHost{
			Name:       joinPatterns(host.Patterns),
			SourceFile: absPath,
			AllOptions: make(map[string][]string),
		}

		// Check for wildcard patterns
		sshHost.IsWildcard = containsWildcard(host.Patterns)

		// Note: kevinburke/ssh_config does not expose per-host position information.
		// We store the top-level config path as SourceFile for all hosts.
		// SourceLine is left at 0 as a limitation of the library.
		// For included files, the library handles Include recursion automatically,
		// but we cannot determine which file each host came from.

		// Extract key-value options
		for _, node := range host.Nodes {
			if kv, ok := node.(*ssh_config.KV); ok {
				key := kv.Key
				value := kv.Value

				// Populate AllOptions
				sshHost.AllOptions[key] = append(sshHost.AllOptions[key], value)

				// Extract named fields for common options
				switch key {
				case "HostName":
					if sshHost.Hostname == "" {
						sshHost.Hostname = value
					}
				case "User":
					if sshHost.User == "" {
						sshHost.User = value
					}
				case "Port":
					if sshHost.Port == "" {
						sshHost.Port = value
					}
				case "IdentityFile":
					sshHost.IdentityFile = append(sshHost.IdentityFile, value)
				}
			}
		}

		// Skip implicit "Host *" entries with no options (default catch-all from library)
		if len(host.Patterns) == 1 && host.Patterns[0].String() == "*" && len(sshHost.AllOptions) == 0 {
			continue
		}

		hosts = append(hosts, sshHost)
	}

	return hosts, nil
}

// joinPatterns combines SSH config patterns into a single string.
// Multiple patterns are space-separated (e.g., "host1 host2").
func joinPatterns(patterns []*ssh_config.Pattern) string {
	var parts []string
	for _, p := range patterns {
		parts = append(parts, p.String())
	}
	return strings.Join(parts, " ")
}

// containsWildcard checks if any pattern contains wildcard characters (* or ?).
func containsWildcard(patterns []*ssh_config.Pattern) bool {
	for _, p := range patterns {
		s := p.String()
		if strings.Contains(s, "*") || strings.Contains(s, "?") {
			return true
		}
	}
	return false
}

// OrganizeHosts separates regular hosts from wildcards and sorts each group alphabetically by Name.
// Pure function, no I/O.
func OrganizeHosts(hosts []SSHHost) (regular, wildcards []SSHHost) {
	for _, host := range hosts {
		if host.IsWildcard {
			wildcards = append(wildcards, host)
		} else {
			regular = append(regular, host)
		}
	}

	// Sort both groups alphabetically by Name
	sort.Slice(regular, func(i, j int) bool {
		return regular[i].Name < regular[j].Name
	})
	sort.Slice(wildcards, func(i, j int) bool {
		return wildcards[i].Name < wildcards[j].Name
	})

	return regular, wildcards
}
