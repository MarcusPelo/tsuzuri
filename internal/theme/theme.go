// Package theme provides application-wide design system tokens, color palettes,
// and lipgloss style constructors.
package theme

import "github.com/charmbracelet/lipgloss"

// Theme encapsulates design system colors and styles.
type Theme struct {
	NormalBg    lipgloss.Color
	InsertBg    lipgloss.Color
	CommandBg   lipgloss.Color
	SidebarBg   lipgloss.Color
	DarkFg      lipgloss.Color
	MutedFg     lipgloss.Color
	TitleFg     lipgloss.Color
	SelectedFg  lipgloss.Color
	SelectedBg  lipgloss.Color
	Border      lipgloss.Color
	BorderFocus lipgloss.Color
}

// DefaultTheme returns the default Gruvbox Dark theme palette.
func DefaultTheme() Theme {
	return Theme{
		NormalBg:    lipgloss.Color("#fabd2f"),
		InsertBg:    lipgloss.Color("#b8bb26"),
		CommandBg:   lipgloss.Color("#8ec07c"),
		SidebarBg:   lipgloss.Color("#d3869b"),
		DarkFg:      lipgloss.Color("#282828"),
		MutedFg:     lipgloss.Color("#928374"),
		TitleFg:     lipgloss.Color("#ebdbb2"),
		SelectedFg:  lipgloss.Color("#fabd2f"),
		SelectedBg:  lipgloss.Color("#3c3836"),
		Border:      lipgloss.Color("#504945"),
		BorderFocus: lipgloss.Color("#fabd2f"),
	}
}
