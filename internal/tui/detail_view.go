package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/florianriquelme/ssherpa/internal/sshconfig"
)

// renderDetailView renders the full-screen detail view for a selected host.
// Shows all SSH config options, source backend, source file, and error info.
func renderDetailView(host *sshconfig.SSHHost, source string, width, height int) string {
	if host == nil {
		return emptyStateStyle.Render("No host selected")
	}

	var b strings.Builder

	// Header: host name
	b.WriteString(detailHeaderStyle.Render(fmt.Sprintf("SSH Host: %s", host.Name)))
	b.WriteString("\n\n")

	// Show parse error prominently if present
	if host.ParseError != nil {
		b.WriteString(warningStyle.Render("⚠ Parse Error:\n"))
		b.WriteString(warningStyle.Render(fmt.Sprintf("  %v\n", host.ParseError)))
		b.WriteString("\n")
	}

	// Backend source
	if source != "" {
		b.WriteString(secondaryStyle.Render(fmt.Sprintf("Source: %s", source)))
		b.WriteString("\n")
	}

	// Source file tracking (for ssh-config backend)
	if host.SourceFile != "" {
		sourceInfo := fmt.Sprintf("Defined in: %s", host.SourceFile)
		if host.SourceLine > 0 {
			sourceInfo = fmt.Sprintf("Defined in: %s:%d", host.SourceFile, host.SourceLine)
		}
		b.WriteString(secondaryStyle.Render(sourceInfo))
		b.WriteString("\n")
	}

	// Separator
	b.WriteString(separatorStyle.Render(strings.Repeat("─", min(width-2, 80))))
	b.WriteString("\n\n")

	// Standard fields section
	b.WriteString(detailLabelStyle.Render("Standard Fields:"))
	b.WriteString("\n")

	writeField := func(label, value string) {
		if value != "" {
			fmt.Fprintf(&b, "  %s %s\n",
				detailLabelStyle.Render(label+":"),
				detailValueStyle.Render(value))
		}
	}

	writeField("Hostname", host.Hostname)
	writeField("User", host.User)
	writeField("Port", host.Port)

	// Identity files
	// TODO: Enhance with key type, fingerprint, and source badge when discoveredKeys are available
	if len(host.IdentityFile) > 0 {
		fmt.Fprintf(&b, "  %s\n", detailLabelStyle.Render("IdentityFile:"))
		for _, file := range host.IdentityFile {
			fmt.Fprintf(&b, "    %s\n", detailValueStyle.Render(file))
		}
	}

	// All options section (sorted alphabetically)
	if len(host.AllOptions) > 0 {
		b.WriteString("\n")
		b.WriteString(detailLabelStyle.Render("All SSH Options:"))
		b.WriteString("\n")

		// Sort keys for consistent display
		keys := make([]string, 0, len(host.AllOptions))
		for key := range host.AllOptions {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			values := host.AllOptions[key]
			if len(values) == 1 {
				b.WriteString(fmt.Sprintf("  %s %s\n",
					detailLabelStyle.Render(key+":"),
					detailValueStyle.Render(values[0])))
			} else {
				b.WriteString(fmt.Sprintf("  %s\n", detailLabelStyle.Render(key+":")))
				for _, val := range values {
					b.WriteString(fmt.Sprintf("    %s\n", detailValueStyle.Render(val)))
				}
			}
		}
	}

	// Footer help text
	b.WriteString("\n")
	b.WriteString(statusBarStyle.Render("K: select key | Esc: back | ↑↓: scroll | q: quit"))

	return b.String()
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
