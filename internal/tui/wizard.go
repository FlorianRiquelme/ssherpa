package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	op "github.com/florianriquelme/sshjesus/internal/backend/onepassword"
	"github.com/florianriquelme/sshjesus/internal/config"
)

// SetupWizard is a Bubbletea model for the first-launch setup flow.
type SetupWizard struct {
	step          int              // Current step in the wizard
	backendChoice string           // Selected backend: "sshconfig", "onepassword", "both"
	tokenInput    textinput.Model  // 1Password service account token input
	token         string           // Resolved token (from env or manual input)
	tokenFromEnv  bool             // Whether the token came from OP_SERVICE_ACCOUNT_TOKEN
	spinner       spinner.Model    // Loading spinner for async operations
	checking      bool             // Whether we're checking 1Password
	checkResult   onePasswordCheckResult // Result of 1Password detection
	cursor        int              // Cursor position for menu selection
	width         int
	height        int
	configPath    string           // Path to save config
	err           error            // Error message for display
	runMigration  bool             // Whether user wants to run migration
	migrationItemCount int         // Number of items found for migration
}

type onePasswordCheckResult struct {
	available  bool
	vaultCount int
	error      string
}

// Wizard steps
const (
	stepWelcome = iota
	stepTokenInput
	stepCheckingOnePassword
	stepOnePasswordSetup
	stepMigrationOffer
	stepSummary
)

// NewSetupWizard creates a new setup wizard.
func NewSetupWizard(configPath string) SetupWizard {
	// Initialize token input (password-masked)
	tokenInput := textinput.New()
	tokenInput.Placeholder = "ops_..."
	tokenInput.CharLimit = 5000
	tokenInput.Width = 60
	tokenInput.EchoMode = textinput.EchoPassword

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accentColor)

	return SetupWizard{
		step:       stepWelcome,
		tokenInput: tokenInput,
		spinner:    s,
		cursor:     0,
		configPath: configPath,
	}
}

// Init initializes the wizard.
func (w SetupWizard) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (w SetupWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height

	case tea.KeyMsg:
		// Global quit handler — always available regardless of step
		if msg.String() == "ctrl+c" {
			return w, tea.Quit
		}

		switch w.step {
		case stepWelcome:
			return w.updateWelcome(msg)

		case stepTokenInput:
			return w.updateTokenInput(msg)

		case stepCheckingOnePassword:
			// No input while checking
			return w, nil

		case stepOnePasswordSetup:
			return w.updateOnePasswordSetup(msg)

		case stepMigrationOffer:
			return w.updateMigrationOffer(msg)

		case stepSummary:
			// Enter to save config and exit wizard
			if msg.String() == "enter" {
				return w, w.saveConfig()
			}
		}

	case spinner.TickMsg:
		if w.checking {
			var cmd tea.Cmd
			w.spinner, cmd = w.spinner.Update(msg)
			return w, cmd
		}

	case onePasswordCheckCompleteMsg:
		// 1Password check completed
		w.checking = false
		w.checkResult = onePasswordCheckResult{
			available:  msg.available,
			vaultCount: msg.vaultCount,
			error:      msg.error,
		}

		// Advance to the setup screen (shows success or failure)
		w.step = stepOnePasswordSetup
		return w, nil

	case configSavedMsg:
		// Config saved successfully - quit wizard
		return w, tea.Quit

	case configSaveErrorMsg:
		// Config save failed - show error
		w.err = msg.err
		return w, nil
	}

	// Forward non-key messages (cursor blink, etc.) to text input when active
	if w.step == stepTokenInput {
		var cmd tea.Cmd
		w.tokenInput, cmd = w.tokenInput.Update(msg)
		return w, cmd
	}

	return w, nil
}

// updateWelcome handles input for the welcome screen.
func (w SetupWizard) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		w.cursor = (w.cursor + 1) % 3
	case "k", "up":
		w.cursor = (w.cursor - 1 + 3) % 3
	case "enter":
		switch w.cursor {
		case 0:
			w.backendChoice = "sshconfig"
			w.step = stepSummary
		case 1:
			w.backendChoice = "onepassword"
			return w.transitionToTokenStep()
		case 2:
			w.backendChoice = "both"
			return w.transitionToTokenStep()
		}
	case "q":
		return w, tea.Quit
	}
	return w, nil
}

// transitionToTokenStep checks for env var token and transitions accordingly.
func (w SetupWizard) transitionToTokenStep() (tea.Model, tea.Cmd) {
	// Check if token is already set via environment variable
	envToken := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	if envToken != "" {
		// Token found in env — skip input, go straight to checking
		w.token = envToken
		w.tokenFromEnv = true
		w.step = stepCheckingOnePassword
		w.checking = true
		return w, tea.Batch(w.spinner.Tick, checkOnePassword(envToken))
	}

	// No env var — show token input
	w.step = stepTokenInput
	cmd := w.tokenInput.Focus()
	return w, cmd
}

