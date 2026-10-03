package preview

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jaisuriya-11/tsuzuri/internal/highlight"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	checkboxRegex   = regexp.MustCompile(`^(\s*)[-*+]\s+\[([ xX])\]\s*(.*)$`)
	bulletRegex     = regexp.MustCompile(`^(\s*)[-*+]\s+(.*)$`)
	numberedRegex   = regexp.MustCompile(`^(\s*)(\d+)[.)]\s+(.*)$`)
	blockquoteRegex = regexp.MustCompile(`^\s*>\s?(.*)$`)
	dividerRegex    = regexp.MustCompile(`^(\-{3,}|\*{3,}|_{3,})$`)
	headingRegex    = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
	tableSepRegex   = regexp.MustCompile(`^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$`)

	htmlCommentRegex = regexp.MustCompile(`<!--.*?-->`)
	htmlAttrRegex    = func(name string) *regexp.Regexp {
		return regexp.MustCompile(`(?i)\b` + name + `\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)
	}
	altAttr = htmlAttrRegex("alt")
)

// styles bundles the per-theme styles used while compiling.
type styles struct {
	th        theme.Theme
	text      lipgloss.Style
	muted     lipgloss.Style
	code      lipgloss.Style
	link      lipgloss.Style
	headings  [6]lipgloss.Style
	quoteBar  lipgloss.Style
	calloutBr lipgloss.Style
	bullet    lipgloss.Style
	codeBox   lipgloss.Style
	codeLang  lipgloss.Style
	rule      lipgloss.Style
}

func newStyles(th theme.Theme) styles {
	return styles{
		th:    th,
		text:  lipgloss.NewStyle().Foreground(th.Fg),
		muted: lipgloss.NewStyle().Foreground(th.GreyFg2),
		code:  lipgloss.NewStyle().Foreground(th.Orange).Background(th.OneBg),
		link:  lipgloss.NewStyle().Foreground(th.Blue).Underline(true),
		headings: [6]lipgloss.Style{
			lipgloss.NewStyle().Bold(true).Foreground(th.Blue),
			lipgloss.NewStyle().Bold(true).Foreground(th.Purple),
			lipgloss.NewStyle().Bold(true).Foreground(th.Green),
			lipgloss.NewStyle().Bold(true).Foreground(th.Yellow),
			lipgloss.NewStyle().Bold(true).Foreground(th.Cyan),
			lipgloss.NewStyle().Bold(true).Foreground(th.GreyFg2),
		},
		quoteBar:  lipgloss.NewStyle().Foreground(th.GreyFg),
		calloutBr: lipgloss.NewStyle().Foreground(th.Blue),
		bullet:    lipgloss.NewStyle().Foreground(th.Blue),
		codeBox: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(th.Line).
			Foreground(th.Fg).
			Padding(0, 1),
		codeLang: lipgloss.NewStyle().Foreground(th.GreyFg2).Italic(true),
		rule:     lipgloss.NewStyle().Foreground(th.Line),
	}
}

// compiler accumulates rendered lines, collapsing runs of blank lines.
type compiler struct {
	st      styles
	width   int
	baseDir string // folder relative image paths resolve against
	hits    []Hit
	// Document positions of the block being rendered.
	lineOffset, fence, fenceEnd int
	cal                         CalendarView
	out                         []string
	para                        []string
}

func (c *compiler) emit(lines ...string) {
	for _, l := range lines {
		if l == "" && (len(c.out) == 0 || c.out[len(c.out)-1] == "") {
			continue
		}
		c.out = append(c.out, l)
	}
}

func (c *compiler) blank() { c.emit("") }

// wrapIndent word-wraps an already styled string to the pane width, prefixing
// the first line with first and continuation lines with rest.
func (c *compiler) wrapIndent(s, first, rest string) []string {
	w := max(c.width-ansi.StringWidth(first), 8)
	lines := strings.Split(ansi.Wrap(s, w, ""), "\n")
	for i := range lines {
		if i == 0 {
			lines[i] = first + lines[i]
		} else {
			lines[i] = rest + lines[i]
		}
	}
	return lines
}

func (c *compiler) flushPara() {
	if len(c.para) == 0 {
		return
	}
	text := strings.Join(c.para, " ")
	c.para = nil
	c.emit(c.wrapIndent(c.inline(text, c.st.text), "", "")...)
}

// Compile parses raw Markdown text and returns a styled ANSI string using the
// given Theme, wrapped to contentWidth columns.
func Compile(input string, th theme.Theme, contentWidth int) string {
	return CompileIn(input, th, contentWidth, "")
}

// CompileIn is Compile for a note stored in baseDir, so local images can be
// found and drawn.
func CompileIn(input string, th theme.Theme, contentWidth int, baseDir string) string {
	return CompileWith(input, th, contentWidth, baseDir, CalendarView{})
}

// CompileWith is CompileIn with a calendar navigation state.
func CompileWith(input string, th theme.Theme, contentWidth int, baseDir string, cal CalendarView) string {
	out, _ := CompileHits(input, th, contentWidth, baseDir, cal)
	return out
}

// CompileHits compiles and also returns the clickable regions of view
// blocks (rows in the output, lines in the original document).
func CompileHits(input string, th theme.Theme, contentWidth int, baseDir string, cal CalendarView) (string, []Hit) {
	if contentWidth < 10 {
		contentWidth = 40
	}
	c := &compiler{st: newStyles(th), width: contentWidth, baseDir: baseDir, cal: cal}

	meta, body := SplitFrontMatter(input)
	c.lineOffset = strings.Count(input, "\n") - strings.Count(body, "\n")
	input = body
	c.header(meta)

	input = htmlCommentRegex.ReplaceAllString(strings.ReplaceAll(input, "\t", "    "), "")
	lines := strings.Split(input, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Fenced code blocks.
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			c.flushPara()
			fence := trimmed[:3]
			lang := strings.TrimSpace(trimmed[3:])
			var code []string
			open := i
			for i++; i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), fence); i++ {
				code = append(code, lines[i])
			}
			c.fence, c.fenceEnd = c.lineOffset+open, c.lineOffset+i
			c.codeBlock(lang, code)
			continue
		}

		// Tables: a header row followed by a separator row.
		if strings.Contains(line, "|") && i+1 < len(lines) && tableSepRegex.MatchString(lines[i+1]) {
			c.flushPara()
			rows := [][]string{splitRow(line)}
			for i += 2; i < len(lines) && strings.Contains(lines[i], "|") && strings.TrimSpace(lines[i]) != ""; i++ {
				rows = append(rows, splitRow(lines[i]))
			}
			i--
			c.table(rows)
			continue
		}

		// A line holding only an image is drawn as a picture.
		if alt, src, ok := standaloneImage(line); ok {
			c.flushPara()
			c.image(alt, src)
			continue
		}

		// Lines made only of HTML layout tags (<div>, </p>, <br> …) vanish;
		// inline HTML elsewhere is handled by the inline parser.
		if isTagOnly(trimmed) {
			if inner := strings.TrimSpace(stripLayoutTags(trimmed)); inner == "" {
				continue
			}
		}

		switch {
		case trimmed == "":
			c.flushPara()
			c.blank()

		case dividerRegex.MatchString(trimmed):
			c.flushPara()
			c.blank()
			c.emit(c.st.rule.Render(strings.Repeat("─", c.width)))
			c.blank()

		case headingRegex.MatchString(trimmed):
			c.flushPara()
			m := headingRegex.FindStringSubmatch(trimmed)
			level := len(m[1])
			st := c.st.headings[level-1]
			prefix := []string{"󰉫 ", "󰉬 ", "󰉭 ", "󰉮 ", "󰉯 ", "󰉰 "}[level-1]
			c.blank()
			c.emit(c.wrapIndent(c.inline(m[2], st), st.Render(prefix), "  ")...)
			if level == 1 {
				c.emit(c.st.headings[0].Render(strings.Repeat("━", min(c.width, max(ansi.StringWidth(m[2])+2, 8)))))
			}
			c.blank()

		case checkboxRegex.MatchString(line):
			c.flushPara()
			m := checkboxRegex.FindStringSubmatch(line)
			indent := strings.Repeat(" ", len(m[1]))
			if m[2] == "x" || m[2] == "X" {
				box := lipgloss.NewStyle().Foreground(c.st.th.Green).Render("󰄲 ")
				text := c.inline(m[3], c.st.muted.Strikethrough(true))
				c.emit(c.wrapIndent(text, indent+box, indent+"  ")...)
			} else {
				box := lipgloss.NewStyle().Foreground(c.st.th.GreyFg2).Render("󰄱 ")
				c.emit(c.wrapIndent(c.inline(m[3], c.st.text), indent+box, indent+"  ")...)
			}

		case bulletRegex.MatchString(line):
			c.flushPara()
			m := bulletRegex.FindStringSubmatch(line)
			depth := len(m[1]) / 2
			indent := strings.Repeat(" ", len(m[1]))
			glyph := []string{"●", "○", "◆", "◇"}[depth%4]
			c.emit(c.wrapIndent(c.inline(m[2], c.st.text), indent+c.st.bullet.Render(glyph)+" ", indent+"  ")...)

		case numberedRegex.MatchString(line):
			c.flushPara()
			m := numberedRegex.FindStringSubmatch(line)
			indent := strings.Repeat(" ", len(m[1]))
			num := c.st.bullet.Bold(true).Render(m[2] + ".")
			pad := strings.Repeat(" ", len(m[2])+2)
			c.emit(c.wrapIndent(c.inline(m[3], c.st.text), indent+num+" ", indent+pad)...)

		case blockquoteRegex.MatchString(line):
			c.flushPara()
			m := blockquoteRegex.FindStringSubmatch(line)
			body := m[1]
			bar := c.st.quoteBar.Render("▎ ")
			st := c.st.muted.Italic(true)
			if kind, rest, ok := callout(body); ok {
				bar = c.st.calloutBr.Render("▎ ")
				c.emit(bar + c.st.calloutBr.Bold(true).Render(kind))
				body = rest
				st = c.st.text
				if strings.TrimSpace(body) == "" {
					continue
				}
			}
			c.emit(c.wrapIndent(c.inline(body, st), bar, bar)...)

		default:
			c.para = append(c.para, trimmed)
		}
	}
	c.flushPara()

	for len(c.out) > 0 && c.out[len(c.out)-1] == "" {
		c.out = c.out[:len(c.out)-1]
	}
	return strings.Join(c.out, "\n"), c.hits
}

// callout recognises GitHub-style "[!NOTE]" admonitions.
func callout(s string) (string, string, bool) {
	t := strings.TrimSpace(s)
	for _, k := range []string{"NOTE", "TIP", "IMPORTANT", "WARNING", "CAUTION"} {
		tag := "[!" + k + "]"
		if strings.HasPrefix(strings.ToUpper(t), tag) {
			icons := map[string]string{"NOTE": "󰋽 ", "TIP": "󰌶 ", "IMPORTANT": "󰅾 ", "WARNING": "󰀪 ", "CAUTION": "󰳦 "}
			return icons[k] + k[:1] + strings.ToLower(k[1:]), t[len(tag):], true
		}
	}
	return "", "", false
}

func (c *compiler) codeBlock(lang string, code []string) {
	if lines, hits, ok := renderBlock(lang, code, c.st.th, c.width, c.cal); ok {
		c.blank()
		base := len(c.out)
		for _, h := range hits {
			h.Row += base
			if h.Line >= 0 {
				h.Line += c.fence + 1
			}
			h.Block, h.End = c.fence, c.fenceEnd
			c.hits = append(c.hits, h)
		}
		for _, l := range lines {
			if ansi.StringWidth(l) > c.width {
				l = ansi.Truncate(l, c.width, "")
			}
			c.out = append(c.out, l) // keep blank rows inside the block
		}
		c.blank()
		return
	}
	c.blank()
	if lang != "" {
		c.emit(c.st.codeLang.Render(" " + lang))
	}
	inner := max(c.width-4, 6)
	colors := highlight.Code(lang, strings.Join(code, "\n"), c.st.th)
	text := lipgloss.NewStyle().Foreground(c.st.th.Fg)
	for i, l := range code {
		painted := highlight.Render([]rune(l), colors[i], text)
		if ansi.StringWidth(painted) > inner {
			painted = ansi.Truncate(painted, inner-1, "…")
		}
		code[i] = painted
	}
	box := c.st.codeBox.Width(inner + 2).Render(strings.Join(code, "\n"))
	c.emit(strings.Split(box, "\n")...)
	c.blank()
}

// header draws the page cover banner and icon from front matter.
func (c *compiler) header(meta map[string]string) {
	if src := meta["cover"]; src != "" {
		path, err := resolveImage(strings.Trim(src, "<>"), c.baseDir)
		var lines []string
		if err == nil {
			lines, err = renderCover(path, c.width, c.st.th.Bg)
		}
		if err != nil {
			c.emit(lipgloss.NewStyle().Foreground(c.st.th.Purple).Render("󰋩 cover") + c.st.muted.Render(" ("+err.Error()+")"))
		} else {
			c.emit(lines...)
		}
		c.blank()
	}
	if icon := meta["icon"]; icon != "" {
		c.emit(" " + icon)
		c.blank()
	}
}

func (c *compiler) image(alt, src string) {
	label := lipgloss.NewStyle().Foreground(c.st.th.Purple)
	path, err := resolveImage(src, c.baseDir)
	var lines []string
	if err == nil {
		lines, err = renderImage(path, c.width, c.st.th.Bg)
	}
	if err != nil {
		name := orDefault(alt, filepath.Base(src))
		c.emit(label.Render("󰋩 "+name) + c.st.muted.Render(" ("+err.Error()+")"))
		return
	}
	c.blank()
	c.emit(lines...)
	if strings.TrimSpace(alt) != "" {
		c.emit(c.st.muted.Italic(true).Render(ui.Truncate(alt, c.width)))
	}
	c.blank()
}

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func (c *compiler) table(rows [][]string) {
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	rendered := make([][]string, len(rows))
	widths := make([]int, cols)
	for i, r := range rows {
		rendered[i] = make([]string, cols)
		for j := 0; j < cols; j++ {
			cell := ""
			if j < len(r) {
				st := c.st.text
				if i == 0 {
					st = st.Bold(true).Foreground(c.st.th.Blue)
				}
				cell = c.inline(r[j], st)
			}
			rendered[i][j] = cell
			widths[j] = max(widths[j], ansi.StringWidth(cell))
		}
	}
	// Shrink the widest columns until the table fits the pane.
	avail := c.width - (3*cols + 1)
	for sum(widths) > avail && avail > cols {
		wi := 0
		for j := range widths {
			if widths[j] > widths[wi] {
				wi = j
			}
		}
		widths[wi]--
	}

	line := c.st.rule
	border := func(l, mid, r string) string {
		parts := make([]string, cols)
		for j, w := range widths {
			parts[j] = strings.Repeat("─", w+2)
		}
		return line.Render(l + strings.Join(parts, mid) + r)
	}
	c.blank()
	c.emit(border("┌", "┬", "┐"))
	for i, r := range rendered {
		var b strings.Builder
		b.WriteString(line.Render("│"))
		for j, cell := range r {
			if ansi.StringWidth(cell) > widths[j] {
				cell = ansi.Truncate(cell, widths[j], "…")
			}
			b.WriteString(" " + cell + strings.Repeat(" ", widths[j]-ansi.StringWidth(cell)) + " ")
			b.WriteString(line.Render("│"))
		}
		c.emit(b.String())
		if i == 0 {
			c.emit(border("├", "┼", "┤"))
		}
	}
	c.emit(border("└", "┴", "┘"))
	c.blank()
}

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}

var layoutTag = regexp.MustCompile(`(?i)</?(div|p|center|span|br|hr|details|summary|picture|source|section|table|tr|td|th|tbody|thead|h[1-6])\b[^>]*>`)

func isTagOnly(s string) bool { return strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">") }

func stripLayoutTags(s string) string { return layoutTag.ReplaceAllString(s, "") }

func attr(re *regexp.Regexp, tag string) string {
	m := re.FindStringSubmatch(tag)
	if m == nil {
		return ""
	}
	for _, v := range m[2:] {
		if v != "" {
			return v
		}
	}
	return ""
}

// inline renders inline Markdown (code, bold, italic, strike, links, images
// and a little inline HTML) by scanning the text once, so styling is never
// applied to text that already contains escape sequences.
func (c *compiler) inline(s string, base lipgloss.Style) string {
	var out strings.Builder
	var plain strings.Builder
	flush := func() {
		if plain.Len() > 0 {
			out.WriteString(base.Render(plain.String()))
			plain.Reset()
		}
	}
	emit := func(rendered string) {
		flush()
		out.WriteString(rendered)
	}

	for i := 0; i < len(s); {
		rest := s[i:]
		switch {
		case rest[0] == '\\' && len(rest) > 1 && strings.ContainsRune("\\`*_{}[]()#+-.!~<>|", rune(rest[1])):
			plain.WriteByte(rest[1])
			i += 2
			continue

		case rest[0] == '`':
			n := 1
			for n < len(rest) && rest[n] == '`' {
				n++
			}
			fence := rest[:n]
			if end := strings.Index(rest[n:], fence); end >= 0 {
				emit(c.st.code.Render(" " + strings.TrimSpace(rest[n:n+end]) + " "))
				i += n + end + n
				continue
			}

		case strings.HasPrefix(rest, "**") || strings.HasPrefix(rest, "__"):
			if end := strings.Index(rest[2:], rest[:2]); end > 0 {
				emit(c.inline(rest[2:2+end], base.Bold(true)))
				i += 2 + end + 2
				continue
			}

		case strings.HasPrefix(rest, "~~"):
			if end := strings.Index(rest[2:], "~~"); end > 0 {
				emit(c.inline(rest[2:2+end], base.Strikethrough(true).Foreground(c.st.th.GreyFg2)))
				i += 2 + end + 2
				continue
			}

		case rest[0] == '*' || (rest[0] == '_' && (i == 0 || s[i-1] == ' ')):
			if end := strings.IndexByte(rest[1:], rest[0]); end > 0 && rest[1] != ' ' {
				emit(c.inline(rest[1:1+end], base.Italic(true)))
				i += 1 + end + 1
				continue
			}

		case strings.HasPrefix(rest, "!["):
			if text, _, n, ok := linkAt(rest[1:]); ok {
				emit(lipgloss.NewStyle().Foreground(c.st.th.Purple).Render("󰋩 " + orDefault(text, "image")))
				i += 1 + n
				continue
			}

		case rest[0] == '[':
			if text, _, n, ok := linkAt(rest); ok {
				emit(c.inline(text, c.st.link))
				i += n
				continue
			}

		case rest[0] == '<':
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				break
			}
			tag := rest[:end+1]
			lower := strings.ToLower(tag)
			switch {
			case strings.HasPrefix(lower, "<http"):
				emit(c.st.link.Render(tag[1 : len(tag)-1]))
			case strings.HasPrefix(lower, "<img"):
				emit(lipgloss.NewStyle().Foreground(c.st.th.Purple).Render("󰋩 " + orDefault(attr(altAttr, tag), "image")))
			case strings.HasPrefix(lower, "<a "):
				closeIdx := strings.Index(strings.ToLower(rest), "</a>")
				if closeIdx > end {
					emit(c.inline(rest[end+1:closeIdx], c.st.link))
					i += closeIdx + len("</a>")
					continue
				}
			case strings.HasPrefix(lower, "<br"):
				plain.WriteByte(' ')
			case strings.HasPrefix(lower, "<kbd>"):
				if ce := strings.Index(strings.ToLower(rest), "</kbd>"); ce > end {
					emit(c.st.code.Render(" " + rest[end+1:ce] + " "))
					i += ce + len("</kbd>")
					continue
				}
			case looksLikeTag(lower):
				// Unknown or layout tag: drop it, keep its text.
			default:
				plain.WriteByte('<')
				i++
				continue
			}
			i += end + 1
			continue
		}

		plain.WriteByte(rest[0])
		i++
	}
	flush()
	return out.String()
}

func looksLikeTag(lower string) bool {
	if len(lower) < 3 {
		return false
	}
	ch := lower[1]
	if ch == '/' && len(lower) > 3 {
		ch = lower[2]
	}
	return ch >= 'a' && ch <= 'z'
}

// linkAt parses "[text](url)" at the start of s, returning the byte length.
func linkAt(s string) (text, url string, n int, ok bool) {
	if !strings.HasPrefix(s, "[") {
		return "", "", 0, false
	}
	depth := 0
	closeText := -1
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				closeText = i
			}
		}
		if closeText >= 0 {
			break
		}
	}
	if closeText < 0 || closeText+1 >= len(s) || s[closeText+1] != '(' {
		return "", "", 0, false
	}
	end := strings.IndexByte(s[closeText+1:], ')')
	if end < 0 {
		return "", "", 0, false
	}
	return s[1:closeText], s[closeText+2 : closeText+1+end], closeText + 1 + end + 1, true
}

func orDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}
