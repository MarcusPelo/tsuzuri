// Package theme provides application-wide design system tokens, color palettes,
// and lipgloss style constructors.
package theme

import "github.com/charmbracelet/lipgloss"

// Theme encapsulates design system colors. The palette mirrors NvChad's
// default "onedark" base46 theme so the UI reads like an NvChad session.
type Theme struct {
	// Base surfaces (darkest to lightest).
	DarkerBg  lipgloss.Color // tree / tabufline fill
	Bg        lipgloss.Color // editor background
	Bg2       lipgloss.Color // inactive tab, input fields
	OneBg     lipgloss.Color // cursorline, selection
	OneBg2    lipgloss.Color
	OneBg3    lipgloss.Color
	LightBg   lipgloss.Color // statusline file block
	StatusBg  lipgloss.Color // statusline fill
	Line      lipgloss.Color // separators, indent guides
	Grey      lipgloss.Color
	GreyFg    lipgloss.Color
	GreyFg2   lipgloss.Color
	LightGrey lipgloss.Color
	Fg        lipgloss.Color

	// Accents.
	Red        lipgloss.Color
	Pink       lipgloss.Color
	Green      lipgloss.Color
	Blue       lipgloss.Color
	NordBlue   lipgloss.Color
	Yellow     lipgloss.Color
	Purple     lipgloss.Color
	DarkPurple lipgloss.Color
	Teal       lipgloss.Color
	Orange     lipgloss.Color
	Cyan       lipgloss.Color

	// Semantic aliases kept for the Markdown compiler and older components.
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

// DefaultTheme returns the NvChad onedark palette.
func DefaultTheme() Theme {
	t := Theme{
		DarkerBg:  lipgloss.Color("#1b1f27"),
		Bg:        lipgloss.Color("#1e222a"),
		Bg2:       lipgloss.Color("#252931"),
		OneBg:     lipgloss.Color("#282c34"),
		OneBg2:    lipgloss.Color("#353b45"),
		OneBg3:    lipgloss.Color("#373b43"),
		LightBg:   lipgloss.Color("#2d3139"),
		StatusBg:  lipgloss.Color("#22262e"),
		Line:      lipgloss.Color("#31353d"),
		Grey:      lipgloss.Color("#42464e"),
		GreyFg:    lipgloss.Color("#565c64"),
		GreyFg2:   lipgloss.Color("#6f737b"),
		LightGrey: lipgloss.Color("#6f737b"),
		Fg:        lipgloss.Color("#abb2bf"),

		Red:        lipgloss.Color("#e06c75"),
		Pink:       lipgloss.Color("#ff75a0"),
		Green:      lipgloss.Color("#98c379"),
		Blue:       lipgloss.Color("#61afef"),
		NordBlue:   lipgloss.Color("#81a1c1"),
		Yellow:     lipgloss.Color("#e7c787"),
		Purple:     lipgloss.Color("#de98fd"),
		DarkPurple: lipgloss.Color("#c882e7"),
		Teal:       lipgloss.Color("#519aba"),
		Orange:     lipgloss.Color("#fca2aa"),
		Cyan:       lipgloss.Color("#a3b8ef"),
	}

	t.NormalBg = t.Blue
	t.InsertBg = t.DarkPurple
	t.CommandBg = t.Green
	t.SidebarBg = t.Purple
	t.DarkFg = t.Bg
	t.MutedFg = t.GreyFg2
	t.TitleFg = t.Fg
	t.SelectedFg = t.Blue
	t.SelectedBg = t.OneBg
	t.Border = t.Line
	t.BorderFocus = t.Blue
	return t
}
