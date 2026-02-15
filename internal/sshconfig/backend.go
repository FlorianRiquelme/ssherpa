package sshconfig

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/domain"
	"github.com/florianriquelme/ssherpa/internal/errors"
)

// Backend implements backend.Backend interface for SSH config files.
// Read-only backend that parses ~/.ssh/config and exposes hosts as domain.Server.
// Does NOT implement backend.Writer interface.
type Backend struct {
	hosts  []SSHHost
	closed bool
	mu     sync.RWMutex
}

// Compile-time interface verification
var _ backend.Backend = (*Backend)(nil)

// New creates a new sshconfig backend by parsing the SSH config file at configPath.
func New(configPath string) (*Backend, error) {
	hosts, err := ParseSSHConfig(configPath)
	if err != nil {
		return nil, &errors.BackendError{
			Op:      "New",
			Backend: "sshconfig",
			Err:     err,
		}
	}

	return &Backend{
		hosts: hosts,
	}, nil
}

// GetServer retrieves a server by ID (Host pattern name).
func (b *Backend) GetServer(ctx context.Context, id string) (*domain.Server, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, &errors.BackendError{
			Op:      "GetServer",
			Backend: "sshconfig",
			Err:     errors.ErrBackendUnavailable,
		}
	}

	// Linear search through hosts (config files are small)
	for _, host := range b.hosts {
		if host.Name == id {
			// Return a copy (copy-on-read pattern)
			server := b.toServer(host)
			return &server, nil
		}
	}

	return nil, &errors.BackendError{
		Op:      "GetServer",
		Backend: "sshconfig",
		Err:     errors.ErrServerNotFound,
	}
}

// ListServers returns all hosts as domain.Server entries.
// Includes ALL hosts, even those with ParseError (shown with warning indicator in TUI).
func (b *Backend) ListServers(ctx context.Context) ([]*domain.Server, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, &errors.BackendError{
			Op:      "ListServers",
			Backend: "sshconfig",
			Err:     errors.ErrBackendUnavailable,
		}
	}

	// Convert all hosts to domain.Server
	servers := make([]*domain.Server, 0, len(b.hosts))
	for _, host := range b.hosts {
		server := b.toServer(host)
		servers = append(servers, &server)
	}

	return servers, nil
}

// GetProject always returns ErrProjectNotFound (SSH config has no projects).
func (b *Backend) GetProject(ctx context.Context, id string) (*domain.Project, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, &errors.BackendError{
			Op:      "GetProject",
			Backend: "sshconfig",
			Err:     errors.ErrBackendUnavailable,
		}
	}

	return nil, &errors.BackendError{
		Op:      "GetProject",
		Backend: "sshconfig",
		Err:     errors.ErrProjectNotFound,
	}
}

// ListProjects returns an empty slice (SSH config has no projects).
func (b *Backend) ListProjects(ctx context.Context) ([]*domain.Project, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, &errors.BackendError{
			Op:      "ListProjects",
			Backend: "sshconfig",
			Err:     errors.ErrBackendUnavailable,
		}
	}

	return []*domain.Project{}, nil
}

// GetCredential always returns ErrCredentialNotFound (SSH config has no credentials).
func (b *Backend) GetCredential(ctx context.Context, id string) (*domain.Credential, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, &errors.BackendError{
			Op:      "GetCredential",
			Backend: "sshconfig",
			Err:     errors.ErrBackendUnavailable,
		}
	}

	return nil, &errors.BackendError{
		Op:      "GetCredential",
		Backend: "sshconfig",
		Err:     errors.ErrCredentialNotFound,
	}
}

// ListCredentials returns an empty slice (SSH config has no credentials).
func (b *Backend) ListCredentials(ctx context.Context) ([]*domain.Credential, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil, &errors.BackendError{
			Op:      "ListCredentials",
			Backend: "sshconfig",
			Err:     errors.ErrBackendUnavailable,
		}
	}

	return []*domain.Credential{}, nil
}

// Close marks the backend as closed. Future operations return ErrBackendUnavailable.
func (b *Backend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	return nil
}

// toServer converts an SSHHost to a domain.Server.
// Private helper used by GetServer and ListServers.
func (b *Backend) toServer(host SSHHost) domain.Server {
	server := domain.Server{
		ID:          host.Name,
		DisplayName: host.Name,
		Host:        host.Hostname,
		User:        host.User,
		Port:        parsePort(host.Port),
		Tags:        []string{}, // SSH config has no tag concept
		Source:      "ssh-config",
	}

	// Use Host as fallback if Hostname is empty (SSH behavior)
	if server.Host == "" {
		server.Host = host.Name
	}

	// Extract first IdentityFile if available
	if len(host.IdentityFile) > 0 {
		server.IdentityFile = host.IdentityFile[0]
	}

	// Extract ProxyJump if available
	if proxyJump, ok := host.AllOptions["ProxyJump"]; ok && len(proxyJump) > 0 {
		server.Proxy = proxyJump[0]
	}

	// Set Notes with source file information
	if host.ParseError != nil {
		server.Notes = fmt.Sprintf("Parse error: %v (Source: %s:%d)",
			host.ParseError, host.SourceFile, host.SourceLine)
	} else if host.SourceFile != "" {
		if host.SourceLine > 0 {
			server.Notes = fmt.Sprintf("Source: %s:%d", host.SourceFile, host.SourceLine)
		} else {
			server.Notes = fmt.Sprintf("Source: %s", host.SourceFile)
		}
	}

	return server
}

// parsePort parses a port string to int, defaulting to 22 if empty or invalid.
func parsePort(portStr string) int {
	if portStr == "" {
		return 22
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return 22
	}

	return port
}
