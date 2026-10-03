// Package sidebar renders the nvim-tree style workspace explorer (<Sidebar/>).
package sidebar

import (
	"path"
	"strings"

	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Rows above the tree: workspace title, search field, spacer.
const treeTop = 3

// TreeItem represents a flattened visible node in the sidebar tree.
type TreeItem struct {
	Page        core.Page
	Level       int
	HasChildren bool
	Expanded    bool
	// IsLast reports whether this is the last child of its parent; Guides[i]
	// reports whether ancestor level i still has siblings below (draw "│").
	IsLast bool
	Guides []bool
}

// Model represents the Sidebar component state.
type Model struct {
	theme       theme.Theme
	pages       []core.Page
	byID        map[string]core.Page
	expanded    map[string]bool
	cursor      int
	offset      int
	focused     bool
	renaming    bool
	renameInput textinput.Model
	searching   bool
	searchInput textinput.Model
	workspace   string
	activeID    string
	modified    map[string]bool
	width       int
	height      int
}

// New constructs the explorer.
func New(th theme.Theme) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "name"
	ti.CharLimit = 128

	si := textinput.New()
	si.Prompt = ""
	si.Placeholder = "Search notes"
	si.CharLimit = 64

	return Model{
		theme:       th,
		byID:        map[string]core.Page{},
		expanded:    map[string]bool{},
		modified:    map[string]bool{},
		renameInput: ti,
		searchInput: si,
		workspace:   "workspace",
	}
}

// SetSize updates dimensions cleanly, clamping negative values to zero.
func (m *Model) SetSize(w, h int) {
	m.width = max(w, 0)
	m.height = max(h, 0)
	m.ensureVisible()
}

// SetWorkspaceName sets the label shown above the tree.
func (m *Model) SetWorkspaceName(name string) {
	if name != "" {
		m.workspace = name
	}
}

// SetActiveID marks the page shown in the editor.
func (m *Model) SetActiveID(id string) { m.activeID = id }

// SetModified marks which pages have unsaved edits.
func (m *Model) SetModified(ids map[string]bool) {
	if ids == nil {
		ids = map[string]bool{}
	}
	m.modified = ids
}

func (m Model) isExpanded(id string) bool { return m.expanded[id] }

func (m Model) query() string {
	return strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
}

// VisibleItems constructs the flattened, filtered list of rows displayed.
func (m Model) VisibleItems() []TreeItem {
	if q := m.query(); q != "" {
		return m.searchResults(q)
	}

	children := make(map[string][]core.Page)
	var roots []core.Page
	for _, p := range m.pages {
		if p.ParentID == "" {
			roots = append(roots, p)
		} else {
			children[p.ParentID] = append(children[p.ParentID], p)
		}
	}

	var items []TreeItem
	var walk func(list []core.Page, level int, guides []bool)
	walk = func(list []core.Page, level int, guides []bool) {
		for i, p := range list {
			kids := children[p.ID]
			last := i == len(list)-1
			exp := m.isExpanded(p.ID)
			items = append(items, TreeItem{
				Page:        p,
				Level:       level,
				HasChildren: len(kids) > 0,
				Expanded:    exp,
				IsLast:      last,
				Guides:      append([]bool(nil), guides...),
			})
			if len(kids) > 0 && exp {
				walk(kids, level+1, append(guides, !last))
			}
		}
	}
	walk(roots, 0, nil)
	return items
}

// searchResults lists every note whose path contains all query terms.
func (m Model) searchResults(q string) []TreeItem {
	terms := strings.Fields(q)
	var items []TreeItem
	for _, p := range m.pages {
		if p.IsFolder {
			continue
		}
		hay := strings.ToLower(p.ID + " " + p.Title)
		ok := true
		for _, t := range terms {
			if !strings.Contains(hay, t) {
				ok = false
				break
			}
		}
		if ok {
			items = append(items, TreeItem{Page: p})
		}
	}
	return items
}

func (m *Model) clampCursor() {
	n := len(m.VisibleItems())
	if n == 0 {
		m.cursor = 0
		return
	}
	m.cursor = max(0, min(m.cursor, n-1))
}

func (m Model) treeHeight() int { return max(m.height-treeTop, 1) }

// ensureVisible scrolls so the cursor row is on screen.
func (m *Model) ensureVisible() {
	m.clampCursor()
	h := m.treeHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	n := len(m.VisibleItems())
	m.offset = max(0, min(m.offset, max(0, n-h)))
}

// SetPages updates the list of pages and clamps the selection cursor.
func (m *Model) SetPages(pages []core.Page) {
	m.pages = pages
	m.byID = make(map[string]core.Page, len(pages))
	for _, p := range pages {
		m.byID[p.ID] = p
	}
	m.ensureVisible()
}

