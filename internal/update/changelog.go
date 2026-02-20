package update

import (
	"regexp"
	"strings"
)

// versionHeaderRe matches "## [0.2.0] - 2026-02-20" or "## [0.2.0]"
var versionHeaderRe = regexp.MustCompile(`^## \[(\d+\.\d+\.\d+)\](?:\s*-\s*(\S+))?`)

// categoryHeaderRe matches "### Added", "### Fixed", etc.
var categoryHeaderRe = regexp.MustCompile(`^### (.+)`)

// ParseChangelog extracts changelog entries between currentVersion (exclusive)
// and latestVersion (inclusive), ordered newest-first.
func ParseChangelog(markdown, currentVersion, latestVersion string) ([]VersionChanges, error) {
	if markdown == "" {
		return nil, nil
	}

	lines := strings.Split(markdown, "\n")

	// Parse all version blocks
	var allVersions []VersionChanges
	var current *VersionChanges
	var currentSection *ChangeSection

	for _, line := range lines {
		// Check for version header
		if m := versionHeaderRe.FindStringSubmatch(line); m != nil {
			// Save previous version
			if current != nil {
				if currentSection != nil && len(currentSection.Entries) > 0 {
					current.Sections = append(current.Sections, *currentSection)
				}
				allVersions = append(allVersions, *current)
			}

			ver := m[1]
			date := ""
			if len(m) > 2 {
				date = m[2]
			}

			current = &VersionChanges{Version: ver, Date: date}
			currentSection = nil
			continue
		}

		// Check for category header
		if m := categoryHeaderRe.FindStringSubmatch(line); m != nil && current != nil {
			// Save previous section
			if currentSection != nil && len(currentSection.Entries) > 0 {
				current.Sections = append(current.Sections, *currentSection)
			}
			currentSection = &ChangeSection{Category: m[1]}
			continue
		}

		// Check for bullet entry
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") && currentSection != nil {
			entry := strings.TrimPrefix(trimmed, "- ")
			currentSection.Entries = append(currentSection.Entries, entry)
		}
	}

	// Save last version
	if current != nil {
		if currentSection != nil && len(currentSection.Entries) > 0 {
			current.Sections = append(current.Sections, *currentSection)
		}
		allVersions = append(allVersions, *current)
	}

	// Filter: include versions > currentVersion and <= latestVersion
	var result []VersionChanges
	for _, vc := range allVersions {
		if compareVersions(vc.Version, currentVersion) > 0 && compareVersions(vc.Version, latestVersion) <= 0 {
			result = append(result, vc)
		}
	}

	return result, nil
}

// compareVersions does a simple semver comparison.
// Returns -1, 0, or 1 (a < b, a == b, a > b).
func compareVersions(a, b string) int {
	aParts := parseVersionParts(a)
	bParts := parseVersionParts(b)

	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

// parseVersionParts splits "1.2.3" into [1, 2, 3].
func parseVersionParts(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		for _, ch := range p {
			if ch >= '0' && ch <= '9' {
				result[i] = result[i]*10 + int(ch-'0')
			}
		}
	}
	return result
}
