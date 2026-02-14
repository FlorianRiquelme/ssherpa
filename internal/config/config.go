package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
	"github.com/florianriquelme/sshjesus/internal/errors"
)

// Config represents the application configuration.
type Config struct {
	Version int    `toml:"version"` // Config schema version for future migrations
	Backend string `toml:"backend"` // Backend identifier: "sshconfig", "onepassword", "mock"
}

// DefaultConfig returns a config with sensible defaults.
// Empty Backend means setup wizard is needed (deferred to Phase 2+).
func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Backend: "", // Empty = needs setup
	}
}

// Validate checks if the config is valid.
// Empty Backend is invalid (setup wizard needed, but that's Phase 2+).
func (c *Config) Validate() error {
	if c.Backend == "" {
		return fmt.Errorf("config validation failed: backend must be non-empty")
	}
	return nil
}

// DefaultPath returns the default config file path using XDG config directories.
// Creates parent directories if they don't exist.
func DefaultPath() (string, error) {
	path, err := xdg.ConfigFile("sshjesus/config.toml")
	if err != nil {
		return "", fmt.Errorf("failed to determine default config path: %w", err)
	}
	return path, nil
}

// Load reads and parses a config file from the given path.
// If path is empty, searches for config in XDG config directories.
// Returns ErrConfigNotFound if no config file exists.
// Returns malformed error if TOML parsing fails.
func Load(path string) (*Config, error) {
	// If no path provided, search XDG config directories
	if path == "" {
		searchPath, err := xdg.SearchConfigFile("sshjesus/config.toml")
		if err != nil {
			// SearchConfigFile returns error if file not found
			return nil, errors.ErrConfigNotFound
		}
		path = searchPath
	} else {
		// If path provided, check if file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, errors.ErrConfigNotFound
		}
	}

	// Decode TOML file
	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("malformed config file at %s: %w", path, err)
	}

	return &cfg, nil
}

// Save writes the config to the given path as TOML.
// If path is empty, uses DefaultPath() (XDG config directory).
// Creates parent directories if needed.
func Save(cfg *Config, path string) error {
	// If no path provided, use default XDG path
	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
		path = defaultPath
	}

	// Encode config as TOML
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file at %s: %w", path, err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config as TOML: %w", err)
	}

	return nil
}
