package onepassword

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommandExecutor abstracts command execution for testability.
type CommandExecutor interface {
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// defaultExecutor implements CommandExecutor using os/exec.
type defaultExecutor struct{}

func (e *defaultExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "OP_BIOMETRIC_UNLOCK_ENABLED=true")
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return stdout, exitErr.Stderr, err
		}
		return stdout, nil, err
	}
	return stdout, nil, nil
}

// CLIClient implements the Client interface using the op CLI.
type CLIClient struct {
	opPath   string
	account  string // 1Password account identifier (e.g. "my.1password.com")
	executor CommandExecutor
}

// NewCLIClient creates a new CLI-based 1Password client.
// It resolves the op binary location and verifies it exists.
func NewCLIClient() (*CLIClient, error) {
	opPath, err := exec.LookPath("op")
	if err != nil {
		return nil, fmt.Errorf("op CLI not found in PATH: %w", err)
	}

	return &CLIClient{
		opPath:   opPath,
		executor: &defaultExecutor{},
	}, nil
}

// NewCLIClientWithAccount creates a new CLI-based 1Password client with an account identifier.
// The account is prepended as --account to all op commands, ensuring the correct
// 1Password account is used (matches the raycast-1password-extension approach).
func NewCLIClientWithAccount(account string) (*CLIClient, error) {
	opPath, err := exec.LookPath("op")
	if err != nil {
		return nil, fmt.Errorf("op CLI not found in PATH: %w", err)
	}

	return &CLIClient{
		opPath:   opPath,
		account:  account,
		executor: &defaultExecutor{},
	}, nil
}

// runOP executes an op command with the given arguments and returns the stdout.
// When an account is configured, --account is prepended to all commands.
func (c *CLIClient) runOP(ctx context.Context, args ...string) ([]byte, error) {
	if c.account != "" {
		args = append([]string{"--account", c.account}, args...)
	}
	stdout, stderr, err := c.executor.Run(ctx, c.opPath, args...)
	if err != nil {
		// Include stderr in error message for debugging
		if len(stderr) > 0 {
			return nil, fmt.Errorf("op command failed: %w (stderr: %s)", err, string(stderr))
		}
		return nil, fmt.Errorf("op command failed: %w", err)
	}
	return stdout, nil
}

// ListVaults retrieves all accessible vaults using the op CLI.
func (c *CLIClient) ListVaults(ctx context.Context) ([]Vault, error) {
	output, err := c.runOP(ctx, "vault", "list", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to list vaults: %w", err)
	}

	var cliVaults []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.Unmarshal(output, &cliVaults); err != nil {
		return nil, fmt.Errorf("failed to parse vault list response: %w", err)
	}

	vaults := make([]Vault, 0, len(cliVaults))
	for _, v := range cliVaults {
		vaults = append(vaults, Vault{
			ID:   v.ID,
			Name: v.Name,
		})
	}

	return vaults, nil
}

// ListItems retrieves all items in a vault using the op CLI.
// Note: This uses the same N+1 pattern as the SDK client for consistency.
func (c *CLIClient) ListItems(ctx context.Context, vaultID string) ([]Item, error) {
	output, err := c.runOP(ctx, "item", "list", "--vault", vaultID, "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to list items in vault %s: %w", vaultID, err)
	}

	var itemOverviews []struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(output, &itemOverviews); err != nil {
		return nil, fmt.Errorf("failed to parse item list response: %w", err)
	}

	// Fetch full details for each item (N+1 pattern like SDK)
	items := make([]Item, 0, len(itemOverviews))
	for _, overview := range itemOverviews {
		item, err := c.GetItem(ctx, vaultID, overview.ID)
		if err != nil {
			// Skip items we can't fetch (might be deleted concurrently)
			continue
		}
		items = append(items, *item)
	}

	return items, nil
}

