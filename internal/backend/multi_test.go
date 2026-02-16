package backend_test

import (
	"context"
	"testing"

	"github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/backend/mock"
	"github.com/florianriquelme/ssherpa/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiBackend_MergesServers(t *testing.T) {
	// Backend A has servers 1 and 2
	backendA := mock.New()
	backendA.Seed([]*domain.Server{
		{ID: "srv1", DisplayName: "server1", Host: "host1.example.com"},
		{ID: "srv2", DisplayName: "server2", Host: "host2.example.com"},
	}, nil, nil)

	// Backend B has server 3
	backendB := mock.New()
	backendB.Seed([]*domain.Server{
		{ID: "srv3", DisplayName: "server3", Host: "host3.example.com"},
	}, nil, nil)

	// Create multi-backend
	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// List servers - should get all 3
	ctx := context.Background()
	servers, err := multi.ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 3)

	// Verify all servers present
	names := make(map[string]bool)
	for _, srv := range servers {
		names[srv.DisplayName] = true
	}
	assert.True(t, names["server1"])
	assert.True(t, names["server2"])
	assert.True(t, names["server3"])
}

func TestMultiBackend_DeduplicatesByPriority(t *testing.T) {
	// Backend A has "prod-web" with host1
	backendA := mock.New()
	backendA.Seed([]*domain.Server{
		{ID: "srv1", DisplayName: "prod-web", Host: "host1.example.com"},
		{ID: "srv2", DisplayName: "staging", Host: "staging.example.com"},
	}, nil, nil)

	// Backend B also has "prod-web" but with host2 (higher priority)
	backendB := mock.New()
	backendB.Seed([]*domain.Server{
		{ID: "srv3", DisplayName: "prod-web", Host: "host2.example.com"},
	}, nil, nil)

	// Create multi-backend with A, then B (B has higher priority)
	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// List servers - should get 2 servers (one "prod-web", one "staging")
	ctx := context.Background()
	servers, err := multi.ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 2)

	// Find the "prod-web" server
	var prodWeb *domain.Server
	for _, srv := range servers {
		if srv.DisplayName == "prod-web" {
			prodWeb = srv
			break
		}
	}

	// Should be from backend B (host2)
	require.NotNil(t, prodWeb)
	assert.Equal(t, "host2.example.com", prodWeb.Host, "Should use higher-priority backend's version")
	assert.Equal(t, "srv3", prodWeb.ID)
}

func TestMultiBackend_CaseInsensitiveDedup(t *testing.T) {
	// Backend A has "Prod-Web" (mixed case)
	backendA := mock.New()
	backendA.Seed([]*domain.Server{
		{ID: "srv1", DisplayName: "Prod-Web", Host: "host1.example.com"},
	}, nil, nil)

	// Backend B has "prod-web" (lowercase) - should be treated as duplicate
	backendB := mock.New()
	backendB.Seed([]*domain.Server{
		{ID: "srv2", DisplayName: "prod-web", Host: "host2.example.com"},
	}, nil, nil)

	// Create multi-backend with A, then B (B has higher priority)
	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// List servers - should get only 1 server (deduped)
	ctx := context.Background()
	servers, err := multi.ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 1)

	// Should be from backend B (higher priority)
	assert.Equal(t, "srv2", servers[0].ID)
	assert.Equal(t, "host2.example.com", servers[0].Host)
}

func TestMultiBackend_WriterDelegation(t *testing.T) {
	// Backend A is Writer-capable (mock implements Writer)
	backendA := mock.New()

	// Backend B is also Writer-capable
	backendB := mock.New()

	// Create multi-backend with A, then B
	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// Create a server via multi-backend
	ctx := context.Background()
	newServer := &domain.Server{
		ID:          "new-srv",
		DisplayName: "new-server",
		Host:        "new.example.com",
	}

	// Type assert to Writer interface
	writer, ok := interface{}(multi).(backend.Writer)
	require.True(t, ok, "MultiBackend should implement Writer interface")

	err := writer.CreateServer(ctx, newServer)
	require.NoError(t, err)

	// Should have delegated to backend A (first Writer-capable)
	serverInA, err := backendA.GetServer(ctx, "new-srv")
	require.NoError(t, err)
	assert.Equal(t, "new-server", serverInA.DisplayName)

	// Should NOT be in backend B
	_, err = backendB.GetServer(ctx, "new-srv")
	assert.Error(t, err, "Should not be in backend B")
}