// SetSelectedID moves the cursor to id, expanding its ancestors (like
// nvim-tree's "find file") so the row is visible.
func (m *Model) SetSelectedID(id string) {
	for p, ok := m.byID[id]; ok && p.ParentID != ""; p, ok = m.byID[p.ParentID] {
		m.expanded[p.ParentID] = true
	}
	for i, item := range m.VisibleItems() {
		if item.Page.ID == id {
			m.cursor = i
			break
		}
	}
	m.ensureVisible()
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
func (m Model) IsRenaming() bool { return m.renaming }

// IsSearching returns whether the search field has focus.
func (m Model) IsSearching() bool { return m.searching }

// IsBusy reports whether the explorer is capturing typed text.
func (m Model) IsBusy() bool { return m.renaming || m.searching }

// SelectedPage returns the currently selected page item safely.
func (m Model) SelectedPage() (core.Page, bool) {
	items := m.VisibleItems()
	if m.cursor < 0 || m.cursor >= len(items) {
		return core.Page{}, false
	}
	return items[m.cursor].Page, true
}

// ContextParentID is the parent a new note should go under: the selected
// folder itself, or the folder holding the selected note.
func (m Model) ContextParentID() string {
	p, ok := m.SelectedPage()
	if !ok {
		return ""
	}
	if p.IsFolder {
		return p.ID
	}
	return p.ParentID
}

// StartRenaming activates inline title editing for the currently selected item.
func (m *Model) StartRenaming() tea.Cmd {
	p, ok := m.SelectedPage()
	if !ok {
		return nil
	}
	m.renaming = true
	m.renameInput.SetValue(p.Title)
	m.renameInput.CursorEnd()
	return m.renameInput.Focus()
}

// StartSearch focuses the search field.
func (m *Model) StartSearch() tea.Cmd {
	m.searching = true
	m.cursor = 0
	m.offset = 0
	return m.searchInput.Focus()
}

func (m *Model) stopSearch(clear bool) {
	m.searching = false
	m.searchInput.Blur()
	if clear {
		m.searchInput.SetValue("")
	}
	m.ensureVisible()
}

func selectCmd(id string) tea.Cmd {
	return func() tea.Msg { return core.PageSelectedMsg{ID: id} }
}

// activate is nvim-tree's "edit" action: toggle a folder, open a note.
func (m *Model) activate(item TreeItem) tea.Cmd {
	if item.HasChildren || item.Page.IsFolder {
		if item.HasChildren {
			m.expanded[item.Page.ID] = !m.isExpanded(item.Page.ID)
			m.ensureVisible()
		}
		return nil
	}
	return selectCmd(item.Page.ID)
}

// Scroll moves the view by n rows, keeping the cursor on screen.
func (m *Model) Scroll(n int) {
	items := len(m.VisibleItems())
	h := m.treeHeight()
	m.offset = max(0, min(m.offset+n, max(0, items-h)))
	if m.cursor < m.offset {
		m.cursor = m.offset
	}
	if m.cursor >= m.offset+h {
		m.cursor = m.offset + h - 1
	}
	m.clampCursor()
}

// Update processes key and mouse messages (mouse coordinates must be relative
// to the sidebar's top-left corner) and emits core domain events.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if mouse, ok := msg.(tea.MouseMsg); ok {
		return m.handleMouse(mouse)
	}

	var cmd tea.Cmd
	key, isKey := msg.(tea.KeyMsg)

	if m.renaming {
		if isKey {
			switch key.String() {
			case "enter":
				newTitle := strings.TrimSpace(m.renameInput.Value())
				m.renaming = false
				m.renameInput.Blur()
				if p, ok := m.SelectedPage(); ok && newTitle != "" {
					newTitle = strings.TrimSuffix(newTitle, ".md")
					p.Title = newTitle
					return m, func() tea.Msg { return core.PageUpdatedMsg{Page: p} }
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

	if m.searching {
		if isKey {
			items := m.VisibleItems()
			switch key.String() {
			case "esc":
				m.stopSearch(true)
				return m, nil
			case "enter":
				m.clampCursor()
				var open tea.Cmd
				if m.cursor < len(items) {
					open = selectCmd(items[m.cursor].Page.ID)
				}
				id := ""
				if open != nil {
					id = items[m.cursor].Page.ID
				}
				m.stopSearch(true)
				if id != "" {
					m.SetSelectedID(id)
				}
				return m, open
			case "down", "ctrl+n", "ctrl+j":
				m.cursor++
				m.ensureVisible()
				return m, nil
			case "up", "ctrl+p", "ctrl+k":
				m.cursor--
				m.ensureVisible()
				return m, nil
			}
		}
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.cursor = 0
		m.ensureVisible()
		return m, cmd
	}

	if !isKey {
		return m, nil
	}

	items := m.VisibleItems()
	var curr TreeItem
	hasCurr := m.cursor >= 0 && m.cursor < len(items)
	if hasCurr {
		curr = items[m.cursor]
	}

	switch key.String() {
	case "/":
		return m, m.StartSearch()

	case "up", "k":
		m.cursor--
	case "down", "j":
		m.cursor++
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(items) - 1
	case "ctrl+d", "pgdown":
		m.cursor += m.treeHeight() / 2
	case "ctrl+u", "pgup":
		m.cursor -= m.treeHeight() / 2

	case "enter", "o":
		if hasCurr {
			if key.String() == "o" && !curr.Page.IsFolder {
				return m, selectCmd(curr.Page.ID)
			}
			return m, m.activate(curr)
		}
	case "l", "right":
		if hasCurr {
			if curr.HasChildren && !curr.Expanded {
				m.expanded[curr.Page.ID] = true
			} else if !curr.HasChildren && !curr.Page.IsFolder {
				return m, selectCmd(curr.Page.ID)
			}
		}
	case "h", "left":
		if hasCurr {
			if curr.HasChildren && curr.Expanded {
				m.expanded[curr.Page.ID] = false
			} else if curr.Page.ParentID != "" {
				// Jump to the parent, like nvim-tree's close-node.
				for i, it := range items {
					if it.Page.ID == curr.Page.ParentID {
						m.cursor = i
						break
					}
				}
			}
		}
	case "z", " ", "space", "tab":
		if hasCurr && curr.HasChildren {
			m.expanded[curr.Page.ID] = !curr.Expanded
		}
	case "W":
		m.expanded = map[string]bool{}

	case "r", "e":
		return m, m.StartRenaming()
	case "n", "%":
		parent := m.ContextParentID()
		return m, func() tea.Msg { return core.PageCreatedMsg{Page: core.Page{ParentID: parent}} }
	case "a":
		if hasCurr {
			parentID := curr.Page.ID
			m.expanded[parentID] = true
			return m, func() tea.Msg { return core.PageCreatedMsg{Page: core.Page{ParentID: parentID}} }
		}
		return m, func() tea.Msg { return core.PageCreatedMsg{} }
	case "d", "x", "delete":
		if hasCurr {
			id := curr.Page.ID
			return m, func() tea.Msg { return core.PageDeletedMsg{ID: id} }
		}
	}

	m.ensureVisible()
	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.Scroll(-3)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.Scroll(3)
		return m, nil
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	if msg.Y == 1 {
		return m, m.StartSearch()
	}
	if msg.Y < treeTop {
		return m, nil
	}
	if m.renaming {
		m.renaming = false
		m.renameInput.Blur()
	}
	idx := m.offset + msg.Y - treeTop
	items := m.VisibleItems()
	if idx < 0 || idx >= len(items) {
		return m, nil
	}
	m.cursor = idx
	item := items[idx]
	if m.searching || m.query() != "" {
		m.stopSearch(true)
		m.SetSelectedID(item.Page.ID)
		return m, selectCmd(item.Page.ID)
	}
	// Clicking the chevron (or a folder anywhere) toggles; clicking a note opens it.
	arrowX := 1 + item.Level*2
	if item.HasChildren && (item.Page.IsFolder || msg.X <= arrowX+1) {
		m.expanded[item.Page.ID] = !item.Expanded
		m.ensureVisible()
		return m, nil
	}
	return m, m.activate(item)
}

// displayName shows real filenames, like nvim-tree.
func displayName(p core.Page) string {
	if p.IsFolder {
		return p.Title
	}
	return p.Title + ".md"
}

// View renders the explorer at exactly width × height.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	th := m.theme
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	w := m.width

	var rows []string

	// 1. Workspace title.
	title := bg.Foreground(th.Blue).Bold(true).Render(" 󰉖 " + ui.Truncate(strings.ToUpper(m.workspace), w-4))
	rows = append(rows, ui.FitLine(title, w, bg))

	// 2. Search field: a single painted line, so it can never misalign.
	fieldW := max(w-2, 4)
	field := lipgloss.NewStyle().Background(th.Bg2)
	var inner string
	if m.searching {
		m.searchInput.Width = max(fieldW-5, 1)
		m.searchInput.TextStyle = field.Foreground(th.Fg)
		m.searchInput.PlaceholderStyle = field.Foreground(th.GreyFg)
		m.searchInput.Cursor.Style = lipgloss.NewStyle().Foreground(th.Blue)
		inner = field.Foreground(th.Blue).Render(" 󰍉 ") + m.searchInput.View()
	} else if q := m.searchInput.Value(); q != "" {
		inner = field.Foreground(th.Blue).Render(" 󰍉 ") + field.Foreground(th.Fg).Render(ui.Truncate(q, fieldW-5))
	} else {
		left := field.Foreground(th.GreyFg).Render(" 󰍉 Search notes")
		hint := field.Foreground(th.Grey).Render("/ ")
		inner = left + field.Render(strings.Repeat(" ", max(fieldW-lipgloss.Width(left)-lipgloss.Width(hint), 0))) + hint
	}
	rows = append(rows, bg.Render(" ")+ui.FitLine(inner, fieldW, field)+bg.Render(" "))
	rows = append(rows, "")

	// 3. Tree.
	items := m.VisibleItems()
	h := m.treeHeight()
	if len(items) == 0 {
		msg := "Empty workspace — press n"
		if m.query() != "" {
			msg = "No matching notes"
		}
		rows = append(rows, bg.Foreground(th.GreyFg).Italic(true).Render("  "+ui.Truncate(msg, w-3)))
	}
	end := min(m.offset+h, len(items))
	for i := m.offset; i < end; i++ {
		rows = append(rows, m.renderItem(items[i], i == m.cursor))
	}

	return ui.Fit(strings.Join(rows, "\n"), w, m.height, bg)
}

func (m Model) renderItem(item TreeItem, selected bool) string {
	th := m.theme
	rowBg := th.DarkerBg
	if selected {
		rowBg = th.Bg2
		if m.focused {
			rowBg = th.OneBg2
		}
	}
	base := lipgloss.NewStyle().Background(rowBg)
	guide := base.Foreground(th.Line)

	var b strings.Builder
	b.WriteString(base.Render(" "))

	// Indent guides.
	if m.query() == "" {
		for i := 0; i < item.Level; i++ {
			switch {
			case i < item.Level-1 && i < len(item.Guides) && item.Guides[i]:
				b.WriteString(guide.Render("│ "))
			case i < item.Level-1:
				b.WriteString(base.Render("  "))
			case item.IsLast:
				b.WriteString(guide.Render("└ "))
			default:
				b.WriteString(guide.Render("│ "))
			}
		}
	}

	// Chevron + icon + name.
	name := displayName(item.Page)
	var arrow, icon string
	iconStyle := base.Foreground(th.NordBlue)
	nameStyle := base.Foreground(th.Fg)
	const (
		chevronRight = " "
		chevronDown  = " "
		folderClosed = " "
		folderOpen   = " "
		folderEmpty  = " "
		markdownIcon = " "
	)
	switch {
	case item.Page.IsFolder:
		icon = folderClosed
		if item.HasChildren {
			arrow = chevronRight
			if item.Expanded {
				arrow = chevronDown
				icon = folderOpen
			}
		} else {
			arrow = "  "
			icon = folderEmpty
		}
		iconStyle = base.Foreground(th.Blue)
		nameStyle = base.Foreground(th.Blue)
	case item.HasChildren:
		arrow = chevronRight
		if item.Expanded {
			arrow = chevronDown
		}
		icon = markdownIcon
	default:
		arrow = "  "
		icon = markdownIcon
	}
	if m.query() != "" {
		arrow = ""
	}
	if item.Page.ID == m.activeID {
		nameStyle = nameStyle.Bold(true).Foreground(th.Green)
	} else if selected && m.focused {
		nameStyle = nameStyle.Bold(true)
	}

	b.WriteString(base.Foreground(th.GreyFg).Render(arrow))
	b.WriteString(iconStyle.Render(icon))

	right := ""
	if m.modified[item.Page.ID] {
		right = base.Foreground(th.Yellow).Render(" ● ")
	}
	used := lipgloss.Width(b.String()) + lipgloss.Width(right)

	if m.renaming && selected {
		m.renameInput.Width = max(m.width-used-2, 1)
		m.renameInput.TextStyle = base.Foreground(th.Fg)
		b.WriteString(m.renameInput.View())
		return ui.FitLine(b.String(), m.width, base)
	}

	avail := m.width - used
	label := name
	if m.query() != "" {
		if dir := path.Dir(item.Page.ID); dir != "." {
			label = name + "  " + dir
		}
	}
	label = ui.Truncate(label, max(avail, 1))
	if m.query() != "" && len(label) > len(name) && strings.HasPrefix(label, name) {
		b.WriteString(nameStyle.Render(name))
		b.WriteString(base.Foreground(th.GreyFg).Render(label[len(name):]))
	} else {
		b.WriteString(nameStyle.Render(label))
	}
	pad := m.width - lipgloss.Width(b.String()) - lipgloss.Width(right)
	if pad > 0 {
		b.WriteString(base.Render(strings.Repeat(" ", pad)))
	}
	b.WriteString(right)
	return ui.FitLine(b.String(), m.width, base)
}
