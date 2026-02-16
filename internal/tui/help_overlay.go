package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpOverlay is a scrollable overlay showing the 1Password field reference.
type HelpOverlay struct {
	viewport viewport.Model
	width    int
	height   int
}

// NewHelpOverlay creates a new help overlay with the given dimensions.
func NewHelpOverlay(width, height int) HelpOverlay {
	// Create viewport for scrollable content
	vp := viewport.New(68, height-8) // Leave room for border and footer
	vp.SetContent(RenderFieldReference())

	return HelpOverlay{
		viewport: vp,
		width:    width,
		height:   height,
	}
}

// Update handles viewport scrolling.
func (h HelpOverlay) Update(msg tea.Msg) (HelpOverlay, tea.Cmd) {
	var cmd tea.Cmd
	h.viewport, cmd = h.viewport.Update(msg)
	return h, cmd
}

// View renders the help overlay with border and footer.
func (h HelpOverlay) View() string {
	footer := helpFooterStyle.Render("Esc or ?: close | ↑/↓: scroll")
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		h.viewport.View(),
		"",
		footer,
	)
	return helpOverlayStyle.Render(content)
}

// RenderFieldReference returns the styled 1Password field reference content.
func RenderFieldReference() string {
	title := formTitleStyle.Render("1Password Field Reference")

	itemPropsSection := lipgloss.JoinVertical(
		lipgloss.Left,
		formLabelStyle.Render("Item Properties:"),
		"  • Title: Display name for the server (becomes the alias)",
		"  • Category: Must be \"Server\"",
		"  • Tag: Must include \"ssherpa\" (case-insensitive)",
		"",
	)

	// Build field table
	table := buildFieldTable()

	exampleSection := lipgloss.JoinVertical(
		lipgloss.Left,
		formLabelStyle.Render("Example:"),
		"",
		"  Title: Production API Server",
		"  Category: Server",
		"  Tags: ssherpa, production",
		"",
		"  Fields:",
		"    hostname: api.example.com",
		"    user: deploy",
		"    port: 2222",
		"    project_tags: api,backend",
		"",
	)

	noteSection := secondaryStyle.Render("Note: Field names are case-insensitive")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		itemPropsSection,
		table,
		"",
		exampleSection,
		noteSection,
	)
}

// RenderFieldReferencePlain returns the plain-text 1Password field reference (no ANSI codes).
func RenderFieldReferencePlain() string {
	var b strings.Builder

	b.WriteString("1Password Field Reference\n")
	b.WriteString("=========================\n\n")

	b.WriteString("Item Properties:\n")
	b.WriteString("  • Title: Display name for the server (becomes the alias)\n")
	b.WriteString("  • Category: Must be \"Server\"\n")
	b.WriteString("  • Tag: Must include \"ssherpa\" (case-insensitive)\n\n")

	b.WriteString("Fields:\n")
	b.WriteString("  Field                 Required  Default       Description\n")
	b.WriteString("  ─────────────────────────────────────────────────────────────────────────\n")
	b.WriteString("  hostname              yes       -             Server hostname or IP address\n")
	b.WriteString("  user                  yes       -             SSH username\n")
	b.WriteString("  port                  no        22            SSH port number\n")
	b.WriteString("  identity_file         no        SSH default   Path to SSH private key file\n")
	b.WriteString("  proxy_jump            no        -             Bastion/jump host for ProxyJump\n")
	b.WriteString("  project_tags          no        -             Comma-separated project tags (e.g. \"web,api\")\n")
	b.WriteString("  remote_project_path   no        -             Remote path to cd into on connect\n")
	b.WriteString("  forward_agent         no        -             Enable SSH agent forwarding (noted, not yet mapped)\n")
	b.WriteString("  extra_config          no        -             Additional SSH config directives (noted, not yet mapped)\n\n")

	b.WriteString("Example:\n")
	b.WriteString("  Title: Production API Server\n")
	b.WriteString("  Category: Server\n")
	b.WriteString("  Tags: ssherpa, production\n\n")
	b.WriteString("  Fields:\n")
	b.WriteString("    hostname: api.example.com\n")
	b.WriteString("    user: deploy\n")
	b.WriteString("    port: 2222\n")
	b.WriteString("    project_tags: api,backend\n\n")

	b.WriteString("Note: Field names are case-insensitive\n")

	return b.String()
}

// buildFieldTable creates a styled table of fields.
func buildFieldTable() string {
	header := lipgloss.JoinHorizontal(
		lipgloss.Left,
		formLabelStyle.Render("Field"),
		strings.Repeat(" ", 16), // padding
		formLabelStyle.Render("Required"),
		strings.Repeat(" ", 2),
		formLabelStyle.Render("Default"),
		strings.Repeat(" ", 7),
		formLabelStyle.Render("Description"),
	)

	separator := secondaryStyle.Render(strings.Repeat("─", 68))

	rows := []string{
		buildFieldRow("hostname", "yes", "-", "Server hostname or IP address"),
		buildFieldRow("user", "yes", "-", "SSH username"),
		buildFieldRow("port", "no", "22", "SSH port number"),
		buildFieldRow("identity_file", "no", "SSH default", "Path to SSH private key file"),
		buildFieldRow("proxy_jump", "no", "-", "Bastion/jump host for ProxyJump"),
		buildFieldRow("project_tags", "no", "-", "Comma-separated project tags (e.g. \"web,api\")"),
		buildFieldRow("remote_project_path", "no", "-", "Remote path to cd into on connect"),
		buildFieldRow("forward_agent", "no", "-", "Enable SSH agent forwarding (noted, not yet mapped)"),
		buildFieldRow("extra_config", "no", "-", "Additional SSH config directives (noted, not yet mapped)"),
	}

	parts := []string{header, separator}
	parts = append(parts, rows...)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// buildFieldRow creates a single row in the field table.
func buildFieldRow(field, required, defaultVal, description string) string {
	// Column widths: field=20, required=10, default=14, description=remaining
	fieldCol := field + strings.Repeat(" ", max(0, 20-len(field)))
	reqCol := required + strings.Repeat(" ", max(0, 10-len(required)))
	defCol := defaultVal + strings.Repeat(" ", max(0, 14-len(defaultVal)))

	// Highlight required fields
	var styledField string
	if required == "yes" {
		styledField = formRequiredStyle.Render(fieldCol)
	} else {
		styledField = secondaryStyle.Render(fieldCol)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		styledField,
		secondaryStyle.Render(reqCol),
		secondaryStyle.Render(defCol),
		description,
	)
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
