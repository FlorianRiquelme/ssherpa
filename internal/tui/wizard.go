package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/ssherpa/internal/config"
)

// SetupWizard is a Bubbletea model for the first-launch setup flow.
type SetupWizard struct {
	step               int                    // Current step in the wizard
	backendChoice      string                 // Selected backend: "sshconfig", "onepassword", "both"
	spinner            spinner.Model          // Loading spinner for async operations
	checking           bool                   // Whether we're checking 1Password CLI
	checkResult        onePasswordCheckResult // Result of 1Password CLI detection
	cursor             int                    // Cursor position for menu selection
	width              int
	height             int
	configPath         string // Path to save config
	err                error  // Error message for display
	runMigration       bool   // Whether user wants to run migration
	migrationItemCount int    // Number of items found for migration

	// Vault selection fields
	vaults           []vaultInfo // Available vaults from 1Password
	selectedVaultIdx int         // Selected vault index in list

	// Sample entry fields
	creatingSample bool   // Whether we're creating a sample entry
	sampleCreated  bool   // Whether sample was successfully created
	sampleError    string // Error message if sample creation failed
}

type vaultInfo struct {
	ID   string
	Name string
}

type onePasswordCheckResult struct {
	available bool
	vaults    []vaultInfo
	error     string
}

// Wizard steps
const (
	stepWelcome = iota
	stepCheckingOnePassword
	stepOnePasswordSetup
	stepVaultSelection
	stepSampleOffer
	stepCreatingSample
	stepMigrationOffer
	stepSummary
)

// NewSetupWizard creates a new setup wizard.
func NewSetupWizard(configPath string) SetupWizard {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accentColor)

	return SetupWizard{
		step:       stepWelcome,
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

		case stepCheckingOnePassword:
			// No input while checking
			return w, nil

		case stepOnePasswordSetup:
			return w.updateOnePasswordSetup(msg)

		case stepVaultSelection:
			return w.updateVaultSelection(msg)

		case stepSampleOffer:
			return w.updateSampleOffer(msg)

		case stepCreatingSample:
			// No input while creating
			return w, nil

		case stepMigrationOffer:
			return w.updateMigrationOffer(msg)

		case stepSummary:
			// Enter to save config and exit wizard
			if msg.String() == "enter" {
				return w, w.saveConfig()
			}
		}

	case spinner.TickMsg:
		if w.checking || w.creatingSample {
			var cmd tea.Cmd
			w.spinner, cmd = w.spinner.Update(msg)
			return w, cmd
		}

	case onePasswordCheckCompleteMsg:
		// 1Password check completed
		w.checking = false
		w.checkResult = onePasswordCheckResult(msg)
		w.vaults = msg.vaults

		// Advance to the setup screen (shows success or failure)
		w.step = stepOnePasswordSetup
		return w, nil

	case sampleEntryCreatedMsg:
		// Sample entry creation completed
		w.creatingSample = false
		if msg.err != "" {
			w.sampleError = msg.err
		} else {
			w.sampleCreated = true
		}
		w.step = stepSummary
		return w, nil

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
		switch w.cursor {
		case 0:
			w.backendChoice = "sshconfig"
			w.step = stepSummary
		case 1:
			w.backendChoice = "onepassword"
			return w.transitionToOpCheck()
		case 2:
			w.backendChoice = "both"
			return w.transitionToOpCheck()
		}
	case "q":
		return w, tea.Quit
	}
	return w, nil
}

// transitionToOpCheck starts the 1Password CLI check flow.
func (w SetupWizard) transitionToOpCheck() (tea.Model, tea.Cmd) {
	w.step = stepCheckingOnePassword
	w.checking = true
	return w, tea.Batch(w.spinner.Tick, checkOpCLI())
}

