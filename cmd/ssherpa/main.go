package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
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
	fieldsFlag := flag.Bool("fields", false, "Show 1Password field reference")
	flag.Parse()

	// Handle --version flag
	if *versionFlag {
		fmt.Println(version.Detailed())
		os.Exit(0)
	}

	// Handle --fields flag
	if *fieldsFlag {
		fmt.Print(tui.RenderFieldReferencePlain())
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

	// Run setup wizard if: --setup flag, no config, or no backend configured
	if *setupFlag || cfg == nil || cfg.Backend == "" {
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

	// If still no backend after wizard (user quit), exit gracefully
	if cfg == nil || cfg.Backend == "" {
		fmt.Fprintln(os.Stderr, "No backend configured. Run 'ssherpa --setup' to configure.")
		os.Exit(1)
	}

	// Determine backend from config
	backendType := cfg.Backend

	// Backend validation
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
	returnToTUI := cfg.ReturnToTUI

	// Detect current project from git (Phase 4)
	currentProjectID, err := project.DetectCurrentProject()
	if err != nil {
		// This should never error per design, but handle it gracefully
		currentProjectID = ""
	}

	// Get projects from config (Phase 4)
	projects := cfg.Projects

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
		opAccountName := cfg.OnePassword.AccountName

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

		opAccountNameBoth := cfg.OnePassword.AccountName

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
		opBackend.StartPolling(0, statusCallback) // 0 = use default interval from env or 5m
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
