package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/domain"
	"github.com/florianriquelme/ssherpa/internal/sshconfig"
	"github.com/florianriquelme/ssherpa/internal/sshkey"
)

// FormMode represents whether we're adding or editing a host.
type FormMode int

const (
	FormAdd FormMode = iota
	FormEdit
)

// ServerForm is a full-screen form for adding/editing SSH connections.
type ServerForm struct {
	mode          FormMode
	fields        []formField
	focusIndex    int
	configPath    string
	originalAlias string // For edit mode: original alias to find block
	saving        bool   // True while DNS check or save in progress
	saveError     string // Error from save attempt
	dnsError      string // Error from DNS check (non-blocking warning)
	spinner       spinner.Model
	backendWriter backend.Writer // Optional: if set, routes writes through backend instead of sshconfig
	originalID    string         // For backend edit mode: original server ID
	selectedKey   *sshkey.SSHKey // Currently selected SSH key (nil = None)
}

// formField represents a single field in the form.
type formField struct {
	label      string
	input      textinput.Model
	textarea   textarea.Model
	isTextarea bool
	required   bool
	validator  func(string) string
	errorMsg   string
}

// NewServerForm creates a new form in add mode with empty fields.
func NewServerForm(configPath string) ServerForm {
	fields := make([]formField, 6)

	// Alias field
	aliasInput := textinput.New()
	aliasInput.Placeholder = "e.g. my-server"
	aliasInput.Focus()
	fields[0] = formField{
		label:     "Alias",
		input:     aliasInput,
		required:  true,
		validator: validateAlias,
	}

	// Hostname field
	hostnameInput := textinput.New()
	hostnameInput.Placeholder = "e.g. 192.168.1.100 or server.example.com"
	fields[1] = formField{
		label:     "Hostname",
		input:     hostnameInput,
		required:  true,
		validator: validateHostname,
	}

	// User field
	userInput := textinput.New()
	userInput.Placeholder = "e.g. root"
	fields[2] = formField{
		label:     "User",
		input:     userInput,
		required:  true,
		validator: validateUser,
	}

	// Port field
	portInput := textinput.New()
	portInput.Placeholder = "22 (default)"
	portInput.CharLimit = 5
	fields[3] = formField{
		label:     "Port",
		input:     portInput,
		required:  false,
		validator: validatePort,
	}

	// IdentityFile field (read-only display, opens picker on Enter/Space)
	identityInput := textinput.New()
	identityInput.Placeholder = "None (SSH default) - Press Enter to select key"
	fields[4] = formField{
		label:     "IdentityFile",
		input:     identityInput,
		required:  false,
		validator: nil, // No validation needed, this is display-only
	}

	// Extra Config field (textarea)
	extraTextarea := textarea.New()
	extraTextarea.Placeholder = "Additional SSH directives (e.g. ProxyJump bastion)"
	extraTextarea.SetHeight(3)
	extraTextarea.ShowLineNumbers = false
	fields[5] = formField{
		label:      "Extra Config",
		textarea:   extraTextarea,
		isTextarea: true,
		required:   false,
	}

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accentColor)

	return ServerForm{
		mode:       FormAdd,
		fields:     fields,
		focusIndex: 0,
		configPath: configPath,
		spinner:    s,
	}
}

// NewEditServerForm creates a new form in edit mode, pre-filled with host data.
func NewEditServerForm(configPath string, host sshconfig.SSHHost) ServerForm {
	form := NewServerForm(configPath)
	form.mode = FormEdit
	form.originalAlias = host.Name

	// Pre-fill Alias
	form.fields[0].input.SetValue(host.Name)

	// Pre-fill Hostname
	form.fields[1].input.SetValue(host.Hostname)

	// Pre-fill User
	form.fields[2].input.SetValue(host.User)

	// Pre-fill Port (only if not default 22)
	if host.Port != "" && host.Port != "22" {
		form.fields[3].input.SetValue(host.Port)
	}

	// Pre-fill IdentityFile (first one)
	if len(host.IdentityFile) > 0 {
		form.fields[4].input.SetValue(host.IdentityFile[0])
	}

	// Pre-fill ExtraConfig (all other options)
	extraLines := buildExtraConfig(host)
	if extraLines != "" {
		form.fields[5].textarea.SetValue(extraLines)
	}

	return form
}

// buildExtraConfig extracts non-standard SSH options from AllOptions.
func buildExtraConfig(host sshconfig.SSHHost) string {
	var lines []string
	standardKeys := map[string]bool{
		"HostName":     true,
		"User":         true,
		"Port":         true,
		"IdentityFile": true,
	}

	for key, values := range host.AllOptions {
		if standardKeys[key] {
			continue
		}
		for _, val := range values {
			lines = append(lines, fmt.Sprintf("%s %s", key, val))
		}
	}

	return strings.Join(lines, "\n")
}