// updateOnePasswordSetup handles input for 1Password setup.
func (w SetupWizard) updateOnePasswordSetup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if w.checkResult.available && len(w.vaults) > 0 {
			// Proceed to vault selection
			w.step = stepVaultSelection
			w.cursor = 0
		} else if w.checkResult.available {
			// No vaults found, skip to summary
			w.step = stepSummary
		} else {
			// Fall back to SSH config only
			w.backendChoice = "sshconfig"
			w.step = stepSummary
		}
		return w, nil
	case "esc":
		// Go back to welcome to choose different backend
		w.step = stepWelcome
		w.checkResult = onePasswordCheckResult{}
		return w, nil
	}
	return w, nil
}

// updateVaultSelection handles input for vault selection.
func (w SetupWizard) updateVaultSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if len(w.vaults) > 0 {
			w.cursor = (w.cursor + 1) % len(w.vaults)
		}
	case "k", "up":
		if len(w.vaults) > 0 {
			w.cursor = (w.cursor - 1 + len(w.vaults)) % len(w.vaults)
		}
	case "enter":
		if len(w.vaults) > 0 {
			w.selectedVaultIdx = w.cursor
			w.step = stepSampleOffer
			w.cursor = 0
		}
	case "esc":
		w.step = stepOnePasswordSetup
	}
	return w, nil
}

// updateSampleOffer handles input for the sample entry offer.
func (w SetupWizard) updateSampleOffer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		w.cursor = (w.cursor + 1) % 2
	case "k", "up":
		w.cursor = (w.cursor - 1 + 2) % 2
	case "enter":
		if w.cursor == 0 {
			// Create sample entry
			vault := w.vaults[w.selectedVaultIdx]
			w.step = stepCreatingSample
			w.creatingSample = true
			return w, tea.Batch(w.spinner.Tick, createSampleEntry(vault.ID))
		}
		// Skip sample entry
		w.step = stepSummary
	case "esc":
		w.step = stepVaultSelection
		w.cursor = w.selectedVaultIdx
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
	case stepCheckingOnePassword:
		return w.renderCheckingOnePassword()
	case stepOnePasswordSetup:
		return w.renderOnePasswordSetup()
	case stepVaultSelection:
		return w.renderVaultSelection()
	case stepSampleOffer:
		return w.renderSampleOffer()
	case stepCreatingSample:
		return w.renderCreatingSample()
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

	title := titleStyle.Render("Welcome to ssherpa!")
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

// renderCheckingOnePassword renders the 1Password CLI detection screen.
func (w SetupWizard) renderCheckingOnePassword() string {
	var b strings.Builder

	title := titleStyle.Render("1Password Setup")
	b.WriteString(title + "\n\n")

	b.WriteString(fmt.Sprintf("  %s Checking 1Password CLI...\n", w.spinner.View()))

	return wizardBoxStyle.Render(b.String())
}

// renderOnePasswordSetup renders the 1Password CLI configuration screen.
func (w SetupWizard) renderOnePasswordSetup() string {
	var b strings.Builder

	title := titleStyle.Render("1Password Setup")
	b.WriteString(title + "\n\n")

	if w.checkResult.available {
		b.WriteString(wizardSuccessStyle.Render("  1Password CLI is ready") + "\n")
		b.WriteString(fmt.Sprintf("  Found %d vault(s)\n\n", len(w.vaults)))

		// Show entry template
		b.WriteString("  ssherpa stores servers as 1Password items:\n\n")

		templateStyle := lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#555555", Dark: "#aaaaaa"}).
			PaddingLeft(4)

		template := "" +
			"Title:     My Server\n" +
			"Category:  Server\n" +
			"Tag:       ssherpa\n" +
			"Fields:\n" +
			"  hostname        dev.example.com   (required)\n" +
			"  user            ubuntu            (required)\n" +
			"  port            22                (optional)\n" +
			"  identity_file   ~/.ssh/id_ed25519 (optional)\n" +
			"  proxy_jump      bastion.host      (optional)"

		b.WriteString(templateStyle.Render(template) + "\n\n")

		b.WriteString("  Press Enter to select a vault")
	} else {
		b.WriteString(wizardErrorStyle.Render("  Could not connect to 1Password CLI") + "\n\n")
		if w.checkResult.error != "" {
			b.WriteString(wizardDimStyle.Render("  "+w.checkResult.error) + "\n\n")
		}
		b.WriteString("  Setup instructions:\n")
		b.WriteString("    1. Install 1Password CLI: https://developer.1password.com/docs/cli/get-started/\n")
		b.WriteString("    2. Install 1Password desktop app\n")
		b.WriteString("    3. Enable CLI integration in Settings > Developer\n")
		b.WriteString("    4. Sign in to 1Password desktop app\n")
		b.WriteString("    5. If using op outside ssherpa, set OP_BIOMETRIC_UNLOCK_ENABLED=true\n\n")
		b.WriteString("  Press Enter to use SSH Config only, or Esc to go back")
	}

	return wizardBoxStyle.Render(b.String())
}

