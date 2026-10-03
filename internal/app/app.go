// Package app acts as the root orchestrator component (<App/>).
package app

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jaisuriya-11/tsuzuri/internal/content"
	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/dashboard"
	"github.com/jaisuriya-11/tsuzuri/internal/preview"
	"github.com/jaisuriya-11/tsuzuri/internal/sidebar"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"

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
	store    *core.Store
	theme    theme.Theme
	keys     KeyMap
	focus    focus
	viewMode viewMode

	width  int
	height int

	sidebarOpen   bool
	previewOpen   bool
	showKeymap    bool
	leaderPending bool

	// Open tabs, in order. Unsaved text lives in the buffers, never on disk.
	buffers  []*buffer
	active   string
	draftSeq int

	confirm *confirmDialog
	saveAs  *saveAsDialog
	finder  *finder
	themes  *themePicker
	browser *fileBrowser

	configPath string // where the theme choice is saved ("" = don't save)
	quitting   bool   // "Save All" before quitting is in progress

	status    string
	statusErr bool

	dashboard dashboard.Model
	sidebar   sidebar.Model
	content   content.Model
	preview   preview.Model
}

// Option customises the app at construction.
type Option func(*Model)

// WithTheme starts with the named base46 theme (unknown names are ignored).
func WithTheme(name string) Option {
	return func(m *Model) {
		if th, ok := theme.Get(name); ok {
			m.applyTheme(th, false)
		}
	}
}

// WithConfigPath saves theme changes to the given config file.
func WithConfigPath(path string) Option {
	return func(m *Model) { m.configPath = path }
}

