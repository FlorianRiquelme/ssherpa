package onepassword

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/florianriquelme/ssherpa/internal/domain"
)

// ItemToServer converts a 1Password item to a domain.Server.
// Returns error if required fields (hostname, user) are missing.
func ItemToServer(item *Item) (*domain.Server, error) {
	server := &domain.Server{
		ID:          item.ID,
		DisplayName: item.Title,
		VaultID:     item.VaultID,
		Port:        22, // default port
		Source:      "1password",
	}

	// Extract fields by title (case-insensitive)
	for _, field := range item.Fields {
		title := strings.ToLower(field.Title)
		value := field.Value

		switch title {
		case "hostname":
			server.Host = value
		case "user":
			server.User = value
		case "port":
			if port, err := strconv.Atoi(value); err == nil {
				server.Port = port
			}
		case "identity_file":
			server.IdentityFile = value
		case "remote_project_path":
			server.RemoteProjectPath = value
		case "project_tags":
			if value != "" {
				// Split comma-separated tags
				tags := strings.Split(value, ",")
				server.ProjectIDs = make([]string, 0, len(tags))
				for _, tag := range tags {
					trimmed := strings.TrimSpace(tag)
					if trimmed != "" {
						server.ProjectIDs = append(server.ProjectIDs, trimmed)
					}
				}
			}
		case "proxy_jump":
			server.Proxy = value
		case "forward_agent":
			// Store as note or ignore (not a direct Server field)
		case "extra_config":
			// Store as note or ignore (not a direct Server field)
		}
	}

	// Validate required fields
	if server.Host == "" {
		return nil, fmt.Errorf("item %q (id: %s) missing required field: hostname", item.Title, item.ID)
	}
	if server.User == "" {
		return nil, fmt.Errorf("item %q (id: %s) missing required field: user", item.Title, item.ID)
	}

	return server, nil
}

// ServerToItem converts a domain.Server to a 1Password item.
// The vaultID parameter specifies which vault the item belongs to.
func ServerToItem(server *domain.Server, vaultID string) *Item {
	item := &Item{
		ID:       server.ID,
		Title:    server.DisplayName,
		VaultID:  vaultID,
		Category: "server",
		Tags:     make([]string, 0),
		Fields:   make([]ItemField, 0),
	}

	// Ensure "ssherpa" tag is present (deduplicate)
	hasSshjesusTag := false
	for _, tag := range server.Tags {
		if strings.EqualFold(tag, "ssherpa") {
			hasSshjesusTag = true
		} else {
			item.Tags = append(item.Tags, tag)
		}
	}
	if !hasSshjesusTag {
		item.Tags = append([]string{"ssherpa"}, item.Tags...)
	} else {
		item.Tags = append([]string{"ssherpa"}, item.Tags...)
	}

	// Add fields
	item.Fields = append(item.Fields, ItemField{
		Title:     "hostname",
		Value:     server.Host,
		FieldType: "Text",
	})

	item.Fields = append(item.Fields, ItemField{
		Title:     "user",
		Value:     server.User,
		FieldType: "Text",
	})

	if server.Port != 22 && server.Port != 0 {
		item.Fields = append(item.Fields, ItemField{
			Title:     "port",
			Value:     strconv.Itoa(server.Port),
			FieldType: "Text",
		})
	}

	if server.IdentityFile != "" {
		item.Fields = append(item.Fields, ItemField{
			Title:     "identity_file",
			Value:     server.IdentityFile,
			FieldType: "Text",
		})
	}

	if server.RemoteProjectPath != "" {
		item.Fields = append(item.Fields, ItemField{
			Title:     "remote_project_path",
			Value:     server.RemoteProjectPath,
			FieldType: "Text",
		})
	}

	if len(server.ProjectIDs) > 0 {
		item.Fields = append(item.Fields, ItemField{
			Title:     "project_tags",
			Value:     strings.Join(server.ProjectIDs, ","),
			FieldType: "Text",
		})
	}

	if server.Proxy != "" {
		item.Fields = append(item.Fields, ItemField{
			Title:     "proxy_jump",
			Value:     server.Proxy,
			FieldType: "Text",
		})
	}

	return item
}

// HasSshjesusTag checks if the tags slice contains "ssherpa" (case-insensitive).
func HasSshjesusTag(tags []string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, "ssherpa") {
			return true
		}
	}
	return false
}
