package project

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractOrgRepo_SSHUrl(t *testing.T) {
	result, err := ExtractOrgRepo("git@github.com:acme/backend-api.git")
	require.NoError(t, err)
	assert.Equal(t, "acme/backend-api", result)
}

func TestExtractOrgRepo_HTTPSUrl(t *testing.T) {
	result, err := ExtractOrgRepo("https://github.com/acme/backend-api.git")
	require.NoError(t, err)
	assert.Equal(t, "acme/backend-api", result)
}

func TestExtractOrgRepo_HTTPSNoGitSuffix(t *testing.T) {
	result, err := ExtractOrgRepo("https://github.com/acme/backend-api")
	require.NoError(t, err)
	assert.Equal(t, "acme/backend-api", result)
}

func TestExtractOrgRepo_GitLabNestedGroup(t *testing.T) {
	result, err := ExtractOrgRepo("git@gitlab.com:company/team/service.git")
	require.NoError(t, err)
	assert.Equal(t, "company/team/service", result)
}

func TestExtractOrgRepo_EmptyUrl(t *testing.T) {
	result, err := ExtractOrgRepo("")
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestExtractOrgRepo_MalformedUrl(t *testing.T) {
	// Should handle gracefully - must not panic
	assert.NotPanics(t, func() {
		result, err := ExtractOrgRepo("not-a-url")
		// git-urls library interprets "not-a-url" as a valid path
		// This is acceptable - the key requirement is no panic
		assert.NoError(t, err)
		// The result could be the string itself (treated as path) or empty
		// Either behavior is acceptable as long as no panic occurs
		assert.NotEmpty(t, result) // In this case, library returns "not-a-url"
	})
}

func TestDetectCurrentProject_NotGitRepo(t *testing.T) {
	// Create temp dir with no .git
	tempDir := t.TempDir()

	// Change to temp dir
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Run detection - should return empty string, no error
	result, err := DetectCurrentProject()
	assert.NoError(t, err, "Non-git directory should not error")
	assert.Equal(t, "", result, "Non-git directory should return empty string")
}

func TestDetectCurrentProject_NoOriginRemote(t *testing.T) {
	// Create temp dir with git init but no origin
	tempDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err := cmd.Run()
	require.NoError(t, err)

	// Change to temp dir
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Run detection - should return empty string, no error
	result, err := DetectCurrentProject()
	assert.NoError(t, err, "Missing origin should not error")
	assert.Equal(t, "", result, "Missing origin should return empty string")
}

func TestDetectCurrentProject_WithOrigin(t *testing.T) {
	// Create temp dir with git init and origin
	tempDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err := cmd.Run()
	require.NoError(t, err)

	// Configure git user (required for commits in some git versions)
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	// Add origin remote
	cmd = exec.Command("git", "remote", "add", "origin", "git@github.com:test-org/test-repo.git")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	// Change to temp dir
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Run detection - should extract org/repo
	result, err := DetectCurrentProject()
	assert.NoError(t, err)
	assert.Equal(t, "test-org/test-repo", result)
}

func TestDetectCurrentProject_HTTPSOrigin(t *testing.T) {
	// Create temp dir with git init and HTTPS origin
	tempDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err := cmd.Run()
	require.NoError(t, err)

	// Add HTTPS origin remote
	cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/test-org/test-repo.git")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	// Change to temp dir
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Run detection - should extract org/repo
	result, err := DetectCurrentProject()
	assert.NoError(t, err)
	assert.Equal(t, "test-org/test-repo", result)
}
