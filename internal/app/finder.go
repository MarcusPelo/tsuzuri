package app

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// finder is the global, Telescope-style note picker. It searches every
// Markdown note under the workspace root, recursively: file names (fuzzy)
// first, then the text inside the notes (like live grep).
type finder struct {
	input   textinput.Model
	root    string
	notes   []core.Page
	matches []finderMatch
	sel     int
	offset  int
	lines   map[string][]string // note text, split into lines
}

// maxTextHits caps text matches so huge workspaces stay responsive.
const maxTextHits = 500

type finderMatch struct {
	page  core.Page
	score int
	pos   []int // matched byte offsets into page.ID (name) or text (line hit)
	line  int   // 1-based line of a text hit; 0 for a file-name match
	text  string
}

func (m *Model) openFinder() tea.Cmd {
	th := m.theme
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = "Search file names and text in every note…"
	in.CharLimit = 120
	field := lipgloss.NewStyle().Background(th.Bg2)
	in.TextStyle = field.Foreground(th.Fg)
	in.PlaceholderStyle = field.Foreground(th.GreyFg)
	in.Cursor.Style = lipgloss.NewStyle().Foreground(th.Blue)
	in.Cursor.TextStyle = field.Foreground(th.Fg)

	var notes []core.Page
	for _, p := range m.store.List() {
		if !p.IsFolder {
			notes = append(notes, p)
		}
	}
	f := &finder{input: in, root: filepath.Base(m.store.Root()), notes: notes, lines: map[string][]string{}}
	for _, p := range notes {
		if full, err := m.store.Get(p.ID); err == nil {
			f.lines[p.ID] = strings.Split(strings.ReplaceAll(full.Content, "\t", "    "), "\n")
		}
	}
	f.refilter()
	m.finder = f
	m.leaderPending = false
	return f.input.Focus()
}

// fuzzyScore matches query as a subsequence of target (case-insensitive).
// Consecutive characters and matches at word or path boundaries score higher,
// and matches inside the file name beat matches in the folder path.
func fuzzyScore(query, target string) (int, []int, bool) {
	if query == "" {
		return 0, nil, true
	}
	q := []rune(strings.ToLower(query))
	t := strings.ToLower(target)
	nameStart := strings.LastIndex(t, "/") + 1

	var pos []int
	score, qi, prev := 0, 0, -2
	for i, r := range t {
		if qi >= len(q) {
			break
		}
		if r != q[qi] {
			continue
		}
		pos = append(pos, i)
		s := 1
		if i == prev+1 {
			s += 5
		}
		if i == 0 || i == nameStart || isBoundary(t[i-1]) {
			s += 4
		}
		if i >= nameStart {
			s += 2
		}
		score += s
		prev = i
		qi++
	}
	if qi < len(q) {
		return 0, nil, false
	}
	return score - len(t)/10, pos, true
}

