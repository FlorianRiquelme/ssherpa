package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/florianriquelme/sshjesus/internal/config"
	"github.com/florianriquelme/sshjesus/internal/errors"
	"github.com/florianriquelme/sshjesus/internal/tui"
)

func main() {
	// Load config (optional for Phase 2 — SSH config is the default)
	cfg, err := config.Load("")
	if err != nil && err != errors.ErrConfigNotFound {
		// Real error (not just missing config)
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Determine backend from config (or default to sshconfig)
	backend := "sshconfig"
	if cfg != nil && cfg.Backend != "" {
		backend = cfg.Backend
	}

	// Only sshconfig backend is supported in Phase 2
	if backend != "sshconfig" {
		fmt.Fprintf(os.Stderr, "Backend '%s' not yet supported. Using sshconfig.\n", backend)
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

	// Create TUI model with new parameters
	model := tui.New(sshConfigPath, historyPath, returnToTUI)

	// Run TUI with alt screen (doesn't pollute terminal history)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
