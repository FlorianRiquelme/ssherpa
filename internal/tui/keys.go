package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the TUI.
// Implements help.KeyMap interface for automatic help text generation.
type KeyMap struct {
	// Navigation
	Up           key.Binding
	Down         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GoToTop      key.Binding
	GoToBottom   key.Binding

	// Actions
	Connect       key.Binding
	Details       key.Binding
	Search        key.Binding
	AssignProject key.Binding
	SelectKey     key.Binding
	AddServer     key.Binding
	EditServer    key.Binding
	DeleteServer  key.Binding
	Undo          key.Binding
	SignIn        key.Binding
	Help          key.Binding
	Quit          key.Binding
	ClearSearch   key.Binding
	ForceQuit     key.Binding
}

// ShortHelp returns the key bindings shown in the mini help view (list mode).
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Connect, k.Details, k.Search, k.AddServer, k.EditServer, k.DeleteServer, k.Help, k.SignIn, k.Quit}
}

// FullHelp returns the key bindings shown in the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},                    // Navigation group
		{k.HalfPageUp, k.HalfPageDown, k.GoToTop, k.GoToBottom}, // Advanced navigation
		{k.Connect, k.Details, k.Search, k.AssignProject, k.SelectKey, k.AddServer, k.EditServer, k.DeleteServer, k.Quit}, // Actions group
		{k.Undo, k.Help, k.SignIn}, // Additional actions
	}
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "½ page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "½ page down"),
		),
		GoToTop: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g/home", "top"),
		),
		GoToBottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G/end", "bottom"),
		),

		// Actions
		Connect: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect"),
		),
		Details: key.NewBinding(
			key.WithKeys("tab", "i"),
			key.WithHelp("tab/i", "details"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		AssignProject: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "project"),
		),
		SelectKey: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("K", "ssh key"),
		),
		AddServer: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		EditServer: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		DeleteServer: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Undo: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "undo"),
		),
		SignIn: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "authenticate"),
			key.WithDisabled(),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ClearSearch: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit search"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			// No help text - hidden from help display
		),
	}
}

// SearchKeyMap provides key bindings for search mode help display.
type SearchKeyMap struct {
	ClearSearch key.Binding
}

// ShortHelp returns the key bindings for search mode mini help.
func (k SearchKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.ClearSearch}
}

// FullHelp returns the key bindings for search mode full help.
func (k SearchKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.ClearSearch}}
}
