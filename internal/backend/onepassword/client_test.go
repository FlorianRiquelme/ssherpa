package onepassword

import (
	"context"
	"fmt"
	"sync"
)

// MockClient is an in-memory implementation of Client for testing.
type MockClient struct {
	mu      sync.RWMutex
	vaults  map[string]*Vault
	items   map[string]*Item // keyed by itemID
	errors  map[string]error  // configurable errors by operation
	closed  bool
}

// NewMockClient creates a new MockClient with empty storage.
func NewMockClient() *MockClient {
	return &MockClient{
		vaults: make(map[string]*Vault),
		items:  make(map[string]*Item),
		errors: make(map[string]error),
	}
}

// AddVault adds a vault to the mock storage.
func (m *MockClient) AddVault(vault Vault) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vaults[vault.ID] = &vault
}

// AddItem adds an item to the mock storage.
func (m *MockClient) AddItem(item Item) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[item.ID] = &item
}

// SetError configures an error to be returned by a specific operation.
// Operations: "ListVaults", "ListItems", "GetItem", "CreateItem", "UpdateItem", "DeleteItem"
func (m *MockClient) SetError(operation string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[operation] = err
}

// ClearError removes a configured error for an operation.
func (m *MockClient) ClearError(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.errors, operation)
}

func (m *MockClient) checkError(operation string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return fmt.Errorf("client is closed")
	}
	if err, ok := m.errors[operation]; ok {
		return err
	}
	return nil
}

func (m *MockClient) ListVaults(ctx context.Context) ([]Vault, error) {
	if err := m.checkError("ListVaults"); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	vaults := make([]Vault, 0, len(m.vaults))
	for _, v := range m.vaults {
		vaults = append(vaults, *v)
	}
	return vaults, nil
}

func (m *MockClient) ListItems(ctx context.Context, vaultID string) ([]Item, error) {
	if err := m.checkError("ListItems"); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check vault exists
	if _, ok := m.vaults[vaultID]; !ok {
		return nil, fmt.Errorf("vault not found: %s", vaultID)
	}

	items := make([]Item, 0)
	for _, item := range m.items {
		if item.VaultID == vaultID {
			items = append(items, *item)
		}
	}
	return items, nil
}

func (m *MockClient) GetItem(ctx context.Context, vaultID, itemID string) (*Item, error) {
	if err := m.checkError("GetItem"); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.items[itemID]
	if !ok {
		return nil, fmt.Errorf("item not found: %s", itemID)
	}
	if item.VaultID != vaultID {
		return nil, fmt.Errorf("item %s not in vault %s", itemID, vaultID)
	}

	// Return copy
	itemCopy := *item
	itemCopy.Fields = make([]ItemField, len(item.Fields))
	copy(itemCopy.Fields, item.Fields)
	itemCopy.Tags = make([]string, len(item.Tags))
	copy(itemCopy.Tags, item.Tags)

	return &itemCopy, nil
}

func (m *MockClient) CreateItem(ctx context.Context, item *Item) (*Item, error) {
	if err := m.checkError("CreateItem"); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check vault exists
	if _, ok := m.vaults[item.VaultID]; !ok {
		return nil, fmt.Errorf("vault not found: %s", item.VaultID)
	}

	// Check for duplicate ID
	if _, exists := m.items[item.ID]; exists {
		return nil, fmt.Errorf("item already exists: %s", item.ID)
	}

	// Store copy
	itemCopy := *item
	itemCopy.Fields = make([]ItemField, len(item.Fields))
	copy(itemCopy.Fields, item.Fields)
	itemCopy.Tags = make([]string, len(item.Tags))
	copy(itemCopy.Tags, item.Tags)

	m.items[item.ID] = &itemCopy

	// Return copy
	result := itemCopy
	result.Fields = make([]ItemField, len(itemCopy.Fields))
	copy(result.Fields, itemCopy.Fields)
	result.Tags = make([]string, len(itemCopy.Tags))
	copy(result.Tags, itemCopy.Tags)

	return &result, nil
}

func (m *MockClient) UpdateItem(ctx context.Context, item *Item) (*Item, error) {
	if err := m.checkError("UpdateItem"); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check item exists
	if _, exists := m.items[item.ID]; !exists {
		return nil, fmt.Errorf("item not found: %s", item.ID)
	}

	// Store copy
	itemCopy := *item
	itemCopy.Fields = make([]ItemField, len(item.Fields))
	copy(itemCopy.Fields, item.Fields)
	itemCopy.Tags = make([]string, len(item.Tags))
	copy(itemCopy.Tags, item.Tags)

	m.items[item.ID] = &itemCopy

	// Return copy
	result := itemCopy
	result.Fields = make([]ItemField, len(itemCopy.Fields))
	copy(result.Fields, itemCopy.Fields)
	result.Tags = make([]string, len(itemCopy.Tags))
	copy(result.Tags, itemCopy.Tags)

	return &result, nil
}

func (m *MockClient) DeleteItem(ctx context.Context, vaultID, itemID string) error {
	if err := m.checkError("DeleteItem"); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.items[itemID]
	if !ok {
		return fmt.Errorf("item not found: %s", itemID)
	}
	if item.VaultID != vaultID {
		return fmt.Errorf("item %s not in vault %s", itemID, vaultID)
	}

	delete(m.items, itemID)
	return nil
}

func (m *MockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}