// updateTokenInput handles input for the service account token step.
func (w SetupWizard) updateTokenInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		token := strings.TrimSpace(w.tokenInput.Value())
		if token == "" {
			return w, nil
		}
		w.token = token
		w.tokenFromEnv = false
		w.tokenInput.Blur()
		w.step = stepCheckingOnePassword
		w.checking = true
		return w, tea.Batch(w.spinner.Tick, checkOnePassword(token))
	case "esc":
		w.step = stepWelcome
		w.tokenInput.Blur()
		w.tokenInput.SetValue("")
		return w, nil
	default:
		var cmd tea.Cmd
		w.tokenInput, cmd = w.tokenInput.Update(msg)
		return w, cmd
	}
}

// updateOnePasswordSetup handles input for 1Password setup.
func (w SetupWizard) updateOnePasswordSetup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if w.checkResult.available {
			w.step = stepSummary
		} else {
			// Fall back to SSH config only
			w.backendChoice = "sshconfig"
			w.step = stepSummary
		}
		return w, nil
	case "esc":
		// Go back to token input to retry
		w.step = stepTokenInput
		w.checkResult = onePasswordCheckResult{}
		cmd := w.tokenInput.Focus()
		return w, cmd
	}
	return w, nil
}

// updateMigrationOffer handles input for migration offer screen.
func (w SetupWizard) updateMigrationOffer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		w.cursor = (w.cursor + 1) % 2
	case "k", "up":
		w.cursor = (w.cursor - 1 + 2) % 2
	case "enter":
		w.runMigration = (w.cursor == 0)
		w.step = stepSummary
	case "esc":
		w.runMigration = false
		w.step = stepSummary
	}
	return w, nil
}

// View renders the current step.
func (w SetupWizard) View() string {
	switch w.step {
	case stepWelcome:
		return w.renderWelcome()
	case stepTokenInput:
		return w.renderTokenInput()
	case stepCheckingOnePassword:
		return w.renderCheckingOnePassword()
	case stepOnePasswordSetup:
		return w.renderOnePasswordSetup()
	case stepMigrationOffer:
		return w.renderMigrationOffer()
	case stepSummary:
		return w.renderSummary()
	default:
		return "Unknown step"
	}
}

// renderWelcome renders the backend selection screen.
func (w SetupWizard) renderWelcome() string {
	var b strings.Builder

	title := titleStyle.Render("Welcome to sshjesus!")
	b.WriteString(title + "\n\n")

	b.WriteString("Choose your backend:\n\n")

	options := []string{
		"SSH Config only      Uses ~/.ssh/config (already working)",
		"1Password            Store servers in 1Password for team sharing",
		"Both                 SSH Config + 1Password (recommended for teams)",
	}

	for i, opt := range options {
		cursor := "  "
		if i == w.cursor {
			cursor = "> "
			opt = selectedStyle.Render(opt)
		}
		b.WriteString(cursor + opt + "\n")
	}

	b.WriteString("\n")
	b.WriteString(wizardDimStyle.Render("Use j/k to navigate, Enter to select, q to quit"))

	return wizardBoxStyle.Render(b.String())
}

// renderTokenInput renders the service account token input screen.
func (w SetupWizard) renderTokenInput() string {
	var b strings.Builder

	title := titleStyle.Render("1Password Service Account")
	b.WriteString(title + "\n\n")

	b.WriteString("A service account token is required to connect to 1Password.\n\n")
	b.WriteString("Create one at: https://my.1password.com/developer-tools/infrastructure-secrets/serviceaccount\n\n")
	b.WriteString("Paste your token:\n\n")
	b.WriteString("  " + w.tokenInput.View() + "\n\n")
	b.WriteString(wizardDimStyle.Render("Tip: set OP_SERVICE_ACCOUNT_TOKEN in your shell profile to skip this step.") + "\n\n")
	b.WriteString(wizardDimStyle.Render("Press Enter to continue, Esc to go back"))

	return wizardBoxStyle.Render(b.String())
}

// renderCheckingOnePassword renders the 1Password detection screen.
func (w SetupWizard) renderCheckingOnePassword() string {
	var b strings.Builder

	title := titleStyle.Render("1Password Setup")
	b.WriteString(title + "\n\n")

	b.WriteString(fmt.Sprintf("  %s Connecting to 1Password...\n", w.spinner.View()))

	return wizardBoxStyle.Render(b.String())
}

// renderOnePasswordSetup renders the 1Password configuration screen.
func (w SetupWizard) renderOnePasswordSetup() string {
	var b strings.Builder

	title := titleStyle.Render("1Password Setup")
	b.WriteString(title + "\n\n")

	if w.checkResult.available {
		b.WriteString(wizardSuccessStyle.Render("✓ Connected to 1Password") + "\n\n")
		b.WriteString(fmt.Sprintf("  Found %d vault(s)\n\n", w.checkResult.vaultCount))
		b.WriteString("Press Enter to continue")
	} else {
		b.WriteString(wizardErrorStyle.Render("✗ Could not connect to 1Password") + "\n\n")
		if w.checkResult.error != "" {
			b.WriteString(wizardDimStyle.Render(w.checkResult.error) + "\n\n")
		}
		b.WriteString("Verify that:\n")
		b.WriteString("  1. The service account token is valid\n")
		b.WriteString("  2. The service account has vault access permissions\n\n")
		b.WriteString("Press Enter to use SSH Config only, or Esc to try again")
	}

	return wizardBoxStyle.Render(b.String())
}

