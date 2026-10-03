// Package content renders the Markdown editor pane (<Body/>), a Vim-style
// editor. The file name lives in the tabline only.
package content

import (
	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/textarea"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"
	"regexp"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the Content component state.
type Model struct {
	theme    theme.Theme
	page     core.Page
	draft    bool
	textarea textarea.Model
	cmdInput textinput.Model
	mode     VimMode
	pending  string // first key of a two-key Vim command ("g", "d")
	slash    *slashMenu
	width    int
	height   int
	focused  bool
}

// New constructs a Content Model.
func New(th theme.Theme) Model {
	ta := textarea.New()
	ta.Placeholder = "Start writing… (i to insert)"
	ta.Prompt = ""
	ta.ShowLineNumbers = true
	ta.EndOfBufferCharacter = '~'
	ta.CharLimit = 0
	ta.MaxHeight = 0
	ta.Cursor.Style = lipgloss.NewStyle().Foreground(th.Fg)
	configureTextareaStyles(&ta, th)

	return Model{
		theme:    th,
		textarea: ta,
		cmdInput: createCommandInput(th),
		mode:     ModeNormal,
	}
}

// SetTheme switches colours.
func (m *Model) SetTheme(th theme.Theme) {
	m.theme = th
	m.textarea.Cursor.Style = lipgloss.NewStyle().Foreground(th.Fg)
	configureTextareaStyles(&m.textarea, th)
	value := m.cmdInput.Value()
	m.cmdInput = createCommandInput(th)
	m.cmdInput.SetValue(value)
	if m.mode == ModeCommand && m.focused {
		m.cmdInput.Focus()
	}
}

// Init initializes the textarea model commands.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// SetSize updates editor dimensions safely.
func (m *Model) SetSize(w, h int) {
	m.width = max(w, 0)
	m.height = max(h, 0)
	m.textarea.SetWidth(max(w-1, 10))
	m.textarea.SetHeight(max(h, 1))
}

// SetPage loads a document into the editor. draft marks an unsaved buffer
// that has no file yet.
func (m *Model) SetPage(p core.Page) {
	m.SetBuffer(p, false)
}

// SetBuffer loads a buffer's text, resetting the cursor to the top.
func (m *Model) SetBuffer(p core.Page, draft bool) {
	m.page = p
	m.draft = draft
	m.pending = ""
	m.slash = nil
	m.textarea.SetValue(p.Content)
	m.textarea.GotoTop()
}

// SetFocused sets focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if !focused {
		m.slash = nil
		m.Blur()
		if m.mode == ModeCommand {
			m.mode = ModeNormal
		}
	}
}

// Focus gives the editor keyboard focus in its current mode.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	if m.mode == ModeCommand {
		return m.cmdInput.Focus()
	}
	return m.focusTextarea()
}

func (m *Model) focusTextarea() tea.Cmd {
	cmd := m.textarea.Focus()
	cursorMode := cursor.CursorStatic
	if m.mode == ModeInsert {
		cursorMode = cursor.CursorBlink
	}
	return tea.Batch(cmd, m.textarea.Cursor.SetMode(cursorMode))
}

// Blur blurs inputs.
func (m *Model) Blur() {
	m.textarea.Blur()
	m.cmdInput.Blur()
}

// Mode returns the active Vim mode.
func (m Model) Mode() VimMode { return m.mode }

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
func (m Model) Value() string { return m.textarea.Value() }

// HasPage reports whether a document is loaded.
func (m Model) HasPage() bool { return m.page.ID != "" }

// CursorPosition returns the 1-based line and column of the cursor.
func (m Model) CursorPosition() (int, int) { return m.textarea.CursorPosition() }

// LineCount returns the number of lines in the buffer.
func (m Model) LineCount() int { return m.textarea.LineCount() }

// CommandView renders the ":" prompt for the command-line row.
func (m Model) CommandView(width int) string {
	m.cmdInput.Width = max(width-2, 1)
	return m.cmdInput.View()
}

// EnterInsert switches to INSERT mode (used by the app, e.g. for a new buffer).
func (m *Model) EnterInsert() tea.Cmd {
	m.mode = ModeInsert
	if !m.focused {
		return nil
	}
	return m.focusTextarea()
}

// ExitInsert returns to NORMAL mode (e.g. when focus leaves the editor).
func (m *Model) ExitInsert() {
	if m.mode == ModeInsert {
		m.mode = ModeNormal
	}
	m.slash = nil
}

func (m *Model) enterInsert() tea.Cmd {
	m.mode = ModeInsert
	return m.focusTextarea()
}

func (m *Model) halfPage() int { return max(m.textarea.Height()/2, 1) }

// listItem matches Markdown list and to-do lines, which Tab nests.
var listItem = regexp.MustCompile(`^\s*([-*+]|\d+[.)])\s`)

// indent handles Tab / Shift+Tab in INSERT mode: list lines are nested or
// un-nested as a whole; elsewhere Tab inserts two spaces.
func (m *Model) indent(in bool) {
	const width = 2
	switch {
	case !in:
		m.textarea.OutdentLine(width)
	case listItem.MatchString(m.textarea.CurrentLine()):
		m.textarea.IndentLine(width)
	default:
		m.textarea.InsertString("  ")
	}
	m.textarea.EnsureVisible()
}

// GotoLine puts the cursor on 1-based line n, scrolled into view.
func (m *Model) GotoLine(n int) { m.textarea.GotoLine(n) }

