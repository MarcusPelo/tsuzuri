// Package sidebar renders the Notion-style categorized navigation sidebar (<Sidebar/>).
package sidebar

import (
	"strings"

	"tsuzuri/internal/core"
	"tsuzuri/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TreeItem represents a flattened visible node in the sidebar tree.
type TreeItem struct {
	Page        core.Page
	Level       int
	HasChildren bool
	Expanded    bool
	IsRecent    bool
}

// Model represents the Sidebar component state.
type Model struct {
	theme       theme.Theme
	pages       []core.Page
	expanded    map[string]bool
	cursor      int
	focused     bool
	renaming    bool
	renameInput textinput.Model
	searching   bool
	searchInput textinput.Model
	width       int
	height      int
}

// New constructs a Notion-style Sidebar Model.
func New(th theme.Theme) Model {
	ti := textinput.New()
	ti.Prompt = "󰏫  "
	ti.Placeholder = "Page title..."
	ti.CharLimit = 64
	ti.Width = 24

	si := textinput.New()
	si.Prompt = "  "
	si.Placeholder = "Search"
	si.CharLimit = 32

	return Model{
		theme:       th,
		pages:       make([]core.Page, 0),
		expanded:    make(map[string]bool),
		cursor:      0,
		renaming:    false,
		renameInput: ti,
		searching:   false,
		searchInput: si,
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

// isExpanded returns whether a page ID folder is expanded (defaults to true).
func (m Model) isExpanded(id string) bool {
	exp, ok := m.expanded[id]
	if !ok {
		return true // Default folders to expanded
	}
	return exp
}

// VisibleItems constructs the flattened, filtered list of items displayed in the sidebar.
func (m Model) VisibleItems() []TreeItem {
	var items []TreeItem

	// Map of parentID -> children
	childrenMap := make(map[string][]core.Page)
	var rootPages []core.Page

	for _, p := range m.pages {
		if p.ParentID == "" {
			rootPages = append(rootPages, p)
		} else {
			childrenMap[p.ParentID] = append(childrenMap[p.ParentID], p)
		}
	}

	searchQuery := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))

	// Recursive tree flattener
	var appendTree func(page core.Page, level int)
	appendTree = func(page core.Page, level int) {
		children := childrenMap[page.ID]
		hasChildren := len(children) > 0
		expanded := m.isExpanded(page.ID)

		if searchQuery == "" || strings.Contains(strings.ToLower(page.Title), searchQuery) {
			items = append(items, TreeItem{
				Page:        page,
				Level:       level,
				HasChildren: hasChildren,
				Expanded:    expanded,
			})
		}

		if hasChildren && (expanded || searchQuery != "") {
			for _, child := range children {
				appendTree(child, level+1)
			}
		}
	}

	for _, root := range rootPages {
		appendTree(root, 0)
	}

	return items
}

// clampCursor ensures m.cursor is always within bounds.
func (m *Model) clampCursor() {
	items := m.VisibleItems()
	if len(items) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(items) {
		m.cursor = len(items) - 1
	}
}

// SetPages updates the list of pages and clamps the selection cursor.
func (m *Model) SetPages(pages []core.Page) {
	m.pages = pages
	m.clampCursor()
}

// SetSelectedID selects the item corresponding to the given page ID.
func (m *Model) SetSelectedID(id string) {
	items := m.VisibleItems()
	for i, item := range items {
		if item.Page.ID == id {
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
	if !focused {
		m.searching = false
		m.searchInput.Blur()
		m.renaming = false
		m.renameInput.Blur()
	}
}

// IsRenaming returns whether inline renaming is active.
func (m Model) IsRenaming() bool {
	return m.renaming
}

// IsSearching returns whether search input is active.
func (m Model) IsSearching() bool {
	return m.searching
}

// SelectedPage returns the currently selected page item safely.
func (m Model) SelectedPage() (core.Page, bool) {
	items := m.VisibleItems()
	if len(items) == 0 || m.cursor < 0 || m.cursor >= len(items) {
		return core.Page{}, false
	}
	return items[m.cursor].Page, true
}

// StartRenaming activates inline title editing for the currently selected item.
func (m *Model) StartRenaming() tea.Cmd {
	if p, ok := m.SelectedPage(); ok {
		m.renaming = true
		m.renameInput.SetValue(p.Title)
		return m.renameInput.Focus()
	}
	return nil
}

// Update processes Bubble Tea key messages and emits core domain events.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.clampCursor()

	// Handle Inline Renaming Mode
	if m.renaming {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				newTitle := strings.TrimSpace(m.renameInput.Value())
				m.renaming = false
				m.renameInput.Blur()

				if newTitle != "" {
					if p, ok := m.SelectedPage(); ok {
						p.Title = newTitle
						return m, func() tea.Msg {
							return core.PageUpdatedMsg{Page: p}
						}
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

	// Handle Search Input Mode
	if m.searching {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter", "esc":
				m.searching = false
				m.searchInput.Blur()
				m.clampCursor()
				if p, ok := m.SelectedPage(); ok {
					selectedID := p.ID
					return m, func() tea.Msg { return core.PageSelectedMsg{ID: selectedID} }
				}
				return m, nil
			}
		}
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.clampCursor()
		return m, cmd
	}

	items := m.VisibleItems()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			// Open Search
			m.searching = true
			m.searchInput.SetValue("")
			m.searchInput.Focus()
			return m, textinput.Blink

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
			if m.cursor < len(items)-1 {
				m.cursor++
				m.clampCursor()
				if p, ok := m.SelectedPage(); ok {
					selectedID := p.ID
					return m, func() tea.Msg { return core.PageSelectedMsg{ID: selectedID} }
				}
			}

		case "enter":
			if len(items) > 0 && m.cursor >= 0 && m.cursor < len(items) {
				curr := items[m.cursor]
				if curr.HasChildren {
					m.expanded[curr.Page.ID] = !m.isExpanded(curr.Page.ID)
				}
				selectedID := curr.Page.ID
				return m, func() tea.Msg { return core.PageSelectedMsg{ID: selectedID} }
			}

		case "r", "e":
			// Inline edit / rename
			return m, m.StartRenaming()

		case "z", "space":
			// Toggle folder expand/collapse
			if len(items) > 0 && m.cursor >= 0 && m.cursor < len(items) {
				curr := items[m.cursor]
				if curr.HasChildren {
					m.expanded[curr.Page.ID] = !m.isExpanded(curr.Page.ID)
					return m, nil
				}
			}

		case "l", "right":
			if len(items) > 0 && m.cursor >= 0 && m.cursor < len(items) {
				curr := items[m.cursor]
				if curr.HasChildren && !m.isExpanded(curr.Page.ID) {
					m.expanded[curr.Page.ID] = true
					return m, nil
				}
			}

		case "h", "left":
			if len(items) > 0 && m.cursor >= 0 && m.cursor < len(items) {
				curr := items[m.cursor]
				if curr.HasChildren && m.isExpanded(curr.Page.ID) {
					m.expanded[curr.Page.ID] = false
					return m, nil
				}
			}

		case "n":
			// Create new top-level page
			return m, func() tea.Msg {
				return core.PageCreatedMsg{Page: core.Page{Title: ""}}
			}

		case "a":
			// Create new sub-page under current selection
			if p, ok := m.SelectedPage(); ok {
				parentID := p.ID
				m.expanded[parentID] = true // ensure parent is expanded
				return m, func() tea.Msg {
					return core.PageCreatedMsg{Page: core.Page{Title: "", ParentID: parentID}}
				}
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

// getPageIcon returns the consistent Nerd Font document icon for a page.
func getPageIcon(p core.Page, index int) string {
	if p.Icon != "" {
		return p.Icon
	}
	return ""
}

// View renders the NvChad-style sidebar (Image 2 style) without workspace bar clutter.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	var b strings.Builder

	// 1. Search Bar at the very top (wrapped in a rounded box)
	boxContentWidth := m.width - 5
	if boxContentWidth < 10 {
		boxContentWidth = 10
	}

	searchBorderColor := m.theme.Border
	if m.searching {
		searchBorderColor = m.theme.BorderFocus
	}

	boxStyle := lipgloss.NewStyle().
		Width(boxContentWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(searchBorderColor)

	var searchBox string
	if m.searching {
		m.searchInput.Width = boxContentWidth - 3
		searchBox = boxStyle.Render(" " + m.searchInput.View())
	} else {
		searchIcon := lipgloss.NewStyle().Foreground(m.theme.NormalBg).Render("")
		searchLabel := lipgloss.NewStyle().Foreground(m.theme.MutedFg).Render("Search")
		slashHint := lipgloss.NewStyle().Foreground(m.theme.Border).Render("/")

		leftText := " " + searchIcon + " " + searchLabel
		rightText := slashHint + " "
		leftW := lipgloss.Width(leftText)
		rightW := lipgloss.Width(rightText)
		gap := boxContentWidth - leftW - rightW
		if gap < 1 {
			gap = 1
		}
		inner := leftText + strings.Repeat(" ", gap) + rightText
		searchBox = boxStyle.Render(inner)
	}
	b.WriteString(" " + searchBox + "\n\n")

	// 2. File Tree (NvChad Image 2 style: ›  assets / ⌵  src /    file)
	items := m.VisibleItems()
	normalStyle := lipgloss.NewStyle().Foreground(m.theme.TitleFg)

	rowWidth := m.width - 2
	if rowWidth < 10 {
		rowWidth = 10
	}

	for i, item := range items {
		displayTitle := item.Page.Title
		if displayTitle == "" {
			displayTitle = "Untitled"
		}

		// Indentation for nested items (2 spaces per level)
		indent := strings.Repeat("  ", item.Level)

		if i == m.cursor {
			if m.renaming {
				b.WriteString(" " + indent + "  ❯ " + m.renameInput.View() + "\n")
				continue
			}
			if m.focused {
				var prefix, iconPart, titlePart string
				if item.HasChildren {
					folderIcon := " "
					chevron := " "
					if item.Expanded {
						folderIcon = " "
						chevron = " "
					}
					prefix = lipgloss.NewStyle().Foreground(m.theme.SelectedFg).Background(m.theme.SelectedBg).Render(" " + indent + chevron)
					iconPart = lipgloss.NewStyle().Foreground(m.theme.SidebarBg).Background(m.theme.SelectedBg).Render(folderIcon)
					titlePart = lipgloss.NewStyle().Bold(true).Foreground(m.theme.SelectedFg).Background(m.theme.SelectedBg).Render(displayTitle)
				} else {
					fileIcon := getPageIcon(item.Page, i)
					prefix = lipgloss.NewStyle().Background(m.theme.SelectedBg).Render(" " + indent + "  ")
					iconPart = lipgloss.NewStyle().Foreground(m.theme.NormalBg).Background(m.theme.SelectedBg).Render(fileIcon + " ")
					titlePart = lipgloss.NewStyle().Bold(true).Foreground(m.theme.SelectedFg).Background(m.theme.SelectedBg).Render(displayTitle)
				}

				line := prefix + iconPart + titlePart
				lineW := lipgloss.Width(line)
				pad := rowWidth - lineW
				if pad > 0 {
					line += lipgloss.NewStyle().Background(m.theme.SelectedBg).Render(strings.Repeat(" ", pad))
				}
				b.WriteString(line + "\n")
				continue
			}
		}

		// Unfocused or unselected row
		var row string
		if item.HasChildren {
			folderIcon := " "
			chevron := " "
			if item.Expanded {
				folderIcon = " "
				chevron = " "
			}
			folderStyle := lipgloss.NewStyle().Foreground(m.theme.SidebarBg)
			row = " " + indent + chevron + folderStyle.Render(folderIcon) + displayTitle
		} else {
			fileIcon := getPageIcon(item.Page, i)
			iconStyle := lipgloss.NewStyle().Foreground(m.theme.NormalBg)
			row = " " + indent + "  " + iconStyle.Render(fileIcon+" ") + displayTitle
		}
		b.WriteString(normalStyle.Render(row) + "\n")
	}

	// 3. Clean Action Buttons at bottom (compact, never wraps)
	b.WriteString("\n")
	btnStyle := lipgloss.NewStyle().Foreground(m.theme.MutedFg)
	helperText := "󰏫 n new    a sub"
	if m.width >= 32 {
		helperText = "󰏫 n: new    a: sub   󰌌 r: edit"
	}
	b.WriteString(" " + btnStyle.Render(helperText) + "\n")

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
		Padding(1, 0).
		Render(b.String())
}