func TestMultiBackend_CloseAll(t *testing.T) {
	backendA := mock.New()
	backendB := mock.New()

	multi := backend.NewMultiBackend(backendA, backendB)

	// Close multi-backend
	err := multi.Close()
	require.NoError(t, err)

	// Both backends should be closed
	ctx := context.Background()
	_, err = backendA.ListServers(ctx)
	assert.Error(t, err, "Backend A should be closed")

	_, err = backendB.ListServers(ctx)
	assert.Error(t, err, "Backend B should be closed")
}

func TestMultiBackend_GetServer_HighestPriorityFirst(t *testing.T) {
	// Backend A has server with ID "srv1"
	backendA := mock.New()
	backendA.Seed([]*domain.Server{
		{ID: "srv1", DisplayName: "serverA", Host: "hostA.example.com"},
	}, nil, nil)

	// Backend B also has server with ID "srv1" (different data)
	backendB := mock.New()
	backendB.Seed([]*domain.Server{
		{ID: "srv1", DisplayName: "serverB", Host: "hostB.example.com"},
	}, nil, nil)

	// Create multi-backend with A, then B (B has higher priority)
	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// GetServer should try highest priority first (backend B)
	ctx := context.Background()
	server, err := multi.GetServer(ctx, "srv1")
	require.NoError(t, err)
	assert.Equal(t, "serverB", server.DisplayName, "Should return from highest-priority backend")
	assert.Equal(t, "hostB.example.com", server.Host)
}