// ScrollBy scrolls the text by n lines, even while the pane is unfocused.
func (m *Model) ScrollBy(n int) { m.textarea.ScrollBy(n) }

// Update processes key and mouse messages (mouse coordinates relative to the
// pane's top-left corner) and handles Vim editing modes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if mouse, ok := msg.(tea.MouseMsg); ok {
		return m.handleMouse(mouse)
	}
	if m.page.ID == "" {
		return m, nil
	}

	switch m.mode {
	case ModeInsert:
		k, isKey := msg.(tea.KeyMsg)
		if isKey && m.slash != nil {
			if cmd, handled := m.updateSlash(k); handled {
				return m, cmd
			}
		}
		if isKey && (k.String() == "tab" || k.String() == "shift+tab") {
			m.indent(k.String() == "tab")
			return m, nil
		}
		if isKey && k.String() == "esc" {
			m.mode = ModeNormal
			m.textarea.CharLeft()
			return m, m.focusTextarea()
		}
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		m.textarea.EnsureVisible()
		if isKey {
			if m.slash != nil {
				m.afterSlashKey()
			} else {
				m.maybeOpenSlash(k)
			}
		}
		return m, cmd

	case ModeCommand:
		k, ok := msg.(tea.KeyMsg)
		if ok {
			switch k.String() {
			case "esc":
				m.mode = ModeNormal
				m.cmdInput.Blur()
				return m, m.focusTextarea()
			case "backspace":
				if m.cmdInput.Value() == "" {
					m.mode = ModeNormal
					m.cmdInput.Blur()
					return m, m.focusTextarea()
				}
			case "enter":
				out := parseVimCommand(m.cmdInput.Value(), m.textarea.Value())
				m.mode = ModeNormal
				m.cmdInput.Blur()
				focus := m.focusTextarea()
				if out == nil {
					return m, focus
				}
				return m, tea.Batch(focus, func() tea.Msg { return out })
			}
		}
		var cmd tea.Cmd
		m.cmdInput, cmd = m.cmdInput.Update(msg)
		return m, cmd
	}

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	return m.handleNormalKey(k)
}

func (m Model) handleNormalKey(k tea.KeyMsg) (Model, tea.Cmd) {
	ta := &m.textarea
	s := k.String()

	if m.pending != "" {
		combo := m.pending + s
		m.pending = ""
		switch combo {
		case "gg":
			ta.GotoTop()
		case "dd":
			ta.DeleteLine()
		}
		return m, nil
	}

	switch s {
	case "i":
		return m, m.enterInsert()
	case "a":
		ta.CharRight()
		return m, m.enterInsert()
	case "A":
		ta.CursorEnd()
		return m, m.enterInsert()
	case "I":
		ta.CursorStart()
		return m, m.enterInsert()
	case "o":
		ta.OpenLineBelow()
		return m, m.enterInsert()
	case "O":
		ta.OpenLineAbove()
		return m, m.enterInsert()
	case ":":
		m.mode = ModeCommand
		m.cmdInput.SetValue("")
		m.textarea.Blur()
		return m, m.cmdInput.Focus()

	case "h", "left":
		ta.CharLeft()
	case "l", "right":
		ta.CharRight()
	case "j", "down", "enter":
		ta.MoveCursorBy(1)
	case "k", "up":
		ta.MoveCursorBy(-1)
	case "w":
		ta.WordForward()
	case "b":
		ta.WordBackward()
	case "0", "^", "home":
		ta.CursorStart()
	case "$", "end":
		ta.CursorEnd()
	case "G":
		ta.GotoBottom()
	case "g", "d":
		m.pending = s
	case "ctrl+d":
		ta.ScrollBy(m.halfPage())
		ta.MoveCursorBy(m.halfPage())
	case "ctrl+u":
		ta.ScrollBy(-m.halfPage())
		ta.MoveCursorBy(-m.halfPage())
	case "pgdown", "ctrl+f":
		ta.MoveCursorBy(ta.Height())
	case "pgup", "ctrl+b":
		ta.MoveCursorBy(-ta.Height())
	case "ctrl+e":
		ta.ScrollBy(1)
	case "ctrl+y":
		ta.ScrollBy(-1)
	case "x", "delete":
		ta.DeleteCharForward()
	}
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	if m.page.ID == "" {
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.textarea.ScrollBy(-3)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.textarea.ScrollBy(3)
		return m, nil
	case tea.MouseButtonLeft:
		if m.slash != nil && msg.Action == tea.MouseActionPress {
			if cmd, hit := m.clickSlash(msg.X, msg.Y); hit {
				return m, cmd
			}
		}
		if msg.Action == tea.MouseActionPress {
			m.textarea.ClickAt(msg.X, msg.Y)
		}
	}
	return m, nil
}

// View renders the editor at exactly width × height.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	plain := lipgloss.NewStyle()

	if m.page.ID == "" {
		th := m.theme
		lines := []string{
			lipgloss.NewStyle().Foreground(th.Grey).Render("󰠮"),
			"",
			lipgloss.NewStyle().Foreground(th.GreyFg2).Bold(true).Render("No note open"),
			"",
			lipgloss.NewStyle().Foreground(th.GreyFg).Render("Ctrl+N  new note    /  search    Tab  explorer"),
		}
		block := lipgloss.JoinVertical(lipgloss.Center, lines...)
		return ui.Fit(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block), m.width, m.height, plain)
	}

	return ui.Fit(m.textarea.View(), m.width, m.height, plain)
}
