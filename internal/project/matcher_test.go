package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuggestProjects_ExactSubdomainMatch(t *testing.T) {
	projects := []ProjectMember{
		{
			ProjectID:   "acme/backend",
			ProjectName: "Backend API",
			Hostnames:   []string{"api-prod-01.acme.com"},
		},
	}

	suggestions := SuggestProjects("api-prod-02.acme.com", projects)

	assert.Len(t, suggestions, 1, "should suggest the matching project")
	assert.Equal(t, "acme/backend", suggestions[0].ProjectID)
	assert.Greater(t, suggestions[0].Score, 0.7, "score should be high for similar hostnames")
}

func TestSuggestProjects_SimilarPrefix(t *testing.T) {
	projects := []ProjectMember{
		{
			ProjectID:   "acme/database",
			ProjectName: "Database",
			Hostnames:   []string{"db-prod.acme.com"},
		},
	}

	suggestions := SuggestProjects("db-staging.acme.com", projects)

	assert.Len(t, suggestions, 1, "should suggest project with similar prefix")
	assert.Equal(t, "acme/database", suggestions[0].ProjectID)
	assert.Greater(t, suggestions[0].Score, 0.7, "should meet threshold despite different environment")
}

func TestSuggestProjects_Unrelated(t *testing.T) {
	projects := []ProjectMember{
		{
			ProjectID:   "acme/backend",
			ProjectName: "Backend",
			Hostnames:   []string{"api-prod.acme.com"},
		},
	}

	suggestions := SuggestProjects("mail.other.com", projects)

	assert.Empty(t, suggestions, "should not suggest unrelated projects")
}

func TestSuggestProjects_MultipleProjects(t *testing.T) {
	projects := []ProjectMember{
		{
			ProjectID:   "acme/backend",
			ProjectName: "Backend",
			Hostnames:   []string{"api-prod-01.acme.com"},
		},
		{
			ProjectID:   "acme/frontend",
			ProjectName: "Frontend",
			Hostnames:   []string{"api-prod-05.acme.com"},
		},
	}

	suggestions := SuggestProjects("api-prod-02.acme.com", projects)

	assert.Len(t, suggestions, 2, "should return both matching projects")
	// Should be sorted by score descending
	assert.Greater(t, suggestions[0].Score, suggestions[1].Score, "suggestions should be sorted by score")
}

func TestSuggestProjects_MaxThree(t *testing.T) {
	projects := []ProjectMember{
		{
			ProjectID:   "proj1",
			ProjectName: "Project 1",
			Hostnames:   []string{"api-prod-01.acme.com"},
		},
		{
			ProjectID:   "proj2",
			ProjectName: "Project 2",
			Hostnames:   []string{"api-prod-02.acme.com"},
		},
		{
			ProjectID:   "proj3",
			ProjectName: "Project 3",
			Hostnames:   []string{"api-prod-03.acme.com"},
		},
		{
			ProjectID:   "proj4",
			ProjectName: "Project 4",
			Hostnames:   []string{"api-prod-04.acme.com"},
		},
	}

	suggestions := SuggestProjects("api-prod-10.acme.com", projects)

	assert.LessOrEqual(t, len(suggestions), 3, "should return at most 3 suggestions")
}

func TestSuggestProjects_NoProjects(t *testing.T) {
	suggestions := SuggestProjects("api-prod.acme.com", []ProjectMember{})

	assert.Empty(t, suggestions, "should return empty for no projects")
}

func TestSuggestProjects_IgnoreNumericSuffix(t *testing.T) {
	projects := []ProjectMember{
		{
			ProjectID:   "acme/backend",
			ProjectName: "Backend",
			Hostnames:   []string{"api-prod-01.acme.com"},
		},
	}

	suggestions := SuggestProjects("api-prod-99.acme.com", projects)

	assert.Len(t, suggestions, 1, "should treat numeric suffixes as equivalent")
	assert.Greater(t, suggestions[0].Score, 0.9, "score should be very high when only numeric suffix differs")
}
