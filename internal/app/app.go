// Package app acts as the root orchestrator component (<App/>).
package app

import (
	"fmt"
	"strings"

	"github.com/jaisuriya-11/tsuzuri/internal/content"
	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/dashboard"
	"github.com/jaisuriya-11/tsuzuri/internal/header"
	"github.com/jaisuriya-11/tsuzuri/internal/preview"
	"github.com/jaisuriya-11/tsuzuri/internal/sidebar"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewMode int

const (
	viewModeDashboard viewMode = iota
	viewModeWorkspace
)

type focus int

const (
	focusSidebar focus = iota
	focusEditor
	focusPreview
)

// draftIDPrefix marks a synthetic buffer ID for a page that has never been
// saved, so it has no file on disk (and no sidebar tree entry) yet.
const draftIDPrefix = "draft:"

func isDraftID(id string) bool {
	return strings.HasPrefix(id, draftIDPrefix)
}

// Model represents the root App container state.
type Model struct {
	store         *core.Store
	theme         theme.Theme
	keys          KeyMap
	focus         focus
	viewMode      viewMode
	selectedID    string
	statusMessage string

	width  int
	height int

	sidebarOpen   bool
	showKeymap    bool
	leaderPending bool

	// openBuffers is the ordered list of page IDs currently open as editor
	// tabs (VSCode-style: only pages you've actually opened, not every file
	// in the workspace). Entries may be draftIDPrefix-prefixed synthetic IDs
	// for unsaved new pages.
	openBuffers []string
	drafts      map[string]core.Page
	draftSeq    int

	dashboard dashboard.Model
	header    header.Model
	sidebar   sidebar.Model
	content   content.Model
	preview   preview.Model
}

// New constructs the root App container initialized with a core.Store backed
// by a real workspace directory on disk. The workspace may already contain
// notes from a previous session (or none at all) — nothing is seeded.
func New(store *core.Store) Model {
	th := theme.DefaultTheme()
	km := DefaultKeyMap()

	db := dashboard.New(th)
	hd := header.New(th)
	sb := sidebar.New(th)
	ct := content.New(th)
	pv := preview.New(th)

	pages := store.List()
	openable := nonFolderPages(pages)
	sb.SetPages(pages)

	m := Model{
		store:       store,
		theme:       th,
		keys:        km,
		focus:       focusEditor,
		viewMode:    viewModeDashboard,
		sidebarOpen: true,
		dashboard:   db,
		header:      hd,
		sidebar:     sb,
		content:     ct,
		preview:     pv,
	}

	if len(openable) > 0 {
		m.displayPage(openable[0])
	}

	m.content.SetFocused(true)
	m.dashboard.SetRecentPages(openable)

	return m
}

// nonFolderPages filters out plain folder nodes, returning only pages that
// have real Markdown content and can be opened in the editor.
func nonFolderPages(pages []core.Page) []core.Page {
	openable := make([]core.Page, 0, len(pages))
	for _, p := range pages {
		if !p.IsFolder {
			openable = append(openable, p)
		}
	}
	return openable
}

// Init initializes child component commands.
func (m Model) Init() tea.Cmd {
	return m.content.Init()
}

// updateLayout recalculates dimensions across child panes.
func (m *Model) updateLayout() {
	headerHeight := 0
	footerHeight := 1
	sidebarW, editorW, previewW, bodyH := CalculateLayout(m.width, m.height, headerHeight, footerHeight, m.sidebarOpen)

	m.dashboard.SetSize(m.width, m.height)
	m.header.SetSize(m.width, headerHeight)
	m.sidebar.SetSize(sidebarW, bodyH)
	m.content.SetSize(editorW, bodyH)
	m.preview.SetSize(previewW, bodyH)
}

// lookupPage resolves a buffer ID to its Page, whether it's a saved page on
// disk or an unsaved draft held only in memory.
func (m *Model) lookupPage(id string) (core.Page, bool) {
	if isDraftID(id) {
		p, ok := m.drafts[id]
		return p, ok
	}
	p, err := m.store.Get(id)
	if err != nil || p.IsFolder {
		return core.Page{}, false
	}
	return p, true
}

// addBuffer ensures id is present in the open-buffers list, appending it at
// the end if it's new (VSCode-style: tabs persist in the order opened).
func (m *Model) addBuffer(id string) {
	for _, b := range m.openBuffers {
		if b == id {
			return
		}
	}
	m.openBuffers = append(m.openBuffers, id)
}

// renameBuffer swaps an open tab from oldID to newID in place, used both for
// on-disk renames and for a draft's first save (synthetic ID -> real ID).
func (m *Model) renameBuffer(oldID, newID string) {
	for i, b := range m.openBuffers {
		if b == oldID {
			m.openBuffers[i] = newID
			return
		}
	}
}

// removeBuffer closes a tab, discarding its draft content if it was never saved.
func (m *Model) removeBuffer(id string) {
	if isDraftID(id) {
		delete(m.drafts, id)
	}
	for i, b := range m.openBuffers {
		if b == id {
			m.openBuffers = append(m.openBuffers[:i], m.openBuffers[i+1:]...)
			return
		}
	}
}

// resolveBuffers loads full Page data for every open tab, silently dropping
// any that no longer resolve (e.g. deleted from disk in another way).
func (m *Model) resolveBuffers() []core.Page {
	pages := make([]core.Page, 0, len(m.openBuffers))
	valid := make([]string, 0, len(m.openBuffers))
	for _, id := range m.openBuffers {
		if p, ok := m.lookupPage(id); ok {
			pages = append(pages, p)
			valid = append(valid, id)
		}
	}
	m.openBuffers = valid
	return pages
}

// displayPage wires p up as the active document across sidebar selection,
// open tabs, content, header, and preview, without touching focus or view mode.
func (m *Model) displayPage(p core.Page) {
	m.selectedID = p.ID
	m.addBuffer(p.ID)
	m.content.SetPages(m.resolveBuffers())
	m.sidebar.SetSelectedID(p.ID)
	m.content.SetPage(p)
	m.header.SetPage(p)
	m.preview.SetPage(p)
}

// openPage displays id (a saved page or an open draft) and switches focus
// into the editor so the user can start typing immediately.
func (m *Model) openPage(id string) tea.Cmd {
	p, ok := m.lookupPage(id)
	if !ok {
		return nil
	}
	m.displayPage(p)
	m.viewMode = viewModeWorkspace
	m.focus = focusEditor
	m.sidebar.SetFocused(false)
	m.preview.SetFocused(false)
	return m.content.Focus()
}

// newDraft starts an unsaved in-memory page under parentID. Nothing touches
// disk until the user explicitly saves it with :w or :wq.
func (m *Model) newDraft(parentID string) core.Page {
	m.draftSeq++
	id := fmt.Sprintf("%s%d", draftIDPrefix, m.draftSeq)
	p := core.Page{ID: id, Title: "Untitled", ParentID: parentID}
	if m.drafts == nil {
		m.drafts = make(map[string]core.Page)
	}
	m.drafts[id] = p
	return p
}

// createPage opens a brand-new unsaved draft under parentID for editing.
func (m *Model) createPage(parentID string) tea.Cmd {
	draft := m.newDraft(parentID)
	return m.openPage(draft.ID)
}

// saveActiveBuffer persists content to the active buffer: a first :w on a
// draft materializes it on disk for the very first time (and swaps its tab
// over to the real ID); saving an already-persisted page just rewrites it.
func (m *Model) saveActiveBuffer(content string) {
	if m.selectedID == "" {
		m.statusMessage = "No page open to save"
		return
	}

	if isDraftID(m.selectedID) {
		draft, ok := m.drafts[m.selectedID]
		if !ok {
			m.statusMessage = "Error saving page: draft no longer exists"
			return
		}
		created, err := m.store.Create(draft.Title, draft.ParentID)
		if err != nil {
			m.statusMessage = "Error saving page: " + err.Error()
			return
		}
		created.Content = content
		saved, err := m.store.Update(created)
		if err != nil {
			m.statusMessage = "Error saving page: " + err.Error()
			return
		}
		oldID := m.selectedID
		delete(m.drafts, oldID)
		m.renameBuffer(oldID, saved.ID)
		m.sidebar.SetPages(m.store.List())
		m.displayPage(saved)
		m.statusMessage = "Saved " + saved.Title + ".md"
		return
	}

	p, err := m.store.Get(m.selectedID)
	if err != nil {
		m.statusMessage = "Error saving page: " + err.Error()
		return
	}
	p.Content = content
	updated, err := m.store.Update(p)
	if err != nil {
		m.statusMessage = "Error saving page: " + err.Error()
		return
	}
	m.renameBuffer(m.selectedID, updated.ID)
	m.sidebar.SetPages(m.store.List())
	m.displayPage(updated)
	m.statusMessage = "Saved " + updated.Title + ".md"
}

// closeActiveBuffer closes the current tab (discarding unsaved draft content,
// exactly like closing an unsaved tab in VSCode) and focuses the tab that
// now sits in its place, VSCode-style.
func (m *Model) closeActiveBuffer() tea.Cmd {
	if m.selectedID == "" {
		return nil
	}

	idx := -1
	for i, b := range m.openBuffers {
		if b == m.selectedID {
			idx = i
			break
		}
	}

	m.removeBuffer(m.selectedID)
	remaining := m.resolveBuffers()
	m.content.SetPages(remaining)

	if len(remaining) == 0 {
		m.selectedID = ""
		m.content.SetPage(core.Page{})
		m.header.SetPage(core.Page{})
		m.preview.SetPage(core.Page{})
		return nil
	}

	nextIdx := idx
	if nextIdx >= len(remaining) {
		nextIdx = len(remaining) - 1
	}
	if nextIdx < 0 {
		nextIdx = 0
	}
	return m.openPage(remaining[nextIdx].ID)
}

// Update routes events down and handles events emitted up from child components.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.updateLayout()
	}

	// 1. Handle Keymap Modal dismiss
	if m.showKeymap {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "esc", "q", "space", "enter":
				m.showKeymap = false
				return m, nil
			}
		}
		return m, nil
	}

	// 2. Leader key detection (Space + h) or (?) for Keymap Modal
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		canUseLeader := m.viewMode == viewModeWorkspace &&
			((m.focus == focusSidebar && !m.sidebar.IsRenaming() && !m.sidebar.IsSearching()) ||
				(m.focus == focusPreview) ||
				(m.focus == focusEditor && m.content.Mode() == content.ModeNormal))

		if canUseLeader {
			if m.leaderPending {
				m.leaderPending = false
				if keyMsg.String() == "h" {
					m.showKeymap = true
					return m, nil
				}
			} else if keyMsg.String() == " " {
				m.leaderPending = true
				return m, nil
			} else if keyMsg.String() == "?" {
				m.showKeymap = true
				return m, nil
			}
		} else {
			m.leaderPending = false
		}
	}

	// 3. Dashboard View Handling
	if m.viewMode == viewModeDashboard {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keys.Quit) || msg.String() == "q":
				return m, tea.Quit

			case msg.String() == "n":
				cmds = append(cmds, m.createPage(""))
				return m, tea.Batch(cmds...)

			case msg.String() == "f" || key.Matches(msg, m.keys.Tab):
				m.viewMode = viewModeWorkspace
				m.focus = focusSidebar
				m.sidebar.SetFocused(true)
				m.content.SetFocused(false)
				m.preview.SetFocused(false)
				return m, nil

			case msg.String() == "enter":
				action := m.dashboard.SelectedAction()
				switch action {
				case dashboard.ActionNewPage:
					cmds = append(cmds, m.createPage(""))
					return m, tea.Batch(cmds...)

				case dashboard.ActionBrowse:
					m.viewMode = viewModeWorkspace
					m.focus = focusSidebar
					m.sidebar.SetFocused(true)
					m.content.SetFocused(false)
					m.preview.SetFocused(false)
					return m, nil

				case dashboard.ActionQuit:
					return m, tea.Quit
				}

			case msg.String() >= "1" && msg.String() <= "9":
				idx := int(msg.String()[0] - '1')
				openable := nonFolderPages(m.store.List())
				if idx < len(openable) {
					cmds = append(cmds, m.openPage(openable[idx].ID))
					return m, tea.Batch(cmds...)
				}
			}
		}

		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd
	}

	// 4. Workspace View Handling
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.ToggleSidebar):
			m.sidebarOpen = !m.sidebarOpen
			if !m.sidebarOpen && m.focus == focusSidebar {
				m.focus = focusEditor
				m.sidebar.SetFocused(false)
				m.content.SetFocused(true)
			}
			m.updateLayout()
			return m, nil

		case key.Matches(msg, m.keys.Dashboard):
			m.dashboard.SetRecentPages(nonFolderPages(m.store.List()))
			m.viewMode = viewModeDashboard
			return m, nil

		case msg.String() == "esc" && m.focus == focusSidebar && !m.sidebar.IsRenaming() && !m.sidebar.IsSearching():
			m.dashboard.SetRecentPages(nonFolderPages(m.store.List()))
			m.viewMode = viewModeDashboard
			return m, nil

		case msg.String() == "esc" && m.focus == focusPreview:
			m.focus = focusEditor
			m.preview.SetFocused(false)
			m.content.SetFocused(true)
			return m, nil

		case key.Matches(msg, m.keys.Tab):
			switch m.focus {
			case focusSidebar:
				if !m.sidebar.IsRenaming() && !m.sidebar.IsSearching() {
					m.focus = focusEditor
					m.sidebar.SetFocused(false)
					m.preview.SetFocused(false)
					cmds = append(cmds, m.content.Focus())
				}
			case focusEditor:
				if m.content.Mode() == content.ModeNormal {
					m.focus = focusPreview
					m.sidebar.SetFocused(false)
					m.content.SetFocused(false)
					m.preview.SetFocused(true)
				}
			case focusPreview:
				if m.sidebarOpen {
					m.focus = focusSidebar
					m.sidebar.SetFocused(true)
					m.content.SetFocused(false)
					m.preview.SetFocused(false)
				} else {
					m.focus = focusEditor
					m.sidebar.SetFocused(false)
					m.preview.SetFocused(false)
					cmds = append(cmds, m.content.Focus())
				}
			}

		case msg.String() == "x" && m.focus == focusEditor && m.content.Mode() == content.ModeNormal:
			cmds = append(cmds, m.closeActiveBuffer())
			return m, tea.Batch(cmds...)

		case msg.String() == "]" || msg.String() == "[":
			canCycle := (m.focus == focusEditor && m.content.Mode() == content.ModeNormal) ||
				(m.focus == focusSidebar && !m.sidebar.IsRenaming() && !m.sidebar.IsSearching()) ||
				(m.focus == focusPreview)
			if canCycle {
				open := m.resolveBuffers()
				if len(open) > 1 {
					currIdx := 0
					for i, p := range open {
						if p.ID == m.selectedID {
							currIdx = i
							break
						}
					}
					nextIdx := currIdx + 1
					if msg.String() == "[" {
						nextIdx = currIdx - 1 + len(open)
					}
					nextIdx %= len(open)
					cmds = append(cmds, m.openPage(open[nextIdx].ID))
					return m, tea.Batch(cmds...)
				}
			}

		case key.Matches(msg, m.keys.NewPage):
			cmds = append(cmds, m.createPage(""))
		}

	// Domain message bus
	case core.PageSelectedMsg:
		if cmd := m.openPage(msg.ID); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case core.PageCreatedMsg:
		m.statusMessage = ""
		cmds = append(cmds, m.createPage(msg.Page.ParentID))

	case core.PageUpdatedMsg:
		updated, err := m.store.Update(msg.Page)
		if err != nil {
			m.statusMessage = "Error updating page: " + err.Error()
		} else {
			m.sidebar.SetPages(m.store.List())
			m.sidebar.SetSelectedID(updated.ID)
			if msg.Page.ID == m.selectedID {
				m.renameBuffer(msg.Page.ID, updated.ID)
				m.selectedID = updated.ID
				m.header.SetPage(updated)
				m.preview.SetTitle(updated.Title)
			} else {
				m.renameBuffer(msg.Page.ID, updated.ID)
			}
			m.content.SetPages(m.resolveBuffers())
		}

	case core.PageDeletedMsg:
		if err := m.store.Delete(msg.ID); err != nil {
			m.statusMessage = "Error deleting page: " + err.Error()
		} else {
			pages := m.store.List()
			m.sidebar.SetPages(pages)
			m.removeBuffer(msg.ID)

			if msg.ID != m.selectedID {
				m.content.SetPages(m.resolveBuffers())
				break
			}

			remaining := m.resolveBuffers()
			m.content.SetPages(remaining)
			if len(remaining) > 0 {
				cmds = append(cmds, m.openPage(remaining[0].ID))
			} else {
				m.selectedID = ""
				m.content.SetPage(core.Page{})
				m.header.SetPage(core.Page{})
				m.preview.SetPage(core.Page{})
				if len(pages) == 0 {
					m.dashboard.SetRecentPages(pages)
					m.viewMode = viewModeDashboard
				}
			}
		}

	case core.VimSaveMsg:
		m.saveActiveBuffer(msg.Content)

	case core.VimQuitMsg:
		if msg.Save && m.selectedID != "" {
			m.saveActiveBuffer(m.content.Value())
		}
		return m, tea.Quit
	}

	// Update Header component mode indicator
	switch m.focus {
	case focusSidebar:
		m.header.SetPaneFocus("SIDEBAR", m.content.ModeString())
	case focusPreview:
		m.header.SetPaneFocus("PREVIEW", m.content.ModeString())
	default:
		m.header.SetPaneFocus("EDITOR", m.content.ModeString())
	}

	if m.focus == focusSidebar {
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.focus == focusPreview {
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		cmds = append(cmds, cmd)
		// Real-time Markdown compilation on every keystroke!
		m.preview.SetContent(m.content.Value())
	}

	return m, tea.Batch(cmds...)
}