// New constructs the root App container for a workspace on disk. Nothing is
// seeded and nothing is opened until the user asks.
func New(store *core.Store, opts ...Option) Model {
	th := theme.DefaultTheme()

	m := Model{
		store:       store,
		theme:       th,
		keys:        DefaultKeyMap(),
		focus:       focusSidebar,
		viewMode:    viewModeDashboard,
		sidebarOpen: true,
		previewOpen: true,
		dashboard:   dashboard.New(th),
		sidebar:     sidebar.New(th),
		content:     content.New(th),
		preview:     preview.New(th),
	}
	m.sidebar.SetWorkspaceName(filepath.Base(store.Root()))
	m.dashboard.SetWorkspace(tildePath(store.Root()))
	m.reloadTree()
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func tildePath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// Init initializes child component commands.
func (m Model) Init() tea.Cmd {
	return m.content.Init()
}

func (m *Model) layout() Layout {
	return CalculateLayout(m.width, m.height, m.sidebarOpen, m.previewOpen)
}

// updateLayout recalculates dimensions across child panes.
func (m *Model) updateLayout() {
	l := m.layout()
	m.dashboard.SetSize(m.width, max(m.height-footerHeight, 1))
	m.sidebar.SetSize(l.SidebarW, l.BodyH)
	m.content.SetSize(l.EditorW, l.BodyH)
	m.preview.SetSize(l.PreviewW, l.BodyH)
	if m.focus == focusPreview && l.PreviewW == 0 {
		m.focusPane(focusEditor)
	}
}

func (m *Model) reloadTree() {
	pages := m.store.List()
	m.sidebar.SetPages(pages)
	m.dashboard.SetRecentPages(pages)
}

func (m *Model) setStatus(s string) { m.status, m.statusErr = s, false }
func (m *Model) setError(s string)  { m.status, m.statusErr = s, true }

// focusPane moves keyboard focus, opening the target pane if it was hidden.
func (m *Model) focusPane(f focus) tea.Cmd {
	m.viewMode = viewModeWorkspace
	if f == focusSidebar && !m.sidebarOpen {
		m.sidebarOpen = true
		m.updateLayout()
	}
	if f == focusPreview && m.layout().PreviewW == 0 {
		f = focusEditor
	}
	m.focus = f
	m.sidebar.SetFocused(f == focusSidebar)
	m.preview.SetFocused(f == focusPreview)
	if f == focusEditor {
		return m.content.Focus()
	}
	m.content.SetFocused(false)
	return nil
}

// panes lists the visible panes left to right.
func (m *Model) panes() []focus {
	l := m.layout()
	var out []focus
	if l.SidebarW > 0 {
		out = append(out, focusSidebar)
	}
	out = append(out, focusEditor)
	if l.PreviewW > 0 {
		out = append(out, focusPreview)
	}
	return out
}

func (m *Model) movePane(delta int, wrap bool) tea.Cmd {
	ps := m.panes()
	i := 0
	for j, p := range ps {
		if p == m.focus {
			i = j
		}
	}
	i += delta
	if wrap {
		i = (i + len(ps)) % len(ps)
	} else {
		i = max(0, min(i, len(ps)-1))
	}
	return m.focusPane(ps[i])
}

// toggleSidebar is Ctrl+B: show and focus the explorer, or hide it when it
// already has focus. Works from every editor mode.
func (m *Model) toggleSidebar() tea.Cmd {
	if m.sidebarOpen && m.focus == focusSidebar && m.viewMode == viewModeWorkspace {
		m.sidebarOpen = false
		m.updateLayout()
		return m.focusPane(focusEditor)
	}
	if m.content.Mode() == content.ModeInsert {
		m.content.ExitInsert()
	}
	return m.focusPane(focusSidebar)
}

func (m *Model) togglePreview() tea.Cmd {
	m.previewOpen = !m.previewOpen
	m.updateLayout()
	if m.previewOpen && m.layout().PreviewW == 0 {
		m.setStatus("Window too narrow for the preview")
	}
	if m.focus == focusPreview && !m.previewOpen {
		return m.focusPane(focusEditor)
	}
	return nil
}

func (m *Model) startFind() tea.Cmd { return m.openFinder() }

func (m *Model) goHome() {
	m.stashActive()
	m.reloadTree()
	m.viewMode = viewModeDashboard
	m.content.SetFocused(false)
	m.sidebar.SetFocused(false)
}

// Update routes events down and handles events emitted up from child components.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.updateLayout()
		return m, nil
	case core.StatusMsg:
		m.status, m.statusErr = msg.Text, msg.Error
		return m, nil
	case tea.KeyMsg:
		m.status = ""
	}

	// Modal layers own all input while open.
	if m.saveAs != nil {
		return m, m.saveAs.update(&m, msg)
	}
	if m.confirm != nil {
		switch msg.(type) {
		case tea.KeyMsg, tea.MouseMsg:
			cmd, _ := m.confirm.update(&m, msg)
			return m, cmd
		}
		return m, nil
	}
	if m.finder != nil {
		return m, m.finder.update(&m, msg)
	}
	if m.themes != nil {
		return m, m.themes.update(&m, msg)
	}
	if m.browser != nil {
		return m, m.browser.update(&m, msg)
	}
	if m.showKeymap {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc", "q", "?", "enter", " ", "space":
				m.showKeymap = false
			case "ctrl+c":
				return m, m.requestQuit(false)
			}
		case tea.MouseMsg:
			if msg.Action == tea.MouseActionPress {
				m.showKeymap = false
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case core.PageSelectedMsg:
		if msg.KeepFocus {
			keep := m.focus
			m.openFile(msg.ID)
			return m, m.focusPane(keep)
		}
		return m, m.openFile(msg.ID)
	case core.FindRequestMsg:
		return m, m.openFinder()
	case core.SlashActionMsg:
		switch msg.Action {
		case "page":
			parent := m.sidebar.ContextParentID()
			if b := m.activeBuffer(); b != nil && !b.draft() {
				parent = b.id
			}
			return m, m.newDraft(parent)
		case "link":
			return m, m.openLinkPicker()
		}
		if kind, ok := strings.CutPrefix(msg.Action, "media:"); ok {
			return m, m.pickMedia(kind)
		}
		return m, nil
	case mediaPickedMsg:
		switch {
		case msg.unavailable:
			return m, m.openFileBrowser(msg.kind)
		case msg.err != nil:
			m.setError("File dialog failed: " + msg.err.Error())
		case msg.cancelled || msg.path == "":
			m.setStatus("")
		default:
			m.attachMedia(msg.kind, msg.path)
		}
		return m, nil
	case core.ThemeMsg:
		if msg.Name == "" {
			return m, m.openThemePicker()
		}
		m.setThemeByName(msg.Name)
		return m, nil
	case core.PageCreatedMsg:
		return m, m.newDraft(msg.Page.ParentID)
	case core.PageUpdatedMsg:
		m.renamePage(msg.Page.ID, msg.Page.Title)
		return m, nil
	case core.PageDeletedMsg:
		m.confirmDelete(msg.ID)
		return m, nil
	case core.VimSaveMsg:
		return m, m.handleSave(msg)
	case core.VimQuitMsg:
		return m, m.handleQuit(msg)
	case core.VimCloseBufferMsg:
		return m, m.closeBuffer(m.active, msg.Force)
	case core.VimNewBufferMsg:
		return m, m.newDraft(m.sidebar.ContextParentID())
	case tea.MouseMsg:
		return m, m.handleMouse(msg)
	case tea.KeyMsg:
		return m, m.handleKey(msg)
	}

	// Anything else (cursor blinks etc.) goes to the components that animate.
	var c1, c2 tea.Cmd
	m.sidebar, c1 = m.sidebar.Update(msg)
	m.content, c2 = m.content.Update(msg)
	return m, tea.Batch(c1, c2)
}

