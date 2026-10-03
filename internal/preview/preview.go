package preview

import (
	"fmt"

	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the real-time compiled Markdown preview component.
type Model struct {
	theme      theme.Theme
	viewport   viewport.Model
	width      int
	height     int
	pageID     string
	title      string
	rawContent string
	focused    bool
	ready      bool
}

// New constructs a Preview Model.
func New(th theme.Theme) Model {
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true

	return Model{
		theme:    th,
		viewport: vp,
		title:    "Untitled",
	}
}

// SetSize updates the layout dimensions for the preview pane and viewport.
func (m *Model) SetSize(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	m.width = w
	m.height = h

	vpWidth := w - 2
	if vpWidth < 10 {
		vpWidth = 10
	}

	// 1 line for header tab
	vpHeight := h - 1
	if vpHeight < 2 {
		vpHeight = 2
	}

	m.viewport.Width = vpWidth
	m.viewport.Height = vpHeight

	if !m.ready {
		m.ready = true
	}

	// Recompile with the new width
	m.recompile()
}

// ScrollStatus returns a formatted indicator of the viewport scroll position.
func (m Model) ScrollStatus() string {
	if m.viewport.AtTop() {
		return "Top"
	}
	if m.viewport.AtBottom() {
		return "Bot"
	}
	pct := m.viewport.ScrollPercent() * 100
	return fmt.Sprintf("%3.0f%%", pct)
}

// SetPage updates both the title and content from a domain Page entity.
func (m *Model) SetPage(p core.Page) {
	m.pageID = p.ID
	m.title = p.Title
	if m.title == "" {
		m.title = "Untitled"
	}
	m.SetContent(p.Content)
}

// SetContent recompiles the raw Markdown into styled ANSI text in real time.
func (m *Model) SetContent(content string) {
	m.rawContent = content
	m.recompile()
}

// SetTitle updates the preview pane title.
func (m *Model) SetTitle(title string) {
	if title == "" {
		title = "Untitled"
	}
	m.title = title
}

// SetFocused updates whether the preview pane has keyboard focus.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// recompile parses the current rawContent with the compiler.
func (m *Model) recompile() {
	vpWidth := m.viewport.Width
	if vpWidth <= 0 {
		vpWidth = m.width - 2
	}
	compiled := Compile(m.rawContent, m.theme, vpWidth)
	m.viewport.SetContent(compiled)
}

// Update processes Bubble Tea events for scrolling the preview.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.viewport.LineDown(1)
			return m, nil
		case "k", "up":
			m.viewport.LineUp(1)
			return m, nil
		case "d", "ctrl+d":
			m.viewport.HalfPageDown()
			return m, nil
		case "u", "ctrl+u":
			m.viewport.HalfPageUp()
			return m, nil
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the compiled Markdown live preview pane.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	tabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.DarkFg).
		Background(m.theme.CommandBg).
		Padding(0, 1)

	headerTab := tabStyle.Render("󰈈 preview")

	vpView := m.viewport.View()
	if m.pageID == "" {
		placeholder := lipgloss.NewStyle().Italic(true).Foreground(m.theme.MutedFg).Render("Nothing to preview")
		vpView = lipgloss.Place(m.viewport.Width, m.viewport.Height, lipgloss.Center, lipgloss.Center, placeholder)
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, headerTab, vpView)

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		MaxHeight(m.height).
		Padding(0, 1).
		Render(inner)
}
