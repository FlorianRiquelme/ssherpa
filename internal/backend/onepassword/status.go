package onepassword

import (
	"context"
	"strings"

	backendpkg "github.com/florianriquelme/sshjesus/internal/backend"
	"github.com/florianriquelme/sshjesus/internal/domain"
	"github.com/florianriquelme/sshjesus/internal/errors"
	"github.com/florianriquelme/sshjesus/internal/sync"
)

// GetStatus returns the current backend status (thread-safe).
func (b *Backend) GetStatus() backendpkg.BackendStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.status
}

// setStatus updates the backend status (thread-safe).
func (b *Backend) setStatus(s backendpkg.BackendStatus) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.status = s
}

// SyncFromOnePassword attempts to sync servers from 1Password.
// On success: sets status to Available, populates cache, writes to TOML cache.
// On error: inspects error type to set status to Locked or Unavailable.
func (b *Backend) SyncFromOnePassword(ctx context.Context) error {
	// Try to list vaults as a health check
	vaults, err := b.client.ListVaults(ctx)
	if err != nil {
		// Inspect error to determine if locked vs unavailable
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "session expired") || strings.Contains(errStr, "locked") {
			b.setStatus(backendpkg.StatusLocked)
		} else {
			b.setStatus(backendpkg.StatusUnavailable)
		}
		return &errors.BackendError{
			Op:      "SyncFromOnePassword",
			Backend: "onepassword",
			Err:     err,
		}
	}

	// Fetch all tagged servers
	servers := make([]*domain.Server, 0)
	for _, vault := range vaults {
		items, err := b.client.ListItems(ctx, vault.ID)
		if err != nil {
			// Skip vaults that error (permission issues, etc.)
			continue
		}

		for _, item := range items {
			if !HasSshjesusTag(item.Tags) {
				continue
			}

			server, err := ItemToServer(&item)
			if err != nil {
				// Skip items that can't be converted (malformed data)
				continue
			}

			servers = append(servers, server)
		}
	}

	// Update cache
	b.mu.Lock()
	b.servers = servers
	b.status = backendpkg.StatusAvailable
	b.mu.Unlock()

	// Write to TOML cache for offline fallback
	if b.cachePath != "" {
		if err := sync.WriteTOMLCache(servers, b.cachePath); err != nil {
			// Log but don't fail sync - cache write is best-effort
			// In production, would use proper logging here
		}
	}

	return nil
}

// LoadFromCache loads servers from the TOML cache file.
// This is called when 1Password is unavailable on startup.
func (b *Backend) LoadFromCache() error {
	if b.cachePath == "" {
		return &errors.BackendError{
			Op:      "LoadFromCache",
			Backend: "onepassword",
			Err:     errors.New("cache path not set"),
		}
	}

	servers, err := sync.ReadTOMLCache(b.cachePath)
	if err != nil {
		return &errors.BackendError{
			Op:      "LoadFromCache",
			Backend: "onepassword",
			Err:     err,
		}
	}

	b.mu.Lock()
	b.servers = servers
	b.mu.Unlock()

	return nil
}