func (m *Model) handleSave(msg core.VimSaveMsg) tea.Cmd {
	b := m.activeBuffer()
	if b == nil {
		m.setError("E32: No file name")
		return nil
	}
	if msg.Path != "" {
		dir, name := path.Split(strings.TrimPrefix(filepath.ToSlash(msg.Path), "/"))
		if err := m.saveBufferAs(b, dir, name); err != nil {
			m.setError(err.Error())
		}
		return nil
	}
	return m.saveBuffer(b, nil)
}

func (m *Model) handleQuit(msg core.VimQuitMsg) tea.Cmd {
	if msg.Save {
		if b := m.activeBuffer(); b != nil {
			return m.saveBuffer(b, func(m *Model) tea.Cmd { return m.requestQuit(false) })
		}
	}
	return m.requestQuit(msg.Force)
}

func (m *Model) renamePage(id, title string) {
	m.stashActive()
	updated, err := m.store.Rename(id, title)
	if err != nil {
		m.setError("Rename failed: " + err.Error())
		return
	}
	activeBefore := m.active
	m.remapBuffers(id, updated.ID)
	m.reloadTree()
	m.sidebar.SetSelectedID(updated.ID)
	if b := m.activeBuffer(); b != nil && m.active != activeBefore {
		m.content.SetBuffer(b.page(), false)
		m.preview.SetPage(b.page())
		m.sidebar.SetActiveID(b.id)
	}
	m.setStatus("Renamed to " + updated.ID)
}

