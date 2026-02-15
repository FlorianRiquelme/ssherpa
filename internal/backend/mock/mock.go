package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/domain"
	"github.com/florianriquelme/ssherpa/internal/errors"
)

// Backend implements backend.Backend and backend.Writer with thread-safe in-memory storage.
type Backend struct {
	mu          sync.RWMutex
	servers     map[string]*domain.Server
	projects    map[string]*domain.Project
	credentials map[string]*domain.Credential
	closed      bool
}

// Compile-time interface verification
var _ backend.Backend = (*Backend)(nil)
var _ backend.Writer = (*Backend)(nil)

// New creates a new mock backend with initialized storage.
func New() *Backend {
	return &Backend{
		servers:     make(map[string]*domain.Server),
		projects:    make(map[string]*domain.Project),
		credentials: make(map[string]*domain.Credential),
		closed:      false,
	}
}

// Seed populates the backend with initial data (helper for tests).
// Stores copies to prevent external mutation.
func (b *Backend) Seed(servers []*domain.Server, projects []*domain.Project, credentials []*domain.Credential) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, srv := range servers {
		srvCopy := *srv
		b.servers[srv.ID] = &srvCopy
	}

	for _, prj := range projects {
		prjCopy := *prj
		b.projects[prj.ID] = &prjCopy
	}

	for _, cred := range credentials {
		credCopy := *cred
		b.credentials[cred.ID] = &credCopy
	}
}

// checkClosed returns ErrBackendUnavailable if backend is closed.
// Must be called while holding at least a read lock.
func (b *Backend) checkClosed() error {
	if b.closed {
		return &errors.BackendError{
			Op:      "Backend",
			Backend: "mock",
			Err:     errors.ErrBackendUnavailable,
		}
	}
	return nil
}

// ===== Server Methods =====

// GetServer retrieves a server by ID.
// Returns a copy to prevent external mutation.
func (b *Backend) GetServer(ctx context.Context, id string) (*domain.Server, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	server, exists := b.servers[id]
	if !exists {
		return nil, &errors.BackendError{
			Op:      "GetServer",
			Backend: "mock",
			Err:     errors.ErrServerNotFound,
		}
	}

	// Return copy to prevent external mutation
	serverCopy := *server
	return &serverCopy, nil
}

// ListServers returns all servers.
// Returns empty slice (not nil) when no servers exist.
// Returns copies to prevent external mutation.
func (b *Backend) ListServers(ctx context.Context) ([]*domain.Server, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	// Return empty slice instead of nil
	if len(b.servers) == 0 {
		return []*domain.Server{}, nil
	}

	servers := make([]*domain.Server, 0, len(b.servers))
	for _, srv := range b.servers {
		srvCopy := *srv
		servers = append(servers, &srvCopy)
	}

	return servers, nil
}

// CreateServer creates a new server.
// Stores a copy to prevent caller mutation.
func (b *Backend) CreateServer(ctx context.Context, server *domain.Server) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if _, exists := b.servers[server.ID]; exists {
		return &errors.BackendError{
			Op:      "CreateServer",
			Backend: "mock",
			Err:     errors.ErrDuplicateID,
		}
	}

	// Store copy to prevent caller mutation
	serverCopy := *server
	b.servers[server.ID] = &serverCopy

	return nil
}

// UpdateServer updates an existing server.
func (b *Backend) UpdateServer(ctx context.Context, server *domain.Server) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if _, exists := b.servers[server.ID]; !exists {
		return &errors.BackendError{
			Op:      "UpdateServer",
			Backend: "mock",
			Err:     errors.ErrServerNotFound,
		}
	}

	// Store copy to prevent caller mutation
	serverCopy := *server
	b.servers[server.ID] = &serverCopy

	return nil
}

// DeleteServer deletes a server by ID.
func (b *Backend) DeleteServer(ctx context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if _, exists := b.servers[id]; !exists {
		return &errors.BackendError{
			Op:      "DeleteServer",
			Backend: "mock",
			Err:     errors.ErrServerNotFound,
		}
	}

	delete(b.servers, id)
	return nil
}

// ===== Project Methods =====

// GetProject retrieves a project by ID.
// Returns a copy to prevent external mutation.
func (b *Backend) GetProject(ctx context.Context, id string) (*domain.Project, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	project, exists := b.projects[id]
	if !exists {
		return nil, &errors.BackendError{
			Op:      "GetProject",
			Backend: "mock",
			Err:     errors.ErrProjectNotFound,
		}
	}

	// Return copy to prevent external mutation
	projectCopy := *project
	return &projectCopy, nil
}

