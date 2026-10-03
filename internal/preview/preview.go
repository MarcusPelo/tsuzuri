package preview

import (
	"fmt"
	"strings"

	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"

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
	baseDir    string
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
	m.width = max(w, 0)
	m.height = max(h, 0)

	// One column of padding on each side.
	m.viewport.Width = max(w-2, 10)
	m.viewport.Height = max(h, 1)
	m.ready = true
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

// SetBaseDir sets the folder the note lives in (for relative image paths).
func (m *Model) SetBaseDir(dir string) {
	if dir != m.baseDir {
		m.baseDir = dir
		m.recompile()
	}
}

// SetTheme switches colours and re-renders.
func (m *Model) SetTheme(th theme.Theme) {
	m.theme = th
	m.recompile()
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
	compiled := CompileIn(m.rawContent, m.theme, vpWidth, m.baseDir)
	m.viewport.SetContent(compiled)
}

// ScrollBy scrolls the preview by n lines (negative scrolls up).
func (m *Model) ScrollBy(n int) {
	if n > 0 {
		m.viewport.ScrollDown(n)
	} else {
		m.viewport.ScrollUp(-n)
	}
}

// Update processes key and mouse events for scrolling the preview.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.ScrollBy(-3)
		case tea.MouseButtonWheelDown:
			m.ScrollBy(3)
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.viewport.ScrollDown(1)
		case "k", "up":
			m.viewport.ScrollUp(1)
		case "d", "ctrl+d":
			m.viewport.HalfPageDown()
		case "u", "ctrl+u":
			m.viewport.HalfPageUp()
		case "pgdown", "ctrl+f", " ":
			m.viewport.PageDown()
		case "pgup", "ctrl+b":
			m.viewport.PageUp()
		case "g", "home":
			m.viewport.GotoTop()
		case "G", "end":
			m.viewport.GotoBottom()
		}
	}
	return m, nil
}

// View renders the compiled Markdown preview at exactly width × height.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	th := m.theme
	plain := lipgloss.NewStyle()

	var body string
	if m.pageID == "" {
		msg := lipgloss.NewStyle().Foreground(th.GreyFg).Render("Nothing to preview")
		body = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	} else {
		body = ui.Fit(m.viewport.View(), m.width-2, m.height, plain)
		lines := strings.Split(body, "\n")
		for i, l := range lines {
			lines[i] = " " + l + " "
		}
		body = strings.Join(lines, "\n")
	}
	return ui.Fit(body, m.width, m.height, plain)
}
