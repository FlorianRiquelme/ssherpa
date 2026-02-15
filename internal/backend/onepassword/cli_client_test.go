package onepassword

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

// mockExecutor implements CommandExecutor for testing.
type mockExecutor struct {
	responses map[string]mockResponse
	callCount map[string]int
}

type mockResponse struct {
	stdout []byte
	stderr []byte
	err    error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		responses: make(map[string]mockResponse),
		callCount: make(map[string]int),
	}
}

func (m *mockExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	key := name + " " + concatenateArgs(args)
	m.callCount[key]++

	if resp, ok := m.responses[key]; ok {
		return resp.stdout, resp.stderr, resp.err
	}

	return nil, nil, errors.New("no mock response configured for: " + key)
}

func (m *mockExecutor) setResponse(name string, args []string, stdout, stderr []byte, err error) {
	key := name + " " + concatenateArgs(args)
	m.responses[key] = mockResponse{
		stdout: stdout,
		stderr: stderr,
		err:    err,
	}
}

func (m *mockExecutor) getCallCount(name string, args []string) int {
	key := name + " " + concatenateArgs(args)
	return m.callCount[key]
}

func concatenateArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}

func TestNewCLIClient(t *testing.T) {
	// This test will fail if 'op' is not in PATH
	// In a real environment, you'd use build tags or skip this
	client, err := NewCLIClient()
	if err != nil {
		t.Skipf("op CLI not available: %v", err)
	}

	if client.opPath == "" {
		t.Error("expected opPath to be set")
	}
	if client.executor == nil {
		t.Error("expected executor to be set")
	}
}

func TestListVaults(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantErr   bool
		wantCount int
	}{
		{
			name: "successful list",
			response: `[
				{"id": "vault1", "name": "Personal"},
				{"id": "vault2", "name": "Work"}
			]`,
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:      "empty vault list",
			response:  `[]`,
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:     "invalid json",
			response: `{invalid}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockExecutor()
			client := &CLIClient{
				opPath:   "op",
				executor: mock,
			}

			var mockErr error
			if tt.wantErr && tt.response == `{invalid}` {
				mockErr = nil // JSON parsing will fail
			}

			mock.setResponse("op", []string{"vault", "list", "--format", "json"}, []byte(tt.response), nil, mockErr)

			vaults, err := client.ListVaults(context.Background())

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(vaults) != tt.wantCount {
				t.Errorf("expected %d vaults, got %d", tt.wantCount, len(vaults))
			}
		})
	}
}

func TestListItems(t *testing.T) {
	tests := []struct {
		name           string
		vaultID        string
		listResponse   string
		getItemResp    string
		wantErr        bool
		wantCount      int
		expectGetCalls int
	}{
		{
			name:    "successful list with items",
			vaultID: "vault1",
			listResponse: `[
				{"id": "item1"},
				{"id": "item2"}
			]`,
			getItemResp: `{
				"id": "item1",
				"title": "Server 1",
				"category": "SERVER",
				"tags": ["ssherpa"],
				"vault": {"id": "vault1"},
				"fields": [
					{"id": "f1", "label": "hostname", "type": "STRING", "value": "example.com"}
				]
			}`,
			wantErr:        false,
			wantCount:      2,
			expectGetCalls: 2,
		},
		{
			name:         "empty item list",
			vaultID:      "vault1",
			listResponse: `[]`,
			wantErr:      false,
			wantCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockExecutor()
			client := &CLIClient{
				opPath:   "op",
				executor: mock,
			}

			mock.setResponse("op", []string{"item", "list", "--vault", tt.vaultID, "--format", "json"}, []byte(tt.listResponse), nil, nil)

			if tt.expectGetCalls > 0 {
				mock.setResponse("op", []string{"item", "get", "item1", "--vault", tt.vaultID, "--format", "json"}, []byte(tt.getItemResp), nil, nil)
				mock.setResponse("op", []string{"item", "get", "item2", "--vault", tt.vaultID, "--format", "json"}, []byte(tt.getItemResp), nil, nil)
			}

			items, err := client.ListItems(context.Background(), tt.vaultID)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(items) != tt.wantCount {
				t.Errorf("expected %d items, got %d", tt.wantCount, len(items))
			}
		})
	}
}

func TestGetItem(t *testing.T) {
	tests := []struct {
		name       string
		vaultID    string
		itemID     string
		response   string
		wantErr    bool
		wantTitle  string
		wantFields int
	}{
		{
			name:    "successful get",
			vaultID: "vault1",
			itemID:  "item1",
			response: `{
				"id": "item1",
				"title": "Test Server",
				"category": "SERVER",
				"tags": ["ssherpa", "prod"],
				"vault": {"id": "vault1"},
				"fields": [
					{"id": "f1", "label": "hostname", "type": "STRING", "value": "example.com", "section": {"id": "s1"}},
					{"id": "f2", "label": "user", "type": "STRING", "value": "root"},
					{"id": "f3", "label": "password", "type": "CONCEALED", "value": "secret123"}
				]
			}`,
			wantErr:    false,
			wantTitle:  "Test Server",
			wantFields: 3,
		},
		{
			name:     "invalid json",
			vaultID:  "vault1",
			itemID:   "item1",
			response: `{invalid}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockExecutor()
			client := &CLIClient{
				opPath:   "op",
				executor: mock,
			}

			mock.setResponse("op", []string{"item", "get", tt.itemID, "--vault", tt.vaultID, "--format", "json"}, []byte(tt.response), nil, nil)

			item, err := client.GetItem(context.Background(), tt.vaultID, tt.itemID)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr {
				if item.Title != tt.wantTitle {
					t.Errorf("expected title %q, got %q", tt.wantTitle, item.Title)
				}
				if len(item.Fields) != tt.wantFields {
					t.Errorf("expected %d fields, got %d", tt.wantFields, len(item.Fields))
				}
				// Verify category is lowercased
				if item.Category != "server" {
					t.Errorf("expected category 'server', got %q", item.Category)
				}
				// Verify field type mapping
				if len(item.Fields) >= 3 {
					if item.Fields[2].FieldType != "Concealed" {
						t.Errorf("expected field type 'Concealed', got %q", item.Fields[2].FieldType)
					}
				}
			}
		})
	}
}

