// Package theme provides application-wide design system tokens, color palettes,
// and lipgloss style constructors.
package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme encapsulates design system colors. The palette mirrors NvChad's
// base46 themes (onedark by default) so the UI reads like an NvChad session.
type Theme struct {
	Name  string
	Light bool

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
	Folder     lipgloss.Color

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

// DefaultName is the theme used when none is configured.
const DefaultName = "onedark"

// palette is one NvChad base46 "base_30" colour table.
type palette struct {
	Light      bool   `json:"light"`
	Fg         string `json:"fg"`
	DarkerBg   string `json:"darker_bg"`
	Bg         string `json:"bg"`
	Bg2        string `json:"bg2"`
	OneBg      string `json:"one_bg"`
	OneBg2     string `json:"one_bg2"`
	OneBg3     string `json:"one_bg3"`
	Grey       string `json:"grey"`
	GreyFg     string `json:"grey_fg"`
	GreyFg2    string `json:"grey_fg2"`
	LightGrey  string `json:"light_grey"`
	Red        string `json:"red"`
	Pink       string `json:"pink"`
	Line       string `json:"line"`
	Green      string `json:"green"`
	NordBlue   string `json:"nord_blue"`
	Blue       string `json:"blue"`
	Yellow     string `json:"yellow"`
	Purple     string `json:"purple"`
	DarkPurple string `json:"dark_purple"`
	Teal       string `json:"teal"`
	Orange     string `json:"orange"`
	Cyan       string `json:"cyan"`
	StatusBg   string `json:"status_bg"`
	LightBg    string `json:"light_bg"`
	Folder     string `json:"folder"`
}

// UserThemesDir returns $XDG_CONFIG_HOME/tsuzuri/themes.
func UserThemesDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "tsuzuri", "themes")
}

func loadUserThemes() map[string]palette {
	dir := UserThemesDir()
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	userThemes := make(map[string]palette)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		name := strings.TrimSuffix(strings.ToLower(e.Name()), ".json")
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var p palette
		if err := json.Unmarshal(data, &p); err == nil && p.Fg != "" && p.Bg != "" {
			userThemes[name] = p
		}
	}
	return userThemes
}

// Names lists every available theme (bundled and user-defined), sorted.
func Names() []string {
	namesMap := make(map[string]struct{}, len(palettes))
	for n := range palettes {
		namesMap[n] = struct{}{}
	}
	for n := range loadUserThemes() {
		namesMap[n] = struct{}{}
	}
	names := make([]string, 0, len(namesMap))
	for n := range namesMap {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// DefaultTheme returns the NvChad onedark theme.
func DefaultTheme() Theme {
	t, _ := Get(DefaultName)
	return t
}

// Get returns the named theme; ok is false for unknown names.
func Get(name string) (Theme, bool) {
	norm := strings.ToLower(strings.TrimSpace(name))
	p, ok := palettes[norm]
	if !ok {
		u := loadUserThemes()
		p, ok = u[norm]
		if !ok {
			return Theme{}, false
		}
	}
	c := func(hex string) lipgloss.Color { return lipgloss.Color(hex) }
	t := Theme{
		Name:  name,
		Light: p.Light,

		DarkerBg:  c(p.DarkerBg),
		Bg:        c(p.Bg),
		Bg2:       c(p.Bg2),
		OneBg:     c(p.OneBg),
		OneBg2:    c(p.OneBg2),
		OneBg3:    c(p.OneBg3),
		LightBg:   c(p.LightBg),
		StatusBg:  c(p.StatusBg),
		Line:      c(p.Line),
		Grey:      c(p.Grey),
		GreyFg:    c(p.GreyFg),
		GreyFg2:   c(p.GreyFg2),
		LightGrey: c(p.LightGrey),
		Fg:        c(p.Fg),

		Red:        c(p.Red),
		Pink:       c(p.Pink),
		Green:      c(p.Green),
		Blue:       c(p.Blue),
		NordBlue:   c(p.NordBlue),
		Yellow:     c(p.Yellow),
		Purple:     c(p.Purple),
		DarkPurple: c(p.DarkPurple),
		Teal:       c(p.Teal),
		Orange:     c(p.Orange),
		Cyan:       c(p.Cyan),
		Folder:     c(p.Folder),
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
	return t, true
}