func isBoundary(b byte) bool {
	r := rune(b)
	return b < 0x80 && !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func (f *finder) refilter() {
	q := strings.TrimSpace(f.input.Value())
	f.matches = f.matches[:0]
	for _, p := range f.notes {
		if score, pos, ok := fuzzyScore(strings.ReplaceAll(q, " ", ""), p.ID); ok {
			f.matches = append(f.matches, finderMatch{page: p, score: score, pos: pos})
		}
	}
	sort.SliceStable(f.matches, func(i, j int) bool {
		if f.matches[i].score != f.matches[j].score {
			return f.matches[i].score > f.matches[j].score
		}
		return f.matches[i].page.UpdatedAt.After(f.matches[j].page.UpdatedAt)
	})

	// Text inside notes (case-insensitive), in path order, after name hits.
	if needle := strings.ToLower(q); len([]rune(needle)) >= 2 {
		byPath := append([]core.Page(nil), f.notes...)
		sort.Slice(byPath, func(i, j int) bool { return byPath[i].ID < byPath[j].ID })
		hits := 0
	scan:
		for _, p := range byPath {
			for n, l := range f.lines[p.ID] {
				lower := strings.ToLower(l)
				idx := strings.Index(lower, needle)
				if idx < 0 || len(lower) != len(l) {
					continue
				}
				trimmed := strings.TrimLeft(l, " ")
				shift := len(l) - len(trimmed)
				var pos []int
				for k := idx; k < idx+len(needle); k++ {
					pos = append(pos, k-shift)
				}
				f.matches = append(f.matches, finderMatch{page: p, pos: pos, line: n + 1, text: trimmed})
				if hits++; hits >= maxTextHits {
					break scan
				}
			}
		}
	}
	f.sel, f.offset = 0, 0
}

func (f *finder) move(delta, visible int) {
	if len(f.matches) == 0 {
		return
	}
	f.sel = (f.sel + delta + len(f.matches)) % len(f.matches)
	if f.sel < f.offset {
		f.offset = f.sel
	}
	if visible > 0 && f.sel >= f.offset+visible {
		f.offset = f.sel - visible + 1
	}
}

func (f *finder) selected() (core.Page, bool) {
	if f.sel < 0 || f.sel >= len(f.matches) {
		return core.Page{}, false
	}
	return f.matches[f.sel].page, true
}

// finderGeom is the finder's on-screen box.
type finderGeom struct {
	x, y, w, h  int
	listW       int // results column width (inner)
	previewW    int // 0 when too narrow for a preview
	listTop     int // first result row, relative to the inner area
	listRows    int
	innerHeight int
}

func finderLayout(W, H int) finderGeom {
	w := min(max(W*4/5, 50), 130)
	w = min(w, W-2)
	h := min(max(H*7/10, 12), 34)
	h = min(h, H-2)
	g := finderGeom{w: w, h: h}
	g.x, g.y = ui.Center(W, H, w, h)
	inner := w - 2
	g.innerHeight = h - 2
	if inner >= 90 {
		g.listW = inner * 9 / 20
		g.previewW = inner - g.listW - 1
	} else {
		g.listW = inner
	}
	g.listTop = 3
	g.listRows = max(g.innerHeight-g.listTop-1, 1)
	return g
}

func (m *Model) finderOpen(f *finder, newTab bool) tea.Cmd {
	p, ok := f.selected()
	if !ok {
		return nil
	}
	line := f.matches[f.sel].line
	m.finder = nil
	var cmd tea.Cmd
	if newTab {
		cmd = m.openFile(p.ID)
	} else {
		cmd = m.openInCurrentBuffer(p.ID)
	}
	if line > 0 && m.active == p.ID {
		m.content.GotoLine(line)
	}
	return cmd
}

func (f *finder) update(m *Model, msg tea.Msg) tea.Cmd {
	g := finderLayout(m.width, m.height)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.finder = nil
			return nil
		case "enter":
			return m.finderOpen(f, false)
		case "ctrl+t":
			return m.finderOpen(f, true)
		case "down", "ctrl+n", "ctrl+j":
			f.move(1, g.listRows)
			return nil
		case "up", "ctrl+p", "ctrl+k":
			f.move(-1, g.listRows)
			return nil
		case "pgdown", "ctrl+d":
			f.move(g.listRows/2, g.listRows)
			return nil
		case "pgup", "ctrl+u":
			f.move(-g.listRows/2, g.listRows)
			return nil
		}
		before := f.input.Value()
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		if f.input.Value() != before {
			f.refilter()
		}
		return cmd

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			f.move(-1, g.listRows)
			return nil
		case tea.MouseButtonWheelDown:
			f.move(1, g.listRows)
			return nil
		case tea.MouseButtonLeft:
		default:
			return nil
		}
		inside := msg.X >= g.x && msg.X < g.x+g.w && msg.Y >= g.y && msg.Y < g.y+g.h
		if !inside {
			m.finder = nil
			return nil
		}
		row := msg.Y - g.y - 1 - g.listTop
		col := msg.X - g.x - 1
		if row >= 0 && row < g.listRows && col < g.listW {
			if idx := f.offset + row; idx < len(f.matches) {
				f.sel = idx
				return m.finderOpen(f, false)
			}
		}
		return nil

	default:
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		return cmd
	}
}