func TestMultiBackend_GetServer_NotFound(t *testing.T) {
	backendA := mock.New()
	backendB := mock.New()

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// Try to get a non-existent server
	ctx := context.Background()
	_, err := multi.GetServer(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMultiBackend_ListProjects_Aggregates(t *testing.T) {
	// Backend A has project 1
	backendA := mock.New()
	backendA.Seed(nil, []*domain.Project{
		{ID: "proj1", Name: "Project 1"},
	}, nil)

	// Backend B has project 2
	backendB := mock.New()
	backendB.Seed(nil, []*domain.Project{
		{ID: "proj2", Name: "Project 2"},
	}, nil)

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// List projects - should get both (no dedup for projects)
	ctx := context.Background()
	projects, err := multi.ListProjects(ctx)
	require.NoError(t, err)
	assert.Len(t, projects, 2)
}

func TestMultiBackend_SkipsErrorBackends(t *testing.T) {
	// Backend A has servers
	backendA := mock.New()
	backendA.Seed([]*domain.Server{
		{ID: "srv1", DisplayName: "server1", Host: "host1.example.com"},
	}, nil, nil)

	// Backend B is closed (will error)
	backendB := mock.New()
	backendB.Close()

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// List servers - should only get from backend A (skip B's error)
	ctx := context.Background()
	servers, err := multi.ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 1)
	assert.Equal(t, "server1", servers[0].DisplayName)
}

func TestMultiBackend_UpdateServerDelegation(t *testing.T) {
	// Backend A is Writer-capable
	backendA := mock.New()
	backendA.Seed([]*domain.Server{
		{ID: "srv1", DisplayName: "server1", Host: "host1.example.com"},
	}, nil, nil)

	multi := backend.NewMultiBackend(backendA)
	defer multi.Close()

	// Update server via Writer interface
	ctx := context.Background()
	updatedServer := &domain.Server{
		ID:          "srv1",
		DisplayName: "updated-server",
		Host:        "updated.example.com",
	}

	writer, ok := interface{}(multi).(backend.Writer)
	require.True(t, ok, "MultiBackend should implement Writer interface")

	err := writer.UpdateServer(ctx, updatedServer)
	require.NoError(t, err)

	// Verify update in backend A
	server, err := backendA.GetServer(ctx, "srv1")
	require.NoError(t, err)
	assert.Equal(t, "updated-server", server.DisplayName)
	assert.Equal(t, "updated.example.com", server.Host)
}

func TestMultiBackend_DeleteServerDelegation(t *testing.T) {
	// Backend A is Writer-capable
	backendA := mock.New()
	backendA.Seed([]*domain.Server{
		{ID: "srv1", DisplayName: "server1", Host: "host1.example.com"},
	}, nil, nil)

	multi := backend.NewMultiBackend(backendA)
	defer multi.Close()

	// Delete server via Writer interface
	ctx := context.Background()

	writer, ok := interface{}(multi).(backend.Writer)
	require.True(t, ok, "MultiBackend should implement Writer interface")

	err := writer.DeleteServer(ctx, "srv1")
	require.NoError(t, err)

	// Verify deletion in backend A
	_, err = backendA.GetServer(ctx, "srv1")
	assert.Error(t, err, "Server should be deleted")
}

func TestMultiBackend_FiltersOutSsherpaGeneratedServers(t *testing.T) {
	// Backend A (ssh-config) has two servers:
	// 1. ssherpa-generated mirror (should be filtered out)
	// 2. user-authored entry (should be kept)
	backendA := mock.New()
	backendA.Seed([]*domain.Server{
		{
			ID:          "my-server",
			DisplayName: "my-server",
			Host:        "1.2.3.4",
			Source:      "ssh-config",
			Notes:       "Source: /home/user/.ssh/ssherpa_config:5",
		},
		{
			ID:          "manual-host",
			DisplayName: "manual-host",
			Host:        "5.6.7.8",
			Source:      "ssh-config",
			Notes:       "Source: /home/user/.ssh/config:10",
		},
	}, nil, nil)

	// Backend B (1password) has one server
	backendB := mock.New()
	backendB.Seed([]*domain.Server{
		{
			ID:          "op-abc123",
			DisplayName: "my-server",
			Host:        "1.2.3.4",
			Source:      "1password",
		},
	}, nil, nil)

	// Create multi-backend with A, then B (B has higher priority)
	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// List servers - should get 2 servers (not 3)
	// "manual-host" from SSH config and "my-server" from 1Password
	// The ssherpa_config mirror should be filtered out
	ctx := context.Background()
	servers, err := multi.ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 2, "Should return 2 servers after filtering ssherpa-generated entry")

	// Verify which servers are present
	names := make(map[string]bool)
	sources := make(map[string]string)
	for _, srv := range servers {
		names[srv.DisplayName] = true
		sources[srv.DisplayName] = srv.Source
	}

	// Should have "manual-host" from ssh-config (user-authored)
	assert.True(t, names["manual-host"], "Should include user-authored SSH config entry")
	assert.Equal(t, "ssh-config", sources["manual-host"])

	// Should have "my-server" from 1password (not the ssherpa_config mirror)
	assert.True(t, names["my-server"], "Should include 1Password entry")
	assert.Equal(t, "1password", sources["my-server"], "Should be from 1Password, not ssherpa_config mirror")
}

func TestMultiBackend_RenamedOnePasswordItemNoDuplicate(t *testing.T) {
	// Backend A (ssh-config) has stale mirror with OLD name
	backendA := mock.New()
	backendA.Seed([]*domain.Server{
		{
			ID:          "old-name",
			DisplayName: "old-name",
			Host:        "1.2.3.4",
			Source:      "ssh-config",
			Notes:       "Source: /home/user/.ssh/ssherpa_config:5",
		},
	}, nil, nil)

	// Backend B (1password) has the renamed item with NEW name
	backendB := mock.New()
	backendB.Seed([]*domain.Server{
		{
			ID:          "op-abc123",
			DisplayName: "new-name",
			Host:        "1.2.3.4",
			Source:      "1password",
		},
	}, nil, nil)

	// Create multi-backend with A, then B (B has higher priority)
	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// List servers - should get 1 server with DisplayName="new-name" from 1Password
	// The stale "old-name" mirror should be filtered out
	ctx := context.Background()
	servers, err := multi.ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 1, "Should return 1 server after filtering stale ssherpa-generated entry")

	// Verify it's the renamed 1Password entry
	assert.Equal(t, "new-name", servers[0].DisplayName, "Should have new name from 1Password")
	assert.Equal(t, "1password", servers[0].Source, "Should be from 1Password")
	assert.Equal(t, "op-abc123", servers[0].ID, "Should have 1Password ID")
}

