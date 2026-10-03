package app

import (
	"strings"

	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// panel wraps pre-painted rows (each exactly inner wide) in a rounded border.
func panel(th theme.Theme, rows []string, inner int) string {
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	for i, r := range rows {
		rows[i] = ui.FitLine(r, inner, bg)
	}
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(th.Blue).
		BorderBackground(th.DarkerBg).
		Render(strings.Join(rows, "\n"))
}

// ---------------------------------------------------------------------------
// Confirm dialog

type confirmDialog struct {
	title    string
	message  []string
	buttons  []string // the last button is always "cancel"
	danger   bool     // paint the first button red (e.g. Delete)
	selected int
	onChoose func(m *Model, choice int) tea.Cmd
}

type confirmGeom struct {
	x, y, w, h int
	buttonRow  int
	spans      [][2]int // inner-x start/end per button
}

func (d *confirmDialog) innerWidth() int {
	w := 46
	for _, l := range d.message {
		w = max(w, lipgloss.Width(l)+6)
	}
	return w
}

func (d *confirmDialog) geom(W, H int) confirmGeom {
	iw := d.innerWidth()
	h := 2 + len(d.message) + 2 + 1
	x, y := ui.Center(W, H, iw+2, h+2)
	g := confirmGeom{x: x, y: y, w: iw + 2, h: h + 2, buttonRow: h - 1}
	cx := 2
	for _, b := range d.buttons {
		bw := lipgloss.Width(b) + 4
		g.spans = append(g.spans, [2]int{cx, cx + bw})
		cx += bw + 2
	}
	return g
}

func (d *confirmDialog) view(th theme.Theme, W, H int) (string, int, int) {
	g := d.geom(W, H)
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	icon := bg.Foreground(th.Yellow).Render("  ")
	if d.danger {
		icon = bg.Foreground(th.Red).Render(" 󰆴 ")
	}
	rows := []string{icon + bg.Foreground(th.Fg).Bold(true).Render(d.title), ""}
	for i, l := range d.message {
		fg := th.Fg
		if i > 0 {
			fg = th.GreyFg2
		}
		rows = append(rows, bg.Foreground(fg).Render("  "+l))
	}
	rows = append(rows, "")

	var btns strings.Builder
	btns.WriteString(bg.Render("  "))
	for i, b := range d.buttons {
		st := lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Fg)
		if i == d.selected {
			accent := th.Blue
			if d.danger && i == 0 {
				accent = th.Red
			}
			st = lipgloss.NewStyle().Background(accent).Foreground(th.Bg).Bold(true)
		}
		btns.WriteString(st.Render("  " + b + "  "))
		btns.WriteString(bg.Render("  "))
	}
	rows = append(rows, btns.String())
	return panel(th, rows, g.w-2), g.x, g.y
}

// update handles keys and mouse; it returns done=true once the dialog closed.
func (d *confirmDialog) update(m *Model, msg tea.Msg) (tea.Cmd, bool) {
	choose := func(i int) (tea.Cmd, bool) {
		m.confirm = nil
		if i == len(d.buttons)-1 {
			m.quitting = false
		}
		return d.onChoose(m, i), true
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "shift+tab", "up", "k":
			d.selected = (d.selected - 1 + len(d.buttons)) % len(d.buttons)
		case "right", "l", "tab", "down", "j":
			d.selected = (d.selected + 1) % len(d.buttons)
		case "enter", " ":
			return choose(d.selected)
		case "esc", "q":
			return choose(len(d.buttons) - 1)
		case "ctrl+c":
			return tea.Quit, true
		default:
			for i, b := range d.buttons {
				if strings.EqualFold(msg.String(), b[:1]) {
					return choose(i)
				}
			}
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
			return nil, false
		}
		g := d.geom(m.width, m.height)
		if msg.Y != g.y+1+g.buttonRow {
			return nil, false
		}
		for i, sp := range g.spans {
			if x := msg.X - g.x - 1; x >= sp[0] && x < sp[1] {
				return choose(i)
			}
		}
	}
	return nil, false
}

// ---------------------------------------------------------------------------
// Save As dialog

const saveAsSuggestions = 5