// renderVaultSelection renders the vault selection screen.
func (w SetupWizard) renderVaultSelection() string {
	var b strings.Builder

	title := titleStyle.Render("Select Vault")
	b.WriteString(title + "\n\n")

	b.WriteString("  Choose a vault for ssherpa servers:\n\n")

	for i, vault := range w.vaults {
		cursor := "  "
		label := fmt.Sprintf("%-20s", vault.Name)
		if i == w.cursor {
			cursor = "> "
			label = selectedStyle.Render(label)
		}
		b.WriteString("  " + cursor + label + "\n")
	}

	b.WriteString("\n")
	b.WriteString(wizardDimStyle.Render("  Use j/k to navigate, Enter to select, Esc to go back"))

	return wizardBoxStyle.Render(b.String())
}

// renderSampleOffer renders the sample entry creation offer.
func (w SetupWizard) renderSampleOffer() string {
	var b strings.Builder

	vault := w.vaults[w.selectedVaultIdx]

	title := titleStyle.Render("Create Sample Entry")
	b.WriteString(title + "\n\n")

	b.WriteString(fmt.Sprintf("  Create a sample server in \"%s\"?\n", vault.Name))
	b.WriteString("  This verifies the integration works.\n\n")

	options := []string{
		"Yes, create sample entry",
		"No, skip",
	}

	for i, opt := range options {
		cursor := "  "
		if i == w.cursor {
			cursor = "> "
			opt = selectedStyle.Render(opt)
		}
		b.WriteString("  " + cursor + opt + "\n")
	}

	b.WriteString("\n")
	b.WriteString(wizardDimStyle.Render("  Use j/k to navigate, Enter to select, Esc to go back"))

	return wizardBoxStyle.Render(b.String())
}

// renderCreatingSample renders the sample entry creation spinner.
func (w SetupWizard) renderCreatingSample() string {
	var b strings.Builder

	title := titleStyle.Render("Creating Sample Entry")
	b.WriteString(title + "\n\n")

	vault := w.vaults[w.selectedVaultIdx]
	b.WriteString(fmt.Sprintf("  %s Creating sample server in \"%s\"...\n", w.spinner.View(), vault.Name))

	return wizardBoxStyle.Render(b.String())
}