// GetItem retrieves a specific item by ID using the op CLI.
func (c *CLIClient) GetItem(ctx context.Context, vaultID, itemID string) (*Item, error) {
	output, err := c.runOP(ctx, "item", "get", itemID, "--vault", vaultID, "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to get item %s from vault %s: %w", itemID, vaultID, err)
	}

	var cliItem struct {
		ID       string   `json:"id"`
		Title    string   `json:"title"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
		Vault    struct {
			ID string `json:"id"`
		} `json:"vault"`
		Fields []struct {
			ID      string  `json:"id"`
			Label   string  `json:"label"`
			Type    string  `json:"type"`
			Value   string  `json:"value"`
			Section *struct {
				ID string `json:"id"`
			} `json:"section"`
		} `json:"fields"`
	}

	if err := json.Unmarshal(output, &cliItem); err != nil {
		return nil, fmt.Errorf("failed to parse item response: %w", err)
	}

	// Map CLI response to our Item structure
	item := &Item{
		ID:       cliItem.ID,
		Title:    cliItem.Title,
		VaultID:  cliItem.Vault.ID,
		Category: strings.ToLower(cliItem.Category), // CLI returns uppercase like "SERVER"
		Tags:     cliItem.Tags,
		Fields:   make([]ItemField, 0, len(cliItem.Fields)),
	}

	// Map fields with proper field name mapping
	for _, f := range cliItem.Fields {
		var sectionID *string
		if f.Section != nil {
			sectionID = &f.Section.ID
		}

		field := ItemField{
			ID:        f.ID,
			Title:     f.Label, // CLI uses "label" instead of "title"
			SectionID: sectionID,
			Value:     f.Value,
			FieldType: mapCLIFieldType(f.Type), // Map CLI type to our type
		}
		item.Fields = append(item.Fields, field)
	}

	return item, nil
}

// mapCLIFieldType maps CLI field types to our internal field types.
func mapCLIFieldType(cliType string) string {
	// CLI uses different type names than SDK
	switch strings.ToLower(cliType) {
	case "concealed", "password":
		return "Concealed"
	case "string", "text":
		return "Text"
	default:
		return "Text" // Default to Text for unknown types
	}
}

// CreateItem creates a new item in 1Password using the op CLI.
func (c *CLIClient) CreateItem(ctx context.Context, item *Item) (*Item, error) {
	if item == nil {
		return nil, fmt.Errorf("item cannot be nil")
	}

	// Build base command
	args := []string{
		"item", "create",
		"--category", item.Category,
		"--vault", item.VaultID,
		"--title", item.Title,
	}

	// Add tags if present
	if len(item.Tags) > 0 {
		args = append(args, "--tags", strings.Join(item.Tags, ","))
	}

	// Add fields as key=value pairs after the -- separator
	if len(item.Fields) > 0 {
		args = append(args, "--")
		for _, field := range item.Fields {
			fieldArg := fmt.Sprintf("%s=%s", field.Title, field.Value)
			args = append(args, fieldArg)
		}
	}

	args = append(args, "--format", "json")

	output, err := c.runOP(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	// Parse the created item response
	var cliItem struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(output, &cliItem); err != nil {
		return nil, fmt.Errorf("failed to parse create item response: %w", err)
	}

	// Fetch the full item details to return
	return c.GetItem(ctx, item.VaultID, cliItem.ID)
}

// UpdateItem updates an existing item in 1Password using the op CLI.
func (c *CLIClient) UpdateItem(ctx context.Context, item *Item) (*Item, error) {
	if item == nil {
		return nil, fmt.Errorf("item cannot be nil")
	}
	if item.ID == "" {
		return nil, fmt.Errorf("item ID is required for update")
	}

	// Build base command
	args := []string{
		"item", "edit", item.ID,
	}

	// Update title if provided
	if item.Title != "" {
		args = append(args, "--title", item.Title)
	}

	// Update fields as key=value pairs after the -- separator
	if len(item.Fields) > 0 {
		args = append(args, "--")
		for _, field := range item.Fields {
			fieldArg := fmt.Sprintf("%s=%s", field.Title, field.Value)
			args = append(args, fieldArg)
		}
	}

	_, err := c.runOP(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update item %s: %w", item.ID, err)
	}

	// Fetch the updated item to return
	return c.GetItem(ctx, item.VaultID, item.ID)
}

// DeleteItem deletes an item from 1Password using the op CLI.
func (c *CLIClient) DeleteItem(ctx context.Context, vaultID, itemID string) error {
	_, err := c.runOP(ctx, "item", "delete", itemID, "--vault", vaultID)
	if err != nil {
		return fmt.Errorf("failed to delete item %s from vault %s: %w", itemID, vaultID, err)
	}

	return nil
}

// Close releases resources held by the CLI client.
// No-op for CLI client as there are no persistent connections.
func (c *CLIClient) Close() error {
	return nil
}