// Row indices inside the Save As panel (used for mouse hit-testing).
const (
	saRowFolderInput = 3
	saRowSuggest     = 4
	saRowNameInput   = saRowSuggest + saveAsSuggestions + 2
)

type saveAsDialog struct {
	buf    *buffer
	after  func(*Model) tea.Cmd
	folder textinput.Model
	name   textinput.Model
	field  int // 0 folder, 1 name
	dirs   []string
	query  string
	sel    int
	err    string
}

func (m *Model) openSaveAs(b *buffer, after func(*Model) tea.Cmd) {
	th := m.theme
	in := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = placeholder
		ti.CharLimit = 200
		field := lipgloss.NewStyle().Background(th.OneBg)
		ti.TextStyle = field.Foreground(th.Fg)
		ti.PlaceholderStyle = field.Foreground(th.GreyFg)
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(th.Blue)
		ti.Cursor.TextStyle = field.Foreground(th.Fg)
		return ti
	}

	dir := b.dir
	name := b.title
	if !b.draft() {
		dir = strings.TrimSuffix(strings.TrimSuffix(b.id, b.fileName()), "/")
	}

	d := &saveAsDialog{buf: b, after: after, dirs: m.store.Dirs(), sel: -1}
	d.folder = in("/  (workspace root)")
	d.folder.SetValue(dir)
	d.name = in("note name")
	d.name.SetValue(name)
	d.name.CursorEnd()
	d.field = 1
	d.name.Focus()
	for i, s := range d.filtered() {
		if s == dir {
			d.sel = i
		}
	}
	m.saveAs = d
}

func (d *saveAsDialog) filtered() []string {
	q := strings.ToLower(strings.Trim(d.query, "/ "))
	var out []string
	for _, dir := range d.dirs {
		if q == "" || strings.Contains(strings.ToLower(dir), q) {
			out = append(out, dir)
		}
	}
	return out
}

func (d *saveAsDialog) focusField(f int) tea.Cmd {
	d.field = f
	if f == 0 {
		d.name.Blur()
		return d.folder.Focus()
	}
	d.folder.Blur()
	return d.name.Focus()
}

func (d *saveAsDialog) pick(i int) {
	list := d.filtered()
	if i < 0 || i >= len(list) {
		return
	}
	d.sel = i
	d.folder.SetValue(list[i])
	d.folder.CursorEnd()
}

func (d *saveAsDialog) dir() string { return strings.Trim(strings.TrimSpace(d.folder.Value()), "/") }

func (d *saveAsDialog) submit(m *Model) tea.Cmd {
	if err := m.saveBufferAs(d.buf, d.dir(), d.name.Value()); err != nil {
		d.err = err.Error()
		return nil
	}
	m.saveAs = nil
	if d.after != nil {
		return d.after(m)
	}
	return nil
}

func (d *saveAsDialog) update(m *Model, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		d.err = ""
		switch msg.String() {
		case "esc", "ctrl+c":
			m.saveAs = nil
			m.quitting = false
			m.setStatus("Save cancelled")
			return nil
		case "enter":
			return d.submit(m)
		case "tab", "shift+tab":
			return d.focusField(1 - d.field)
		case "up", "ctrl+p":
			if d.field == 0 {
				d.pick(max(d.sel-1, 0))
				return nil
			}
			return d.focusField(0)
		case "down", "ctrl+n":
			if d.field == 0 {
				d.pick(min(d.sel+1, len(d.filtered())-1))
				return nil
			}
			return nil
		}
		var cmd tea.Cmd
		if d.field == 0 {
			d.folder, cmd = d.folder.Update(msg)
			d.query = d.folder.Value()
			d.sel = -1
		} else {
			d.name, cmd = d.name.Update(msg)
		}
		return cmd

	case tea.MouseMsg:
		g := d.geom(m.width, m.height)
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		list := d.filtered()
		start := d.windowStart(len(list))
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if d.field == 0 || d.sel >= 0 {
				d.pick(max(d.sel-1, 0))
			}
			return nil
		case tea.MouseButtonWheelDown:
			d.pick(min(d.sel+1, len(list)-1))
			return nil
		case tea.MouseButtonLeft:
		default:
			return nil
		}
		row := msg.Y - g.y - 1
		switch {
		case row == saRowFolderInput:
			return d.focusField(0)
		case row >= saRowSuggest && row < saRowSuggest+saveAsSuggestions:
			d.pick(start + row - saRowSuggest)
			return nil
		case row == saRowNameInput:
			return d.focusField(1)
		}
	}
	return nil
}

