package content

import (
	"github.com/jaisuriya-11/tsuzuri/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// maxHistory caps the undo steps kept per buffer; the oldest are dropped.
const maxHistory = 200

// snapshot is the buffer text and cursor at one point in time.
type snapshot struct {
	text     string
	row, col int
}

// History is a buffer's undo/redo stacks. Each step is a whole-text
// snapshot: notes are small, and several edits (preview actions, tables,
// front matter) replace the whole text anyway. The app keeps one History
// per tab, so switching tabs or saving keeps it.
type History struct {
	undo, redo []snapshot
	insert     *snapshot // state when the open INSERT session began
	moved      bool      // the last key was u / Ctrl+R, not a new change
}

// NewHistory returns an empty history.
func NewHistory() *History { return &History{} }

// push records s as the state before a change, dropping the redo stack.
func (h *History) push(s snapshot) {
	if n := len(h.undo); n > 0 && h.undo[n-1].text == s.text {
		return
	}
	h.undo = append(h.undo, s)
	if len(h.undo) > maxHistory {
		h.undo = h.undo[len(h.undo)-maxHistory:]
	}
	h.redo = nil
}

// inInsert reports whether an INSERT session is open; its edits become one
// step when it ends.
func (h *History) inInsert() bool { return h.insert != nil }

// beginInsert opens an INSERT session that started from s.
func (h *History) beginInsert(s snapshot) {
	if h.insert == nil {
		h.insert = &s
	}
}

// endInsert closes the INSERT session, recording it as one step if the text
// changed.
func (h *History) endInsert(cur string) {
	if h.insert != nil && h.insert.text != cur {
		h.push(*h.insert)
	}
	h.insert = nil
}

// step pops the state to restore from from and moves the current text onto
// to. Both keep the popped cursor, which sits where the change was made, so
// undo and redo each land the cursor on the change.
func step(from, to *[]snapshot, text string) (snapshot, bool) {
	n := len(*from)
	if n == 0 {
		return snapshot{}, false
	}
	s := (*from)[n-1]
	*from = (*from)[:n-1]
	*to = append(*to, snapshot{text: text, row: s.row, col: s.col})
	return s, true
}

func (m *Model) snapshot() snapshot {
	row, col := m.textarea.RowCol()
	return snapshot{text: m.textarea.Value(), row: row, col: col}
}

func (m *Model) restore(s snapshot) {
	m.textarea.SetValue(s.text)
	m.textarea.SetRowCol(s.row, s.col)
	m.textarea.EnsureVisible()
}

// track runs an edit and records it as one undo step if it changed the
// text. Inside an INSERT session the edit joins that session instead.
func (m *Model) track(edit func()) {
	before := m.snapshot()
	edit()
	if !m.hist.inInsert() && m.textarea.Value() != before.text {
		m.hist.push(before)
	}
}

// undo restores the state before the last change (Vim's u).
func (m *Model) undo() tea.Cmd {
	s, ok := step(&m.hist.undo, &m.hist.redo, m.textarea.Value())
	if !ok {
		return statusCmd("Already at oldest change")
	}
	m.restore(s)
	m.hist.moved = true
	return nil
}

// redo re-applies the last undone change (Vim's Ctrl+R).
func (m *Model) redo() tea.Cmd {
	s, ok := step(&m.hist.redo, &m.hist.undo, m.textarea.Value())
	if !ok {
		return statusCmd("Already at newest change")
	}
	m.restore(s)
	m.hist.moved = true
	return nil
}

func statusCmd(text string) tea.Cmd {
	return func() tea.Msg { return core.StatusMsg{Text: text} }
}
