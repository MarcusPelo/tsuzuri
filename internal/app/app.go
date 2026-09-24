// Package app acts as the root orchestrator component (<App/>).
package app

import (
	"fmt"
	"strings"

	"tsuzuri/internal/content"
	"tsuzuri/internal/core"
	"tsuzuri/internal/dashboard"
	"tsuzuri/internal/header"
	"tsuzuri/internal/preview"
	"tsuzuri/internal/sidebar"
	"tsuzuri/internal/theme"

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

	dashboard dashboard.Model
	header    header.Model
	sidebar   sidebar.Model
	content   content.Model
	preview   preview.Model
}

// New constructs the root App container initialized with a core.Store.
func New(store *core.Store) Model {
	th := theme.DefaultTheme()
	km := DefaultKeyMap()

	db := dashboard.New(th)
	hd := header.New(th)
	sb := sidebar.New(th)
	ct := content.New(th)
	pv := preview.New(th)

	pages := store.List()
	sb.SetPages(pages)
	ct.SetPages(pages)

	var selectedID string
	if len(pages) > 0 {
		selectedID = pages[0].ID
		sb.SetSelectedID(selectedID)
		ct.SetPage(pages[0])
		hd.SetPage(pages[0])
		pv.SetPage(pages[0])
	}

	ct.SetFocused(true)
	db.SetRecentPages(pages)

	return Model{
		store:         store,
		theme:         th,
		keys:          km,
		focus:         focusEditor,
		viewMode:      viewModeDashboard,
		selectedID:    selectedID,
		sidebarOpen:   true,
		showKeymap:    false,
		leaderPending: false,
		dashboard:     db,
		header:        hd,
		sidebar:       sb,
		content:       ct,
		preview:       pv,
	}
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
				newPage, err := m.store.Create("")
				if err != nil {
					m.statusMessage = "Error creating page: " + err.Error()
				} else {
					m.selectedID = newPage.ID
					pages := m.store.List()
					m.sidebar.SetPages(pages)
					m.content.SetPages(pages)
					m.sidebar.SetSelectedID(newPage.ID)
					m.content.SetPage(newPage)
					m.header.SetPage(newPage)
					m.preview.SetPage(newPage)
					m.viewMode = viewModeWorkspace
					m.focus = focusEditor
					m.sidebar.SetFocused(false)
					m.preview.SetFocused(false)
					cmds = append(cmds, m.content.Focus())
				}
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
					newPage, err := m.store.Create("")
					if err != nil {
						m.statusMessage = "Error creating page: " + err.Error()
					} else {
						m.selectedID = newPage.ID
						pages := m.store.List()
						m.sidebar.SetPages(pages)
						m.content.SetPages(pages)
						m.sidebar.SetSelectedID(newPage.ID)
						m.content.SetPage(newPage)
						m.header.SetPage(newPage)
						m.preview.SetPage(newPage)
						m.viewMode = viewModeWorkspace
						m.focus = focusEditor
						m.sidebar.SetFocused(false)
						m.preview.SetFocused(false)
						cmds = append(cmds, m.content.Focus())
					}
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
				pages := m.store.List()
				if idx < len(pages) {
					m.selectedID = pages[idx].ID
					m.sidebar.SetPages(pages)
					m.content.SetPages(pages)
					m.sidebar.SetSelectedID(pages[idx].ID)
					m.content.SetPage(pages[idx])
					m.header.SetPage(pages[idx])
					m.preview.SetPage(pages[idx])
					m.viewMode = viewModeWorkspace
					m.focus = focusEditor
					m.sidebar.SetFocused(false)
					m.preview.SetFocused(false)
					cmds = append(cmds, m.content.Focus())
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
			m.dashboard.SetRecentPages(m.store.List())
			m.viewMode = viewModeDashboard
			return m, nil

		case msg.String() == "esc" && m.focus == focusSidebar && !m.sidebar.IsRenaming() && !m.sidebar.IsSearching():
			m.dashboard.SetRecentPages(m.store.List())
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

		case msg.String() == "]" || msg.String() == "[":
			canCycle := (m.focus == focusEditor && m.content.Mode() == content.ModeNormal) ||
				(m.focus == focusSidebar && !m.sidebar.IsRenaming() && !m.sidebar.IsSearching()) ||
				(m.focus == focusPreview)
			if canCycle {
				pages := m.store.List()
				if len(pages) > 1 {
					currIdx := 0
					for i, p := range pages {
						if p.ID == m.selectedID {
							currIdx = i
							break
						}
					}
					nextIdx := currIdx + 1
					if msg.String() == "[" {
						nextIdx = currIdx - 1 + len(pages)
					}
					nextIdx %= len(pages)
					nextPage := pages[nextIdx]
					m.selectedID = nextPage.ID
					m.sidebar.SetSelectedID(nextPage.ID)
					m.content.SetPage(nextPage)
					m.header.SetPage(nextPage)
					m.preview.SetPage(nextPage)
					m.content.SetPages(pages)
					return m, nil
				}
			}

		case key.Matches(msg, m.keys.NewPage):
			newPage, err := m.store.Create("")
			if err != nil {
				m.statusMessage = "Error creating page: " + err.Error()
			} else {
				m.selectedID = newPage.ID
				pages := m.store.List()
				m.sidebar.SetPages(pages)
				m.content.SetPages(pages)
				m.sidebar.SetSelectedID(newPage.ID)
				m.content.SetPage(newPage)
				m.header.SetPage(newPage)
				m.preview.SetPage(newPage)
				m.focus = focusEditor
				m.sidebar.SetFocused(false)
				m.preview.SetFocused(false)
				cmds = append(cmds, m.content.Focus())
			}
		}

	// Domain message bus
	case core.PageSelectedMsg:
		m.selectedID = msg.ID
		if p, err := m.store.Get(msg.ID); err == nil {
			m.content.SetPage(p)
			m.header.SetPage(p)
			m.preview.SetPage(p)
		} else {
			m.statusMessage = "Error loading page: " + err.Error()
		}

	case core.PageCreatedMsg:
		var newPage core.Page
		var err error
		if msg.Page.ParentID != "" {
			newPage, err = m.store.CreateChild(msg.Page.ParentID, msg.Page.Title)
		} else {
			newPage, err = m.store.Create(msg.Page.Title)
		}
		if err != nil {
			m.statusMessage = "Error creating page: " + err.Error()
		} else {
			m.selectedID = newPage.ID
			pages := m.store.List()
			m.sidebar.SetPages(pages)
			m.content.SetPages(pages)
			m.sidebar.SetSelectedID(newPage.ID)
			m.content.SetPage(newPage)
			m.header.SetPage(newPage)
			m.preview.SetPage(newPage)
			if m.focus == focusSidebar {
				cmds = append(cmds, m.sidebar.StartRenaming())
			} else {
				m.focus = focusEditor
				m.sidebar.SetFocused(false)
				m.preview.SetFocused(false)
				cmds = append(cmds, m.content.Focus())
			}
		}

	case core.PageUpdatedMsg:
		if err := m.store.Update(msg.Page); err != nil {
			m.statusMessage = "Error updating page: " + err.Error()
		} else {
			pages := m.store.List()
			m.sidebar.SetPages(pages)
			m.content.SetPages(pages)
			if msg.Page.ID == m.selectedID {
				m.header.SetPage(msg.Page)
				m.preview.SetTitle(msg.Page.Title)
			}
		}

	case core.PageDeletedMsg:
		if err := m.store.Delete(msg.ID); err != nil {
			m.statusMessage = "Error deleting page: " + err.Error()
		} else {
			pages := m.store.List()
			m.sidebar.SetPages(pages)
			m.content.SetPages(pages)
			if len(pages) > 0 {
				m.sidebar.SetSelectedID(pages[0].ID)
				if p, err := m.store.Get(pages[0].ID); err == nil {
					m.selectedID = p.ID
					m.content.SetPage(p)
					m.header.SetPage(p)
					m.preview.SetPage(p)
				}
			}
		}

	case core.VimSaveMsg:
		if p, err := m.store.Get(m.selectedID); err == nil {
			p.Content = msg.Content
			if updateErr := m.store.Update(p); updateErr != nil {
				m.statusMessage = "Error saving page: " + updateErr.Error()
			} else {
				pages := m.store.List()
				m.sidebar.SetPages(pages)
				m.content.SetPages(pages)
				m.preview.SetContent(msg.Content)
			}
		} else {
			m.statusMessage = "Error saving page: " + err.Error()
		}

	case core.VimQuitMsg:
		if msg.Save {
			if p, err := m.store.Get(m.selectedID); err == nil {
				p.Content = m.content.Value()
				_ = m.store.Update(p)
			}
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
			modeTag = lipgloss.NewStyle().Bold(true).Background(m.theme.NormalBg).Foreground(m.theme.DarkFg).Padding(0, 1).Render(" NORMAL")
		}
	}

	docTitle := "Untitled"
	if p, err := m.store.Get(m.selectedID); err == nil && p.Title != "" {
		docTitle = p.Title
	}
	if !strings.HasSuffix(docTitle, ".md") {
		docTitle += ".md"
	}
	filePill := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.TitleFg).
		Background(m.theme.SelectedBg).
		Padding(0, 1).
		Render(" " + docTitle)

	var hint string
	if m.statusMessage != "" {
		hint = "  " + lipgloss.NewStyle().Foreground(m.theme.NormalBg).Render(m.statusMessage)
	} else {
		hint = "  " + lipgloss.NewStyle().Foreground(m.theme.MutedFg).Render("i insert • Esc normal • :wq save")
	}
	leftContent := modeTag + " " + filePill + hint

	previewStatus := lipgloss.NewStyle().Bold(true).Foreground(m.theme.CommandBg).Render("󰈈 " + m.preview.ScrollStatus())
	fileTypePill := lipgloss.NewStyle().Foreground(m.theme.MutedFg).Render("󰈙 markdown")
	tabHint := lipgloss.NewStyle().Foreground(m.theme.MutedFg).Render("[/] tabs")
	keymapHint := lipgloss.NewStyle().Bold(true).Foreground(m.theme.NormalBg).Render("󰌌 space+h")
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