// renderBottomBar renders the unified bottom status bar across the full terminal width in NvChad NvUI v3.0 style.
func (m Model) renderBottomBar() string {
	var modeTag string
	switch m.focus {
	case focusSidebar:
		modeTag = lipgloss.NewStyle().Bold(true).Background(m.theme.SidebarBg).Foreground(m.theme.DarkFg).Padding(0, 1).Render("󰉋 TREE")
	case focusPreview:
		modeTag = lipgloss.NewStyle().Bold(true).Background(m.theme.CommandBg).Foreground(m.theme.DarkFg).Padding(0, 1).Render("󰈈 PREVIEW")
	default:
		switch m.content.Mode() {
		case content.ModeInsert:
			modeTag = lipgloss.NewStyle().Bold(true).Background(m.theme.InsertBg).Foreground(m.theme.DarkFg).Padding(0, 1).Render("󰏫 INSERT")
		case content.ModeCommand:
			modeTag = lipgloss.NewStyle().Bold(true).Background(m.theme.CommandBg).Foreground(m.theme.DarkFg).Padding(0, 1).Render("󰘳 COMMAND")
		default:
			modeTag = lipgloss.NewStyle().Bold(true).Background(m.theme.NormalBg).Foreground(m.theme.DarkFg).Padding(0, 1).Render(" NORMAL")
		}
	}

	var filePill string
	if p, ok := m.lookupPage(m.selectedID); ok {
		docTitle := p.Title
		if docTitle == "" {
			docTitle = "Untitled"
		}
		if !strings.HasSuffix(docTitle, ".md") {
			docTitle += ".md"
		}
		if isDraftID(m.selectedID) {
			docTitle += " (unsaved)"
		}
		filePill = lipgloss.NewStyle().
			Bold(true).
			Foreground(m.theme.TitleFg).
			Background(m.theme.SelectedBg).
			Padding(0, 1).
			Render(" " + docTitle)
	} else {
		filePill = lipgloss.NewStyle().
			Italic(true).
			Foreground(m.theme.MutedFg).
			Padding(0, 1).
			Render("no page open")
	}

	var hint string
	if m.statusMessage != "" {
		hint = "  " + lipgloss.NewStyle().Foreground(m.theme.NormalBg).Render(m.statusMessage)
	} else {
		hint = "  " + lipgloss.NewStyle().Foreground(m.theme.MutedFg).Render("i insert • Esc normal • :wq save")
	}
	leftContent := modeTag + " " + filePill + hint

	previewStatus := lipgloss.NewStyle().Bold(true).Foreground(m.theme.CommandBg).Render("󰈈 " + m.preview.ScrollStatus())
	fileTypePill := lipgloss.NewStyle().Foreground(m.theme.MutedFg).Render("󰈙 markdown")
	tabHint := lipgloss.NewStyle().Foreground(m.theme.MutedFg).Render("[/] tabs  x close")
	keymapHint := lipgloss.NewStyle().Bold(true).Foreground(m.theme.NormalBg).Render(" space+h")
	rightContent := previewStatus + "   " + fileTypePill + "   " + tabHint + "   " + keymapHint

	leftW := lipgloss.Width(leftContent)
	rightW := lipgloss.Width(rightContent)
	gap := m.width - leftW - rightW - 2
	if gap < 1 {
		gap = 1
	}

	line := leftContent + strings.Repeat(" ", gap) + rightContent
	return lipgloss.NewStyle().
		Width(m.width).
		Height(1).
		Padding(0, 1).
		Background(m.theme.DarkFg).
		Render(line)
}

// View composes child component views into the complete terminal layout.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading…"
	}

	// Guard against terminal window being too small
	if m.width < MinTermWidth || m.height < MinTermHeight {
		warningStyle := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.theme.NormalBg).
			Bold(true)
		return warningStyle.Render(fmt.Sprintf("Terminal window is too small (%dx%d).\nPlease resize to min %dx%d.", m.width, m.height, MinTermWidth, MinTermHeight))
	}

	// 1. Overlay Keymap Modal if open
	if m.showKeymap {
		return RenderKeymapModal(m.theme, m.width, m.height)
	}

	// 2. Dashboard View
	if m.viewMode == viewModeDashboard {
		return m.dashboard.View()
	}

	// 3. Body Panes (Sidebar + Editor + Preview)
	contentView := m.content.View()
	previewView := m.preview.View()

	var bodyView string
	if m.sidebarOpen {
		sidebarView := m.sidebar.View()
		bodyView = lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, contentView, previewView)
	} else {
		bodyView = lipgloss.JoinHorizontal(lipgloss.Top, contentView, previewView)
	}

	// 4. Unified Bottom Status Bar
	statusBarView := m.renderBottomBar()

	return lipgloss.JoinVertical(lipgloss.Left, bodyView, statusBarView)
}
