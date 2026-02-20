package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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

	// Vault discovery fields
	vaults       []vaultDiscovery // Available vaults from 1Password
	totalServers int              // Total ssherpa-tagged servers across all vaults
}

type vaultDiscovery struct {
	ID          string
	Name        string
	ServerCount int
}

type onePasswordCheckResult struct {
	available    bool
	vaults       []vaultDiscovery
	totalServers int
	error        string
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
		w.checking = false
		w.checkResult = onePasswordCheckResult(msg)
		w.vaults = msg.vaults
		w.totalServers = msg.totalServers
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
		if w.checkResult.available {
			w.step = stepSummary
		} else {
			w.backendChoice = "sshconfig"
			w.step = stepSummary
		}
		return w, nil
	case "esc":
		w.step = stepWelcome
		w.checkResult = onePasswordCheckResult{}
		return w, nil
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

		if w.totalServers > 0 {
			b.WriteString("  Auto-discovered ssherpa servers:\n\n")
			for _, vault := range w.vaults {
				if vault.ServerCount > 0 {
					b.WriteString(fmt.Sprintf("    %-20s %d server(s)\n", vault.Name, vault.ServerCount))
				}
			}
			b.WriteString("    " + strings.Repeat("\u2500", 30) + "\n")
			b.WriteString(fmt.Sprintf("    %-20s %d server(s)\n\n", "Total", w.totalServers))
		} else {
			b.WriteString("  No ssherpa-tagged servers found yet.\n")
			b.WriteString("  Create servers in 1Password with the \"ssherpa\" tag,\n")
			b.WriteString("  or add them from the ssherpa TUI after setup.\n\n")
		}

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

		b.WriteString("  Press Enter to continue")
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
		if w.totalServers > 0 {
			b.WriteString(fmt.Sprintf("  Servers: %d discovered\n", w.totalServers))
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

// opEnv returns os environment variables with biometric unlock enabled for 1Password CLI.
func opEnv() []string {
	return append(os.Environ(), "OP_BIOMETRIC_UNLOCK_ENABLED=true")
}

// checkOpCLI verifies that the 1Password CLI is installed and has an active session.
func checkOpCLI() tea.Cmd {
	return func() tea.Msg {
		opPath, err := exec.LookPath("op")
		if err != nil {
			return onePasswordCheckCompleteMsg{
				available: false,
				error:     "1Password CLI (op) not found in PATH",
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, opPath, "vault", "list", "--format", "json")
		cmd.Env = opEnv()
		output, err := cmd.Output()
		if err != nil {
			return onePasswordCheckCompleteMsg{
				available: false,
				error:     "No active 1Password session. Sign in via 1Password desktop app.",
			}
		}

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

		vaults := make([]vaultDiscovery, 0, len(cliVaults))
		totalServers := 0
		for _, v := range cliVaults {
			count := countSsherpaItems(ctx, opPath, v.ID)
			vaults = append(vaults, vaultDiscovery{
				ID:          v.ID,
				Name:        v.Name,
				ServerCount: count,
			})
			totalServers += count
		}

		return onePasswordCheckCompleteMsg{
			available:    true,
			vaults:       vaults,
			totalServers: totalServers,
		}
	}
}

// countSsherpaItems counts items tagged "ssherpa" in a specific vault.
func countSsherpaItems(ctx context.Context, opPath, vaultID string) int {
	cmd := exec.CommandContext(ctx, opPath, "item", "list",
		"--vault", vaultID,
		"--tags", "ssherpa",
		"--format", "json",
	)
	cmd.Env = opEnv()
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	var items []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(output, &items); err != nil {
		return 0
	}
	return len(items)
}

// onePasswordCheckCompleteMsg is sent when 1Password check completes.
type onePasswordCheckCompleteMsg struct {
	available    bool
	vaults       []vaultDiscovery
	totalServers int
	error        string
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