func (f *finder) view(m *Model) (string, int, int) {
	th := m.theme
	g := finderLayout(m.width, m.height)
	inner := g.w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	field := lipgloss.NewStyle().Background(th.Bg2)

	rows := make([]string, 0, g.innerHeight)

	// Title + count.
	title := bg.Foreground(th.Blue).Render(" 󰍉 ") + bg.Foreground(th.Fg).Bold(true).Render("Find Note") + bg.Foreground(th.GreyFg2).Render("  in "+f.root+"/")
	count := bg.Foreground(th.GreyFg2).Render(fmt.Sprintf("%d results · %d notes ", len(f.matches), len(f.notes)))
	rows = append(rows, title+bg.Render(strings.Repeat(" ", max(inner-lipgloss.Width(title)-lipgloss.Width(count), 0)))+count)

	// Prompt.
	f.input.Width = max(inner-8, 1)
	prompt := field.Foreground(th.Blue).Bold(true).Render(" \uf002 ") + f.input.View()
	rows = append(rows, bg.Render(" ")+ui.FitLine(prompt, inner-2, field)+bg.Render(" "))
	rows = append(rows, bg.Foreground(th.Line).Render(strings.Repeat("─", inner)))

	// Results and preview side by side.
	var prevLines []string
	var prevTitle string
	prevStart, hitLine := 0, 0
	if p, ok := f.selected(); ok && g.previewW > 0 {
		prevLines = f.lines[p.ID]
		prevTitle = p.ID
		if hitLine = f.matches[f.sel].line; hitLine > 0 {
			prevTitle = fmt.Sprintf("%s:%d", p.ID, hitLine)
			prevStart = max(hitLine-1-(g.listRows-1)/3, 0)
		}
	}
	sep := bg.Foreground(th.Line).Render("│")
	for r := 0; r < g.listRows; r++ {
		left := f.resultRow(th, f.offset+r, g.listW)
		if r == 0 && len(f.matches) == 0 {
			left = ui.FitLine(bg.Foreground(th.GreyFg).Italic(true).Render("   No matching notes"), g.listW, bg)
		}
		if g.previewW == 0 {
			rows = append(rows, left)
			continue
		}
		var right string
		switch {
		case r == 0 && prevTitle != "":
			right = bg.Foreground(th.Yellow).Render(" 󰈈 ") + bg.Foreground(th.GreyFg2).Render(ui.Truncate(prevTitle, g.previewW-5))
		case r >= 1 && prevStart+r-1 < len(prevLines):
			n := prevStart + r - 1
			right = previewLine(th, prevLines[n], g.previewW)
			if n+1 == hitLine {
				right = ui.FitLine(lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Fg).Render(" "+ui.Truncate(prevLines[n], g.previewW-2)), g.previewW, lipgloss.NewStyle().Background(th.OneBg2))
			}
		}
		rows = append(rows, left+sep+ui.FitLine(right, g.previewW, bg))
	}

	help := "↑↓ move · Enter open here (jumps to the line) · Ctrl+T new tab · Esc close"
	rows = append(rows, bg.Foreground(th.GreyFg).Render(" "+ui.Truncate(help, inner-2)))
	return panel(th, rows, inner), g.x, g.y
}

func (f *finder) resultRow(th theme.Theme, idx, width int) string {
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	if idx >= len(f.matches) {
		return ui.FitLine("", width, bg)
	}
	mt := f.matches[idx]
	selected := idx == f.sel
	rowBg := th.DarkerBg
	if selected {
		rowBg = th.OneBg2
	}
	base := lipgloss.NewStyle().Background(rowBg)
	hit := base.Foreground(th.Blue).Bold(true)

	marker := base.Render("  ")
	if selected {
		marker = base.Foreground(th.Blue).Render("▌ ")
	}
	icon := base.Foreground(th.NordBlue).Render(" ")

	if mt.line > 0 {
		return f.textRow(th, mt, base, hit, marker, width)
	}

	id := mt.page.ID
	nameStart := strings.LastIndex(id, "/") + 1
	matched := map[int]bool{}
	for _, p := range mt.pos {
		matched[p] = true
	}
	render := func(from, to int, normal lipgloss.Style) string {
		var b strings.Builder
		for i, r := range id[from:to] {
			st := normal
			if matched[from+i] {
				st = hit
			}
			b.WriteString(st.Render(string(r)))
		}
		return b.String()
	}

	name := render(nameStart, len(id), base.Foreground(th.Fg).Bold(selected))
	var dir string
	if nameStart > 0 {
		dir = base.Render("  ") + render(0, nameStart-1, base.Foreground(th.GreyFg))
	}
	line := marker + icon + name + dir
	if lipgloss.Width(line) > width {
		line = ui.FitLine(line, width-1, base) + base.Foreground(th.GreyFg).Render("…")
	}
	return ui.FitLine(line, width, base)
}

