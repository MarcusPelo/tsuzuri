// Package content renders the active page view and Markdown editor (<Body/>).
package content

import (
	"strings"

	"tsuzuri/internal/core"
	"tsuzuri/internal/theme"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the Content component state.
type Model struct {
	theme         theme.Theme
	pages         []core.Page
	page          core.Page
	textarea      textarea.Model
	cmdInput      textinput.Model
	mode          VimMode
	statusMessage string
	width         int
	height        int
	focused       bool
}

// New constructs a Content Model.
func New(th theme.Theme) Model {
	ta := textarea.New()
	ta.Placeholder = "Type Markdown text here..."
	ta.Prompt = ""
	ta.ShowLineNumbers = true
	ta.EndOfBufferCharacter = ' '
	ta.CharLimit = 0

	configureTextareaStyles(&ta, th)
	cmdInput := createCommandInput(th)

	return Model{
		theme:    th,
		textarea: ta,
		cmdInput: cmdInput,
		mode:     ModeNormal,
	}
}

// Init initializes the textarea model commands.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// SetSize updates editor dimensions safely.
func (m *Model) SetSize(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	m.width = w
	m.height = h

	taWidth := w - 4
	if taWidth < 10 {
		taWidth = 10
	}
	taHeight := h - 2
	if taHeight < 2 {
		taHeight = 2
	}
	m.textarea.SetWidth(taWidth)
	m.textarea.SetHeight(taHeight)
}

// SetPage sets the page content to be edited.
func (m *Model) SetPage(p core.Page) {
	m.page = p
	m.textarea.SetValue(p.Content)
}

// SetPages updates the list of open buffer pages for the tabufline.
func (m *Model) SetPages(pages []core.Page) {
	m.pages = pages
}

// SetFocused sets focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if !focused {
		m.Blur()
	}
}

// Focus focuses input depending on active mode.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	if m.mode == ModeInsert {
		return m.textarea.Focus()
	} else if m.mode == ModeCommand {
		return m.cmdInput.Focus()
	}
	return nil
}

// Blur blurs inputs.
func (m *Model) Blur() {
	m.textarea.Blur()
	m.cmdInput.Blur()
}

// Mode returns the active Vim mode.
func (m Model) Mode() VimMode {
	return m.mode
}

// ModeString returns the string representation of the active mode.
func (m Model) ModeString() string {
	switch m.mode {
	case ModeInsert:
		return "INSERT"
	case ModeCommand:
		return "COMMAND"
	default:
		return "NORMAL"
	}
}

// Value returns the current textarea content.
func (m Model) Value() string {
	return m.textarea.Value()
}

// Update processes Bubble Tea key events and handles Vim editing modes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.mode {
	case ModeNormal:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			m.statusMessage = ""
			switch msg.String() {
			case "i", "a":
				m.mode = ModeInsert
				m.textarea.Focus()
				return m, textarea.Blink
			case ":":
				m.mode = ModeCommand
				m.cmdInput.SetValue("")
				m.cmdInput.Focus()
				return m, textinput.Blink
			case "j", "down":
				m.textarea.CursorDown()
				return m, nil
			case "k", "up":
				m.textarea.CursorUp()
				return m, nil
			}
		}

	case ModeInsert:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "esc" {
				m.mode = ModeNormal
				m.textarea.Blur()
				return m, nil
			}
		}
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	case ModeCommand:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.mode = ModeNormal
				m.cmdInput.Blur()
				return m, nil
			case "enter":
				saveMsg, quitMsg, statusMsg, isSave, isQuit := parseVimCommand(m.cmdInput.Value())
				m.mode = ModeNormal
				m.cmdInput.Blur()
				m.statusMessage = statusMsg

				var cmds []tea.Cmd
				if isSave {
					contentVal := m.textarea.Value()
					saveMsg.Content = contentVal
					cmds = append(cmds, func() tea.Msg { return saveMsg })
				}
				if isQuit {
					cmds = append(cmds, func() tea.Msg { return quitMsg })
				}
				return m, tea.Batch(cmds...)
			}
		}
		m.cmdInput, cmd = m.cmdInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the content editor layout safely with NvChad-style tabufline across the top.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	m.SetSize(m.width, m.height)

	// Build NvChad-style tabufline across the top of the editor (NO x close buttons)
	var tabs []string
	pagesToRender := m.pages
	if len(pagesToRender) == 0 && m.page.ID != "" {
		pagesToRender = []core.Page{m.page}
	}

	tabActiveStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.DarkFg).
		Background(m.theme.SidebarBg).
		Padding(0, 1)

	tabInactiveStyle := lipgloss.NewStyle().
		Foreground(m.theme.MutedFg).
		Background(lipgloss.Color("#181b21")).
		Padding(0, 1)

	currentTotalWidth := 0
	maxTabsWidth := m.width - 4
	if maxTabsWidth < 10 {
		maxTabsWidth = 10
	}

	for _, p := range pagesToRender {
		tabTitle := p.Title
		if tabTitle == "" {
			tabTitle = "Untitled"
		}
		if !strings.HasSuffix(tabTitle, ".md") {
			tabTitle = tabTitle + ".md"
		}

		var tabStr string
		if p.ID == m.page.ID {
			dirty := ""
			if m.textarea.Value() != m.page.Content {
				dirty = " ●"
			}
			tabStr = tabActiveStyle.Render(" " + tabTitle + dirty)
		} else {
			tabStr = tabInactiveStyle.Render(" " + tabTitle)
		}

		tabW := lipgloss.Width(tabStr)
		if currentTotalWidth+tabW > maxTabsWidth && len(tabs) > 0 {
			break
		}
		tabs = append(tabs, tabStr)
		currentTotalWidth += tabW + 1
	}

	var titleView string
	if len(tabs) > 0 {
		titleView = strings.Join(tabs, " ")
	} else {
		title := m.page.Title
		if title == "" {
			title = "Untitled"
		}
		if !strings.HasSuffix(title, ".md") {
			title = title + ".md"
		}
		titleView = tabActiveStyle.Render(" " + title)
	}

	editorView := m.textarea.View()

	var inner string
	if m.mode == ModeCommand {
		inner = lipgloss.JoinVertical(lipgloss.Left, titleView, editorView, m.cmdInput.View())
	} else if m.statusMessage != "" {
		statusMsg := lipgloss.NewStyle().Foreground(m.theme.NormalBg).Render(m.statusMessage)
		inner = lipgloss.JoinVertical(lipgloss.Left, titleView, editorView, statusMsg)
	} else {
		inner = lipgloss.JoinVertical(lipgloss.Left, titleView, editorView)
	}

	border := m.theme.Border
	if m.focused {
		border = m.theme.BorderFocus
	}

	return lipgloss.NewStyle().
		Width(m.width-1).
		Height(m.height).
		MaxHeight(m.height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(border).
		Padding(0, 1).
		Render(inner)
}
