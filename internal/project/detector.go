package project

import (
	"os/exec"
	"strings"

	giturls "github.com/whilp/git-urls"
)

// ExtractOrgRepo parses a git remote URL and extracts the org/repo identifier.
// Returns empty string for empty input or unparseable URLs.
//
// Examples:
//   - "git@github.com:acme/backend-api.git" -> "acme/backend-api"
//   - "https://github.com/acme/backend-api.git" -> "acme/backend-api"
//   - "git@gitlab.com:company/team/service.git" -> "company/team/service"
func ExtractOrgRepo(remoteURL string) (string, error) {
	// Empty input returns empty result
	if remoteURL == "" {
		return "", nil
	}

	// Parse the URL
	parsed, err := giturls.Parse(remoteURL)
	if err != nil {
		// Malformed URL - return empty string gracefully
		return "", nil
	}

	// Extract path and clean it up
	path := parsed.Path

	// Trim leading slash
	path = strings.TrimPrefix(path, "/")

	// Trim .git suffix
	path = strings.TrimSuffix(path, ".git")

	return path, nil
}

// DetectCurrentProject runs git config to get the origin remote URL and extracts
// the org/repo identifier from it.
//
// Returns empty string (not error) if:
//   - Not in a git repository
//   - No origin remote configured
//   - Remote URL cannot be parsed
//
// This follows the "only origin remote" user decision - we don't check other remotes.
func DetectCurrentProject() (string, error) {
	// Run: git config --get remote.origin.url
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()

	if err != nil {
		// Command failed - not a git repo or no origin remote
		// Per user decision: return empty string, not error
		return "", nil
	}

	// Clean up the output (trim whitespace)
	remoteURL := strings.TrimSpace(string(output))

	// Extract org/repo from the URL
	return ExtractOrgRepo(remoteURL)
}
