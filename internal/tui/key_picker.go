package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/florianriquelme/sshjesus/internal/sshkey"
)

// SSHKeyPicker is a lightweight popup overlay for SSH key selection.
type SSHKeyPicker struct {
	keys           []sshkey.SSHKey
	selected       int
	currentKeyPath string // Currently assigned IdentityFile (for checkmark)
	width          int
	height         int
	serverName     string // Server this picker is for (displayed in title)
}

// NewSSHKeyPicker creates a new SSH key picker overlay.
func NewSSHKeyPicker(
	serverName string,
	keys []sshkey.SSHKey,
	currentKeyPath string, // Current IdentityFile path (empty if none)
) SSHKeyPicker {
	return SSHKeyPicker{
		keys:           keys,
		selected:       0,
		currentKeyPath: currentKeyPath,
		serverName:     serverName,
		width:          70,
		height:         20,
	}
}

// Update handles picker key events.
func (p SSHKeyPicker) Update(msg tea.Msg) (SSHKeyPicker, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// Close picker without changes
			return p, func() tea.Msg { return keyPickerClosedMsg{} }

		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			p.selected++
			// Max index is len(keys) because we have "None" option at index 0
			if p.selected > len(p.keys) {
				p.selected = len(p.keys)
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			p.selected--
			if p.selected < 0 {
				p.selected = 0
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Select key or "None"
			if p.selected == 0 {
				// "None (SSH default)" selected
				return p, func() tea.Msg {
					return keySelectedMsg{path: "", cleared: true}
				}
			} else {
				// A key was selected (adjust index for "None" offset)
				keyIdx := p.selected - 1
				if keyIdx >= 0 && keyIdx < len(p.keys) {
					selectedKey := p.keys[keyIdx]
					return p, func() tea.Msg {
						return keySelectedMsg{path: selectedKey.Path, key: &selectedKey, cleared: false}
					}
				}
			}
		}
	}

	return p, nil
}

// View renders the picker overlay.
func (p SSHKeyPicker) View() string {
	var b strings.Builder

	// Title
	title := pickerTitleStyle.Render(fmt.Sprintf("Select SSH Key: %s", p.serverName))
	b.WriteString(title)
	b.WriteString("\n\n")

	// First item: "None (SSH default)"
	noneSelected := p.selected == 0
	cursor := "  "
	if noneSelected {
		cursor = "> "
	}
	checkmark := "  "
	if p.currentKeyPath == "" {
		checkmark = pickerCheckmarkStyle.Render("✓ ")
	}
	noneStyle := lipgloss.NewStyle()
	if noneSelected {
		noneStyle = pickerSelectedStyle
	}
	b.WriteString(cursor)
	b.WriteString(checkmark)
	b.WriteString(noneStyle.Render("None (SSH default)"))
	b.WriteString("\n")

	// Render each key
	for i, k := range p.keys {
		// Item index in the list (0 = None, 1+ = keys)
		itemIndex := i + 1
		isSelected := p.selected == itemIndex

		// Cursor indicator
		cursor := "  "
		if isSelected {
			cursor = "> "
		}

		// Checkmark if this key is currently assigned
		checkmark := "  "
		if k.Path == p.currentKeyPath {
			checkmark = pickerCheckmarkStyle.Render("✓ ")
		}

		// Key info line
		keyInfo := p.renderKeyInfo(k)

		// Apply selection style
		keyStyle := lipgloss.NewStyle()
		if isSelected {
			keyStyle = pickerSelectedStyle
		}

		b.WriteString(cursor)
		b.WriteString(checkmark)
		b.WriteString(keyStyle.Render(keyInfo))
		b.WriteString("\n")

		// Fingerprint on second line (indented, dimmed)
		if k.Fingerprint != "" {
			fpLine := p.renderFingerprint(k.Fingerprint)
			b.WriteString("    ")
			b.WriteString(keyFingerprintStyle.Render(fpLine))
			b.WriteString("\n")
		}
	}

	// Help text
	b.WriteString("\n")
	help := pickerHelpStyle.Render("up/k down/j: navigate • enter: select • esc: cancel")
	b.WriteString(help)

	// Wrap in border
	content := b.String()
	bordered := pickerBorderStyle.Render(content)

	return bordered
}

// renderKeyInfo renders the key info line: filename type source_badge [missing] [encrypted]
func (p SSHKeyPicker) renderKeyInfo(k sshkey.SSHKey) string {
	var parts []string

	// Filename
	parts = append(parts, k.Filename)

	// Type (if available)
	if k.Type != "" {
		parts = append(parts, fmt.Sprintf("(%s)", k.Type))
	}

	// Source badge with appropriate color
	badge := p.renderSourceBadge(k)
	parts = append(parts, badge)

	// Missing badge
	if k.Missing {
		parts = append(parts, keyMissingBadgeStyle.Render("[missing]"))
	}

	// Encrypted badge
	if k.Encrypted {
		parts = append(parts, keyEncryptedBadgeStyle.Render("[encrypted]"))
	}

	return strings.Join(parts, " ")
}

// renderSourceBadge renders the source badge with appropriate color
func (p SSHKeyPicker) renderSourceBadge(k sshkey.SSHKey) string {
	switch k.Source {
	case sshkey.SourceFile:
		return keySourceFileStyle.Render("[file]")
	case sshkey.SourceAgent:
		return keySourceAgentStyle.Render("[agent]")
	case sshkey.Source1Password:
		return keySource1PStyle.Render("[1password]")
	default:
		return secondaryStyle.Render("[unknown]")
	}
}

// renderFingerprint truncates and formats the fingerprint for display
func (p SSHKeyPicker) renderFingerprint(fp string) string {
	// Truncate to fit width (leave room for indent and border)
	maxLen := p.width - 10
	if len(fp) > maxLen {
		return fp[:maxLen-3] + "..."
	}
	return fp
}
