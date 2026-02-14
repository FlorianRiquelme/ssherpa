package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/florianriquelme/sshjesus/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigNotFound(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "nonexistent.toml")

	cfg, err := Load(nonExistentPath)
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, errors.ErrConfigNotFound)
}

func TestLoadConfigValid(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Write a valid TOML file
	validTOML := `version = 1
backend = "sshconfig"
`
	err := os.WriteFile(configPath, []byte(validTOML), 0644)
	require.NoError(t, err)

	// Load and verify
	cfg, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "sshconfig", cfg.Backend)
}

func TestLoadConfigMalformed(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Write invalid TOML
	invalidTOML := `this is not valid TOML ][[[`
	err := os.WriteFile(configPath, []byte(invalidTOML), 0644)
	require.NoError(t, err)

	// Load should fail with malformed error
	cfg, err := Load(configPath)
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "malformed")
}

func TestSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	cfg := &Config{
		Version: 1,
		Backend: "onepassword",
	}

	err := Save(cfg, configPath)
	require.NoError(t, err)

	// Read file back and verify TOML content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "version = 1")
	assert.Contains(t, contentStr, "backend = \"onepassword\"")
}

func TestSaveAndReload(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	original := &Config{
		Version: 1,
		Backend: "mock",
	}

	// Save
	err := Save(original, configPath)
	require.NoError(t, err)

	// Reload
	reloaded, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, reloaded)

	// Verify round-trip preserves all fields
	assert.Equal(t, original.Version, reloaded.Version)
	assert.Equal(t, original.Backend, reloaded.Backend)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "", cfg.Backend) // Empty = needs setup
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config passes",
			config: &Config{
				Version: 1,
				Backend: "sshconfig",
			},
			wantErr: false,
		},
		{
			name: "empty backend fails",
			config: &Config{
				Version: 1,
				Backend: "",
			},
			wantErr: true,
			errMsg:  "backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigWithProjects_SaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Create config with projects
	original := &Config{
		Version: 1,
		Backend: "sshconfig",
		Projects: []ProjectConfig{
			{
				ID:            "acme/backend-api",
				Name:          "Backend API",
				GitRemoteURLs: []string{"git@github.com:acme/backend-api.git"},
				Color:         "#FF5733",
				ServerNames:   []string{"api-prod", "api-staging"},
			},
			{
				ID:            "example/frontend",
				Name:          "Frontend App",
				GitRemoteURLs: []string{"https://github.com/example/frontend.git"},
				Color:         "", // Empty means auto-generate
				ServerNames:   []string{"web-prod"},
			},
		},
	}

	// Save
	err := Save(original, configPath)
	require.NoError(t, err)

	// Reload
	reloaded, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, reloaded)

	// Verify all fields preserved
	assert.Equal(t, original.Version, reloaded.Version)
	assert.Equal(t, original.Backend, reloaded.Backend)
	require.Len(t, reloaded.Projects, 2)

	// Verify first project
	assert.Equal(t, "acme/backend-api", reloaded.Projects[0].ID)
	assert.Equal(t, "Backend API", reloaded.Projects[0].Name)
	assert.Equal(t, []string{"git@github.com:acme/backend-api.git"}, reloaded.Projects[0].GitRemoteURLs)
	assert.Equal(t, "#FF5733", reloaded.Projects[0].Color)
	assert.Equal(t, []string{"api-prod", "api-staging"}, reloaded.Projects[0].ServerNames)

	// Verify second project
	assert.Equal(t, "example/frontend", reloaded.Projects[1].ID)
	assert.Equal(t, "Frontend App", reloaded.Projects[1].Name)
	assert.Equal(t, []string{"https://github.com/example/frontend.git"}, reloaded.Projects[1].GitRemoteURLs)
	assert.Equal(t, "", reloaded.Projects[1].Color)
	assert.Equal(t, []string{"web-prod"}, reloaded.Projects[1].ServerNames)
}

func TestConfigWithProjects_EmptyProjects(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Config with empty Projects slice
	original := &Config{
		Version:  1,
		Backend:  "onepassword",
		Projects: []ProjectConfig{},
	}

	// Save
	err := Save(original, configPath)
	require.NoError(t, err)

	// Read TOML file
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	contentStr := string(content)

	// Should not have any [[project]] entries
	assert.NotContains(t, contentStr, "[[project]]")

	// Reload
	reloaded, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, reloaded)

	// Verify empty projects preserved (or nil is acceptable)
	if reloaded.Projects != nil {
		assert.Empty(t, reloaded.Projects)
	}
}

func TestConfigWithProjects_MultipleRemoteURLs(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Project with multiple remote URLs
	original := &Config{
		Version: 1,
		Backend: "sshconfig",
		Projects: []ProjectConfig{
			{
				ID:   "company/monorepo",
				Name: "Monorepo",
				GitRemoteURLs: []string{
					"git@github.com:company/monorepo.git",
					"https://github.com/company/monorepo.git",
					"git@gitlab.com:company/monorepo.git",
				},
			},
		},
	}

	// Save and reload
	err := Save(original, configPath)
	require.NoError(t, err)

	reloaded, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, reloaded)

	// Verify multiple URLs preserved
	require.Len(t, reloaded.Projects, 1)
	assert.Equal(t, 3, len(reloaded.Projects[0].GitRemoteURLs))
	assert.Equal(t, original.Projects[0].GitRemoteURLs, reloaded.Projects[0].GitRemoteURLs)
}

func TestConfigWithProjects_ServerNames(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Project with multiple server names
	original := &Config{
		Version: 1,
		Backend: "sshconfig",
		Projects: []ProjectConfig{
			{
				ID:   "acme/backend",
				Name: "Backend Services",
				ServerNames: []string{
					"api-prod-01",
					"api-prod-02",
					"api-staging",
					"api-dev",
				},
			},
		},
	}

	// Save and reload
	err := Save(original, configPath)
	require.NoError(t, err)

	reloaded, err := Load(configPath)
	require.NoError(t, err)
	require.NotNil(t, reloaded)

	// Verify server names preserved
	require.Len(t, reloaded.Projects, 1)
	assert.Equal(t, 4, len(reloaded.Projects[0].ServerNames))
	assert.Equal(t, original.Projects[0].ServerNames, reloaded.Projects[0].ServerNames)
}