func TestCreateItem(t *testing.T) {
	tests := []struct {
		name         string
		item         *Item
		createResp   string
		getResp      string
		wantErr      bool
		expectFields []string
	}{
		{
			name: "successful create",
			item: &Item{
				VaultID:  "vault1",
				Title:    "New Server",
				Category: "server",
				Tags:     []string{"ssherpa"},
				Fields: []ItemField{
					{Title: "hostname", Value: "example.com"},
					{Title: "user", Value: "root"},
				},
			},
			createResp: `{"id": "newitem1"}`,
			getResp: `{
				"id": "newitem1",
				"title": "New Server",
				"category": "SERVER",
				"tags": ["ssherpa"],
				"vault": {"id": "vault1"},
				"fields": [
					{"id": "f1", "label": "hostname", "type": "STRING", "value": "example.com"},
					{"id": "f2", "label": "user", "type": "STRING", "value": "root"}
				]
			}`,
			wantErr:      false,
			expectFields: []string{"hostname=example.com", "user=root"},
		},
		{
			name:    "nil item",
			item:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockExecutor()
			client := &CLIClient{
				opPath:   "op",
				executor: mock,
			}

			if tt.item != nil {
				// Setup create response
				createArgs := []string{
					"item", "create",
					"--category", tt.item.Category,
					"--vault", tt.item.VaultID,
					"--title", tt.item.Title,
				}
				if len(tt.item.Tags) > 0 {
					createArgs = append(createArgs, "--tags", "ssherpa")
				}
				if len(tt.item.Fields) > 0 {
					createArgs = append(createArgs, "--")
					createArgs = append(createArgs, tt.expectFields...)
				}
				createArgs = append(createArgs, "--format", "json")

				mock.setResponse("op", createArgs, []byte(tt.createResp), nil, nil)

				// Setup get response
				mock.setResponse("op", []string{"item", "get", "newitem1", "--vault", tt.item.VaultID, "--format", "json"}, []byte(tt.getResp), nil, nil)
			}

			result, err := client.CreateItem(context.Background(), tt.item)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && result == nil {
				t.Error("expected result but got nil")
			}
		})
	}
}

