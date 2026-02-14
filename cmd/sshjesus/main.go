package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	backendpkg "github.com/florianriquelme/sshjesus/internal/backend"
	"github.com/florianriquelme/sshjesus/internal/config"
	"github.com/florianriquelme/sshjesus/internal/errors"
	"github.com/florianriquelme/sshjesus/internal/project"
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
	backend := "sshconfig"
	if cfg != nil && cfg.Backend != "" {
		backend = cfg.Backend
	}

	// Backend validation happens naturally when backend adapter is created
	if backend != "sshconfig" && backend != "onepassword" && backend != "both" {
		fmt.Fprintf(os.Stderr, "Backend '%s' not supported. Valid options: sshconfig, onepassword, both\n", backend)
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

	// Create TUI model with new parameters
	// For now, pass StatusUnknown (1Password integration in next task)
	model := tui.New(sshConfigPath, historyPath, returnToTUI, currentProjectID, projects, appConfigPath, backendpkg.StatusUnknown)

	// Run TUI with alt screen (doesn't pollute terminal history)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