func TestMultiBackend_GetProject_NotFound(t *testing.T) {
	backendA := mock.New()
	backendB := mock.New()

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	ctx := context.Background()
	_, err := multi.GetProject(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMultiBackend_GetProject_HighestPriorityFirst(t *testing.T) {
	backendA := mock.New()
	backendA.Seed(nil, []*domain.Project{
		{ID: "proj1", Name: "Project A"},
	}, nil)

	backendB := mock.New()
	backendB.Seed(nil, []*domain.Project{
		{ID: "proj1", Name: "Project B"},
	}, nil)

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	ctx := context.Background()
	project, err := multi.GetProject(ctx, "proj1")
	require.NoError(t, err)
	assert.Equal(t, "Project B", project.Name, "Should return from highest-priority backend")
}

func TestMultiBackend_ListCredentials_Aggregates(t *testing.T) {
	backendA := mock.New()
	backendA.Seed(nil, nil, []*domain.Credential{
		{ID: "cred1", Name: "Key A"},
	})

	backendB := mock.New()
	backendB.Seed(nil, nil, []*domain.Credential{
		{ID: "cred2", Name: "Key B"},
	})

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	ctx := context.Background()
	creds, err := multi.ListCredentials(ctx)
	require.NoError(t, err)
	assert.Len(t, creds, 2)
}

func TestMultiBackend_GetCredential_NotFound(t *testing.T) {
	backendA := mock.New()
	backendB := mock.New()

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	ctx := context.Background()
	_, err := multi.GetCredential(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMultiBackend_GetCredential_HighestPriorityFirst(t *testing.T) {
	backendA := mock.New()
	backendA.Seed(nil, nil, []*domain.Credential{
		{ID: "cred1", Name: "Cred A"},
	})

	backendB := mock.New()
	backendB.Seed(nil, nil, []*domain.Credential{
		{ID: "cred1", Name: "Cred B"},
	})

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	ctx := context.Background()
	cred, err := multi.GetCredential(ctx, "cred1")
	require.NoError(t, err)
	assert.Equal(t, "Cred B", cred.Name, "Should return from highest-priority backend")
}

func TestMultiBackend_CreateProjectDelegation(t *testing.T) {
	backendA := mock.New()
	backendB := mock.New()

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	ctx := context.Background()
	project := &domain.Project{ID: "proj-new", Name: "New Project"}

	err := multi.CreateProject(ctx, project)
	require.NoError(t, err)

	// Should delegate to first Writer-capable backend (A)
	p, err := backendA.GetProject(ctx, "proj-new")
	require.NoError(t, err)
	assert.Equal(t, "New Project", p.Name)

	// Should NOT be in B
	_, err = backendB.GetProject(ctx, "proj-new")
	assert.Error(t, err)
}

func TestMultiBackend_CreateCredentialDelegation(t *testing.T) {
	backendA := mock.New()
	backendB := mock.New()

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	ctx := context.Background()
	cred := &domain.Credential{ID: "cred-new", Name: "New Key"}

	err := multi.CreateCredential(ctx, cred)
	require.NoError(t, err)

	// Should delegate to first Writer-capable backend (A)
	c, err := backendA.GetCredential(ctx, "cred-new")
	require.NoError(t, err)
	assert.Equal(t, "New Key", c.Name)

	// Should NOT be in B
	_, err = backendB.GetCredential(ctx, "cred-new")
	assert.Error(t, err)
}

func TestMultiBackend_CloseReturnsFirstError(t *testing.T) {
	backendA := mock.New()
	backendB := mock.New()

	multi := backend.NewMultiBackend(backendA, backendB)

	// Close should succeed when both backends close without error
	err := multi.Close()
	assert.NoError(t, err)
}

func TestMultiBackend_GetOnePasswordBackend_NotFound(t *testing.T) {
	// Mock backends don't implement statusGetter, so should return nil
	backendA := mock.New()
	backendB := mock.New()

	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	result := multi.GetOnePasswordBackend()
	assert.Nil(t, result)
}

func TestMultiBackend_UpdateProjectDelegation(t *testing.T) {
	backendA := mock.New()
	backendA.Seed(nil, []*domain.Project{
		{ID: "proj1", Name: "Original"},
	}, nil)

	multi := backend.NewMultiBackend(backendA)
	defer multi.Close()

	ctx := context.Background()
	updated := &domain.Project{ID: "proj1", Name: "Updated"}

	err := multi.UpdateProject(ctx, updated)
	require.NoError(t, err)

	p, err := backendA.GetProject(ctx, "proj1")
	require.NoError(t, err)
	assert.Equal(t, "Updated", p.Name)
}

func TestMultiBackend_DeleteProjectDelegation(t *testing.T) {
	backendA := mock.New()
	backendA.Seed(nil, []*domain.Project{
		{ID: "proj1", Name: "To Delete"},
	}, nil)

	multi := backend.NewMultiBackend(backendA)
	defer multi.Close()

	ctx := context.Background()
	err := multi.DeleteProject(ctx, "proj1")
	require.NoError(t, err)

	_, err = backendA.GetProject(ctx, "proj1")
	assert.Error(t, err)
}

func TestMultiBackend_UpdateCredentialDelegation(t *testing.T) {
	backendA := mock.New()
	backendA.Seed(nil, nil, []*domain.Credential{
		{ID: "cred1", Name: "Original"},
	})

	multi := backend.NewMultiBackend(backendA)
	defer multi.Close()

	ctx := context.Background()
	updated := &domain.Credential{ID: "cred1", Name: "Updated"}

	err := multi.UpdateCredential(ctx, updated)
	require.NoError(t, err)

	c, err := backendA.GetCredential(ctx, "cred1")
	require.NoError(t, err)
	assert.Equal(t, "Updated", c.Name)
}

func TestMultiBackend_DeleteCredentialDelegation(t *testing.T) {
	backendA := mock.New()
	backendA.Seed(nil, nil, []*domain.Credential{
		{ID: "cred1", Name: "To Delete"},
	})

	multi := backend.NewMultiBackend(backendA)
	defer multi.Close()

	ctx := context.Background()
	err := multi.DeleteCredential(ctx, "cred1")
	require.NoError(t, err)

	_, err = backendA.GetCredential(ctx, "cred1")
	assert.Error(t, err)
}

func TestMultiBackend_PureSshConfigServersNotFiltered(t *testing.T) {
	// Backend A (ssh-config) has user-authored entry (NOT from ssherpa_config)
	backendA := mock.New()
	backendA.Seed([]*domain.Server{
		{
			ID:          "user-host",
			DisplayName: "user-host",
			Host:        "9.8.7.6",
			Source:      "ssh-config",
			Notes:       "Source: /home/user/.ssh/config:3",
		},
	}, nil, nil)

	// Backend B (1password) has no servers
	backendB := mock.New()
	backendB.Seed([]*domain.Server{}, nil, nil)

	// Create multi-backend with A, then B
	multi := backend.NewMultiBackend(backendA, backendB)
	defer multi.Close()

	// List servers - should get 1 server
	// User-authored SSH config entries are never filtered
	ctx := context.Background()
	servers, err := multi.ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 1, "Should return user-authored SSH config entry")

	// Verify it's the user-authored entry
	assert.Equal(t, "user-host", servers[0].DisplayName)
	assert.Equal(t, "ssh-config", servers[0].Source)
	assert.Equal(t, "9.8.7.6", servers[0].Host)
}
