package onepassword

import (
	"context"
	"fmt"
	"time"

	"github.com/1password/onepassword-sdk-go"
)

// Client abstracts 1Password SDK operations for testability.
// This interface enables testing with mock implementations without
// depending on real 1Password vaults.
type Client interface {
	ListVaults(ctx context.Context) ([]Vault, error)
	ListItems(ctx context.Context, vaultID string) ([]Item, error)
	GetItem(ctx context.Context, vaultID, itemID string) (*Item, error)
	CreateItem(ctx context.Context, item *Item) (*Item, error)
	UpdateItem(ctx context.Context, item *Item) (*Item, error)
	DeleteItem(ctx context.Context, vaultID, itemID string) error
	Close() error
}

// Vault represents a 1Password vault with minimal fields needed by sshjesus.
type Vault struct {
	ID   string
	Name string
}

// Item represents a 1Password item with simplified structure.
type Item struct {
	ID       string
	Title    string
	VaultID  string
	Category string
	Tags     []string
	Fields   []ItemField
}

// ItemField represents a field within a 1Password item.
type ItemField struct {
	ID        string
	Title     string
	SectionID *string
	Value     string
	FieldType string // "Text" or "Concealed"
}

// SDKClient wraps the real 1Password SDK client.
type SDKClient struct {
	client *onepassword.Client
}

// NewDesktopAppClient creates a client that integrates with 1Password desktop app.
// Uses desktop app integration for authentication (no tokens needed).
func NewDesktopAppClient(accountName string) (*SDKClient, error) {
	client, err := onepassword.NewClient(
		context.Background(),
		onepassword.WithDesktopAppIntegration(accountName),
		onepassword.WithIntegrationInfo("sshjesus", "v0.1.0"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize 1Password desktop app client: %w", err)
	}

	return &SDKClient{client: client}, nil
}

// NewServiceAccountClient creates a client using a service account token.
// Fallback authentication method when desktop app is unavailable.
func NewServiceAccountClient(token string) (*SDKClient, error) {
	client, err := onepassword.NewClient(
		context.Background(),
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo("sshjesus", "v0.1.0"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize 1Password service account client: %w", err)
	}

	return &SDKClient{client: client}, nil
}

// ListVaults retrieves all accessible vaults.
func (c *SDKClient) ListVaults(ctx context.Context) ([]Vault, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	overviews, err := c.client.Vaults().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list vaults: %w", err)
	}

	vaults := make([]Vault, 0, len(overviews))
	for _, v := range overviews {
		vaults = append(vaults, Vault{
			ID:   v.ID,
			Name: v.Title,
		})
	}

	return vaults, nil
}

// ListItems retrieves all items in a vault.
func (c *SDKClient) ListItems(ctx context.Context, vaultID string) ([]Item, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	overviews, err := c.client.Items().List(ctx, vaultID)
	if err != nil {
		return nil, fmt.Errorf("failed to list items in vault %s: %w", vaultID, err)
	}

	// List returns ItemOverview, need to get full items
	items := make([]Item, 0, len(overviews))
	for _, overview := range overviews {
		item, err := c.GetItem(ctx, vaultID, overview.ID)
		if err != nil {
			// Skip items we can't fetch (might be deleted concurrently)
			continue
		}
		items = append(items, *item)
	}

	return items, nil
}

// GetItem retrieves a specific item by ID.
func (c *SDKClient) GetItem(ctx context.Context, vaultID, itemID string) (*Item, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sdkItem, err := c.client.Items().Get(ctx, vaultID, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get item %s from vault %s: %w", itemID, vaultID, err)
	}

	item := c.convertSDKItem(&sdkItem)
	return &item, nil
}

// CreateItem creates a new item in 1Password.
func (c *SDKClient) CreateItem(ctx context.Context, item *Item) (*Item, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Convert to SDK ItemCreateParams
	params := c.convertToCreateParams(item)

	created, err := c.client.Items().Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	result := c.convertSDKItem(&created)
	return &result, nil
}

// UpdateItem updates an existing item in 1Password.
func (c *SDKClient) UpdateItem(ctx context.Context, item *Item) (*Item, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Convert to SDK Item
	sdkItem := c.convertToSDKItem(item)

	updated, err := c.client.Items().Put(ctx, sdkItem)
	if err != nil {
		return nil, fmt.Errorf("failed to update item %s: %w", item.ID, err)
	}

	result := c.convertSDKItem(&updated)
	return &result, nil
}

// DeleteItem deletes an item from 1Password.
func (c *SDKClient) DeleteItem(ctx context.Context, vaultID, itemID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := c.client.Items().Delete(ctx, vaultID, itemID)
	if err != nil {
		return fmt.Errorf("failed to delete item %s from vault %s: %w", itemID, vaultID, err)
	}

	return nil
}

// Close releases resources held by the SDK client.
func (c *SDKClient) Close() error {
	// The SDK client doesn't have a Close method in the current version,
	// but we implement this for future compatibility and interface consistency.
	return nil
}

// convertSDKItem converts SDK item to our simplified Item struct.
func (c *SDKClient) convertSDKItem(sdkItem *onepassword.Item) Item {
	item := Item{
		ID:       sdkItem.ID,
		Title:    sdkItem.Title,
		VaultID:  sdkItem.VaultID,
		Category: string(sdkItem.Category),
		Tags:     sdkItem.Tags,
		Fields:   make([]ItemField, 0, len(sdkItem.Fields)),
	}

	for _, f := range sdkItem.Fields {
		item.Fields = append(item.Fields, ItemField{
			ID:        f.ID,
			Title:     f.Title,
			SectionID: f.SectionID,
			Value:     f.Value,
			FieldType: string(f.FieldType),
		})
	}

	return item
}

// convertToSDKItem converts our Item to SDK Item format (for Put).
func (c *SDKClient) convertToSDKItem(item *Item) onepassword.Item {
	fields := make([]onepassword.ItemField, 0, len(item.Fields))
	for _, f := range item.Fields {
		fields = append(fields, onepassword.ItemField{
			ID:        f.ID,
			Title:     f.Title,
			SectionID: f.SectionID,
			Value:     f.Value,
			FieldType: onepassword.ItemFieldType(f.FieldType),
		})
	}

	return onepassword.Item{
		ID:       item.ID,
		Title:    item.Title,
		VaultID:  item.VaultID,
		Category: onepassword.ItemCategory(item.Category),
		Tags:     item.Tags,
		Fields:   fields,
	}
}

// convertToCreateParams converts our Item to SDK ItemCreateParams format (for Create).
func (c *SDKClient) convertToCreateParams(item *Item) onepassword.ItemCreateParams {
	fields := make([]onepassword.ItemField, 0, len(item.Fields))
	for _, f := range item.Fields {
		fields = append(fields, onepassword.ItemField{
			ID:        f.ID,
			Title:     f.Title,
			SectionID: f.SectionID,
			Value:     f.Value,
			FieldType: onepassword.ItemFieldType(f.FieldType),
		})
	}

	return onepassword.ItemCreateParams{
		Title:    item.Title,
		VaultID:  item.VaultID,
		Category: onepassword.ItemCategory(item.Category),
		Tags:     item.Tags,
		Fields:   fields,
	}
}
