package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/sshjesus/internal/config"
)

// SetupWizard is a Bubbletea model for the first-launch setup flow.
type SetupWizard struct {
	step          int              // Current step in the wizard
	backendChoice string           // Selected backend: "sshconfig", "onepassword", "both"
	accountInput  textinput.Model  // 1Password account name input
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
	stepCheckingOnePassword
	stepOnePasswordSetup
	stepMigrationOffer
	stepSummary
)

// NewSetupWizard creates a new setup wizard.
func NewSetupWizard(configPath string) SetupWizard {
	// Initialize account input
	accountInput := textinput.New()
	accountInput.Placeholder = "your-account@1password.com"
	accountInput.CharLimit = 100
	accountInput.Width = 40

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accentColor)

	return SetupWizard{
		step:         stepWelcome,
		accountInput: accountInput,
		spinner:      s,
		cursor:       0, // Default to first option
		configPath:   configPath,
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
		switch w.step {
		case stepWelcome:
			return w.updateWelcome(msg)

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
				// Save config before exiting
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

		if msg.available {
			// Success - move to setup
			return w, nil
		} else {
			// Failed - offer fallback or retry
			return w, nil
		}

	case configSavedMsg:
		// Config saved successfully - quit wizard
		return w, tea.Quit

	case configSaveErrorMsg:
		// Config save failed - show error
		w.err = msg.err
		return w, nil
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
		// User selected a backend
		switch w.cursor {
		case 0:
			w.backendChoice = "sshconfig"
			w.step = stepSummary // Skip to summary, no 1Password setup needed
		case 1:
			w.backendChoice = "onepassword"
			w.step = stepCheckingOnePassword
			w.checking = true
			return w, tea.Batch(w.spinner.Tick, checkOnePassword())
		case 2:
			w.backendChoice = "both"
			w.step = stepCheckingOnePassword
			w.checking = true
			return w, tea.Batch(w.spinner.Tick, checkOnePassword())
		}
	case "q", "ctrl+c":
		return w, tea.Quit
	}
	return w, nil
}

// updateOnePasswordSetup handles input for 1Password setup.
func (w SetupWizard) updateOnePasswordSetup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if !w.checkResult.available {
			// Fall back to SSH config only
			w.backendChoice = "sshconfig"
			w.step = stepSummary
			return w, nil
		}
		// Cancel if 1Password is available (go back to welcome)
		w.step = stepWelcome
		w.accountInput.SetValue("")
		return w, nil
	case "enter":
		if w.checkResult.available {
			// Save account name and proceed to migration offer
			// For now, skip migration offer and go to summary
			w.step = stepSummary
			return w, nil
		} else {
			// Fall back to SSH config only
			w.backendChoice = "sshconfig"
			w.step = stepSummary
			return w, nil
		}
	default:
		// Pass to text input
		var cmd tea.Cmd
		w.accountInput, cmd = w.accountInput.Update(msg)
		return w, cmd
	}
}

// updateMigrationOffer handles input for migration offer screen.
func (w SetupWizard) updateMigrationOffer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		w.cursor = (w.cursor + 1) % 2
	case "k", "up":
		w.cursor = (w.cursor - 1 + 2) % 2
	case "enter":
		// User made choice
		w.runMigration = (w.cursor == 0)
		w.step = stepSummary
	case "esc":
		// Skip migration
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

// renderCheckingOnePassword renders the 1Password detection screen.
func (w SetupWizard) renderCheckingOnePassword() string {
	var b strings.Builder

	title := titleStyle.Render("1Password Setup")
	b.WriteString(title + "\n\n")

	b.WriteString(fmt.Sprintf("  %s Detecting 1Password...\n", w.spinner.View()))

	return wizardBoxStyle.Render(b.String())
}

// renderOnePasswordSetup renders the 1Password configuration screen.
func (w SetupWizard) renderOnePasswordSetup() string {
	var b strings.Builder

	title := titleStyle.Render("1Password Setup")
	b.WriteString(title + "\n\n")

	if w.checkResult.available {
		// 1Password detected successfully
		b.WriteString(wizardSuccessStyle.Render("✓ 1Password desktop app found") + "\n\n")
		b.WriteString(fmt.Sprintf("  Found %d vault(s)\n\n", w.checkResult.vaultCount))
		b.WriteString("Press Enter to continue")
	} else {
		// 1Password not detected
		b.WriteString(wizardErrorStyle.Render("✗ 1Password desktop app not found") + "\n\n")
		b.WriteString("To use the 1Password backend:\n")
		b.WriteString("  1. Install 1Password 8 from https://1password.com\n")
		b.WriteString("  2. Enable the desktop app integration in 1Password settings\n")
		b.WriteString("  3. Re-run sshjesus\n\n")
		b.WriteString("Press Enter to use SSH Config only, or Esc to quit")
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
		if w.accountInput.Value() != "" {
			b.WriteString(fmt.Sprintf("Account: %s\n", w.accountInput.Value()))
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
	b.WriteString(fmt.Sprintf("Config saved to: %s\n", wizardDimStyle.Render(configPath)))

	b.WriteString("\n")
	b.WriteString(wizardDimStyle.Render("Press Enter to start sshjesus"))

	return wizardBoxStyle.Render(b.String())
}

// Styles for wizard (reuse existing + add wizard-specific)
var (
	wizardDimStyle     = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#999999", Dark: "#666666"})
	wizardBoxStyle     = lipgloss.NewStyle().Padding(2, 4).Border(lipgloss.RoundedBorder()).BorderForeground(accentColor)
	wizardSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	wizardErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
)

// checkOnePassword checks if 1Password desktop app is available.
func checkOnePassword() tea.Cmd {
	return func() tea.Msg {
		// TODO: In Task 2, implement actual 1Password detection using the client
		// For now, simulate detection failure (no client integrated yet)
		return onePasswordCheckCompleteMsg{
			available:  false,
			vaultCount: 0,
			error:      "1Password desktop app not detected (simulated)",
		}
	}
}

// onePasswordCheckCompleteMsg is sent when 1Password check completes.
type onePasswordCheckCompleteMsg struct {
	available  bool
	vaultCount int
	error      string
}

// wizardCompleteMsg is sent when the wizard completes successfully.
type wizardCompleteMsg struct {
	backendChoice string
	accountName   string
	runMigration  bool
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
		// Create config with selected backend
		cfg := config.DefaultConfig()
		cfg.Backend = w.backendChoice

		// Add 1Password settings if applicable
		if w.backendChoice == "onepassword" || w.backendChoice == "both" {
			cfg.OnePassword.AccountName = w.accountInput.Value()
			cfg.MigrationDone = w.runMigration // Mark migration as done if ran
		}

		// Save config
		err := config.Save(cfg, w.configPath)
		if err != nil {
			return configSaveErrorMsg{err: err}
		}

		return configSavedMsg{}
	}
}
