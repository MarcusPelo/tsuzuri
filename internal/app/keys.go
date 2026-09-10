package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines application-wide global keyboard shortcuts.
type KeyMap struct {
	Quit    key.Binding
	Tab     key.Binding
	NewPage key.Binding
}

// DefaultKeyMap returns the default keybindings for Tsuzuri.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		NewPage: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "new page"),
		),
	}
}