// Update handles form messages.
func (f ServerForm) Update(msg tea.Msg) (ServerForm, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If saving, ignore all keys except force quit
		if f.saving {
			return f, nil
		}

		switch msg.String() {
		case "esc":
			// Cancel form
			return f, func() tea.Msg { return formCancelledMsg{} }

		case "ctrl+s":
			// Save from any field
			return f, f.handleSave()

		case "tab":
			// Move to next field
			f.blurCurrentField()
			f.validateCurrentField()
			f.focusIndex = (f.focusIndex + 1) % len(f.fields)
			f.focusCurrentField()
			return f, nil

		case "shift+tab":
			// Move to previous field
			f.blurCurrentField()
			f.validateCurrentField()
			f.focusIndex = (f.focusIndex - 1 + len(f.fields)) % len(f.fields)
			f.focusCurrentField()
			return f, nil

		case "j":
			// j: move to next field (only if not in textarea)
			if !f.fields[f.focusIndex].isTextarea {
				f.blurCurrentField()
				f.validateCurrentField()
				f.focusIndex = (f.focusIndex + 1) % len(f.fields)
				f.focusCurrentField()
				return f, nil
			}

		case "k":
			// k: move to previous field (only if not in textarea)
			if !f.fields[f.focusIndex].isTextarea {
				f.blurCurrentField()
				f.validateCurrentField()
				f.focusIndex = (f.focusIndex - 1 + len(f.fields)) % len(f.fields)
				f.focusCurrentField()
				return f, nil
			}

		case "enter", " ":
			// Special handling for IdentityFile field (index 4): open key picker
			if f.focusIndex == 4 {
				// Request model to open key picker
				currentKeyPath := f.fields[4].input.Value()
				return f, func() tea.Msg {
					return formRequestKeyPickerMsg{currentKeyPath: currentKeyPath}
				}
			}

			// Enter on last field OR if not in textarea: save
			if msg.String() == "enter" && !f.fields[f.focusIndex].isTextarea {
				return f, f.handleSave()
			}
		}

		// Pass key to focused field (skip IdentityFile field - it's display-only)
		if f.focusIndex == 4 {
			// Don't update IdentityFile input - it's controlled by key selection
			return f, nil
		}

		if f.fields[f.focusIndex].isTextarea {
			var cmd tea.Cmd
			f.fields[f.focusIndex].textarea, cmd = f.fields[f.focusIndex].textarea.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			var cmd tea.Cmd
			f.fields[f.focusIndex].input, cmd = f.fields[f.focusIndex].input.Update(msg)
			cmds = append(cmds, cmd)
		}

	case dnsCheckResultMsg:
		// DNS check completed
		f.saving = false
		if msg.err != nil {
			// DNS failed - show warning but continue with save
			f.dnsError = msg.err.Error()
		} else {
			f.dnsError = ""
		}
		// Proceed with actual save
		return f, f.performSave()

	case spinner.TickMsg:
		if f.saving {
			var cmd tea.Cmd
			f.spinner, cmd = f.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return f, tea.Batch(cmds...)
}

// blurCurrentField removes focus from current field.
func (f *ServerForm) blurCurrentField() {
	if f.fields[f.focusIndex].isTextarea {
		f.fields[f.focusIndex].textarea.Blur()
	} else {
		f.fields[f.focusIndex].input.Blur()
	}
}

// focusCurrentField gives focus to current field.
func (f *ServerForm) focusCurrentField() {
	if f.fields[f.focusIndex].isTextarea {
		f.fields[f.focusIndex].textarea.Focus()
	} else {
		f.fields[f.focusIndex].input.Focus()
	}
}

// validateCurrentField validates the current field and stores error.
func (f *ServerForm) validateCurrentField() {
	field := &f.fields[f.focusIndex]
	if field.validator == nil {
		return
	}

	var value string
	if field.isTextarea {
		value = field.textarea.Value()
	} else {
		value = field.input.Value()
	}

	field.errorMsg = field.validator(value)
}

// validateAllFields validates all fields and returns true if all valid.
func (f *ServerForm) validateAllFields() bool {
	allValid := true
	for i := range f.fields {
		field := &f.fields[i]
		if field.validator == nil {
			continue
		}

		var value string
		if field.isTextarea {
			value = field.textarea.Value()
		} else {
			value = field.input.Value()
		}

		field.errorMsg = field.validator(value)
		if field.errorMsg != "" {
			allValid = false
		}
	}
	return allValid
}

// handleSave initiates the save process (validation + DNS check).
func (f *ServerForm) handleSave() tea.Cmd {
	// Validate all fields
	if !f.validateAllFields() {
		// Find first invalid field and focus it
		for i, field := range f.fields {
			if field.errorMsg != "" {
				f.blurCurrentField()
				f.focusIndex = i
				f.focusCurrentField()
				break
			}
		}
		return nil
	}

	// Start DNS check
	f.saving = true
	f.saveError = ""
	hostname := strings.TrimSpace(f.fields[1].input.Value())
	return tea.Batch(
		dnsCheckCmd(hostname),
		f.spinner.Tick,
	)
}

// performSave writes the entry to SSH config or backend.
func (f *ServerForm) performSave() tea.Cmd {
	// If backend writer is set, route through it
	if f.backendWriter != nil {
		return f.performBackendSave()
	}

	// Build HostEntry from form fields
	entry := sshconfig.HostEntry{
		Alias:        strings.TrimSpace(f.fields[0].input.Value()),
		Hostname:     strings.TrimSpace(f.fields[1].input.Value()),
		User:         strings.TrimSpace(f.fields[2].input.Value()),
		Port:         strings.TrimSpace(f.fields[3].input.Value()),
		IdentityFile: strings.TrimSpace(f.fields[4].input.Value()),
		ExtraConfig:  strings.TrimSpace(f.fields[5].textarea.Value()),
	}

	// Perform add or edit
	var err error
	if f.mode == FormAdd {
		err = sshconfig.AddHost(f.configPath, entry)
	} else {
		err = sshconfig.EditHost(f.configPath, f.originalAlias, entry)
	}

	if err != nil {
		f.saveError = err.Error()
		f.saving = false
		return nil
	}

	// Success - send serverSavedMsg
	return func() tea.Msg {
		return serverSavedMsg{alias: entry.Alias}
	}
}

// performBackendSave writes the entry through the backend writer.
func (f *ServerForm) performBackendSave() tea.Cmd {
	// Build domain.Server from form fields
	alias := strings.TrimSpace(f.fields[0].input.Value())
	hostname := strings.TrimSpace(f.fields[1].input.Value())
	user := strings.TrimSpace(f.fields[2].input.Value())
	portStr := strings.TrimSpace(f.fields[3].input.Value())
	identityFile := strings.TrimSpace(f.fields[4].input.Value())

	// Parse port (default 22)
	port := 22
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p <= 65535 {
			port = p
		}
	}

	server := &domain.Server{
		ID:           alias, // For new servers, ID = DisplayName
		DisplayName:  alias,
		Host:         hostname,
		User:         user,
		Port:         port,
		IdentityFile: identityFile,
		Tags:         []string{},
	}

	// Perform add or edit
	ctx := context.Background()
	var err error
	if f.mode == FormAdd {
		err = f.backendWriter.CreateServer(ctx, server)
	} else {
		// For edit mode, use originalID if set, otherwise use alias
		if f.originalID != "" {
			server.ID = f.originalID
		}
		err = f.backendWriter.UpdateServer(ctx, server)
	}

	if err != nil {
		f.saveError = err.Error()
		f.saving = false
		return nil
	}

	// Success - send BackendServersUpdatedMsg to trigger reload
	return func() tea.Msg {
		return BackendServersUpdatedMsg{}
	}
}

