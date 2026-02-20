package update

import "time"

const (
	// GitHub repository coordinates
	repoOwner = "FlorianRiquelme"
	repoName  = "ssherpa"

	// Cache expiry duration
	checkInterval = 24 * time.Hour
)

// CacheData is persisted to ~/.cache/ssherpa/update.json
type CacheData struct {
	LastCheck        time.Time `json:"last_check"`
	LatestVersion    string    `json:"latest_version"`
	ChangelogMD      string    `json:"changelog_md"`
	DismissedVersion string    `json:"dismissed_version"`
}

// VersionChanges represents a single release's changelog entries.
type VersionChanges struct {
	Version  string           // e.g., "0.3.0"
	Date     string           // e.g., "2026-02-20"
	Sections []ChangeSection
}

// ChangeSection groups entries under a changelog category.
type ChangeSection struct {
	Category string   // "Added", "Changed", "Fixed", etc.
	Entries  []string // Individual bullet points
}

// UpdateInfo carries the result of an update check to the TUI.
type UpdateInfo struct {
	LatestVersion string
	Changes       []VersionChanges
}

// githubRelease is the minimal JSON shape from GitHub's releases/latest endpoint.
type githubRelease struct {
	TagName string `json:"tag_name"` // e.g., "v0.3.0"
}
