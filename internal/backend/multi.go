package backend

import (
	"context"
	"strings"
	"sync"

	"github.com/florianriquelme/sshjesus/internal/domain"
	"github.com/florianriquelme/sshjesus/internal/errors"
)

// MultiBackend aggregates servers from multiple Backend implementations.
// Implements Backend interface. Delegates writes to the first Writer-capable backend.
//
// Priority order matters: later backends win conflicts when servers have duplicate DisplayNames.
// For deduplication, DisplayName comparison is case-insensitive.
type MultiBackend struct {
	backends []Backend
	mu       sync.RWMutex
}

// Ensure MultiBackend implements Backend interface.
var _ Backend = (*MultiBackend)(nil)

// NewMultiBackend creates a new multi-backend aggregator.
// Backends are provided in priority order: later backends win conflicts.
// Example: NewMultiBackend(sshconfigBackend, onepasswordBackend) -> 1Password wins duplicates.
func NewMultiBackend(backends ...Backend) *MultiBackend {
	return &MultiBackend{
		backends: backends,
	}
}

// ListServers aggregates servers from all backends.
// When multiple backends have servers with the same DisplayName (case-insensitive),
// the server from the higher-priority backend (later in the list) is returned.
func (m *MultiBackend) ListServers(ctx context.Context) ([]*domain.Server, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect servers from all backends
	allServers := make([]*domain.Server, 0)
	for _, backend := range m.backends {
		servers, err := backend.ListServers(ctx)
		if err != nil {
			// Skip backends that error (e.g., offline backend)
			continue
		}
		allServers = append(allServers, servers...)
	}

	// Deduplicate by DisplayName (case-insensitive), keeping last occurrence (highest priority)
	// Use map to track: lowercase DisplayName -> Server
	deduped := make(map[string]*domain.Server)
	for _, server := range allServers {
		key := strings.ToLower(server.DisplayName)
		deduped[key] = server
	}

	// Convert map back to slice
	result := make([]*domain.Server, 0, len(deduped))
	for _, server := range deduped {
		result = append(result, server)
	}

	return result, nil
}

// GetServer retrieves a server by ID from backends in reverse priority order (highest first).
// Returns the first match found.
func (m *MultiBackend) GetServer(ctx context.Context, id string) (*domain.Server, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try backends in reverse order (highest priority first)
	for i := len(m.backends) - 1; i >= 0; i-- {
		server, err := m.backends[i].GetServer(ctx, id)
		if err == nil && server != nil {
			return server, nil
		}
		// Continue to next backend if not found or error
	}

	return nil, &errors.BackendError{
		Op:      "GetServer",
		Backend: "multi",
		Err:     errors.New("server not found in any backend"),
	}
}

// ListProjects aggregates projects from all backends (no deduplication).
func (m *MultiBackend) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allProjects := make([]*domain.Project, 0)
	for _, backend := range m.backends {
		projects, err := backend.ListProjects(ctx)
		if err != nil {
			// Skip backends that error
			continue
		}
		allProjects = append(allProjects, projects...)
	}

	return allProjects, nil
}

// GetProject retrieves a project by ID from backends in reverse priority order.
func (m *MultiBackend) GetProject(ctx context.Context, id string) (*domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try backends in reverse order (highest priority first)
	for i := len(m.backends) - 1; i >= 0; i-- {
		project, err := m.backends[i].GetProject(ctx, id)
		if err == nil && project != nil {
			return project, nil
		}
	}

	return nil, &errors.BackendError{
		Op:      "GetProject",
		Backend: "multi",
		Err:     errors.New("project not found in any backend"),
	}
}

// ListCredentials aggregates credentials from all backends (no deduplication).
func (m *MultiBackend) ListCredentials(ctx context.Context) ([]*domain.Credential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allCredentials := make([]*domain.Credential, 0)
	for _, backend := range m.backends {
		credentials, err := backend.ListCredentials(ctx)
		if err != nil {
			// Skip backends that error
			continue
		}
		allCredentials = append(allCredentials, credentials...)
	}

	return allCredentials, nil
}

// GetCredential retrieves a credential by ID from backends in reverse priority order.
func (m *MultiBackend) GetCredential(ctx context.Context, id string) (*domain.Credential, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try backends in reverse order (highest priority first)
	for i := len(m.backends) - 1; i >= 0; i-- {
		credential, err := m.backends[i].GetCredential(ctx, id)
		if err == nil && credential != nil {
			return credential, nil
		}
	}

	return nil, &errors.BackendError{
		Op:      "GetCredential",
		Backend: "multi",
		Err:     errors.New("credential not found in any backend"),
	}
}

