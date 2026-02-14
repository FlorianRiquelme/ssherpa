package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectColor_Deterministic(t *testing.T) {
	// Same project ID should produce same color on repeated calls
	projectID := "acme/backend-api"

	color1 := ProjectColor(projectID)
	color2 := ProjectColor(projectID)
	color3 := ProjectColor(projectID)

	assert.Equal(t, color1.Light, color2.Light, "Light color should be deterministic")
	assert.Equal(t, color1.Dark, color2.Dark, "Dark color should be deterministic")
	assert.Equal(t, color2.Light, color3.Light, "Light color should be consistent across multiple calls")
	assert.Equal(t, color2.Dark, color3.Dark, "Dark color should be consistent across multiple calls")
}

func TestProjectColor_DifferentProjects(t *testing.T) {
	// Different project IDs should produce different hues
	projectIDs := []string{
		"acme/backend-api",
		"example/frontend",
		"company/service",
		"org/microservice",
		"team/application",
		"group/project",
	}

	colors := make(map[string]string)

	for _, id := range projectIDs {
		color := ProjectColor(id)
		colors[id] = color.Light
		assert.NotEmpty(t, color.Light, "Color should not be empty for project: %s", id)
		assert.NotEmpty(t, color.Dark, "Dark color should not be empty for project: %s", id)
	}

	// Verify not all colors are identical
	// Count unique colors
	uniqueColors := make(map[string]bool)
	for _, color := range colors {
		uniqueColors[color] = true
	}

	assert.Greater(t, len(uniqueColors), 1, "Different project IDs should produce different colors")
}

func TestProjectColor_EmptyID(t *testing.T) {
	// Empty string should not panic and should return a valid color
	assert.NotPanics(t, func() {
		color := ProjectColor("")
		assert.NotEmpty(t, color.Light, "Empty ID should still produce a color")
		assert.NotEmpty(t, color.Dark, "Empty ID should still produce a dark color")
	})
}

func TestProjectColor_ValidHexFormat(t *testing.T) {
	// Colors should be valid hex strings
	projectID := "test/project"

	color := ProjectColor(projectID)

	// Check that light and dark colors start with # and have valid hex format
	assert.Regexp(t, `^#[0-9A-Fa-f]{6}$`, color.Light, "Light color should be valid hex format")
	assert.Regexp(t, `^#[0-9A-Fa-f]{6}$`, color.Dark, "Dark color should be valid hex format")
}
