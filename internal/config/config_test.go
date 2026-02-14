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
