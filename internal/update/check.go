package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// releaseAPIURL returns the GitHub releases/latest API endpoint.
func releaseAPIURL() string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
}

// changelogRawURL returns the raw CHANGELOG.md URL for a given version tag.
func changelogRawURL(version string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/v%s/CHANGELOG.md", repoOwner, repoName, version)
}

// CheckForUpdate is the main entry point. It reads the cache, checks GitHub if
// expired, and returns UpdateInfo if a new version is available (and not dismissed).
// Returns nil if up-to-date or dismissed.
func CheckForUpdate(currentVersion string) (*UpdateInfo, error) {
	if currentVersion == "dev" {
		return nil, nil // Skip check for dev builds
	}

	cachePath := defaultCachePath()
	cache, err := readCache(cachePath)
	if err != nil {
		// Cache read failed — proceed without cache
		cache = CacheData{}
	}

	// Check if cached result is still fresh
	if !cacheExpired(cache) {
		// Use cached data
		if cache.LatestVersion == "" || compareVersions(cache.LatestVersion, currentVersion) <= 0 {
			return nil, nil // Up to date
		}
		if cache.DismissedVersion == cache.LatestVersion {
			return nil, nil // Dismissed
		}
		// Parse cached changelog
		changes, err := ParseChangelog(cache.ChangelogMD, currentVersion, cache.LatestVersion)
		if err != nil {
			return nil, nil
		}
		return &UpdateInfo{
			LatestVersion: cache.LatestVersion,
			Changes:       changes,
		}, nil
	}

	// Cache expired — check remote
	info, err := checkRemote(currentVersion, releaseAPIURL(), "")
	if err != nil {
		return nil, err
	}

	// Update cache
	newCache := CacheData{
		LastCheck:        time.Now(),
		DismissedVersion: cache.DismissedVersion, // Preserve dismiss state
	}
	if info != nil {
		newCache.LatestVersion = info.LatestVersion
		// Fetch and cache the changelog
		changelogURL := changelogRawURL(info.LatestVersion)
		md, _ := fetchBody(changelogURL) // Best-effort
		newCache.ChangelogMD = md

		// Re-parse with full changelog if we got it
		if md != "" {
			changes, _ := ParseChangelog(md, currentVersion, info.LatestVersion)
			info.Changes = changes
		}
	}
	_ = writeCache(cachePath, newCache) // Best-effort cache write

	// Check dismiss after refresh
	if info != nil && newCache.DismissedVersion == info.LatestVersion {
		return nil, nil
	}

	return info, nil
}

// checkRemote hits the GitHub API and optionally fetches the changelog.
// changelogURL can override the default (for testing). Pass "" for production default.
func checkRemote(currentVersion, apiURL, changelogURL string) (*UpdateInfo, error) {
	// Fetch latest release
	body, err := fetchBody(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	var release githubRelease
	if err := json.Unmarshal([]byte(body), &release); err != nil {
		return nil, fmt.Errorf("failed to parse release response: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if compareVersions(latestVersion, currentVersion) <= 0 {
		return nil, nil // Already up to date
	}

	// Fetch changelog
	if changelogURL == "" {
		changelogURL = changelogRawURL(latestVersion)
	}
	changelogMD, _ := fetchBody(changelogURL) // Best-effort

	changes, _ := ParseChangelog(changelogMD, currentVersion, latestVersion)

	return &UpdateInfo{
		LatestVersion: latestVersion,
		Changes:       changes,
	}, nil
}

// DismissVersion writes the dismissed version to cache.
func DismissVersion(version string) {
	cachePath := defaultCachePath()
	cache, _ := readCache(cachePath)
	cache.DismissedVersion = version
	_ = writeCache(cachePath, cache)
}

// fetchBody performs a GET request and returns the body as a string.
func fetchBody(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ssherpa-update-checker")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
