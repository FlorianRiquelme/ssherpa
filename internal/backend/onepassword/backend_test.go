package onepassword

import (
	"context"
	"fmt"
	"testing"

	"github.com/florianriquelme/sshjesus/internal/backend"
	"github.com/florianriquelme/sshjesus/internal/domain"
	"github.com/florianriquelme/sshjesus/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListServers(t *testing.T) {
	client := NewMockClient()

	// Setup: 2 vaults with 3 items (2 tagged, 1 not)
	client.AddVault(Vault{ID: "vault-1", Name: "Personal"})
	client.AddVault(Vault{ID: "vault-2", Name: "Work"})

	client.AddItem(Item{
		ID:       "item-1",
		Title:    "Server 1",
		VaultID:  "vault-1",
		Category: "server",
		Tags:     []string{"sshjesus", "production"},
		Fields: []ItemField{
			{Title: "hostname", Value: "server1.example.com", FieldType: "Text"},
			{Title: "user", Value: "admin", FieldType: "Text"},
		},
	})

	client.AddItem(Item{
		ID:       "item-2",
		Title:    "Server 2",
		VaultID:  "vault-2",
		Category: "server",
		Tags:     []string{"sshjesus", "dev"},
		Fields: []ItemField{
			{Title: "hostname", Value: "server2.example.com", FieldType: "Text"},
			{Title: "user", Value: "ubuntu", FieldType: "Text"},
		},
	})

	client.AddItem(Item{
		ID:       "item-3",
		Title:    "Not Tagged",
		VaultID:  "vault-1",
		Category: "server",
		Tags:     []string{"other"},
		Fields: []ItemField{
			{Title: "hostname", Value: "server3.example.com", FieldType: "Text"},
			{Title: "user", Value: "root", FieldType: "Text"},
		},
	})

	b := New(client)
	ctx := context.Background()

	servers, err := b.ListServers(ctx)
	require.NoError(t, err)

	// Should return exactly 2 servers (filtered by sshjesus tag)
	assert.Len(t, servers, 2)

	// Verify server IDs
	ids := make([]string, len(servers))
	for i, s := range servers {
		ids[i] = s.ID
	}
	assert.Contains(t, ids, "item-1")
	assert.Contains(t, ids, "item-2")
	assert.NotContains(t, ids, "item-3")
}

func TestListServersSkipsErrorVaults(t *testing.T) {
	client := NewMockClient()

	// Setup: 2 vaults, one will error
	client.AddVault(Vault{ID: "vault-good", Name: "Good Vault"})
	client.AddVault(Vault{ID: "vault-bad", Name: "Bad Vault"})

	client.AddItem(Item{
		ID:       "item-good",
		Title:    "Good Server",
		VaultID:  "vault-good",
		Category: "server",
		Tags:     []string{"sshjesus"},
		Fields: []ItemField{
			{Title: "hostname", Value: "good.example.com", FieldType: "Text"},
			{Title: "user", Value: "admin", FieldType: "Text"},
		},
	})

	// Configure error for bad vault specifically
	client.SetVaultError("vault-bad", fmt.Errorf("permission denied"))

	b := New(client)
	ctx := context.Background()

	// Should succeed but skip the bad vault
	servers, err := b.ListServers(ctx)
	require.NoError(t, err)

	// Should still return server from good vault
	assert.Len(t, servers, 1)
	assert.Equal(t, "item-good", servers[0].ID)
}

func TestGetServerFound(t *testing.T) {
	client := NewMockClient()
	client.AddVault(Vault{ID: "vault-1", Name: "Personal"})
	client.AddItem(Item{
		ID:       "item-1",
		Title:    "Test Server",
		VaultID:  "vault-1",
		Category: "server",
		Tags:     []string{"sshjesus"},
		Fields: []ItemField{
			{Title: "hostname", Value: "test.example.com", FieldType: "Text"},
			{Title: "user", Value: "testuser", FieldType: "Text"},
		},
	})

	b := New(client)
	ctx := context.Background()

	// First list to populate cache
	_, err := b.ListServers(ctx)
	require.NoError(t, err)

	// Get specific server
	server, err := b.GetServer(ctx, "item-1")
	require.NoError(t, err)
	assert.Equal(t, "item-1", server.ID)
	assert.Equal(t, "test.example.com", server.Host)
}