func TestUpdateItem(t *testing.T) {
	tests := []struct {
		name    string
		item    *Item
		getResp string
		wantErr bool
	}{
		{
			name: "successful update",
			item: &Item{
				ID:      "item1",
				VaultID: "vault1",
				Title:   "Updated Server",
				Fields: []ItemField{
					{Title: "hostname", Value: "updated.example.com"},
				},
			},
			getResp: `{
				"id": "item1",
				"title": "Updated Server",
				"category": "SERVER",
				"vault": {"id": "vault1"},
				"fields": [
					{"id": "f1", "label": "hostname", "type": "STRING", "value": "updated.example.com"}
				]
			}`,
			wantErr: false,
		},
		{
			name:    "nil item",
			item:    nil,
			wantErr: true,
		},
		{
			name: "missing item ID",
			item: &Item{
				VaultID: "vault1",
				Title:   "Server",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockExecutor()
			client := &CLIClient{
				opPath:   "op",
				executor: mock,
			}

			if tt.item != nil && tt.item.ID != "" {
				// Setup update response
				updateArgs := []string{"item", "edit", tt.item.ID}
				if tt.item.Title != "" {
					updateArgs = append(updateArgs, "--title", tt.item.Title)
				}
				if len(tt.item.Fields) > 0 {
					updateArgs = append(updateArgs, "--", "hostname=updated.example.com")
				}

				mock.setResponse("op", updateArgs, []byte(""), nil, nil)

				// Setup get response
				mock.setResponse("op", []string{"item", "get", tt.item.ID, "--vault", tt.item.VaultID, "--format", "json"}, []byte(tt.getResp), nil, nil)
			}

			result, err := client.UpdateItem(context.Background(), tt.item)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantErr && result == nil {
				t.Error("expected result but got nil")
			}
		})
	}
}

func TestDeleteItem(t *testing.T) {
	tests := []struct {
		name     string
		vaultID  string
		itemID   string
		mockErr  error
		wantErr  bool
		stderr   []byte
	}{
		{
			name:    "successful delete",
			vaultID: "vault1",
			itemID:  "item1",
			wantErr: false,
		},
		{
			name:    "delete with error",
			vaultID: "vault1",
			itemID:  "item1",
			mockErr: &exec.ExitError{},
			stderr:  []byte("item not found"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockExecutor()
			client := &CLIClient{
				opPath:   "op",
				executor: mock,
			}

			mock.setResponse("op", []string{"item", "delete", tt.itemID, "--vault", tt.vaultID}, []byte(""), tt.stderr, tt.mockErr)

			err := client.DeleteItem(context.Background(), tt.vaultID, tt.itemID)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestClose(t *testing.T) {
	client := &CLIClient{
		opPath:   "op",
		executor: newMockExecutor(),
	}

	err := client.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got: %v", err)
	}
}

func TestMapCLIFieldType(t *testing.T) {
	tests := []struct {
		cliType  string
		expected string
	}{
		{"CONCEALED", "Concealed"},
		{"concealed", "Concealed"},
		{"PASSWORD", "Concealed"},
		{"STRING", "Text"},
		{"string", "Text"},
		{"TEXT", "Text"},
		{"text", "Text"},
		{"unknown", "Text"},
		{"", "Text"},
	}

	for _, tt := range tests {
		t.Run(tt.cliType, func(t *testing.T) {
			result := mapCLIFieldType(tt.cliType)
			if result != tt.expected {
				t.Errorf("mapCLIFieldType(%q) = %q, expected %q", tt.cliType, result, tt.expected)
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		mockErr  error
		stderr   string
		wantErr  bool
	}{
		{
			name:    "command execution error with stderr",
			method:  "ListVaults",
			mockErr: &exec.ExitError{},
			stderr:  "session expired",
			wantErr: true,
		},
		{
			name:    "command execution error without stderr",
			method:  "ListVaults",
			mockErr: errors.New("command failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockExecutor()
			client := &CLIClient{
				opPath:   "op",
				executor: mock,
			}

			mock.setResponse("op", []string{"vault", "list", "--format", "json"}, nil, []byte(tt.stderr), tt.mockErr)

			_, err := client.ListVaults(context.Background())

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