// View renders the form.
func (f ServerForm) View() string {
	var b strings.Builder

	// Title
	title := "Add SSH Connection"
	if f.mode == FormEdit {
		title = fmt.Sprintf("Edit SSH Connection: %s", f.originalAlias)
	}
	b.WriteString(formTitleStyle.Render(title))
	b.WriteString("\n\n")

	// Render each field
	for i, field := range f.fields {
		// Label with required indicator
		labelText := field.label
		if field.required {
			labelText += " " + formRequiredStyle.Render("*")
		}
		b.WriteString(formLabelStyle.Render(labelText))
		b.WriteString("\n")

		// Input or textarea
		if field.isTextarea {
			b.WriteString(field.textarea.View())
		} else {
			b.WriteString(field.input.View())
		}
		b.WriteString("\n")

		// Error message (if any)
		if field.errorMsg != "" {
			b.WriteString(formErrorStyle.Render("⚠ " + field.errorMsg))
			b.WriteString("\n")
		}

		// DNS warning for hostname field
		if i == 1 && f.dnsError != "" {
			b.WriteString(formDnsWarningStyle.Render("⚠ " + f.dnsError + " (non-blocking)"))
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	// Save error (if any)
	if f.saveError != "" {
		b.WriteString(formErrorStyle.Render("Save failed: " + f.saveError))
		b.WriteString("\n\n")
	}

	// Footer
	if f.saving {
		b.WriteString(formSavingStyle.Render(f.spinner.View() + " Checking hostname..."))
	} else {
		b.WriteString(renderHintRow([]shortcutHint{
			{key: "tab", desc: "next field"},
			{key: "ctrl+s", desc: "save"},
			{key: "esc", desc: "cancel"},
		}))
	}

	return b.String()
}
