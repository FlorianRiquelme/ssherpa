package mock

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/domain"
	backendErrors "github.com/florianriquelme/ssherpa/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Server CRUD Tests =====

func TestCreateServer(t *testing.T) {
	b := New()
	ctx := context.Background()

	server := &domain.Server{
		ID:          "srv-1",
		Host:        "example.com",
		User:        "admin",
		Port:        22,
		DisplayName: "Test Server",
	}

	err := b.CreateServer(ctx, server)
	require.NoError(t, err)

	// Retrieve by ID and verify fields match
	retrieved, err := b.GetServer(ctx, "srv-1")
	require.NoError(t, err)
	assert.Equal(t, server.ID, retrieved.ID)
	assert.Equal(t, server.Host, retrieved.Host)
	assert.Equal(t, server.User, retrieved.User)
	assert.Equal(t, server.Port, retrieved.Port)
	assert.Equal(t, server.DisplayName, retrieved.DisplayName)
}

func TestCreateServerDuplicate(t *testing.T) {
	b := New()
	ctx := context.Background()

	server := &domain.Server{
		ID:          "srv-1",
		Host:        "example.com",
		User:        "admin",
		Port:        22,
		DisplayName: "Test Server",
	}

	err := b.CreateServer(ctx, server)
	require.NoError(t, err)

	// Second create with same ID should fail
	err = b.CreateServer(ctx, server)
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrDuplicateID))
}

func TestGetServerNotFound(t *testing.T) {
	b := New()
	ctx := context.Background()

	server, err := b.GetServer(ctx, "nonexistent")
	require.Error(t, err)
	assert.Nil(t, server)
	assert.True(t, errors.Is(err, backendErrors.ErrServerNotFound))
}

func TestListServers(t *testing.T) {
	b := New()
	ctx := context.Background()

	servers := []*domain.Server{
		{ID: "srv-1", Host: "host1.com", User: "user1", Port: 22, DisplayName: "Server 1"},
		{ID: "srv-2", Host: "host2.com", User: "user2", Port: 22, DisplayName: "Server 2"},
		{ID: "srv-3", Host: "host3.com", User: "user3", Port: 22, DisplayName: "Server 3"},
	}

	for _, srv := range servers {
		err := b.CreateServer(ctx, srv)
		require.NoError(t, err)
	}

	list, err := b.ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestListServersEmpty(t *testing.T) {
	b := New()
	ctx := context.Background()

	list, err := b.ListServers(ctx)
	require.NoError(t, err)
	assert.NotNil(t, list)
	assert.Len(t, list, 0)
}

func TestUpdateServer(t *testing.T) {
	b := New()
	ctx := context.Background()

	server := &domain.Server{
		ID:          "srv-1",
		Host:        "example.com",
		User:        "admin",
		Port:        22,
		DisplayName: "Test Server",
	}

	err := b.CreateServer(ctx, server)
	require.NoError(t, err)

	// Update fields
	updated := &domain.Server{
		ID:          "srv-1",
		Host:        "updated.com",
		User:        "root",
		Port:        2222,
		DisplayName: "Updated Server",
	}

	err = b.UpdateServer(ctx, updated)
	require.NoError(t, err)

	// Retrieve and verify changes
	retrieved, err := b.GetServer(ctx, "srv-1")
	require.NoError(t, err)
	assert.Equal(t, "updated.com", retrieved.Host)
	assert.Equal(t, "root", retrieved.User)
	assert.Equal(t, 2222, retrieved.Port)
	assert.Equal(t, "Updated Server", retrieved.DisplayName)
}

func TestUpdateServerNotFound(t *testing.T) {
	b := New()
	ctx := context.Background()

	server := &domain.Server{
		ID:          "nonexistent",
		Host:        "example.com",
		User:        "admin",
		Port:        22,
		DisplayName: "Test Server",
	}

	err := b.UpdateServer(ctx, server)
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrServerNotFound))
}

