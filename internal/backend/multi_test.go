package backend_test

import (
	"context"
	"testing"

	"github.com/florianriquelme/sshjesus/internal/backend"
	"github.com/florianriquelme/sshjesus/internal/backend/mock"
	"github.com/florianriquelme/sshjesus/internal/domain"
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
