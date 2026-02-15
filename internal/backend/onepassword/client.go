package onepassword

import (
	"context"
)

// Client abstracts 1Password operations for testability.
// This interface enables testing with mock implementations without
// depending on real 1Password vaults or CLI tools.
type Client interface {
	ListVaults(ctx context.Context) ([]Vault, error)
	ListItems(ctx context.Context, vaultID string) ([]Item, error)
	GetItem(ctx context.Context, vaultID, itemID string) (*Item, error)
	CreateItem(ctx context.Context, item *Item) (*Item, error)
	UpdateItem(ctx context.Context, item *Item) (*Item, error)
	DeleteItem(ctx context.Context, vaultID, itemID string) error
	Close() error
}

// Vault represents a 1Password vault with minimal fields needed by ssherpa.
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