// ListProjects returns all projects.
// Returns empty slice (not nil) when no projects exist.
// Returns copies to prevent external mutation.
func (b *Backend) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	// Return empty slice instead of nil
	if len(b.projects) == 0 {
		return []*domain.Project{}, nil
	}

	projects := make([]*domain.Project, 0, len(b.projects))
	for _, prj := range b.projects {
		prjCopy := *prj
		projects = append(projects, &prjCopy)
	}

	return projects, nil
}

// CreateProject creates a new project.
// Stores a copy to prevent caller mutation.
func (b *Backend) CreateProject(ctx context.Context, project *domain.Project) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if _, exists := b.projects[project.ID]; exists {
		return &errors.BackendError{
			Op:      "CreateProject",
			Backend: "mock",
			Err:     errors.ErrDuplicateID,
		}
	}

	// Store copy to prevent caller mutation
	projectCopy := *project
	b.projects[project.ID] = &projectCopy

	return nil
}

// UpdateProject updates an existing project.
func (b *Backend) UpdateProject(ctx context.Context, project *domain.Project) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if _, exists := b.projects[project.ID]; !exists {
		return &errors.BackendError{
			Op:      "UpdateProject",
			Backend: "mock",
			Err:     errors.ErrProjectNotFound,
		}
	}

	// Store copy to prevent caller mutation
	projectCopy := *project
	b.projects[project.ID] = &projectCopy

	return nil
}

// DeleteProject deletes a project by ID.
func (b *Backend) DeleteProject(ctx context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if _, exists := b.projects[id]; !exists {
		return &errors.BackendError{
			Op:      "DeleteProject",
			Backend: "mock",
			Err:     errors.ErrProjectNotFound,
		}
	}

	delete(b.projects, id)
	return nil
}

// ===== Credential Methods =====

// GetCredential retrieves a credential by ID.
// Returns a copy to prevent external mutation.
func (b *Backend) GetCredential(ctx context.Context, id string) (*domain.Credential, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	credential, exists := b.credentials[id]
	if !exists {
		return nil, &errors.BackendError{
			Op:      "GetCredential",
			Backend: "mock",
			Err:     errors.ErrCredentialNotFound,
		}
	}

	// Return copy to prevent external mutation
	credentialCopy := *credential
	return &credentialCopy, nil
}

// ListCredentials returns all credentials.
// Returns empty slice (not nil) when no credentials exist.
// Returns copies to prevent external mutation.
func (b *Backend) ListCredentials(ctx context.Context) ([]*domain.Credential, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if err := b.checkClosed(); err != nil {
		return nil, err
	}

	// Return empty slice instead of nil
	if len(b.credentials) == 0 {
		return []*domain.Credential{}, nil
	}

	credentials := make([]*domain.Credential, 0, len(b.credentials))
	for _, cred := range b.credentials {
		credCopy := *cred
		credentials = append(credentials, &credCopy)
	}

	return credentials, nil
}

// CreateCredential creates a new credential.
// Stores a copy to prevent caller mutation.
func (b *Backend) CreateCredential(ctx context.Context, credential *domain.Credential) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if _, exists := b.credentials[credential.ID]; exists {
		return &errors.BackendError{
			Op:      "CreateCredential",
			Backend: "mock",
			Err:     errors.ErrDuplicateID,
		}
	}

	// Store copy to prevent caller mutation
	credentialCopy := *credential
	b.credentials[credential.ID] = &credentialCopy

	return nil
}

// UpdateCredential updates an existing credential.
func (b *Backend) UpdateCredential(ctx context.Context, credential *domain.Credential) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if _, exists := b.credentials[credential.ID]; !exists {
		return &errors.BackendError{
			Op:      "UpdateCredential",
			Backend: "mock",
			Err:     errors.ErrCredentialNotFound,
		}
	}

	// Store copy to prevent caller mutation
	credentialCopy := *credential
	b.credentials[credential.ID] = &credentialCopy

	return nil
}

// DeleteCredential deletes a credential by ID.
func (b *Backend) DeleteCredential(ctx context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if _, exists := b.credentials[id]; !exists {
		return &errors.BackendError{
			Op:      "DeleteCredential",
			Backend: "mock",
			Err:     errors.ErrCredentialNotFound,
		}
	}

	delete(b.credentials, id)
	return nil
}

// ===== Lifecycle Methods =====

// Close closes the backend and makes all operations return ErrBackendUnavailable.
func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	return nil
}

// String returns a human-readable representation of the backend.
func (b *Backend) String() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return fmt.Sprintf("Mock Backend (servers: %d, projects: %d, credentials: %d, closed: %v)",
		len(b.servers), len(b.projects), len(b.credentials), b.closed)
}
