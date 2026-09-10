// Package sidebar renders the page list and navigation controls (<Sidebar/>).
package sidebar

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"tsuzuri/internal/core"
	"tsuzuri/internal/theme"
)

// Model represents the Sidebar component state.
type Model struct {
	theme       theme.Theme
	pages       []core.Page
	cursor      int
	focused     bool
	renaming    bool
	renameInput textinput.Model
	width       int
	height      int
}

// New constructs a Sidebar Model.
func New(th theme.Theme) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 32

	return Model{
		theme:       th,
		pages:       make([]core.Page, 0),
		cursor:      0,
		renaming:    false,
		renameInput: ti,
	}
}

// SetSize updates dimensions cleanly, clamping negative values to zero.
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

// clampCursor ensures m.cursor is always within bounds [0, len(m.pages)-1] (or 0 if empty).
func (m *Model) clampCursor() {
	if len(m.pages) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.pages) {
		m.cursor = len(m.pages) - 1
	}
}

// SetPages updates the list of pages rendered in the sidebar and clamps the selection cursor.
func (m *Model) SetPages(pages []core.Page) {
	m.pages = pages
	m.clampCursor()
}

// SetSelectedID selects the item corresponding to the given page ID.
func (m *Model) SetSelectedID(id string) {
	for i, p := range m.pages {
		if p.ID == id {
			m.cursor = i
			m.clampCursor()
			return
		}
	}
	m.clampCursor()
}

// SetFocused sets focus styling state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// IsRenaming returns whether inline renaming is active.
func (m Model) IsRenaming() bool {
	return m.renaming
}

// SelectedPage returns the currently selected page item safely.
func (m Model) SelectedPage() (core.Page, bool) {
	if len(m.pages) == 0 {
		return core.Page{}, false
	}
	if m.cursor < 0 || m.cursor >= len(m.pages) {
		return core.Page{}, false
	}
	return m.pages[m.cursor], true
}

// Update processes Bubble Tea key messages and emits core domain events safely.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.clampCursor()

	if m.renaming {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				newTitle := strings.TrimSpace(m.renameInput.Value())
				m.renaming = false
				m.renameInput.Blur()

				if newTitle != "" && len(m.pages) > 0 && m.cursor >= 0 && m.cursor < len(m.pages) {
					m.pages[m.cursor].Title = newTitle
					updatedPage := m.pages[m.cursor]
					return m, func() tea.Msg {
						return core.PageUpdatedMsg{Page: updatedPage}
					}
				}
				return m, nil
			case "esc":
				m.renaming = false
				m.renameInput.Blur()
				return m, nil
			}
		}
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.clampCursor()
				if p, ok := m.SelectedPage(); ok {
					selectedID := p.ID
					return m, func() tea.Msg { return core.PageSelectedMsg{ID: selectedID} }
				}
			}
		case "down", "j":
			if m.cursor < len(m.pages)-1 {
				m.cursor++
				m.clampCursor()
				if p, ok := m.SelectedPage(); ok {
					selectedID := p.ID
					return m, func() tea.Msg { return core.PageSelectedMsg{ID: selectedID} }
				}
			}
		case "enter":
			if p, ok := m.SelectedPage(); ok {
				m.renaming = true
				m.renameInput.SetValue(p.Title)
				m.renameInput.Focus()
				return m, textinput.Blink
			}
		case "n", "a":
			return m, func() tea.Msg {
				return core.PageCreatedMsg{Page: core.Page{Title: ""}}
			}
		case "d", "x":
			if p, ok := m.SelectedPage(); ok && len(m.pages) > 1 {
				deletedID := p.ID
				return m, func() tea.Msg { return core.PageDeletedMsg{ID: deletedID} }
			}
		}
	}

	return m, nil
}

// View renders the sidebar layout safely.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).
		Foreground(m.theme.TitleFg)
	title := titleStyle.Render("Tsuzuri")

	var b strings.Builder
	b.WriteString(title + "\n\n")

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.SelectedFg).
		Background(m.theme.SelectedBg)

	normalStyle := lipgloss.NewStyle().
		Foreground(m.theme.TitleFg)

	for i, page := range m.pages {
		displayTitle := page.Title
		if displayTitle == "" {
			displayTitle = "Untitled"
		}

		if i == m.cursor {
			if m.renaming {
				b.WriteString("> " + m.renameInput.View() + "\n")
				continue
			}
			if m.focused {
				b.WriteString(selectedStyle.Render("> "+displayTitle) + "\n")
			} else {
				b.WriteString(normalStyle.Render("• "+displayTitle) + "\n")
			}
		} else {
			b.WriteString(normalStyle.Render("  "+displayTitle) + "\n")
		}
	}

	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(m.theme.MutedFg)
	b.WriteString(helpStyle.Render("[↑/↓] Select\n[Enter] Rename\n[n] New doc\n[d] Delete\n[Tab] Switch"))

	border := m.theme.Border
	if m.focused {
		border = m.theme.BorderFocus
	}

	return lipgloss.NewStyle().
		Width(m.width-1).
		Height(m.height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(border).
		Padding(1, 1).
		Render(b.String())
}
