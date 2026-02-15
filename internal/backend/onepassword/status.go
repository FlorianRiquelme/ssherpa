package onepassword

import (
	"context"
	"strings"

	backendpkg "github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/domain"
	"github.com/florianriquelme/ssherpa/internal/errors"
	"github.com/florianriquelme/ssherpa/internal/sync"
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

// SyncFromBackend implements backend.Syncer.
// It delegates to SyncFromOnePassword for the actual sync logic.
func (b *Backend) SyncFromBackend(ctx context.Context) error {
	return b.SyncFromOnePassword(ctx)
}

// SyncFromOnePassword attempts to sync servers from 1Password.
// On success: sets status to Available, populates cache, writes to TOML cache.
// On error: inspects error type to set status to Locked or Unavailable.
func (b *Backend) SyncFromOnePassword(ctx context.Context) error {
	// Try to list vaults as a health check
	vaults, err := b.client.ListVaults(ctx)
	if err != nil {
		// Inspect error to determine locked vs not-signed-in vs unavailable
		errStr := strings.ToLower(err.Error())
		switch {
		case strings.Contains(errStr, "session expired") || strings.Contains(errStr, "locked"):
			b.setStatus(backendpkg.StatusLocked)
		case strings.Contains(errStr, "not signed in") ||
			strings.Contains(errStr, "not currently signed in") ||
			strings.Contains(errStr, "no active session") ||
			strings.Contains(errStr, "signin"):
			b.setStatus(backendpkg.StatusNotSignedIn)
		default:
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
