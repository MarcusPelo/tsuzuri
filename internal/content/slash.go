package content

import (
	"strings"

	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// slashItem is one entry of the Notion-style "/" block menu.
type slashItem struct {
	section string
	icon    string
	label   string
	hint    string
	aliases string
	before  string // inserted before the cursor
	after   string // inserted after the cursor
	action  string // "page" / "link" are handled by the app
	block   bool   // starts its own line
}

var slashItems = []slashItem{
	{section: "Basic blocks", icon: "\U000f026b", label: "Heading 1", hint: "#", aliases: "h1 title", before: "# ", block: true},
	{section: "Basic blocks", icon: "\U000f026c", label: "Heading 2", hint: "##", aliases: "h2 subtitle", before: "## ", block: true},
	{section: "Basic blocks", icon: "\U000f026d", label: "Heading 3", hint: "###", aliases: "h3", before: "### ", block: true},
	{section: "Basic blocks", icon: "\U000f026e", label: "Heading 4", hint: "####", aliases: "h4", before: "#### ", block: true},
	{section: "Basic blocks", icon: "\U000f0279", label: "Bulleted list", hint: "-", aliases: "bullet ul unordered", before: "- ", block: true},
	{section: "Basic blocks", icon: "\U000f027b", label: "Numbered list", hint: "1.", aliases: "ol ordered number", before: "1. ", block: true},
	{section: "Basic blocks", icon: "\U000f0135", label: "To-do list", hint: "[]", aliases: "todo task checkbox check", before: "- [ ] ", block: true},
	{section: "Basic blocks", icon: "\U000f0142", label: "Toggle list", hint: ">", aliases: "details collapse", before: "<details>\n<summary>", after: "</summary>\n\n</details>", block: true},
	{section: "Basic blocks", icon: "\U000f0219", label: "Page", aliases: "subpage sub-note note new", action: "page"},
	{section: "Basic blocks", icon: "\U000f02fd", label: "Callout", aliases: "note tip info admonition", before: "> [!NOTE]\n> ", block: true},
	{section: "Basic blocks", icon: "\U000f027e", label: "Quote", hint: "\"", aliases: "blockquote", before: "> ", block: true},
	{section: "Basic blocks", icon: "\U000f04eb", label: "Table", aliases: "grid", before: "| ", after: " | Column 2 |\n| --- | --- |\n|  |  |", block: true},
	{section: "Basic blocks", icon: "\U000f0374", label: "Divider", hint: "---", aliases: "hr rule line separator", before: "---\n", block: true},
	{section: "Basic blocks", icon: "\U000f0337", label: "Link to page", aliases: "mention reference note", action: "link"},
	{section: "Media", icon: "\U000f02e9", label: "Image", aliases: "picture photo img", before: "![", after: "](image.png)"},
	{section: "Media", icon: "\U000f0567", label: "Video", aliases: "movie youtube", before: "[▶ ", after: "](https://)"},
	{section: "Media", icon: "\U000f075a", label: "Audio", aliases: "sound music", before: "[♪ ", after: "](audio.mp3)"},
	{section: "Media", icon: "\U000f0169", label: "Code", hint: "```", aliases: "snippet block pre", before: "```\n", after: "\n```", block: true},
}

const (
	slashWidth   = 36
	slashMaxRows = 12
)

type slashMenu struct {
	row, col int // position of the "/" that opened the menu
	sel      int
	offset   int
}

// SlashOpen reports whether the "/" block menu is showing.
func (m Model) SlashOpen() bool { return m.slash != nil }

func (m *Model) slashQuery() (string, bool) {
	row, col := m.textarea.RowCol()
	if m.slash == nil || row != m.slash.row || col <= m.slash.col {
		return "", false
	}
	before := []rune(m.textarea.LineBeforeCursor())
	if m.slash.col >= len(before) || before[m.slash.col] != '/' {
		return "", false
	}
	q := string(before[m.slash.col+1:])
	if strings.Contains(q, "  ") || strings.HasPrefix(q, " ") {
		return "", false
	}
	return q, true
}

func (m *Model) slashMatches() []slashItem {
	q, _ := m.slashQuery()
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return slashItems
	}
	var out []slashItem
	for _, it := range slashItems {
		if strings.Contains(strings.ToLower(it.label+" "+it.aliases), q) {
			out = append(out, it)
		}
	}
	return out
}

// maybeOpenSlash opens the menu right after a "/" typed at the start of a
// line or after whitespace, as in Notion.
func (m *Model) maybeOpenSlash(k tea.KeyMsg) {
	if k.Type != tea.KeyRunes || string(k.Runes) != "/" {
		return
	}
	before := []rune(m.textarea.LineBeforeCursor())
	n := len(before)
	if n == 0 || before[n-1] != '/' {
		return
	}
	if n >= 2 && before[n-2] != ' ' {
		return
	}
	row, _ := m.textarea.RowCol()
	m.slash = &slashMenu{row: row, col: n - 1}
}

// updateSlash handles a key while the menu is open. handled=false lets the
// key reach the editor (typing filters the menu).
func (m *Model) updateSlash(k tea.KeyMsg) (tea.Cmd, bool) {
	items := m.slashMatches()
	switch k.String() {
	case "esc":
		m.slash = nil
		return nil, true
	case "up", "ctrl+p", "ctrl+k":
		if len(items) > 0 {
			m.slash.sel = (m.slash.sel - 1 + len(items)) % len(items)
			m.scrollSlash()
		}
		return nil, true
	case "down", "ctrl+n", "ctrl+j":
		if len(items) > 0 {
			m.slash.sel = (m.slash.sel + 1) % len(items)
			m.scrollSlash()
		}
		return nil, true
	case "enter", "tab":
		if len(items) == 0 {
			m.slash = nil
			return nil, k.String() == "tab"
		}
		return m.applySlash(items[min(m.slash.sel, len(items)-1)]), true
	}
	return nil, false
}