func TestGetServerNotFound(t *testing.T) {
	client := NewMockClient()
	client.AddVault(Vault{ID: "vault-1", Name: "Personal"})

	b := New(client)
	ctx := context.Background()

	// List to ensure cache is populated
	_, err := b.ListServers(ctx)
	require.NoError(t, err)

	// Try to get non-existent server
	_, err = b.GetServer(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrServerNotFound))
}

func TestCreateServer(t *testing.T) {
	client := NewMockClient()
	client.AddVault(Vault{ID: "vault-1", Name: "Personal"})

	b := New(client)
	ctx := context.Background()

	server := &domain.Server{
		ID:          "new-server",
		DisplayName: "New Server",
		Host:        "new.example.com",
		User:        "admin",
		Port:        2222,
		VaultID:     "vault-1",
	}

	err := b.CreateServer(ctx, server)
	require.NoError(t, err)

	// Verify it appears in list
	servers, err := b.ListServers(ctx)
	require.NoError(t, err)

	found := false
	for _, s := range servers {
		if s.ID == "new-server" {
			found = true
			assert.Equal(t, "New Server", s.DisplayName)
			assert.Equal(t, "new.example.com", s.Host)
			break
		}
	}
	assert.True(t, found, "Created server should appear in ListServers")
}

func TestUpdateServer(t *testing.T) {
	client := NewMockClient()
	client.AddVault(Vault{ID: "vault-1", Name: "Personal"})
	client.AddItem(Item{
		ID:       "item-1",
		Title:    "Old Title",
		VaultID:  "vault-1",
		Category: "server",
		Tags:     []string{"sshjesus"},
		Fields: []ItemField{
			{Title: "hostname", Value: "old.example.com", FieldType: "Text"},
			{Title: "user", Value: "olduser", FieldType: "Text"},
		},
	})

	b := New(client)
	ctx := context.Background()

	// Populate cache
	_, err := b.ListServers(ctx)
	require.NoError(t, err)

	// Update server
	updated := &domain.Server{
		ID:          "item-1",
		DisplayName: "New Title",
		Host:        "new.example.com",
		User:        "newuser",
		Port:        22,
		VaultID:     "vault-1",
	}

	err = b.UpdateServer(ctx, updated)
	require.NoError(t, err)

	// Verify update
	server, err := b.GetServer(ctx, "item-1")
	require.NoError(t, err)
	assert.Equal(t, "New Title", server.DisplayName)
	assert.Equal(t, "new.example.com", server.Host)
	assert.Equal(t, "newuser", server.User)
}

func TestDeleteServer(t *testing.T) {
	client := NewMockClient()
	client.AddVault(Vault{ID: "vault-1", Name: "Personal"})
	client.AddItem(Item{
		ID:       "item-1",
		Title:    "Server to Delete",
		VaultID:  "vault-1",
		Category: "server",
		Tags:     []string{"sshjesus"},
		Fields: []ItemField{
			{Title: "hostname", Value: "delete.example.com", FieldType: "Text"},
			{Title: "user", Value: "admin", FieldType: "Text"},
		},
	})

	b := New(client)
	ctx := context.Background()

	// Populate cache
	servers, err := b.ListServers(ctx)
	require.NoError(t, err)
	assert.Len(t, servers, 1)

	// Delete server
	err = b.DeleteServer(ctx, "item-1")
	require.NoError(t, err)

	// Verify deletion
	_, err = b.GetServer(ctx, "item-1")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrServerNotFound))
}

func TestClosedBackendReturnsError(t *testing.T) {
	client := NewMockClient()
	client.AddVault(Vault{ID: "vault-1", Name: "Personal"})

	b := New(client)
	ctx := context.Background()

	// Close backend
	err := b.Close()
	require.NoError(t, err)

	// All operations should return ErrBackendUnavailable
	_, err = b.ListServers(ctx)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))

	_, err = b.GetServer(ctx, "any-id")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))

	server := &domain.Server{
		ID:      "test",
		Host:    "test.com",
		User:    "test",
		VaultID: "vault-1",
	}

	err = b.CreateServer(ctx, server)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))

	err = b.UpdateServer(ctx, server)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))

	err = b.DeleteServer(ctx, "any-id")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errors.ErrBackendUnavailable))
}

func TestInterfaceCompliance(t *testing.T) {
	client := NewMockClient()
	b := New(client)

	// Verify interface implementations at compile time
	var _ backend.Backend = b
	var _ backend.Writer = b

	// If we get here, the test passes
	assert.True(t, true)
}
