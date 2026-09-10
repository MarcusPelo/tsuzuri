// Package app acts as the root orchestrator component (<App/>).
package app

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"tsuzuri/internal/content"
	"tsuzuri/internal/core"
	"tsuzuri/internal/header"
	"tsuzuri/internal/sidebar"
	"tsuzuri/internal/theme"
)

type focus int

const (
	focusSidebar focus = iota
	focusContent
)

// Model represents the root App container state.
type Model struct {
	store         *core.Store
	theme         theme.Theme
	keys          KeyMap
	focus         focus
	selectedID    string
	statusMessage string

	width  int
	height int

	header  header.Model
	sidebar sidebar.Model
	content content.Model
}

// New constructs the root App container initialized with a core.Store.
func New(store *core.Store) Model {
	th := theme.DefaultTheme()
	km := DefaultKeyMap()

	hd := header.New(th)
	sb := sidebar.New(th)
	ct := content.New(th)

	pages := store.List()
	sb.SetPages(pages)

	var selectedID string
	if len(pages) > 0 {
		selectedID = pages[0].ID
		sb.SetSelectedID(selectedID)
		ct.SetPage(pages[0])
		hd.SetPage(pages[0])
	}

	ct.SetFocused(true)

	return Model{
		store:      store,
		theme:      th,
		keys:       km,
		focus:      focusContent,
		selectedID: selectedID,
		header:     hd,
		sidebar:    sb,
		content:    ct,
	}
}

// Init initializes child component commands.
func (m Model) Init() tea.Cmd {
	return m.content.Init()
}

// Update routes events down and handles events emitted up from child components.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		headerHeight := 1
		sidebarW, contentW, contentH := CalculateLayout(msg.Width, msg.Height, headerHeight)

		m.header.SetSize(msg.Width, headerHeight)
		m.sidebar.SetSize(sidebarW, contentH)
		m.content.SetSize(contentW, contentH)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Tab):
			if m.focus == focusSidebar {
				if !m.sidebar.IsRenaming() {
					m.focus = focusContent
					m.sidebar.SetFocused(false)
					cmds = append(cmds, m.content.Focus())
				}
			} else {
				if m.content.Mode() == content.ModeNormal {
					m.focus = focusSidebar
					m.sidebar.SetFocused(true)
					m.content.SetFocused(false)
				}
			}

		case key.Matches(msg, m.keys.NewPage):
			newPage, err := m.store.Create("")
			if err != nil {
				m.statusMessage = "Error creating page: " + err.Error()
			} else {
				m.selectedID = newPage.ID
				m.sidebar.SetPages(m.store.List())
				m.sidebar.SetSelectedID(newPage.ID)
				m.content.SetPage(newPage)
				m.header.SetPage(newPage)
				m.focus = focusContent
				m.sidebar.SetFocused(false)
				cmds = append(cmds, m.content.Focus())
			}
		}

	// Catch custom domain event messages bubbled up from children
	case core.PageSelectedMsg:
		m.selectedID = msg.ID
		if p, err := m.store.Get(msg.ID); err == nil {
			m.content.SetPage(p)
			m.header.SetPage(p)
		} else {
			m.statusMessage = "Error loading page: " + err.Error()
		}

	case core.PageCreatedMsg:
		newPage, err := m.store.Create(msg.Page.Title)
		if err != nil {
			m.statusMessage = "Error creating page: " + err.Error()
		} else {
			m.selectedID = newPage.ID
			m.sidebar.SetPages(m.store.List())
			m.sidebar.SetSelectedID(newPage.ID)
			m.content.SetPage(newPage)
			m.header.SetPage(newPage)
			m.focus = focusContent
			m.sidebar.SetFocused(false)
			cmds = append(cmds, m.content.Focus())
		}

	case core.PageUpdatedMsg:
		if err := m.store.Update(msg.Page); err != nil {
			m.statusMessage = "Error updating page: " + err.Error()
		} else {
			m.sidebar.SetPages(m.store.List())
			if msg.Page.ID == m.selectedID {
				m.header.SetPage(msg.Page)
			}
		}

	case core.PageDeletedMsg:
		if err := m.store.Delete(msg.ID); err != nil {
			m.statusMessage = "Error deleting page: " + err.Error()
		} else {
			pages := m.store.List()
			m.sidebar.SetPages(pages)
			if len(pages) > 0 {
				m.sidebar.SetSelectedID(pages[0].ID)
				if p, err := m.store.Get(pages[0].ID); err == nil {
					m.selectedID = p.ID
					m.content.SetPage(p)
					m.header.SetPage(p)
				}
			}
		}

	case core.VimSaveMsg:
		if p, err := m.store.Get(m.selectedID); err == nil {
			p.Content = msg.Content
			if updateErr := m.store.Update(p); updateErr != nil {
				m.statusMessage = "Error saving page: " + updateErr.Error()
			} else {
				m.sidebar.SetPages(m.store.List())
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

	// Update active focused component model
	m.header.SetMode(m.content.ModeString(), m.focus == focusSidebar)

	if m.focus == focusSidebar {
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View composes child component views into the complete terminal layout, or renders a warning if terminal is too small.
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

	headerView := m.header.View()
	sidebarView := m.sidebar.View()
	contentView := m.content.View()

	bodyView := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, contentView)
	return lipgloss.JoinVertical(lipgloss.Left, headerView, bodyView)
}
