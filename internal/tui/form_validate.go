package tui

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// validateAlias validates the SSH host alias field.
// Returns empty string if valid, error message otherwise.
func validateAlias(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "Alias is required"
	}
	if strings.Contains(trimmed, " ") {
		return "Alias cannot contain spaces"
	}
	if strings.HasPrefix(trimmed, "#") {
		return "Alias cannot start with #"
	}
	return ""
}

// validateHostname validates the hostname field.
// Returns empty string if valid, error message otherwise.
func validateHostname(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "Hostname is required"
	}
	if strings.Contains(trimmed, " ") {
		return "Hostname cannot contain spaces"
	}
	return ""
}

// validateUser validates the user field.
// Returns empty string if valid, error message otherwise.
func validateUser(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "User is required"
	}
	if strings.Contains(trimmed, " ") {
		return "User cannot contain spaces"
	}
	return ""
}

// validatePort validates the port field.
// Returns empty string if valid, error message otherwise.
// Port is optional (empty is valid).
func validatePort(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		// Empty port is valid (use SSH default 22)
		return ""
	}

	// Parse as integer
	port, err := strconv.Atoi(trimmed)
	if err != nil {
		return "Port must be a number between 1 and 65535"
	}

	// Check range
	if port < 1 || port > 65535 {
		return "Port must be a number between 1 and 65535"
	}

	return ""
}

// checkDNS performs a DNS lookup on the hostname with a 2-second timeout.
// Returns nil on success, error on failure.
func checkDNS(hostname string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := net.DefaultResolver.LookupHost(ctx, hostname)
	if err != nil {
		return fmt.Errorf("DNS lookup failed: %w", err)
	}

	return nil
}

// dnsCheckResultMsg is sent after async DNS check completes.
type dnsCheckResultMsg struct {
	hostname string
	err      error
}

// dnsCheckCmd returns a tea.Cmd that performs DNS lookup asynchronously.
func dnsCheckCmd(hostname string) tea.Cmd {
	return func() tea.Msg {
		err := checkDNS(hostname)
		return dnsCheckResultMsg{
			hostname: hostname,
			err:      err,
		}
	}
}
