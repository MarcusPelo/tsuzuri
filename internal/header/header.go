// Package header renders the top application status bar (<Header/>).
package header

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"tsuzuri/internal/core"
	"tsuzuri/internal/theme"
)

// Model represents the Header component state.
type Model struct {
	theme        theme.Theme
	page         core.Page
	vimModeStr   string
	focusSidebar bool
	width        int
	height       int
	rightText    string
}

// New constructs a Header Model with default settings.
func New(th theme.Theme) Model {
	return Model{
		theme:     th,
		rightText: "[Tab] Switch Pane  [Ctrl+N] New",
	}
}

// SetSize updates layout bounds for the header.
func (m *Model) SetSize(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	m.width = w
	m.height = h
}

// SetPage sets the current active page for title rendering.
func (m *Model) SetPage(p core.Page) {
	m.page = p
}

// SetMode sets the active Vim mode badge and focus status.
func (m *Model) SetMode(modeStr string, focusSidebar bool) {
	m.vimModeStr = modeStr
	m.focusSidebar = focusSidebar
}

// Update processes Bubble Tea messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

// View renders the header string safely.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	modeStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)

	var modeTag string
	if m.focusSidebar {
		modeTag = modeStyle.Background(m.theme.SidebarBg).Foreground(m.theme.DarkFg).Render("SIDEBAR")
	} else {
		switch m.vimModeStr {
		case "INSERT":
			modeTag = modeStyle.Background(m.theme.InsertBg).Foreground(m.theme.DarkFg).Render("-- INSERT --")
		case "COMMAND":
			modeTag = modeStyle.Background(m.theme.CommandBg).Foreground(m.theme.DarkFg).Render("-- COMMAND --")
		default:
			modeTag = modeStyle.Background(m.theme.NormalBg).Foreground(m.theme.DarkFg).Render("-- NORMAL --")
		}
	}

	title := m.page.Title
	if title == "" {
		title = "Untitled"
	}
	leftText := modeTag + "  " + title

	leftWidth := lipgloss.Width(leftText)
	rightWidth := lipgloss.Width(m.rightText)
	gap := m.width - leftWidth - rightWidth - 2
	if gap < 1 {
		gap = 1
	}

	line := leftText + strings.Repeat(" ", gap) + m.rightText

	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Foreground(m.theme.MutedFg).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(m.theme.Border).
		Render(line)
}