// renderMigrationOffer renders the migration offer screen.
func (w SetupWizard) renderMigrationOffer() string {
	var b strings.Builder

	title := titleStyle.Render("Existing Items Found")
	b.WriteString(title + "\n\n")

	b.WriteString(fmt.Sprintf("We found %d SSH/Server items in your 1Password vaults\n", w.migrationItemCount))
	b.WriteString("that are not yet managed by sshjesus.\n\n")

	options := []string{
		"Run migration wizard    Tag existing items for sshjesus",
		"Skip for now            You can migrate later with 'sshjesus migrate'",
	}

	for i, opt := range options {
		cursor := "  "
		if i == w.cursor {
			cursor = "> "
			opt = selectedStyle.Render(opt)
		}
		b.WriteString(cursor + opt + "\n")
	}

	b.WriteString("\n")
	b.WriteString(wizardDimStyle.Render("Use j/k to navigate, Enter to select"))

	return wizardBoxStyle.Render(b.String())
}

// renderSummary renders the setup complete summary.
func (w SetupWizard) renderSummary() string {
	var b strings.Builder

	title := titleStyle.Render("Setup Complete!")
	b.WriteString(title + "\n\n")

	// Show selected backend
	backendName := w.backendChoice
	switch w.backendChoice {
	case "sshconfig":
		backendName = "SSH Config only"
	case "onepassword":
		backendName = "1Password"
	case "both":
		backendName = "SSH Config + 1Password"
	}
	b.WriteString(fmt.Sprintf("Backend: %s\n", wizardSuccessStyle.Render(backendName)))

	// Show 1Password details if applicable
	if w.backendChoice == "onepassword" || w.backendChoice == "both" {
		if w.checkResult.vaultCount > 0 {
			b.WriteString(fmt.Sprintf("Vaults:  %d\n", w.checkResult.vaultCount))
		}
		if !w.tokenFromEnv {
			b.WriteString("\n")
			b.WriteString(wizardErrorStyle.Render("Important:") + " Add this to your shell profile to persist the token:\n")
			b.WriteString(wizardDimStyle.Render("  export OP_SERVICE_ACCOUNT_TOKEN=<your-token>") + "\n")
		}
		if w.runMigration {
			b.WriteString(fmt.Sprintf("Migrated: %d items\n", w.migrationItemCount))
		}
	}

	b.WriteString("\n")
	configPath := w.configPath
	if configPath == "" {
		configPath = "~/.config/sshjesus/config.toml"
	}
	b.WriteString(fmt.Sprintf("Config path: %s\n", wizardDimStyle.Render(configPath)))

	b.WriteString("\n")
	if w.err != nil {
		b.WriteString(wizardErrorStyle.Render(fmt.Sprintf("Error saving config: %s", w.err.Error())) + "\n")
		b.WriteString(wizardDimStyle.Render("Press Enter to retry"))
	} else {
		b.WriteString(wizardDimStyle.Render("Press Enter to start sshjesus"))
	}

	return wizardBoxStyle.Render(b.String())
}

// Styles for wizard (reuse existing + add wizard-specific)
var (
	wizardDimStyle     = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#999999", Dark: "#666666"})
	wizardBoxStyle     = lipgloss.NewStyle().Padding(2, 4).Border(lipgloss.RoundedBorder()).BorderForeground(accentColor)
	wizardSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	wizardErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
)

// checkOnePassword verifies connectivity to 1Password using a service account token.
func checkOnePassword(token string) tea.Cmd {
	return func() tea.Msg {
		client, err := op.NewServiceAccountClient(token)
		if err != nil {
			return onePasswordCheckCompleteMsg{
				available:  false,
				vaultCount: 0,
				error:      err.Error(),
			}
		}
		defer client.Close()

		vaults, err := client.ListVaults(context.Background())
		if err != nil {
			return onePasswordCheckCompleteMsg{
				available:  false,
				vaultCount: 0,
				error:      err.Error(),
			}
		}

		return onePasswordCheckCompleteMsg{
			available:  true,
			vaultCount: len(vaults),
			error:      "",
		}
	}
}

// onePasswordCheckCompleteMsg is sent when 1Password check completes.
type onePasswordCheckCompleteMsg struct {
	available  bool
	vaultCount int
	error      string
}

// configSavedMsg is sent when config is saved successfully.
type configSavedMsg struct{}

// configSaveErrorMsg is sent when config save fails.
type configSaveErrorMsg struct {
	err error
}

// saveConfig saves the wizard configuration to disk.
func (w SetupWizard) saveConfig() tea.Cmd {
	return func() tea.Msg {
		cfg := config.DefaultConfig()
		cfg.Backend = w.backendChoice

		if w.backendChoice == "onepassword" || w.backendChoice == "both" {
			cfg.MigrationDone = w.runMigration
		}

		err := config.Save(cfg, w.configPath)
		if err != nil {
			return configSaveErrorMsg{err: err}
		}

		return configSavedMsg{}
	}
}