func (d *saveAsDialog) windowStart(n int) int {
	if d.sel < saveAsSuggestions {
		return 0
	}
	return min(d.sel-saveAsSuggestions+1, max(n-saveAsSuggestions, 0))
}

type saveAsGeom struct{ x, y, w, h int }

func (d *saveAsDialog) innerWidth(W int) int { return max(min(66, W-6), 36) }

func (d *saveAsDialog) geom(W, H int) saveAsGeom {
	iw := d.innerWidth(W)
	h := saRowNameInput + 5
	x, y := ui.Center(W, H, iw+2, h+2)
	return saveAsGeom{x: x, y: y, w: iw + 2, h: h + 2}
}

func (d *saveAsDialog) view(th theme.Theme, W, H int) (string, int, int) {
	g := d.geom(W, H)
	iw := g.w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	field := lipgloss.NewStyle().Background(th.OneBg)
	label := func(text string, active bool) string {
		fg := th.GreyFg2
		if active {
			fg = th.Blue
		}
		return bg.Foreground(fg).Bold(active).Render("  " + text)
	}
	inputRow := func(view, suffix string, active bool) string {
		w := iw - 4
		bar := field.Foreground(th.Grey).Render(" ▏")
		if active {
			bar = field.Foreground(th.Blue).Render(" ▏")
		}
		suf := field.Foreground(th.GreyFg2).Render(suffix)
		body := ui.FitLine(bar+view, w-lipgloss.Width(suf), field) + suf
		return bg.Render("  ") + body
	}

	rows := []string{
		bg.Foreground(th.Blue).Render(" 󰆓 ") + bg.Foreground(th.Fg).Bold(true).Render("Save As") +
			bg.Foreground(th.GreyFg).Render("   "+d.buf.fileName()),
		"",
		label("Folder", d.field == 0),
	}
	d.folder.Width = iw - 8
	rows = append(rows, inputRow(d.folder.View(), "", d.field == 0))

	list := d.filtered()
	start := d.windowStart(len(list))
	for i := 0; i < saveAsSuggestions; i++ {
		idx := start + i
		if idx >= len(list) {
			if i == 0 {
				rows = append(rows, bg.Foreground(th.GreyFg).Italic(true).Render("     new folder will be created"))
			} else {
				rows = append(rows, "")
			}
			continue
		}
		name := list[idx]
		text := name
		if name == "" {
			text = "/" + "  workspace root"
		}
		st := bg.Foreground(th.GreyFg2)
		ico := bg.Foreground(th.Blue)
		mark := "   "
		if idx == d.sel {
			st = lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Blue).Bold(true)
			ico = lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Blue)
			mark = " ▸ "
		}
		row := st.Render("  "+mark) + ico.Render("󰉋  ") + st.Render(ui.Truncate(text, iw-12))
		if idx == d.sel {
			row = ui.FitLine(row, iw-2, lipgloss.NewStyle().Background(th.OneBg2))
		}
		rows = append(rows, row)
	}
	rows = append(rows, "", label("Name", d.field == 1))
	d.name.Width = iw - 12
	rows = append(rows, inputRow(d.name.View(), " .md ", d.field == 1))
	rows = append(rows, "")

	if d.err != "" {
		rows = append(rows, bg.Foreground(th.Red).Render("    "+ui.Truncate(d.err, iw-6)))
	} else if rel, err := core.NormalizeNotePath(d.dir(), d.name.Value()); err != nil {
		rows = append(rows, bg.Foreground(th.Red).Render("    "+ui.Truncate(err.Error(), iw-6)))
	} else {
		rows = append(rows, bg.Foreground(th.Green).Render("  󰈙  ")+bg.Foreground(th.Fg).Render(ui.Truncate(rel, iw-8)))
	}
	rows = append(rows, "")
	help := "Tab switch field · ↑↓ folder · Enter save · Esc cancel"
	rows = append(rows, bg.Foreground(th.GreyFg).Render("  "+ui.Truncate(help, iw-3)))
	return panel(th, rows, iw), g.x, g.y
}