func (m *Model) confirmDelete(id string) {
	p, err := m.store.Get(id)
	if err != nil {
		m.setError("Not found: " + id)
		return
	}
	msg := []string{"The file is removed from disk. This cannot be undone."}
	if p.IsFolder {
		msg = []string{"The folder and everything inside it are removed from disk.", "This cannot be undone."}
	} else if sidecar := strings.TrimSuffix(id, ".md"); sidecar != id {
		if info, err := os.Stat(filepath.Join(m.store.Root(), filepath.FromSlash(sidecar))); err == nil && info.IsDir() {
			msg = []string{"The note and all of its sub-notes are removed from disk.", "This cannot be undone."}
		}
	}
	m.confirm = &confirmDialog{
		title:   "Delete " + id + "?",
		message: msg,
		buttons: []string{"Delete", "Cancel"},
		danger:  true,
		onChoose: func(m *Model, choice int) tea.Cmd {
			if choice != 0 {
				return nil
			}
			if err := m.store.Delete(id); err != nil {
				m.setError("Delete failed: " + err.Error())
				return nil
			}
			m.dropBuffersUnder(id)
			m.reloadTree()
			m.refreshModified()
			m.setStatus("Deleted " + id)
			if len(m.buffers) == 0 && len(m.store.List()) == 0 {
				m.goHome()
			}
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// Keyboard

func (m *Model) handleKey(k tea.KeyMsg) tea.Cmd {
	s := k.String()
	if key.Matches(k, m.keys.Quit) {
		return m.requestQuit(false)
	}
	if m.viewMode == viewModeDashboard {
		return m.handleDashboardKey(k)
	}

	sidebarBusy := m.focus == focusSidebar && m.sidebar.IsBusy()
	editorBusy := m.focus == focusEditor && m.content.Mode() != content.ModeNormal

	// VSCode-style shortcuts work in every mode.
	switch {
	case key.Matches(k, m.keys.Save):
		if b := m.activeBuffer(); b != nil {
			return m.saveBuffer(b, nil)
		}
		return nil
	case key.Matches(k, m.keys.NewPage) && !sidebarBusy:
		return m.newDraft(m.sidebar.ContextParentID())
	case key.Matches(k, m.keys.ToggleSidebar):
		return m.toggleSidebar()
	case key.Matches(k, m.keys.Find):
		return m.startFind()
	}

	if !sidebarBusy && !editorBusy {
		if m.leaderPending {
			m.leaderPending = false
			return m.handleLeader(s)
		}
		switch s {
		case " ":
			m.leaderPending = true
			return nil
		case "?":
			m.showKeymap = true
			return nil
		case "tab":
			return m.movePane(1, true)
		case "shift+tab":
			return m.movePane(-1, true)
		case "ctrl+h":
			return m.movePane(-1, false)
		case "ctrl+l":
			return m.movePane(1, false)
		case "\\":
			return m.openFinder()
		case "[":
			return m.cycleBuffer(-1)
		case "]":
			return m.cycleBuffer(1)
		case "esc":
			if m.focus != focusEditor && m.activeBuffer() != nil {
				return m.focusPane(focusEditor)
			}
		}
	}
	m.leaderPending = false

	var cmd tea.Cmd
	switch m.focus {
	case focusSidebar:
		m.sidebar, cmd = m.sidebar.Update(k)
	case focusPreview:
		m.preview, cmd = m.preview.Update(k)
	default:
		if m.activeBuffer() == nil && s == "i" {
			return m.newDraft(m.sidebar.ContextParentID())
		}
		m.content, cmd = m.content.Update(k)
		if b := m.activeBuffer(); b != nil {
			m.preview.SetContent(m.content.Value())
			m.refreshModified()
		}
	}
	return cmd
}

// handleLeader runs NvChad-style <Space> mappings.
func (m *Model) handleLeader(s string) tea.Cmd {
	switch s {
	case "e":
		return m.focusPane(focusSidebar)
	case "f":
		return m.startFind()
	case "n":
		return m.newDraft(m.sidebar.ContextParentID())
	case "x":
		return m.closeBuffer(m.active, false)
	case "p":
		return m.togglePreview()
	case "tab":
		return m.cycleBuffer(1)
	case "shift+tab":
		return m.cycleBuffer(-1)
	case "t":
		return m.openThemePicker()
	case "d":
		m.goHome()
	case "h", "?":
		m.showKeymap = true
	case "w":
		if b := m.activeBuffer(); b != nil {
			return m.saveBuffer(b, nil)
		}
	}
	return nil
}

func (m *Model) handleDashboardKey(k tea.KeyMsg) tea.Cmd {
	s := k.String()
	switch s {
	case "n", "ctrl+n":
		return m.newDraft("")
	case "f", "\\", "ctrl+p":
		return m.startFind()
	case "e", "ctrl+b":
		return m.focusPane(focusSidebar)
	case "?":
		m.showKeymap = true
		return nil
	case "t":
		return m.openThemePicker()
	case "q":
		return m.requestQuit(false)
	case "esc":
		if m.activeBuffer() != nil {
			return m.focusPane(focusEditor)
		}
		return nil
	case "enter":
		a, id := m.dashboard.SelectedAction()
		return m.runDashboardAction(a, id)
	}
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		if i := int(s[0] - '1'); i < len(m.dashboard.RecentPages()) {
			return m.openFile(m.dashboard.RecentPages()[i].ID)
		}
		return nil
	}
	m.dashboard, _ = m.dashboard.Update(k)
	return nil
}

func (m *Model) runDashboardAction(a dashboard.Action, id string) tea.Cmd {
	switch a {
	case dashboard.ActionNewPage:
		return m.newDraft("")
	case dashboard.ActionFind:
		return m.startFind()
	case dashboard.ActionBrowse:
		return m.focusPane(focusSidebar)
	case dashboard.ActionKeymap:
		m.showKeymap = true
	case dashboard.ActionQuit:
		return m.requestQuit(false)
	case dashboard.ActionOpenRecent:
		return m.openFile(id)
	case dashboard.ActionThemes:
		return m.openThemePicker()
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mouse

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if m.viewMode == viewModeDashboard {
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if a, id, ok := m.dashboard.Click(msg.X, msg.Y); ok {
				return m.runDashboardAction(a, id)
			}
		}
		return nil
	}
	if msg.Action != tea.MouseActionPress {
		return nil
	}

	l := m.layout()
	if msg.Y == 0 {
		return m.handleTabClick(msg)
	}
	if msg.Y < l.BodyY || msg.Y >= l.BodyY+l.BodyH {
		return nil
	}

	wheel := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
	if !wheel && msg.Button != tea.MouseButtonLeft {
		return nil
	}

	local := msg
	local.Y -= l.BodyY
	var target focus
	switch {
	case msg.X < l.SidebarW:
		target = focusSidebar
	case msg.X >= l.EditorX && msg.X < l.EditorX+l.EditorW:
		target, local.X = focusEditor, msg.X-l.EditorX
	case l.PreviewW > 0 && msg.X >= l.PreviewX:
		target, local.X = focusPreview, msg.X-l.PreviewX
	default:
		return nil // a divider
	}

	var cmds []tea.Cmd
	if !wheel && m.focus != target {
		cmds = append(cmds, m.focusPane(target))
	}
	var cmd tea.Cmd
	switch target {
	case focusSidebar:
		m.sidebar, cmd = m.sidebar.Update(local)
	case focusEditor:
		m.content, cmd = m.content.Update(local)
	case focusPreview:
		m.preview, cmd = m.preview.Update(local)
	}
	return tea.Batch(append(cmds, cmd)...)
}

func (m *Model) handleTabClick(msg tea.MouseMsg) tea.Cmd {
	hit := m.tabAt(msg.X)
	switch msg.Button {
	case tea.MouseButtonMiddle:
		if hit.kind == hitTab || hit.kind == hitClose {
			return m.closeBuffer(hit.id, false)
		}
		return nil
	case tea.MouseButtonWheelUp:
		return m.cycleBuffer(-1)
	case tea.MouseButtonWheelDown:
		return m.cycleBuffer(1)
	case tea.MouseButtonLeft:
	default:
		return nil
	}
	switch hit.kind {
	case hitTab:
		if i := m.bufferIndex(hit.id); i >= 0 {
			m.showBuffer(m.buffers[i])
			return m.focusPane(focusEditor)
		}
	case hitClose:
		return m.closeBuffer(hit.id, false)
	case hitNew:
		return m.newDraft(m.sidebar.ContextParentID())
	case hitExplorer:
		return m.focusPane(focusSidebar)
	}
	return nil
}

// ---------------------------------------------------------------------------
// View

// View composes child component views into the complete terminal layout.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "loading…"
	}
	if m.width < MinTermWidth || m.height < MinTermHeight {
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.theme.Yellow).
			Render(fmt.Sprintf("Terminal too small (%dx%d)\nResize to at least %dx%d", m.width, m.height, MinTermWidth, MinTermHeight))
	}

	var screen string
	if m.viewMode == viewModeDashboard {
		screen = lipgloss.JoinVertical(lipgloss.Left, m.dashboard.View(), m.bottomBar())
	} else {
		l := m.layout()
		div := ui.Column("│", l.BodyH, lipgloss.NewStyle().Foreground(m.theme.Line))
		var cols []string
		if l.SidebarW > 0 {
			cols = append(cols, m.sidebar.View(), div)
		}
		cols = append(cols, m.content.View())
		if l.PreviewW > 0 {
			cols = append(cols, div, m.preview.View())
		}
		tabs, _ := m.tabline()
		screen = lipgloss.JoinVertical(lipgloss.Left,
			tabs,
			lipgloss.JoinHorizontal(lipgloss.Top, cols...),
			m.bottomBar(),
		)
	}

	switch {
	case m.saveAs != nil:
		box, x, y := m.saveAs.view(m.theme, m.width, m.height)
		screen = ui.Overlay(screen, box, x, y)
	case m.confirm != nil:
		box, x, y := m.confirm.view(m.theme, m.width, m.height)
		screen = ui.Overlay(screen, box, x, y)
	case m.finder != nil:
		box, x, y := m.finder.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.themes != nil:
		box, x, y := m.themes.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.browser != nil:
		box, x, y := m.browser.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.viewMode == viewModeWorkspace && m.focus == focusEditor && m.content.SlashOpen():
		if box, x, y, ok := m.content.SlashView(); ok {
			l := m.layout()
			screen = ui.Overlay(screen, box, l.EditorX+x, l.BodyY+y)
		}
	case m.showKeymap:
		box := RenderKeymapModal(m.theme, m.width, m.height)
		x, y := ui.Center(m.width, m.height, lipgloss.Width(box), lipgloss.Height(box))
		screen = ui.Overlay(screen, box, x, y)
	}
	return ui.Paint(screen, lipgloss.NewStyle().Foreground(m.theme.Fg).Background(m.theme.Bg))
}
