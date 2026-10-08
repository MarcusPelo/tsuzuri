package preview

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/jaisuriya-11/tsuzuri/internal/theme"
)

// The helpers below let other renderers (the PDF export) reuse the preview's
// parsing and block drawing.

// controls are the clickable bits of the views, which mean nothing on paper.
var controls = []string{" + New page", calendarNav, "⊕ add question", "+ Add value", "+ New"}

// PrintBlockLines draws a special fenced block (board, calendar, timeline,
// chart, form, flow) as styled terminal lines width cells wide, without its
// clickable controls. ok is false for ordinary code.
func PrintBlockLines(lang string, body []string, th theme.Theme, width int) (lines []string, ok bool) {
	lines, _, ok = renderBlock(lang, body, th, width, CalendarView{})
	for i, l := range lines {
		for _, c := range controls {
			l = strings.ReplaceAll(l, c, strings.Repeat(" ", ansi.StringWidth(c)))
		}
		lines[i] = l
	}
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, ok
}

// MathRows turns the LaTeX lines of a math block into Unicode rows, one per
// equation line.
func MathRows(body []string) []string {
	var rows []string
	for _, eq := range mathLines(body) {
		rows = append(rows, texToUnicode(eq))
	}
	return rows
}

// IsMathLang reports whether a fence name marks a math block.
func IsMathLang(lang string) bool {
	switch lang {
	case "math", "latex", "tex", "katex":
		return true
	}
	return false
}

// StandaloneImage reports whether a whole line is just an image.
func StandaloneImage(line string) (alt, src string, ok bool) { return standaloneImage(line) }

// ResolveImage turns a Markdown image source into a local file path.
func ResolveImage(src, baseDir string) (string, error) { return resolveImage(src, baseDir) }

// LinkAt parses a Markdown link "[text](url)" at the start of s, returning
// its text, target and length.
func LinkAt(s string) (text, url string, n int, ok bool) { return linkAt(s) }

// StripLayoutTags removes block HTML tags such as <div> and <br>.
func StripLayoutTags(s string) string { return stripLayoutTags(s) }
