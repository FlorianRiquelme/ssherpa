package onepassword

import (
	"testing"

	"github.com/florianriquelme/ssherpa/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemToServer_Complete(t *testing.T) {
	item := &Item{
		ID:       "item-123",
		Title:    "Production API",
		VaultID:  "vault-456",
		Category: "server",
		Tags:     []string{"sshjesus", "production"},
		Fields: []ItemField{
			{Title: "hostname", Value: "api.example.com", FieldType: "Text"},
			{Title: "user", Value: "deploy", FieldType: "Text"},
			{Title: "port", Value: "2222", FieldType: "Text"},
			{Title: "identity_file", Value: "/home/user/.ssh/prod_key", FieldType: "Text"},
			{Title: "remote_project_path", Value: "/var/www/app", FieldType: "Text"},
			{Title: "project_tags", Value: "proj-api,proj-backend", FieldType: "Text"},
			{Title: "proxy_jump", Value: "bastion.example.com", FieldType: "Text"},
			{Title: "forward_agent", Value: "true", FieldType: "Text"},
			{Title: "extra_config", Value: "StrictHostKeyChecking=no", FieldType: "Text"},
		},
	}

	server, err := ItemToServer(item)
	require.NoError(t, err)

	assert.Equal(t, "item-123", server.ID)
	assert.Equal(t, "Production API", server.DisplayName)
	assert.Equal(t, "api.example.com", server.Host)
	assert.Equal(t, "deploy", server.User)
	assert.Equal(t, 2222, server.Port)
	assert.Equal(t, "/home/user/.ssh/prod_key", server.IdentityFile)
	assert.Equal(t, "/var/www/app", server.RemoteProjectPath)
	assert.Equal(t, []string{"proj-api", "proj-backend"}, server.ProjectIDs)
	assert.Equal(t, "bastion.example.com", server.Proxy)
	assert.Equal(t, "vault-456", server.VaultID)
}

func TestItemToServer_Minimal(t *testing.T) {
	item := &Item{
		ID:       "item-789",
		Title:    "Dev Server",
		VaultID:  "vault-123",
		Category: "server",
		Tags:     []string{"sshjesus"},
		Fields: []ItemField{
			{Title: "hostname", Value: "dev.example.com", FieldType: "Text"},
			{Title: "user", Value: "ubuntu", FieldType: "Text"},
		},
	}

	server, err := ItemToServer(item)
	require.NoError(t, err)

	assert.Equal(t, "item-789", server.ID)
	assert.Equal(t, "Dev Server", server.DisplayName)
	assert.Equal(t, "dev.example.com", server.Host)
	assert.Equal(t, "ubuntu", server.User)
	assert.Equal(t, 22, server.Port, "Should default to port 22")
	assert.Equal(t, "", server.IdentityFile)
	assert.Equal(t, "", server.RemoteProjectPath)
	assert.Empty(t, server.ProjectIDs)
	assert.Equal(t, "", server.Proxy)
	assert.Equal(t, "vault-123", server.VaultID)
}

func TestItemToServer_MissingHostname(t *testing.T) {
	item := &Item{
		ID:       "item-no-host",
		Title:    "Invalid Server",
		VaultID:  "vault-123",
		Category: "server",
		Tags:     []string{"sshjesus"},
		Fields: []ItemField{
			{Title: "user", Value: "ubuntu", FieldType: "Text"},
		},
	}

	_, err := ItemToServer(item)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hostname")
}

func TestItemToServer_MissingUser(t *testing.T) {
	item := &Item{
		ID:       "item-no-user",
		Title:    "Invalid Server",
		VaultID:  "vault-123",
		Category: "server",
		Tags:     []string{"sshjesus"},
		Fields: []ItemField{
			{Title: "hostname", Value: "test.example.com", FieldType: "Text"},
		},
	}

	_, err := ItemToServer(item)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user")
}

func TestServerToItem_RoundTrip(t *testing.T) {
	original := &domain.Server{
		ID:                "srv-roundtrip",
		DisplayName:       "Roundtrip Server",
		Host:              "roundtrip.example.com",
		User:              "admin",
		Port:              8022,
		IdentityFile:      "/home/user/.ssh/roundtrip_key",
		RemoteProjectPath: "/opt/app",
		ProjectIDs:        []string{"proj-1", "proj-2"},
		Proxy:             "jump.example.com",
		VaultID:           "vault-rt",
	}

	// Convert to item
	item := ServerToItem(original, "vault-rt")

	// Verify item structure
	assert.Equal(t, "srv-roundtrip", item.ID)
	assert.Equal(t, "Roundtrip Server", item.Title)
	assert.Equal(t, "vault-rt", item.VaultID)
	assert.Equal(t, "server", item.Category)
	assert.Contains(t, item.Tags, "sshjesus")

	// Convert back to server
	recovered, err := ItemToServer(item)
	require.NoError(t, err)

	// Verify lossless conversion
	assert.Equal(t, original.ID, recovered.ID)
	assert.Equal(t, original.DisplayName, recovered.DisplayName)
	assert.Equal(t, original.Host, recovered.Host)
	assert.Equal(t, original.User, recovered.User)
	assert.Equal(t, original.Port, recovered.Port)
	assert.Equal(t, original.IdentityFile, recovered.IdentityFile)
	assert.Equal(t, original.RemoteProjectPath, recovered.RemoteProjectPath)
	assert.Equal(t, original.ProjectIDs, recovered.ProjectIDs)
	assert.Equal(t, original.Proxy, recovered.Proxy)
	assert.Equal(t, original.VaultID, recovered.VaultID)
}

func TestHasSshjesusTag_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected bool
	}{
		{
			name:     "lowercase",
			tags:     []string{"sshjesus", "production"},
			expected: true,
		},
		{
			name:     "uppercase",
			tags:     []string{"SSHJESUS", "dev"},
			expected: true,
		},
		{
			name:     "mixed case",
			tags:     []string{"SshJesus", "test"},
			expected: true,
		},
		{
			name:     "camel case",
			tags:     []string{"production", "SshJesus"},
			expected: true,
		},
		{
			name:     "not present",
			tags:     []string{"production", "server"},
			expected: false,
		},
		{
			name:     "empty",
			tags:     []string{},
			expected: false,
		},
		{
			name:     "partial match",
			tags:     []string{"ssh", "jesus"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasSshjesusTag(tt.tags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServerToItem_EnsuresSshjesusTag(t *testing.T) {
	server := &domain.Server{
		ID:          "srv-notag",
		DisplayName: "No Tag Server",
		Host:        "notag.example.com",
		User:        "user",
		Port:        22,
		VaultID:     "vault-123",
	}

	item := ServerToItem(server, "vault-123")

	assert.Contains(t, item.Tags, "sshjesus", "Should always include sshjesus tag")
}

func TestServerToItem_DeduplicatesSshjesusTag(t *testing.T) {
	server := &domain.Server{
		ID:          "srv-dupetag",
		DisplayName: "Dupe Tag Server",
		Host:        "dupe.example.com",
		User:        "user",
		Port:        22,
		Tags:        []string{"sshjesus", "production"},
		VaultID:     "vault-123",
	}

	item := ServerToItem(server, "vault-123")

	// Count occurrences of sshjesus tag
	count := 0
	for _, tag := range item.Tags {
		if tag == "sshjesus" {
			count++
		}
	}

	assert.Equal(t, 1, count, "Should only have one sshjesus tag, not duplicated")
}
