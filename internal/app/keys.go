package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines application-wide global keyboard shortcuts.
type KeyMap struct {
	Quit          key.Binding
	Tab           key.Binding
	NewPage       key.Binding
	Dashboard     key.Binding
	ToggleSidebar key.Binding
	Help          key.Binding
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
		Dashboard: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "dashboard"),
		),
		ToggleSidebar: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("ctrl+b", "toggle sidebar"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}
