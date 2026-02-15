package onepassword

import (
	"context"
	"sync"
	"time"

	backendpkg "github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/domain"
	"github.com/florianriquelme/ssherpa/internal/errors"
)

// Backend implements the backendpkg.Backend and backendpkg.Writer interfaces
// using 1Password as the storage layer.
type Backend struct {
	client    Client                // SDK client (real or mock)
	mu        sync.RWMutex          // Protects cached servers, status, and closed flag
	servers   []*domain.Server      // Cached servers from last sync
	closed    bool                  // Backend closed flag
	status    backendpkg.BackendStatus // Current availability status
	cachePath string                // Path to TOML cache for fallback
	poller    *Poller               // Background availability poller
	lastWrite time.Time             // Last write timestamp for debouncing
}

// Compile-time interface verification
var (
	_ backendpkg.Backend = (*Backend)(nil)
	_ backendpkg.Writer  = (*Backend)(nil)
	_ backendpkg.Syncer  = (*Backend)(nil)
)

// New creates a new 1Password backend with the given client.
// No initial sync is performed - caller should call ListServers to populate cache.
func New(client Client) *Backend {
	return &Backend{
		client:  client,
		servers: make([]*domain.Server, 0),
		status:  backendpkg.StatusUnknown,
	}
}

// NewWithCache creates a new 1Password backend with cache path for offline fallback.
func NewWithCache(client Client, cachePath string) *Backend {
	return &Backend{
		client:    client,
		servers:   make([]*domain.Server, 0),
		status:    backendpkg.StatusUnknown,
		cachePath: cachePath,
	}
}

// checkClosed returns ErrBackendUnavailable if backend is closed.
// Must be called with mu held (either RLock or Lock).
func (b *Backend) checkClosed() error {
	if b.closed {
		return &errors.BackendError{
			Op:      "checkClosed",
			Backend: "onepassword",
			Err:     errors.ErrBackendUnavailable,
		}
	}
	return nil
}

// ListServers returns cached servers (populated by SyncFromOnePassword or LoadFromCache).
// This is status-aware: returns live data when Available, cached data when Locked/Unavailable.
func (b *Backend) ListServers(ctx context.Context) ([]*domain.Server, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	// Return copies (copy-on-read pattern)
	result := make([]*domain.Server, len(b.servers))
	for i, s := range b.servers {
		serverCopy := *s
		result[i] = &serverCopy
	}

	return result, nil
}

// GetServer retrieves a server by ID from the cache.
func (b *Backend) GetServer(ctx context.Context, id string) (*domain.Server, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	// Linear search through cached servers (small lists)
	for _, server := range b.servers {
		if server.ID == id {
			// Return copy
			serverCopy := *server
			return &serverCopy, nil
		}
	}

	return nil, &errors.BackendError{
		Op:      "GetServer",
		Backend: "onepassword",
		Err:     errors.ErrServerNotFound,
	}
}

// ListProjects returns an empty slice (projects are tags on items, not standalone entities).
func (b *Backend) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	return []*domain.Project{}, nil
}

// GetProject returns ErrProjectNotFound (projects are tags, not standalone entities).
func (b *Backend) GetProject(ctx context.Context, id string) (*domain.Project, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	return nil, &errors.BackendError{
		Op:      "GetProject",
		Backend: "onepassword",
		Err:     errors.ErrProjectNotFound,
	}
}

// ListCredentials returns an empty slice (credentials are embedded in items).
func (b *Backend) ListCredentials(ctx context.Context) ([]*domain.Credential, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	return []*domain.Credential{}, nil
}

// GetCredential returns ErrCredentialNotFound (credentials are embedded in items).
func (b *Backend) GetCredential(ctx context.Context, id string) (*domain.Credential, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	return nil, &errors.BackendError{
		Op:      "GetCredential",
		Backend: "onepassword",
		Err:     errors.ErrCredentialNotFound,
	}
}

// Close releases resources held by the backend.
func (b *Backend) Close() error {
	b.mu.Lock()

	if b.closed {
		b.mu.Unlock()
		return nil
	}

	// Stop poller before closing client
	if b.poller != nil {
		poller := b.poller
		b.poller = nil
		b.mu.Unlock() // Unlock before calling Stop() to avoid deadlock
		poller.Stop()
		b.mu.Lock() // Re-lock for closed flag update
	}

	b.closed = true
	b.mu.Unlock()
	return b.client.Close()
}

// CreateServer creates a new server in 1Password.
// The server.VaultID must be set to specify the target vault.
func (b *Backend) CreateServer(ctx context.Context, server *domain.Server) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if server.VaultID == "" {
		return &errors.BackendError{
			Op:      "CreateServer",
			Backend: "onepassword",
			Err:     errors.New("VaultID must be set"),
		}
	}

	// Convert to item
	item := ServerToItem(server, server.VaultID)

	// Create in 1Password
	created, err := b.client.CreateItem(ctx, item)
	if err != nil {
		return &errors.BackendError{
			Op:      "CreateServer",
			Backend: "onepassword",
			Err:     err,
		}
	}

	// Convert back and add to cache
	newServer, err := ItemToServer(created)
	if err != nil {
		return &errors.BackendError{
			Op:      "CreateServer",
			Backend: "onepassword",
			Err:     err,
		}
	}

	b.servers = append(b.servers, newServer)

	// Update last write timestamp
	b.lastWrite = time.Now()

	return nil
}

