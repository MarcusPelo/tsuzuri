package dashboard

import (
	"fmt"
	"strings"

	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Action defines an action triggered by a dashboard menu item.
type Action int

const (
	ActionNewPage Action = iota
	ActionBrowse
	ActionQuit
)

// MenuItem represents a selectable button on the landing page.
type MenuItem struct {
	Key         string
	Icon        string
	Label       string
	Description string
	Action      Action
}

// Model represents the NvChad landing page state.
type Model struct {
	theme       theme.Theme
	width       int
	height      int
	cursor      int
	menuItems   []MenuItem
	recentPages []core.Page
}

var banner = []string{
	`████████╗███████╗██╗   ██╗███████╗██╗   ██╗██████╗ ██╗`,
	`╚══██╔══╝██╔════╝██║   ██║╚══███╔╝██║   ██║██╔══██╗██║`,
	`   ██║   ███████╗██║   ██║  ███╔╝ ██║   ██║██████╔╝██║`,
	`   ██║   ╚════██║██║   ██║ ███╔╝  ██║   ██║██╔══██╗██║`,
	`   ██║   ███████║╚██████╔╝███████╗╚██████╔╝██║  ██║██║`,
	`   ╚═╝   ╚══════╝ ╚═════╝ ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝`,
}

// New constructs a Dashboard model.
func New(th theme.Theme) Model {
	items := []MenuItem{
		{Key: "n", Icon: "󰏫", Label: "New Document", Description: "Create a new workspace note", Action: ActionNewPage},
		{Key: "f", Icon: "󰉋", Label: "Browse Notes", Description: "Open sidebar page explorer", Action: ActionBrowse},
		{Key: "q", Icon: "󰅖", Label: "Quit", Description: "Exit application", Action: ActionQuit},
	}
	return Model{
		theme:     th,
		menuItems: items,
		cursor:    0,
	}
}

// SetSize updates dimensions.
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

// SetRecentPages updates the list of recent pages shown on the dashboard.
func (m *Model) SetRecentPages(pages []core.Page) {
	m.recentPages = pages
}

// SelectedAction returns the action of the currently hovered menu item.
func (m Model) SelectedAction() Action {
	if m.cursor >= 0 && m.cursor < len(m.menuItems) {
		return m.menuItems[m.cursor].Action
	}
	return ActionNewPage
}

// Update handles dashboard navigation and shortcut keys.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.menuItems)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

// View renders the centered NvChad dashboard.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	// 1. ASCII Art Banner
	bannerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.SelectedFg)
	var bannerLines []string
	for _, l := range banner {
		bannerLines = append(bannerLines, bannerStyle.Render(l))
	}
	bannerBlock := strings.Join(bannerLines, "\n")

	// 2. Subtitle
	subtitle := lipgloss.NewStyle().
		Foreground(m.theme.MutedFg).
		Italic(true).
		Render("~ 綴り • Terminal Markdown Notebook ~")

	// 3. Action Menu Items (NvChad style buttons)
	var menuRows []string
	for i, item := range m.menuItems {
		keyBadge := lipgloss.NewStyle().
			Bold(true).
			Foreground(m.theme.DarkFg).
			Background(m.theme.NormalBg).
			Padding(0, 1).
			Render(item.Key)

		prefix := "  "
		itemStyle := lipgloss.NewStyle().Foreground(m.theme.TitleFg)
		if i == m.cursor {
			prefix = lipgloss.NewStyle().Foreground(m.theme.NormalBg).Bold(true).Render("❯ ")
			itemStyle = lipgloss.NewStyle().Bold(true).Foreground(m.theme.NormalBg)
		}

		icon := lipgloss.NewStyle().Foreground(m.theme.SidebarBg).Render(item.Icon)
		label := itemStyle.Render(item.Label)
		desc := lipgloss.NewStyle().Foreground(m.theme.MutedFg).Render(item.Description)

		row := fmt.Sprintf("%s%s  %s  %-18s  %s", prefix, keyBadge, icon, label, desc)
		menuRows = append(menuRows, row)
	}
	menuBlock := strings.Join(menuRows, "\n")

	// 4. Recent Documents Section
	var recentBlock string
	if len(m.recentPages) > 0 {
		var recents []string
		header := lipgloss.NewStyle().Bold(true).Foreground(m.theme.CommandBg).Render("Recent Documents")
		recents = append(recents, header)

		maxRecent := 4
		if len(m.recentPages) < maxRecent {
			maxRecent = len(m.recentPages)
		}
		for i := 0; i < maxRecent; i++ {
			p := m.recentPages[i]
			title := p.Title
			if title == "" {
				title = "Untitled"
			}
			numBadge := lipgloss.NewStyle().Bold(true).Foreground(m.theme.NormalBg).Render(fmt.Sprintf("%d.", i+1))
			docTitle := lipgloss.NewStyle().Foreground(m.theme.TitleFg).Render(title)
			recents = append(recents, fmt.Sprintf("  󰈙 %s %s", numBadge, docTitle))
		}
		recentBlock = strings.Join(recents, "\n")
	}

	// 5. Footer Stats
	footer := lipgloss.NewStyle().
		Foreground(m.theme.MutedFg).
		Render(fmt.Sprintf("󱐋 Tsuzuri  •  %d document(s)  •  j/k navigate  •  Enter select  •  q quit", len(m.recentPages)))

	// Compose stack
	var sections []string
	sections = append(sections, bannerBlock, "", subtitle, "", menuBlock)
	if recentBlock != "" {
		sections = append(sections, "", recentBlock)
	}
	sections = append(sections, "", footer)

	stack := lipgloss.JoinVertical(lipgloss.Center, sections...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, stack)
}