func TestDeleteServer(t *testing.T) {
	b := New()
	ctx := context.Background()

	server := &domain.Server{
		ID:          "srv-1",
		Host:        "example.com",
		User:        "admin",
		Port:        22,
		DisplayName: "Test Server",
	}

	err := b.CreateServer(ctx, server)
	require.NoError(t, err)

	// Delete the server
	err = b.DeleteServer(ctx, "srv-1")
	require.NoError(t, err)

	// Subsequent get should return not found
	_, err = b.GetServer(ctx, "srv-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrServerNotFound))
}

func TestDeleteServerNotFound(t *testing.T) {
	b := New()
	ctx := context.Background()

	err := b.DeleteServer(ctx, "nonexistent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrServerNotFound))
}

// ===== Project CRUD Tests =====

func TestCreateProject(t *testing.T) {
	b := New()
	ctx := context.Background()

	project := &domain.Project{
		ID:          "prj-1",
		Name:        "Test Project",
		Description: "A test project",
	}

	err := b.CreateProject(ctx, project)
	require.NoError(t, err)

	retrieved, err := b.GetProject(ctx, "prj-1")
	require.NoError(t, err)
	assert.Equal(t, project.ID, retrieved.ID)
	assert.Equal(t, project.Name, retrieved.Name)
	assert.Equal(t, project.Description, retrieved.Description)
}

func TestGetProjectNotFound(t *testing.T) {
	b := New()
	ctx := context.Background()

	project, err := b.GetProject(ctx, "nonexistent")
	require.Error(t, err)
	assert.Nil(t, project)
	assert.True(t, errors.Is(err, backendErrors.ErrProjectNotFound))
}

func TestListProjects(t *testing.T) {
	b := New()
	ctx := context.Background()

	projects := []*domain.Project{
		{ID: "prj-1", Name: "Project 1", Description: "Desc 1"},
		{ID: "prj-2", Name: "Project 2", Description: "Desc 2"},
	}

	for _, prj := range projects {
		err := b.CreateProject(ctx, prj)
		require.NoError(t, err)
	}

	list, err := b.ListProjects(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestUpdateProject(t *testing.T) {
	b := New()
	ctx := context.Background()

	project := &domain.Project{
		ID:          "prj-1",
		Name:        "Test Project",
		Description: "Original description",
	}

	err := b.CreateProject(ctx, project)
	require.NoError(t, err)

	updated := &domain.Project{
		ID:          "prj-1",
		Name:        "Updated Project",
		Description: "Updated description",
	}

	err = b.UpdateProject(ctx, updated)
	require.NoError(t, err)

	retrieved, err := b.GetProject(ctx, "prj-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated Project", retrieved.Name)
	assert.Equal(t, "Updated description", retrieved.Description)
}

func TestDeleteProject(t *testing.T) {
	b := New()
	ctx := context.Background()

	project := &domain.Project{
		ID:          "prj-1",
		Name:        "Test Project",
		Description: "Test description",
	}

	err := b.CreateProject(ctx, project)
	require.NoError(t, err)

	err = b.DeleteProject(ctx, "prj-1")
	require.NoError(t, err)

	_, err = b.GetProject(ctx, "prj-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrProjectNotFound))
}

// ===== Credential CRUD Tests =====

func TestCreateCredential(t *testing.T) {
	b := New()
	ctx := context.Background()

	credential := &domain.Credential{
		ID:          "cred-1",
		Name:        "Test Credential",
		Type:        domain.CredentialKeyFile,
		KeyFilePath: "/path/to/key",
	}

	err := b.CreateCredential(ctx, credential)
	require.NoError(t, err)

	retrieved, err := b.GetCredential(ctx, "cred-1")
	require.NoError(t, err)
	assert.Equal(t, credential.ID, retrieved.ID)
	assert.Equal(t, credential.Name, retrieved.Name)
	assert.Equal(t, credential.Type, retrieved.Type)
	assert.Equal(t, credential.KeyFilePath, retrieved.KeyFilePath)
}

func TestGetCredentialNotFound(t *testing.T) {
	b := New()
	ctx := context.Background()

	credential, err := b.GetCredential(ctx, "nonexistent")
	require.Error(t, err)
	assert.Nil(t, credential)
	assert.True(t, errors.Is(err, backendErrors.ErrCredentialNotFound))
}

func TestListCredentials(t *testing.T) {
	b := New()
	ctx := context.Background()

	credentials := []*domain.Credential{
		{ID: "cred-1", Name: "Credential 1", Type: domain.CredentialKeyFile, KeyFilePath: "/path/1"},
		{ID: "cred-2", Name: "Credential 2", Type: domain.CredentialSSHAgent},
	}

	for _, cred := range credentials {
		err := b.CreateCredential(ctx, cred)
		require.NoError(t, err)
	}

	list, err := b.ListCredentials(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestUpdateCredential(t *testing.T) {
	b := New()
	ctx := context.Background()

	credential := &domain.Credential{
		ID:          "cred-1",
		Name:        "Test Credential",
		Type:        domain.CredentialKeyFile,
		KeyFilePath: "/path/to/key",
	}

	err := b.CreateCredential(ctx, credential)
	require.NoError(t, err)

	updated := &domain.Credential{
		ID:          "cred-1",
		Name:        "Updated Credential",
		Type:        domain.CredentialSSHAgent,
		KeyFilePath: "",
	}

	err = b.UpdateCredential(ctx, updated)
	require.NoError(t, err)

	retrieved, err := b.GetCredential(ctx, "cred-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated Credential", retrieved.Name)
	assert.Equal(t, domain.CredentialSSHAgent, retrieved.Type)
}

func TestDeleteCredential(t *testing.T) {
	b := New()
	ctx := context.Background()

	credential := &domain.Credential{
		ID:          "cred-1",
		Name:        "Test Credential",
		Type:        domain.CredentialKeyFile,
		KeyFilePath: "/path/to/key",
	}

	err := b.CreateCredential(ctx, credential)
	require.NoError(t, err)

	err = b.DeleteCredential(ctx, "cred-1")
	require.NoError(t, err)

	_, err = b.GetCredential(ctx, "cred-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrCredentialNotFound))
}

// ===== Error Handling Tests =====

func TestClosedBackend(t *testing.T) {
	b := New()
	ctx := context.Background()

	// Close the backend
	err := b.Close()
	require.NoError(t, err)

	// All operations should return ErrBackendUnavailable
	_, err = b.GetServer(ctx, "srv-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrBackendUnavailable))

	err = b.CreateServer(ctx, &domain.Server{ID: "srv-1"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrBackendUnavailable))

	_, err = b.ListServers(ctx)
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrBackendUnavailable))

	err = b.UpdateServer(ctx, &domain.Server{ID: "srv-1"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrBackendUnavailable))

	err = b.DeleteServer(ctx, "srv-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErrors.ErrBackendUnavailable))
}

func TestErrorChain(t *testing.T) {
	b := New()
	ctx := context.Background()

	_, err := b.GetServer(ctx, "nonexistent")
	require.Error(t, err)

	// Verify errors.Is works through BackendError wrapper
	assert.True(t, errors.Is(err, backendErrors.ErrServerNotFound))
}

func TestErrorAs(t *testing.T) {
	b := New()
	ctx := context.Background()

	_, err := b.GetServer(ctx, "nonexistent")
	require.Error(t, err)

	// Verify errors.As extracts BackendError
	var backendErr *backendErrors.BackendError
	assert.True(t, errors.As(err, &backendErr))
	assert.Equal(t, "GetServer", backendErr.Op)
	assert.Equal(t, "mock", backendErr.Backend)
}

// ===== Thread Safety Tests =====

func TestConcurrentAccess(t *testing.T) {
	b := New()
	ctx := context.Background()

	// Seed some initial data
	for i := 0; i < 10; i++ {
		server := &domain.Server{
			ID:          "srv-" + string(rune('0'+i)),
			Host:        "host.com",
			User:        "user",
			Port:        22,
			DisplayName: "Server",
		}
		err := b.CreateServer(ctx, server)
		require.NoError(t, err)
	}

	// Launch 10 goroutines doing concurrent reads and writes
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()

			// Do various operations
			for j := 0; j < 100; j++ {
				// Read
				_, _ = b.ListServers(ctx)
				_, _ = b.GetServer(ctx, "srv-0")

				// Write (create new server with unique ID)
				server := &domain.Server{
					ID:          "concurrent-" + time.Now().String(),
					Host:        "host.com",
					User:        "user",
					Port:        22,
					DisplayName: "Server",
				}
				_ = b.CreateServer(ctx, server)
			}
		}(i)
	}

	// Wait for all goroutines to complete (no panics or data races)
	wg.Wait()
}

// ===== Copy Semantics Tests =====

func TestGetServerReturnsCopy(t *testing.T) {
	b := New()
	ctx := context.Background()

	server := &domain.Server{
		ID:          "srv-1",
		Host:        "example.com",
		User:        "admin",
		Port:        22,
		DisplayName: "Test Server",
	}

	err := b.CreateServer(ctx, server)
	require.NoError(t, err)

	// Get the server
	retrieved, err := b.GetServer(ctx, "srv-1")
	require.NoError(t, err)

	// Modify returned server
	retrieved.Host = "modified.com"
	retrieved.DisplayName = "Modified"

	// Re-get should return original values
	reRetrieved, err := b.GetServer(ctx, "srv-1")
	require.NoError(t, err)
	assert.Equal(t, "example.com", reRetrieved.Host)
	assert.Equal(t, "Test Server", reRetrieved.DisplayName)
}

func TestCreateServerStoresCopy(t *testing.T) {
	b := New()
	ctx := context.Background()

	server := &domain.Server{
		ID:          "srv-1",
		Host:        "example.com",
		User:        "admin",
		Port:        22,
		DisplayName: "Test Server",
	}

	err := b.CreateServer(ctx, server)
	require.NoError(t, err)

	// Modify original after create
	server.Host = "modified.com"
	server.DisplayName = "Modified"

	// Get should return stored values (not modified)
	retrieved, err := b.GetServer(ctx, "srv-1")
	require.NoError(t, err)
	assert.Equal(t, "example.com", retrieved.Host)
	assert.Equal(t, "Test Server", retrieved.DisplayName)
}

// ===== Interface Verification =====

var _ backend.Backend = (*Backend)(nil)
var _ backend.Writer = (*Backend)(nil)
