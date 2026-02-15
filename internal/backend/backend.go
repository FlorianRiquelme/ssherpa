package backend

import (
	"context"

	"github.com/florianriquelme/sshjesus/internal/domain"
)

// Backend defines the minimal read-only interface that all backends MUST implement.
// Follows the database/sql pattern: minimal required interface, optional capabilities via type assertions.
//
// All backends must support read operations and clean shutdown.
// Write operations are optional (type-assert to Writer interface).
// Filtering is optional (type-assert to Filterer interface).
//
// Example usage:
//
//	backend := someBackend()
//	servers, err := backend.ListServers(ctx)
//
//	// Check if backend supports writes
//	if writer, ok := backend.(Writer); ok {
//	    err := writer.CreateServer(ctx, &server)
//	}
//
//	// Check if backend supports filtering
//	if filterer, ok := backend.(Filterer); ok {
//	    servers, err := filterer.FilterServers(ctx, filters)
//	}
type Backend interface {
	// GetServer retrieves a server by ID.
	GetServer(ctx context.Context, id string) (*domain.Server, error)

	// ListServers retrieves all servers.
	ListServers(ctx context.Context) ([]*domain.Server, error)

	// GetProject retrieves a project by ID.
	GetProject(ctx context.Context, id string) (*domain.Project, error)

	// ListProjects retrieves all projects.
	ListProjects(ctx context.Context) ([]*domain.Project, error)

	// GetCredential retrieves a credential by ID.
	GetCredential(ctx context.Context, id string) (*domain.Credential, error)

	// ListCredentials retrieves all credentials.
	ListCredentials(ctx context.Context) ([]*domain.Credential, error)

	// Close releases any resources held by the backend.
	// Returns error for io.Closer compatibility and future cleanup scenarios.
	Close() error
}

// Writer is an optional interface for backends that support write operations.
// Type-assert Backend to Writer to check if writes are supported.
type Writer interface {
	CreateServer(ctx context.Context, server *domain.Server) error
	UpdateServer(ctx context.Context, server *domain.Server) error
	DeleteServer(ctx context.Context, id string) error

	CreateProject(ctx context.Context, project *domain.Project) error
	UpdateProject(ctx context.Context, project *domain.Project) error
	DeleteProject(ctx context.Context, id string) error

	CreateCredential(ctx context.Context, cred *domain.Credential) error
	UpdateCredential(ctx context.Context, cred *domain.Credential) error
	DeleteCredential(ctx context.Context, id string) error
}

// Filterer is an optional interface for backends that support server-side filtering.
// Type-assert Backend to Filterer to check if filtering is supported.
type Filterer interface {
	FilterServers(ctx context.Context, filters ServerFilter) ([]*domain.Server, error)
}

// Syncer is an optional interface for backends that support on-demand synchronization.
// Type-assert to Syncer to trigger a sync cycle (e.g., after user signs in).
type Syncer interface {
	SyncFromBackend(ctx context.Context) error
	GetStatus() BackendStatus
}

// ServerFilter captures filter criteria for server queries.
// All fields are optional (zero values ignored).
type ServerFilter struct {
	ProjectID string   // filter by project ID
	Tags      []string // filter by tags (servers must have all specified tags)
	Favorite  *bool    // tri-state: nil=any, true=favorites only, false=non-favorites only
	Query     string   // free text search (implementation-defined scope)
}