// renderMigrationOffer renders the migration offer screen.
func (w SetupWizard) renderMigrationOffer() string {
	var b strings.Builder

	title := titleStyle.Render("Existing Items Found")
	b.WriteString(title + "\n\n")

	b.WriteString(fmt.Sprintf("We found %d SSH/Server items in your 1Password vaults\n", w.migrationItemCount))
	b.WriteString("that are not yet managed by ssherpa.\n\n")

	options := []string{
		"Run migration wizard    Tag existing items for ssherpa",
		"Skip for now            You can migrate later with 'ssherpa migrate'",
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
	b.WriteString(fmt.Sprintf("  Backend: %s\n", wizardSuccessStyle.Render(backendName)))

	// Show 1Password details if applicable
	if w.backendChoice == "onepassword" || w.backendChoice == "both" {
		if len(w.vaults) > 0 {
			b.WriteString(fmt.Sprintf("  Vaults:  %d\n", len(w.vaults)))
		}
		if w.selectedVaultIdx < len(w.vaults) {
			b.WriteString(fmt.Sprintf("  Vault:   %s\n", w.vaults[w.selectedVaultIdx].Name))
		}
		if w.sampleCreated {
			b.WriteString(wizardSuccessStyle.Render("  Sample:  Created \"ssherpa-sample\" entry") + "\n")
		}
		if w.sampleError != "" {
			b.WriteString(wizardErrorStyle.Render(fmt.Sprintf("  Sample:  Failed (%s)", w.sampleError)) + "\n")
		}
		if w.runMigration {
			b.WriteString(fmt.Sprintf("  Migrated: %d items\n", w.migrationItemCount))
		}
	}

	b.WriteString("\n")
	configPath := w.configPath
	if configPath == "" {
		configPath = "~/.config/ssherpa/config.toml"
	}
	b.WriteString(fmt.Sprintf("  Config: %s\n", wizardDimStyle.Render(configPath)))

	b.WriteString("\n")
	if w.err != nil {
		b.WriteString(wizardErrorStyle.Render(fmt.Sprintf("  Error saving config: %s", w.err.Error())) + "\n")
		b.WriteString(wizardDimStyle.Render("  Press Enter to retry"))
	} else {
		b.WriteString(wizardDimStyle.Render("  Press Enter to start ssherpa"))
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

// checkOpCLI verifies that the 1Password CLI is installed and has an active session.
func checkOpCLI() tea.Cmd {
	return func() tea.Msg {
		// Step 1: Check if 'op' binary exists
		opPath, err := exec.LookPath("op")
		if err != nil {
			return onePasswordCheckCompleteMsg{
				available: false,
				error:     "1Password CLI (op) not found in PATH",
			}
		}

		// Step 2: Verify session is active by listing vaults
		ctx := context.Background()
		cmd := exec.CommandContext(ctx, opPath, "vault", "list", "--format", "json")
		cmd.Env = append(os.Environ(), "OP_BIOMETRIC_UNLOCK_ENABLED=true")
		output, err := cmd.Output()
		if err != nil {
			return onePasswordCheckCompleteMsg{
				available: false,
				error:     "No active 1Password session. Sign in via 1Password desktop app.",
			}
		}

		// Parse vault list to get names and IDs
		var cliVaults []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(output, &cliVaults); err != nil {
			return onePasswordCheckCompleteMsg{
				available: false,
				error:     fmt.Sprintf("Failed to parse vault list: %v", err),
			}
		}

		vaults := make([]vaultInfo, 0, len(cliVaults))
		for _, v := range cliVaults {
			vaults = append(vaults, vaultInfo{ID: v.ID, Name: v.Name})
		}

		return onePasswordCheckCompleteMsg{
			available: true,
			vaults:    vaults,
		}
	}
}

// onePasswordCheckCompleteMsg is sent when 1Password check completes.
type onePasswordCheckCompleteMsg struct {
	available bool
	vaults    []vaultInfo
	error     string
}

// sampleEntryCreatedMsg is sent when sample entry creation completes.
type sampleEntryCreatedMsg struct {
	err string
}

// createSampleEntry creates a sample server entry in the given vault via op CLI.
func createSampleEntry(vaultID string) tea.Cmd {
	return func() tea.Msg {
		opPath, err := exec.LookPath("op")
		if err != nil {
			return sampleEntryCreatedMsg{err: "op CLI not found"}
		}

		ctx := context.Background()
		cmd := exec.CommandContext(ctx, opPath,
			"item", "create",
			"--category", "server",
			"--vault", vaultID,
			"--title", "ssherpa-sample",
			"--tags", "ssherpa",
			"--", "hostname=example.example.com", "user=ubuntu",
			"--format", "json",
		)
		cmd.Env = append(os.Environ(), "OP_BIOMETRIC_UNLOCK_ENABLED=true")

		if _, err := cmd.Output(); err != nil {
			return sampleEntryCreatedMsg{err: fmt.Sprintf("failed to create item: %v", err)}
		}

		return sampleEntryCreatedMsg{}
	}
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