// Close closes all backends, logging errors but not failing on first error.
func (m *MultiBackend) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, backend := range m.backends {
		if err := backend.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		// Return first error (could be enhanced to return all errors)
		return errs[0]
	}

	return nil
}

// Writer interface delegation

// CreateServer delegates to the first Writer-capable backend.
func (m *MultiBackend) CreateServer(ctx context.Context, server *domain.Server) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, backend := range m.backends {
		if writer, ok := backend.(Writer); ok {
			return writer.CreateServer(ctx, server)
		}
	}

	return &errors.BackendError{
		Op:      "CreateServer",
		Backend: "multi",
		Err:     errors.New("no writer-capable backend available"),
	}
}

// UpdateServer delegates to the first Writer-capable backend.
func (m *MultiBackend) UpdateServer(ctx context.Context, server *domain.Server) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, backend := range m.backends {
		if writer, ok := backend.(Writer); ok {
			return writer.UpdateServer(ctx, server)
		}
	}

	return &errors.BackendError{
		Op:      "UpdateServer",
		Backend: "multi",
		Err:     errors.New("no writer-capable backend available"),
	}
}

// DeleteServer delegates to the first Writer-capable backend.
func (m *MultiBackend) DeleteServer(ctx context.Context, id string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, backend := range m.backends {
		if writer, ok := backend.(Writer); ok {
			return writer.DeleteServer(ctx, id)
		}
	}

	return &errors.BackendError{
		Op:      "DeleteServer",
		Backend: "multi",
		Err:     errors.New("no writer-capable backend available"),
	}
}

// CreateProject delegates to the first Writer-capable backend.
func (m *MultiBackend) CreateProject(ctx context.Context, project *domain.Project) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, backend := range m.backends {
		if writer, ok := backend.(Writer); ok {
			return writer.CreateProject(ctx, project)
		}
	}

	return &errors.BackendError{
		Op:      "CreateProject",
		Backend: "multi",
		Err:     errors.New("no writer-capable backend available"),
	}
}

// UpdateProject delegates to the first Writer-capable backend.
func (m *MultiBackend) UpdateProject(ctx context.Context, project *domain.Project) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, backend := range m.backends {
		if writer, ok := backend.(Writer); ok {
			return writer.UpdateProject(ctx, project)
		}
	}

	return &errors.BackendError{
		Op:      "UpdateProject",
		Backend: "multi",
		Err:     errors.New("no writer-capable backend available"),
	}
}

// DeleteProject delegates to the first Writer-capable backend.
func (m *MultiBackend) DeleteProject(ctx context.Context, id string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, backend := range m.backends {
		if writer, ok := backend.(Writer); ok {
			return writer.DeleteProject(ctx, id)
		}
	}

	return &errors.BackendError{
		Op:      "DeleteProject",
		Backend: "multi",
		Err:     errors.New("no writer-capable backend available"),
	}
}

// CreateCredential delegates to the first Writer-capable backend.
func (m *MultiBackend) CreateCredential(ctx context.Context, cred *domain.Credential) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, backend := range m.backends {
		if writer, ok := backend.(Writer); ok {
			return writer.CreateCredential(ctx, cred)
		}
	}

	return &errors.BackendError{
		Op:      "CreateCredential",
		Backend: "multi",
		Err:     errors.New("no writer-capable backend available"),
	}
}

// UpdateCredential delegates to the first Writer-capable backend.
func (m *MultiBackend) UpdateCredential(ctx context.Context, cred *domain.Credential) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, backend := range m.backends {
		if writer, ok := backend.(Writer); ok {
			return writer.UpdateCredential(ctx, cred)
		}
	}

	return &errors.BackendError{
		Op:      "UpdateCredential",
		Backend: "multi",
		Err:     errors.New("no writer-capable backend available"),
	}
}

// DeleteCredential delegates to the first Writer-capable backend.
func (m *MultiBackend) DeleteCredential(ctx context.Context, id string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, backend := range m.backends {
		if writer, ok := backend.(Writer); ok {
			return writer.DeleteCredential(ctx, id)
		}
	}

	return &errors.BackendError{
		Op:      "DeleteCredential",
		Backend: "multi",
		Err:     errors.New("no writer-capable backend available"),
	}
}

// GetOnePasswordBackend finds and returns the 1Password backend if present.
// Returns nil if no 1Password backend is in the multi-backend.
func (m *MultiBackend) GetOnePasswordBackend() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, backend := range m.backends {
		// Check if this backend has GetStatus method (1Password backend)
		// Use type switch to check for method
		type statusGetter interface {
			GetStatus() interface{}
		}
		if _, ok := backend.(statusGetter); ok {
			return backend
		}
	}

	return nil
}
