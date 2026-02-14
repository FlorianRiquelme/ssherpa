package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	backendpkg "github.com/florianriquelme/sshjesus/internal/backend"
	"github.com/florianriquelme/sshjesus/internal/backend/onepassword"
	"github.com/florianriquelme/sshjesus/internal/config"
	"github.com/florianriquelme/sshjesus/internal/errors"
	"github.com/florianriquelme/sshjesus/internal/project"
	"github.com/florianriquelme/sshjesus/internal/sshconfig"
	"github.com/florianriquelme/sshjesus/internal/sync"
	"github.com/florianriquelme/sshjesus/internal/tui"
)

func main() {
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

	// If no config exists or backend is empty, run setup wizard
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
		historyPath = filepath.Join(homeDir, ".ssh", "sshjesus_history.json")
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
		client, err := onepassword.NewCLIClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating 1Password CLI client: %v\n", err)
			os.Exit(1)
		}

		cachePath := filepath.Join(homeDir, ".ssh", "sshjesus_1password_cache.toml")
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

		client, err := onepassword.NewCLIClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating 1Password CLI client: %v\n", err)
			os.Exit(1)
		}

		cachePath := filepath.Join(homeDir, ".ssh", "sshjesus_1password_cache.toml")
		opBackend = onepassword.NewWithCache(client, cachePath)

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
					includeFile := filepath.Join(homeDir, ".ssh", "sshjesus_config")
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
