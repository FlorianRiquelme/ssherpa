package project

import (
	"regexp"
	"sort"
	"strings"

	"github.com/agnivade/levenshtein"
)

// ProjectMember represents a project with its member hostnames for matching.
type ProjectMember struct {
	ProjectID   string
	ProjectName string
	Hostnames   []string // Hostnames of servers already in this project
}

// Suggestion represents a project suggestion with a relevance score.
type Suggestion struct {
	ProjectID   string
	ProjectName string
	Score       float64 // 0.0 to 1.0
}

// SuggestProjects returns up to 3 project suggestions for a server hostname
// based on hostname pattern similarity with existing project members.
// Only returns suggestions with similarity score >= 70%.
func SuggestProjects(serverHostname string, projects []ProjectMember) []Suggestion {
	if len(projects) == 0 {
		return []Suggestion{}
	}

	type scoredProject struct {
		project ProjectMember
		score   float64
	}

	var scored []scoredProject

	for _, proj := range projects {
		maxScore := 0.0
		for _, hostname := range proj.Hostnames {
			similarity := computeHostnameSimilarity(serverHostname, hostname)
			if similarity > maxScore {
				maxScore = similarity
			}
		}

		if maxScore >= 0.7 {
			scored = append(scored, scoredProject{
				project: proj,
				score:   maxScore,
			})
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Return at most 3
	limit := 3
	if len(scored) < limit {
		limit = len(scored)
	}

	suggestions := make([]Suggestion, limit)
	for i := 0; i < limit; i++ {
		suggestions[i] = Suggestion{
			ProjectID:   scored[i].project.ProjectID,
			ProjectName: scored[i].project.ProjectName,
			Score:       scored[i].score,
		}
	}

	return suggestions
}

var numericSegmentPattern = regexp.MustCompile(`^\d+$`)

// computeHostnameSimilarity calculates similarity between two hostnames.
// Returns a score from 0.0 to 1.0, where 1.0 is identical.
func computeHostnameSimilarity(a, b string) float64 {
	// Split hostnames into segments
	segmentsA := splitHostname(a)
	segmentsB := splitHostname(b)

	// Strip numeric-only segments
	segmentsA = filterNonNumeric(segmentsA)
	segmentsB = filterNonNumeric(segmentsB)

	// If either is empty after filtering, fall back to simple comparison
	if len(segmentsA) == 0 || len(segmentsB) == 0 {
		if a == b {
			return 1.0
		}
		return 0.0
	}

	// Compute weighted similarity
	maxLen := len(segmentsA)
	if len(segmentsB) > maxLen {
		maxLen = len(segmentsB)
	}

	weights := []float64{1.0, 0.8, 0.6, 0.4, 0.2}
	totalWeight := 0.0
	weightedSum := 0.0

	for i := 0; i < maxLen; i++ {
		weight := 0.1 // Default weight for segments beyond initial 5
		if i < len(weights) {
			weight = weights[i]
		}

		var segA, segB string
		if i < len(segmentsA) {
			segA = segmentsA[i]
		}
		if i < len(segmentsB) {
			segB = segmentsB[i]
		}

		similarity := segmentSimilarity(segA, segB)
		weightedSum += similarity * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSum / totalWeight
}

// splitHostname splits a hostname by '.' and '-' into segments.
func splitHostname(hostname string) []string {
	// Replace '-' with '.' for uniform splitting
	normalized := strings.ReplaceAll(hostname, "-", ".")
	parts := strings.Split(normalized, ".")

	// Filter out empty parts
	var segments []string
	for _, part := range parts {
		if part != "" {
			segments = append(segments, strings.ToLower(part))
		}
	}

	return segments
}

// filterNonNumeric removes segments that are purely numeric.
func filterNonNumeric(segments []string) []string {
	var filtered []string
	for _, seg := range segments {
		if !numericSegmentPattern.MatchString(seg) {
			filtered = append(filtered, seg)
		}
	}
	return filtered
}

// segmentSimilarity computes similarity between two segments using Levenshtein distance.
func segmentSimilarity(a, b string) float64 {
	if a == "" && b == "" {
		return 1.0
	}
	if a == "" || b == "" {
		return 0.0
	}
	if a == b {
		return 1.0
	}

	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	distance := levenshtein.ComputeDistance(a, b)
	similarity := 1.0 - (float64(distance) / float64(maxLen))

	if similarity < 0 {
		return 0.0
	}

	return similarity
}
