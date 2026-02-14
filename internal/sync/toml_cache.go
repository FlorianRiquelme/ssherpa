package sync

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/renameio/v2"

	"github.com/florianriquelme/sshjesus/internal/domain"
)

// CachedServer represents a server in the TOML cache.
// This includes sshjesus-specific fields that don't fit in SSH config.
type CachedServer struct {
	ID                string   `toml:"id"`
	DisplayName       string   `toml:"display_name"`
	Host              string   `toml:"host"`
	User              string   `toml:"user"`
	Port              int      `toml:"port"`
	IdentityFile      string   `toml:"identity_file,omitempty"`
	Proxy             string   `toml:"proxy,omitempty"`
	RemoteProjectPath string   `toml:"remote_project_path,omitempty"`
	ProjectIDs        []string `toml:"project_ids,omitempty"`
	VaultID           string   `toml:"vault_id"`
	Tags              []string `toml:"tags,omitempty"`
	Notes             string   `toml:"notes,omitempty"`
}

// TOMLCache represents the entire TOML cache file structure.
type TOMLCache struct {
	LastSync time.Time      `toml:"last_sync"`
	Servers  []CachedServer `toml:"server"`
}

// WriteTOMLCache writes the server list to a TOML cache file.
// This cache stores sshjesus-specific fields that don't fit in SSH config
// (RemoteProjectPath, ProjectIDs, VaultID, etc.).
func WriteTOMLCache(servers []*domain.Server, cachePath string) error {
	// Convert domain.Server list to CachedServer list
	cache := TOMLCache{
		LastSync: time.Now().UTC(),
		Servers:  make([]CachedServer, 0, len(servers)),
	}

	for _, srv := range servers {
		cached := CachedServer{
			ID:                srv.ID,
			DisplayName:       srv.DisplayName,
			Host:              srv.Host,
			User:              srv.User,
			Port:              srv.Port,
			IdentityFile:      srv.IdentityFile,
			Proxy:             srv.Proxy,
			RemoteProjectPath: srv.RemoteProjectPath,
			ProjectIDs:        srv.ProjectIDs,
			VaultID:           srv.VaultID,
			Tags:              srv.Tags,
			Notes:             srv.Notes,
		}
		cache.Servers = append(cache.Servers, cached)
	}

	// Encode as TOML
	tmpFile, err := os.CreateTemp("", "sshjesus-cache-*.toml")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file

	encoder := toml.NewEncoder(tmpFile)
	if err := encoder.Encode(cache); err != nil {
		tmpFile.Close()
		return fmt.Errorf("encode TOML: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Read the temp file content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("read temp file: %w", err)
	}

	// Write atomically using renameio
	if err := renameio.WriteFile(cachePath, content, 0600); err != nil {
		return fmt.Errorf("write TOML cache: %w", err)
	}

	return nil
}

// ReadTOMLCache reads servers from a TOML cache file.
// Used for offline fallback and verifying sync state.
func ReadTOMLCache(cachePath string) ([]*domain.Server, error) {
	// Read the cache file
	content, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("read TOML cache: %w", err)
	}

	// Decode TOML
	var cache TOMLCache
	if err := toml.Unmarshal(content, &cache); err != nil {
		return nil, fmt.Errorf("decode TOML: %w", err)
	}

	// Convert CachedServer list to domain.Server list
	servers := make([]*domain.Server, 0, len(cache.Servers))
	for _, cached := range cache.Servers {
		srv := &domain.Server{
			ID:                cached.ID,
			DisplayName:       cached.DisplayName,
			Host:              cached.Host,
			User:              cached.User,
			Port:              cached.Port,
			IdentityFile:      cached.IdentityFile,
			Proxy:             cached.Proxy,
			RemoteProjectPath: cached.RemoteProjectPath,
			ProjectIDs:        cached.ProjectIDs,
			VaultID:           cached.VaultID,
			Tags:              cached.Tags,
			Notes:             cached.Notes,
		}
		servers = append(servers, srv)
	}

	return servers, nil
}