func previewLine(th theme.Theme, l string, width int) string {
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	trim := strings.TrimSpace(l)
	st := bg.Foreground(th.GreyFg2)
	switch {
	case strings.HasPrefix(trim, "#"):
		st = bg.Foreground(th.Blue).Bold(true)
	case strings.HasPrefix(trim, "- [x]"), strings.HasPrefix(trim, "- [X]"):
		st = bg.Foreground(th.Green)
	case strings.HasPrefix(trim, "- "), strings.HasPrefix(trim, "* "):
		st = bg.Foreground(th.Fg)
	case strings.HasPrefix(trim, ">"):
		st = bg.Foreground(th.Purple)
	case strings.HasPrefix(trim, "```"):
		st = bg.Foreground(th.Yellow)
	}
	return bg.Render(" ") + st.Render(ui.Truncate(l, width-2))
}

// openInCurrentBuffer shows id in the active tab instead of adding a new one
// (Telescope / ":e" behaviour). An already-open file is simply focused, and a
// tab with unsaved changes is never replaced — the file opens next to it.
func (m *Model) openInCurrentBuffer(id string) tea.Cmd {
	if i := m.bufferIndex(id); i >= 0 {
		m.showBuffer(m.buffers[i])
		return m.focusPane(focusEditor)
	}
	cur := m.activeBuffer()
	if cur == nil || m.isDirty(cur) {
		cmd := m.openFile(id)
		if cur != nil {
			m.setStatus(fmt.Sprintf("Opened in a new tab — %s has unsaved changes", cur.fileName()))
		}
		return cmd
	}
	p, err := m.store.Get(id)
	if err != nil || p.IsFolder {
		m.setError("Cannot open " + id)
		return nil
	}
	cur.id, cur.title, cur.saved, cur.text, cur.dir = p.ID, p.Title, p.Content, p.Content, ""
	m.active = ""
	m.showBuffer(cur)
	m.refreshModified()
	return m.focusPane(focusEditor)
}

// textRow renders a text hit: "path:line  …matched text…".
func (f *finder) textRow(th theme.Theme, mt finderMatch, base, hit lipgloss.Style, marker string, width int) string {
	loc := base.Foreground(th.GreyFg2).Render(fmt.Sprintf("%s:%d", mt.page.ID, mt.line))
	icon := base.Foreground(th.Yellow).Render("\uf002 ")
	head := marker + icon + loc + base.Render("  ")
	room := width - lipgloss.Width(head)

	text := mt.text
	start := 0
	if len(mt.pos) > 0 && mt.pos[0] > room/2 {
		start = mt.pos[0] - room/3 // keep the match visible on long lines
		for start > 0 && !utf8.RuneStart(text[start]) {
			start--
		}
	}
	matched := map[int]bool{}
	for _, p := range mt.pos {
		matched[p] = true
	}
	var b strings.Builder
	if start > 0 {
		b.WriteString(base.Foreground(th.GreyFg).Render("…"))
	}
	normal := base.Foreground(th.Fg)
	for i, r := range text[start:] {
		st := normal
		if matched[start+i] {
			st = hit
		}
		b.WriteString(st.Render(string(r)))
	}
	line := head + b.String()
	if lipgloss.Width(line) > width {
		line = ui.FitLine(line, width-1, base) + base.Foreground(th.GreyFg).Render("…")
	}
	return ui.FitLine(line, width, base)
}
