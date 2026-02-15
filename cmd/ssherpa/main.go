package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kevinburke/ssh_config"
	backendpkg "github.com/florianriquelme/ssherpa/internal/backend"
	"github.com/florianriquelme/ssherpa/internal/backend/onepassword"
	"github.com/florianriquelme/ssherpa/internal/config"
	"github.com/florianriquelme/ssherpa/internal/errors"
	"github.com/florianriquelme/ssherpa/internal/project"
	"github.com/florianriquelme/ssherpa/internal/sshconfig"
	"github.com/florianriquelme/ssherpa/internal/sync"
	"github.com/florianriquelme/ssherpa/internal/tui"
	"github.com/florianriquelme/ssherpa/internal/version"
)

func main() {
	// Parse CLI flags BEFORE any backend initialization
	versionFlag := flag.Bool("version", false, "Show version information")
	setupFlag := flag.Bool("setup", false, "Run setup wizard")
	flag.Parse()

	// Handle --version flag
	if *versionFlag {
		fmt.Println(version.Detailed())
		os.Exit(0)
	}

	// Resolve app config path before loading
	appConfigPath, err := config.DefaultPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determining config path: %v\n", err)
		os.Exit(1)
	}

	// Load config (optional for Phase 2 — SSH config is the default)
	cfg, err := config.Load("")
	if err != nil && err != errors.ErrConfigNotFound {
		// Real error (not just missing config)
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Determine if onboarding should run
	shouldRunOnboarding := *setupFlag || (cfg == nil || !cfg.OnboardingDone)

	if shouldRunOnboarding {
		// Initialize config if needed
		if cfg == nil {
			cfg = config.DefaultConfig()
		}

		// Run onboarding flow
		if err := runOnboarding(cfg, appConfigPath, *setupFlag); err != nil {
			fmt.Fprintf(os.Stderr, "Error running onboarding: %v\n", err)
			os.Exit(1)
		}

		// Reload config after onboarding completes
		cfg, err = config.Load("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config after onboarding: %v\n", err)
			os.Exit(1)
		}
	}

	// Fallback: If config still has no backend (edge case), run setup wizard
	if cfg == nil || cfg.Backend == "" {
		wizard := tui.NewSetupWizard(appConfigPath)
		p := tea.NewProgram(wizard, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running setup wizard: %v\n", err)
			os.Exit(1)
		}

		// Reload config after wizard completes
		cfg, err = config.Load("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config after setup: %v\n", err)
			os.Exit(1)
		}
	}

	// Determine backend from config (or default to sshconfig)
	backendType := "sshconfig"
	if cfg != nil && cfg.Backend != "" {
		backendType = cfg.Backend
	}

	// Backend validation happens naturally when backend adapter is created
	if backendType != "sshconfig" && backendType != "onepassword" && backendType != "both" {
		fmt.Fprintf(os.Stderr, "Backend '%s' not supported. Valid options: sshconfig, onepassword, both\n", backendType)
		os.Exit(1)
	}

	// Determine SSH config path and history path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determining home directory: %v\n", err)
		os.Exit(1)
	}
	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")

	// Determine history path
	historyPath := ""
	if homeDir != "" {
		historyPath = filepath.Join(homeDir, ".ssh", "ssherpa_history.json")
	}

	// Get return-to-TUI config option (default: false = exit after SSH)
	returnToTUI := false
	if cfg != nil {
		returnToTUI = cfg.ReturnToTUI
	}

	// Detect current project from git (Phase 4)
	currentProjectID, err := project.DetectCurrentProject()
	if err != nil {
		// This should never error per design, but handle it gracefully
		currentProjectID = ""
	}

	// Get projects from config (Phase 4)
	var projects []config.ProjectConfig
	if cfg != nil {
		projects = cfg.Projects
	}

	// Construct backend based on configuration
	var backend backendpkg.Backend
	var opStatus backendpkg.BackendStatus = backendpkg.StatusUnknown
	var opBackend *onepassword.Backend

	switch backendType {
	case "sshconfig":
		// Pure SSH config backend
		sshBackend, err := sshconfig.New(sshConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating SSH config backend: %v\n", err)
			os.Exit(1)
		}
		backend = sshBackend

	case "onepassword":
		// 1Password backend
		opAccountName := ""
		if cfg != nil {
			opAccountName = cfg.OnePassword.AccountName
		}

		var client *onepassword.CLIClient
		if opAccountName != "" {
			client, err = onepassword.NewCLIClientWithAccount(opAccountName)
		} else {
			client, err = onepassword.NewCLIClient()
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating 1Password CLI client: %v\n", err)
			os.Exit(1)
		}

		cachePath := filepath.Join(homeDir, ".ssh", "ssherpa_1password_cache.toml")
		opBackend = onepassword.NewWithCache(client, cachePath)

		// Load from cache (best-effort, non-fatal) - TUI will show cached data instantly
		if cacheErr := opBackend.LoadFromCache(); cacheErr != nil {
			// No cache available, TUI will start with empty list
			// Background sync will populate data when it completes
		}

		backend = opBackend
		opStatus = opBackend.GetStatus()

	case "both":
		// Multi-backend: SSH config + 1Password
		sshBackend, err := sshconfig.New(sshConfigPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating SSH config backend: %v\n", err)
			os.Exit(1)
		}

		opAccountNameBoth := ""
		if cfg != nil {
			opAccountNameBoth = cfg.OnePassword.AccountName
		}

		var clientBoth *onepassword.CLIClient
		if opAccountNameBoth != "" {
			clientBoth, err = onepassword.NewCLIClientWithAccount(opAccountNameBoth)
		} else {
			clientBoth, err = onepassword.NewCLIClient()
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating 1Password CLI client: %v\n", err)
			os.Exit(1)
		}

		cachePath := filepath.Join(homeDir, ".ssh", "ssherpa_1password_cache.toml")
		opBackend = onepassword.NewWithCache(clientBoth, cachePath)

		// Load from cache (best-effort, non-fatal) - SSH config data is always available
		if cacheErr := opBackend.LoadFromCache(); cacheErr != nil {
			// No cache available, TUI will show SSH config servers only
			// Background sync will add 1Password servers when it completes
		}

		backend = backendpkg.NewMultiBackend(sshBackend, opBackend)
		opStatus = opBackend.GetStatus()
	}

	// Create TUI model with backend status and backend
	model := tui.New(sshConfigPath, historyPath, returnToTUI, currentProjectID, projects, appConfigPath, opStatus, backend)

	// Run TUI with alt screen (doesn't pollute terminal history)
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Start poller if we have a 1Password backend
	if opBackend != nil {
		// Track whether we've generated SSH include file yet
		var sshIncludeGenerated bool

		// Create a callback that sends a message to the TUI program
		statusCallback := func(status backendpkg.BackendStatus) {
			p.Send(tui.OnePasswordStatusMsg{Status: status})

			// On first successful sync, generate SSH include file and notify TUI to refresh
			if status == backendpkg.StatusAvailable && !sshIncludeGenerated {
				servers, err := opBackend.ListServers(context.Background())
				if err == nil {
					includeFile := filepath.Join(homeDir, ".ssh", "ssherpa_config")
					if err := sync.WriteSSHIncludeFile(servers, includeFile); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to write SSH include file: %v\n", err)
					}
					if err := sync.EnsureIncludeDirective(sshConfigPath, includeFile); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Failed to ensure Include directive: %v\n", err)
					}
					sshIncludeGenerated = true

					// Notify TUI to refresh server list
					p.Send(tui.BackendServersUpdatedMsg{})
				}
			}
		}
		opBackend.StartPolling(0, statusCallback) // 0 = use default interval from env or 5s
		defer opBackend.Close()
	} else {
		// Close backend on exit
		defer backend.Close()
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

// runOnboarding executes the first-run onboarding flow.
// It shows a welcome message, detects SSH config hosts, and optionally runs the 1Password setup wizard.
func runOnboarding(cfg *config.Config, appConfigPath string, setupFlag bool) error {
	// Step 1: Welcome + SSH config detection
	fmt.Println("Welcome to ssherpa!")
	fmt.Println()

	// Count hosts in ~/.ssh/config
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")

	hostCount := 0
	if data, err := os.ReadFile(sshConfigPath); err == nil {
		sshCfg, parseErr := ssh_config.Decode(strings.NewReader(string(data)))
		if parseErr == nil {
			// Count non-wildcard Host entries
			for _, host := range sshCfg.Hosts {
				// Skip wildcard patterns
				if len(host.Patterns) > 0 {
					pattern := host.Patterns[0].String()
					if pattern != "*" && !strings.Contains(pattern, "*") {
						hostCount++
					}
				}
			}
		}
	}

	if hostCount > 0 {
		fmt.Printf("Found %d SSH hosts in your config.\n", hostCount)
	} else {
		fmt.Println("No SSH config found.")
	}
	fmt.Println()

	// Step 2: Offer 1Password setup
	fmt.Print("Would you like to set up 1Password integration? [y/N] ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		// Launch the existing setup wizard
		wizard := tui.NewSetupWizard(appConfigPath)
		p := tea.NewProgram(wizard, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("setup wizard failed: %w", err)
		}

		// Reload config after wizard completes (wizard saves config)
		reloadedCfg, err := config.Load("")
		if err != nil {
			return fmt.Errorf("failed to reload config after wizard: %w", err)
		}
		// Copy wizard results to the passed config
		*cfg = *reloadedCfg
	} else {
		// User chose not to use 1Password - save sshconfig-only backend
		cfg.Backend = "sshconfig"
		if err := config.Save(cfg, appConfigPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	// Step 3: Mark onboarding done
	cfg.OnboardingDone = true
	if err := config.Save(cfg, appConfigPath); err != nil {
		return fmt.Errorf("failed to save onboarding state: %w", err)
	}

	fmt.Println("Setup complete! Launching ssherpa...")
	fmt.Println()

	return nil
}