func (m *Model) scrollSlash() {
	s := m.slash
	if s.sel < s.offset {
		s.offset = s.sel
	}
	if s.sel >= s.offset+slashMaxRows {
		s.offset = s.sel - slashMaxRows + 1
	}
}

// afterSlashKey re-validates the menu once the editor consumed a key.
func (m *Model) afterSlashKey() {
	if m.slash == nil {
		return
	}
	if _, ok := m.slashQuery(); !ok {
		m.slash = nil
		return
	}
	if q, _ := m.slashQuery(); strings.HasSuffix(q, " ") && len(m.slashMatches()) == 0 {
		m.slash = nil
		return
	}
	m.slash.sel, m.slash.offset = 0, 0
}

// applySlash replaces "/query" with the chosen block's Markdown.
func (m *Model) applySlash(it slashItem) tea.Cmd {
	q, _ := m.slashQuery()
	ta := &m.textarea
	ta.DeleteBefore(1 + len([]rune(q)))
	m.slash = nil

	if it.action != "" {
		action := it.action
		return func() tea.Msg { return core.SlashActionMsg{Action: action} }
	}
	if it.block && strings.TrimSpace(ta.LineBeforeCursor()) != "" {
		ta.InsertString("\n")
	}
	ta.InsertString(it.before)
	row, col := ta.RowCol()
	if it.after != "" {
		ta.InsertString(it.after)
		ta.SetRowCol(row, col)
	}
	ta.EnsureVisible()
	return nil
}

// InsertText types text at the cursor (used for "Link to page").
func (m *Model) InsertText(s string) {
	m.textarea.InsertString(s)
	m.textarea.EnsureVisible()
}

// slashGeom places the menu under the cursor (or above it when there is no
// room), relative to the editor pane.
func (m Model) slashGeom(rows int) (x, y, w, h int, ok bool) {
	cx, cy, visible := m.textarea.CursorScreen()
	if !visible {
		return 0, 0, 0, 0, false
	}
	w = min(slashWidth, m.width)
	h = rows + 2
	x = min(max(cx-1, 0), max(m.width-w, 0))
	y = cy + 1
	if y+h > m.height && cy-h >= 0 {
		y = cy - h
	}
	return x, y, w, h, true
}

type slashRow struct {
	header string
	item   int // index into matches, -1 for headers
}

func (m Model) slashRows(items []slashItem) []slashRow {
	q, _ := m.slashQuery()
	var rows []slashRow
	last := ""
	for i, it := range items {
		if strings.TrimSpace(q) == "" && it.section != last {
			rows = append(rows, slashRow{header: it.section, item: -1})
			last = it.section
		}
		rows = append(rows, slashRow{item: i})
	}
	return rows
}

// visibleSlashRows returns the rows on screen, keeping the selection visible.
func (m Model) visibleSlashRows() ([]slashRow, []slashItem) {
	items := m.slashMatches()
	rows := m.slashRows(items)
	selRow := 0
	for i, r := range rows {
		if r.item == m.slash.sel {
			selRow = i
		}
	}
	start := 0
	if selRow >= slashMaxRows {
		start = selRow - slashMaxRows + 1
	}
	end := min(start+slashMaxRows, len(rows))
	return rows[start:end], items
}

// SlashView renders the menu and its position relative to the pane.
func (m Model) SlashView() (string, int, int, bool) {
	if m.slash == nil {
		return "", 0, 0, false
	}
	th := m.theme
	rows, items := m.visibleSlashRows()
	if len(items) == 0 {
		rows = nil
	}
	n := max(len(rows), 1)
	x, y, w, _, ok := m.slashGeom(n)
	if !ok {
		return "", 0, 0, false
	}
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)

	var lines []string
	if len(items) == 0 {
		lines = append(lines, ui.FitLine(bg.Foreground(th.GreyFg).Italic(true).Render("  No results"), inner, bg))
	}
	for _, r := range rows {
		if r.item < 0 {
			lines = append(lines, ui.FitLine(bg.Foreground(th.GreyFg2).Render(" "+r.header), inner, bg))
			continue
		}
		it := items[r.item]
		base := bg
		labelFg, iconFg := th.Fg, th.GreyFg2
		if r.item == m.slash.sel {
			base = lipgloss.NewStyle().Background(th.OneBg2)
			iconFg = th.Blue
		}
		left := base.Foreground(iconFg).Render(" "+it.icon+"  ") + base.Foreground(labelFg).Render(it.label)
		right := base.Foreground(th.GreyFg).Render(it.hint + " ")
		gap := inner - lipgloss.Width(left) - lipgloss.Width(right)
		lines = append(lines, left+base.Render(strings.Repeat(" ", max(gap, 1)))+right)
	}
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(th.Line).
		BorderBackground(th.DarkerBg).
		Render(strings.Join(lines, "\n"))
	return box, x, y, true
}

// clickSlash applies the item at pane coordinates (x, y), if any.
func (m *Model) clickSlash(x, y int) (tea.Cmd, bool) {
	rows, items := m.visibleSlashRows()
	bx, by, w, h, ok := m.slashGeom(max(len(rows), 1))
	if !ok || x < bx || x >= bx+w || y < by || y >= by+h {
		m.slash = nil
		return nil, false
	}
	i := y - by - 1
	if i < 0 || i >= len(rows) || rows[i].item < 0 {
		return nil, true
	}
	m.slash.sel = rows[i].item
	return m.applySlash(items[rows[i].item]), true
}
