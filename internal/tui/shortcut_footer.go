package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// shortcutHint represents a single key+description pair.
type shortcutHint struct {
	key  string
	desc string
}

// renderShortcutFooter renders a context-aware multi-line shortcut footer.
// It shows different shortcuts depending on the current view mode and state.
func renderShortcutFooter(mode ViewMode, searchFocused bool, signInEnabled bool, hasUndo bool, hasUpdate bool) string {
	switch {
	case mode == ViewList && searchFocused:
		return renderHintRows([][]shortcutHint{
			{
				{key: "esc", desc: "exit search"},
				{key: "enter", desc: "connect"},
			},
		})

	case mode == ViewDetail:
		return renderHintRows([][]shortcutHint{
			{
				{key: "esc", desc: "back"},
				{key: "K", desc: "ssh key"},
				{key: "↑/↓", desc: "scroll"},
				{key: "q", desc: "quit"},
			},
		})

	default: // ViewList (not searching), or any other mode
		row1 := []shortcutHint{
			{key: "enter", desc: "connect"},
			{key: "/", desc: "search"},
			{key: "a", desc: "add"},
			{key: "e", desc: "edit"},
			{key: "d", desc: "delete"},
			{key: "q", desc: "quit"},
		}
		row2 := []shortcutHint{
			{key: "p", desc: "project"},
			{key: "?", desc: "1pass ref"},
		}
		if hasUndo {
			row2 = append(row2, shortcutHint{key: "u", desc: "undo"})
		}
		if hasUpdate {
			row2 = append(row2, shortcutHint{key: "U", desc: "update"})
		}
		if signInEnabled {
			row2 = append(row2, shortcutHint{key: "s", desc: "authenticate"})
		}
		return renderHintRows([][]shortcutHint{row1, row2})
	}
}

// renderHintRows renders multiple rows of shortcut hints.
func renderHintRows(rows [][]shortcutHint) string {
	var rendered []string
	for _, row := range rows {
		rendered = append(rendered, renderHintRow(row))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rendered...)
}

// renderHintRow renders a single row of shortcut hints separated by " · ".
func renderHintRow(hints []shortcutHint) string {
	sep := shortcutSepStyle.Render(" · ")
	var parts []string
	for _, h := range hints {
		part := shortcutKeyStyle.Render(h.key) + " " + shortcutDescStyle.Render(h.desc)
		parts = append(parts, part)
	}
	return " " + strings.Join(parts, sep)
}