// UpdateServer updates an existing server in 1Password.
func (b *Backend) UpdateServer(ctx context.Context, server *domain.Server) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	// Find server in cache to get VaultID if not set
	var vaultID string
	for _, cached := range b.servers {
		if cached.ID == server.ID {
			vaultID = cached.VaultID
			break
		}
	}

	if vaultID == "" {
		vaultID = server.VaultID
	}

	if vaultID == "" {
		return &errors.BackendError{
			Op:      "UpdateServer",
			Backend: "onepassword",
			Err:     errors.ErrServerNotFound,
		}
	}

	// Get existing item to preserve fields we don't manage
	existing, err := b.client.GetItem(ctx, vaultID, server.ID)
	if err != nil {
		return &errors.BackendError{
			Op:      "UpdateServer",
			Backend: "onepassword",
			Err:     err,
		}
	}

	// Convert server to item (preserving ID and vault)
	updated := ServerToItem(server, vaultID)
	updated.ID = existing.ID
	updated.VaultID = existing.VaultID

	// Update in 1Password
	_, err = b.client.UpdateItem(ctx, updated)
	if err != nil {
		return &errors.BackendError{
			Op:      "UpdateServer",
			Backend: "onepassword",
			Err:     err,
		}
	}

	// Update cache
	for i, cached := range b.servers {
		if cached.ID == server.ID {
			serverCopy := *server
			serverCopy.VaultID = vaultID
			b.servers[i] = &serverCopy
			break
		}
	}

	// Update last write timestamp
	b.lastWrite = time.Now()

	return nil
}

// DeleteServer deletes a server from 1Password.
func (b *Backend) DeleteServer(ctx context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	// Find server in cache to get VaultID
	var vaultID string
	var foundIndex = -1
	for i, server := range b.servers {
		if server.ID == id {
			vaultID = server.VaultID
			foundIndex = i
			break
		}
	}

	if vaultID == "" {
		return &errors.BackendError{
			Op:      "DeleteServer",
			Backend: "onepassword",
			Err:     errors.ErrServerNotFound,
		}
	}

	// Delete from 1Password
	err := b.client.DeleteItem(ctx, vaultID, id)
	if err != nil {
		return &errors.BackendError{
			Op:      "DeleteServer",
			Backend: "onepassword",
			Err:     err,
		}
	}

	// Remove from cache
	if foundIndex >= 0 {
		b.servers = append(b.servers[:foundIndex], b.servers[foundIndex+1:]...)
	}

	// Update last write timestamp
	b.lastWrite = time.Now()

	return nil
}

// CreateProject returns ErrReadOnlyBackend (projects are tags, not standalone entities).
func (b *Backend) CreateProject(ctx context.Context, project *domain.Project) error {
	return &errors.BackendError{
		Op:      "CreateProject",
		Backend: "onepassword",
		Err:     errors.ErrReadOnlyBackend,
	}
}

// UpdateProject returns ErrReadOnlyBackend (projects are tags, not standalone entities).
func (b *Backend) UpdateProject(ctx context.Context, project *domain.Project) error {
	return &errors.BackendError{
		Op:      "UpdateProject",
		Backend: "onepassword",
		Err:     errors.ErrReadOnlyBackend,
	}
}

// DeleteProject returns ErrReadOnlyBackend (projects are tags, not standalone entities).
func (b *Backend) DeleteProject(ctx context.Context, id string) error {
	return &errors.BackendError{
		Op:      "DeleteProject",
		Backend: "onepassword",
		Err:     errors.ErrReadOnlyBackend,
	}
}

// CreateCredential returns ErrReadOnlyBackend (credentials are embedded in items).
func (b *Backend) CreateCredential(ctx context.Context, cred *domain.Credential) error {
	return &errors.BackendError{
		Op:      "CreateCredential",
		Backend: "onepassword",
		Err:     errors.ErrReadOnlyBackend,
	}
}

// UpdateCredential returns ErrReadOnlyBackend (credentials are embedded in items).
func (b *Backend) UpdateCredential(ctx context.Context, cred *domain.Credential) error {
	return &errors.BackendError{
		Op:      "UpdateCredential",
		Backend: "onepassword",
		Err:     errors.ErrReadOnlyBackend,
	}
}

// DeleteCredential returns ErrReadOnlyBackend (credentials are embedded in items).
func (b *Backend) DeleteCredential(ctx context.Context, id string) error {
	return &errors.BackendError{
		Op:      "DeleteCredential",
		Backend: "onepassword",
		Err:     errors.ErrReadOnlyBackend,
	}
}
