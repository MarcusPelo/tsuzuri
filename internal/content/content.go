// Package content renders the Markdown editor pane (<Body/>), a Vim-style
// editor. The file name lives in the tabline only.
package content

import (
	"regexp"
	"strings"

	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/highlight"
	"github.com/jaisuriya-11/tsuzuri/internal/textarea"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"

	"github.com/atotto/clipboard"
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
	hl       *hlCache // shared across copies so View can memoise
	dragging bool
	dragRow  int
	dragCol  int
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
		hl:       &hlCache{},
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
	case ModeVisual:
		return "VISUAL"
	case ModeVisualLine:
		return "V-LINE"
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

// hlCache memoises syntax colours for the last text/theme rendered.
type hlCache struct {
	text  string
	theme string
	cols  highlight.Colors
	valid bool
}

func (c *hlCache) colors(text string, th theme.Theme) highlight.Colors {
	if !c.valid || c.text != text || c.theme != th.Name {
		c.text, c.theme, c.cols, c.valid = text, th.Name, highlight.Markdown(text, th), true
	}
	return c.cols
}

// ReplaceText swaps the whole buffer text, keeping the cursor where it was
// (clamped). Used when the preview edits the note.
func (m *Model) ReplaceText(text string) {
	row, col := m.textarea.RowCol()
	m.textarea.SetValue(text)
	m.textarea.SetRowCol(row, col)
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

	if k, ok := msg.(tea.KeyMsg); ok && m.textarea.HasSelection() &&
		m.mode != ModeVisual && m.mode != ModeVisualLine {
		m.textarea.ClearSelection() // a key press drops a mouse selection
		_ = k
	}

	switch m.mode {
	case ModeVisual, ModeVisualLine:
		if k, ok := msg.(tea.KeyMsg); ok {
			return m.handleVisualKey(k)
		}
		return m, nil
	case ModeInsert:
		k, isKey := msg.(tea.KeyMsg)
		if isKey && m.slash != nil {
			if cmd, handled := m.updateSlash(k); handled {
				return m, cmd
			}
		}
		if isKey && (k.String() == "tab" || k.String() == "shift+tab") {
			if !m.nextCell(k.String() == "tab") {
				m.indent(k.String() == "tab")
			}
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
				if op, ok := tableCommands[strings.TrimSpace(m.cmdInput.Value())]; ok {
					m.mode = ModeNormal
					m.cmdInput.Blur()
					focus := m.focusTextarea()
					if !m.tableOp(op) {
						return m, tea.Batch(focus, func() tea.Msg { return core.StatusMsg{Text: "Not in a table", Error: true} })
					}
					return m, focus
				}
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
		case "yy":
			return m, copyCmd(ta.CurrentLine() + "\n")
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
	case "g", "d", "y":
		m.pending = s
	case "v", "V":
		m.mode = ModeVisual
		if s == "V" {
			m.mode = ModeVisualLine
		}
		ta.SelectLinewise = s == "V"
		ta.StartSelection()
	case "p":
		return m, m.paste()
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
		switch msg.Action {
		case tea.MouseActionPress:
			if m.mode == ModeVisual || m.mode == ModeVisualLine {
				m.mode = ModeNormal
			}
			m.textarea.ClearSelection()
			m.textarea.SelectLinewise = false
			m.textarea.ClickAt(msg.X, msg.Y)
			m.dragRow, m.dragCol = m.textarea.RowCol()
			m.dragging = true
		case tea.MouseActionMotion:
			if !m.dragging {
				break
			}
			m.textarea.ClickAt(msg.X, msg.Y)
			if r, c := m.textarea.RowCol(); !m.textarea.HasSelection() && (r != m.dragRow || c != m.dragCol) {
				m.textarea.SetAnchor(m.dragRow, m.dragCol)
			}
		case tea.MouseActionRelease:
			m.dragging = false
			if m.textarea.HasSelection() {
				// Auto-copy on release, like selecting text in Claude Code.
				return m, copyCmd(m.textarea.SelectedText())
			}
		}
	}
	return m, nil
}

func copyCmd(text string) tea.Cmd {
	if text == "" {
		return nil
	}
	return func() tea.Msg { return core.CopyMsg{Text: text} }
}

// handleVisualKey runs Vim visual mode: motions extend the selection,
// y copies, d/x cut, Esc cancels.
func (m Model) handleVisualKey(k tea.KeyMsg) (Model, tea.Cmd) {
	ta := &m.textarea
	exit := func() {
		m.mode = ModeNormal
		ta.ClearSelection()
		ta.SelectLinewise = false
	}
	switch s := k.String(); s {
	case "esc", "ctrl+c":
		exit()
		return m, nil
	case "v", "V":
		if (s == "v") == (m.mode == ModeVisual) {
			exit()
			return m, nil
		}
		m.mode = map[string]VimMode{"v": ModeVisual, "V": ModeVisualLine}[s]
		ta.SelectLinewise = s == "V"
		return m, nil
	case "y":
		text := ta.SelectedText()
		exit()
		return m, copyCmd(text)
	case "d", "x", "delete":
		text := ta.SelectedText()
		ta.DeleteSelection()
		exit()
		return m, copyCmd(text)
	case "h", "j", "k", "l", "left", "right", "up", "down", "w", "b", "0", "^", "$",
		"home", "end", "G", "g", "ctrl+d", "ctrl+u", "pgdown", "pgup":
		mode := m.mode
		m.mode = ModeNormal
		m, _ = m.handleNormalKey(k)
		m.mode = mode
	}
	return m, nil
}

// paste inserts the system clipboard after the cursor (Vim's p); text
// ending in a newline is pasted as whole lines below.
func (m *Model) paste() tea.Cmd {
	text, err := clipboard.ReadAll()
	if err != nil || text == "" {
		return func() tea.Msg { return core.StatusMsg{Text: "Clipboard is empty or unavailable", Error: err != nil} }
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if strings.HasSuffix(text, "\n") {
		m.textarea.OpenLineBelow()
		m.textarea.InsertString(strings.TrimSuffix(text, "\n"))
	} else {
		m.textarea.CharRight()
		m.textarea.InsertString(text)
	}
	m.textarea.EnsureVisible()
	return nil
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

	m.textarea.LineColors = m.hl.colors(m.textarea.Value(), m.theme)
	return ui.Fit(m.textarea.View(), m.width, m.height, plain)
}
