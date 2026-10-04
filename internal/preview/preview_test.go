package preview_test

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"

	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/preview"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
)

func TestCompiler(t *testing.T) {
	th := theme.DefaultTheme()
	md := `# Main Title
## Section Header
### Sub Header

- [ ] Unfinished task
- [x] Finished task

- Bullet item 1
- Bullet item 2

1. First step
2. Second step

> 💡 This is a callout box

---

` + "```go\nfmt.Println(\"Hello\")\n```" + `

Here is **bold**, *italic*, ` + "`inline_code`" + `, and [Link](https://charm.sh).
`

	compiled := preview.Compile(md, th, 80)

	if !strings.Contains(compiled, "Main Title") {
		t.Errorf("expected compiled output to contain 'Main Title', got %q", compiled)
	}
	if !strings.Contains(compiled, "Section Header") {
		t.Errorf("expected compiled output to contain 'Section Header', got %q", compiled)
	}
	if !strings.Contains(compiled, "󰄱") {
		t.Errorf("expected unchecked checkbox icon '󰄱', got %q", compiled)
	}
	if !strings.Contains(compiled, "\U000f0132") {
		t.Errorf("expected checked checkbox icon, got %q", compiled)
	}
	if !strings.Contains(compiled, "●") {
		t.Errorf("expected bullet point icon, got %q", compiled)
	}
	if !strings.Contains(compiled, "go") {
		t.Errorf("expected code block language badge 'go', got %q", compiled)
	}
}

func TestPreviewComponent(t *testing.T) {
	th := theme.DefaultTheme()
	p := preview.New(th)
	p.SetSize(60, 25)

	page := core.Page{
		ID:      "p1",
		Title:   "Project Plan",
		Content: "# Project Roadmap\n- [ ] Ship live preview\n- [x] Fix terminal scrolling",
	}
	p.SetPage(page)

	view := p.View()
	if !strings.Contains(view, "preview") {
		t.Errorf("expected preview header badge, got %q", view)
	}
	if !strings.Contains(view, "Project Roadmap") {
		t.Errorf("expected compiled content 'Project Roadmap', got %q", view)
	}
	if p.ScrollStatus() != "Top" {
		t.Errorf("expected initial scroll status 'Top', got %q", p.ScrollStatus())
	}
}

func TestCompilerRegressions(t *testing.T) {
	th := theme.DefaultTheme()
	src := `<div align="center">

### ~ Title ~

</div>

<p align="center">
  <a href="https://x/ci"><img src="https://x/badge.svg" alt="CI"></a>
</p>

**Tsuzuri** (綴り — *spelling*) is a notebook. See [Bubble Tea](https://github.com/charmbracelet/bubbletea) and more words to force wrapping across several lines of output.

| Key | Action |
| :-- | :-- |
| ` + "`n`" + ` | New note |
`
	out := preview.Compile(src, th, 40)
	plain := ansi.Strip(out)

	if strings.Contains(plain, "38;2;") || strings.Contains(plain, "[0m") {
		t.Fatalf("raw escape codes leaked into output:\n%s", plain)
	}
	for _, bad := range []string{"<div", "</div>", "<p", "<a ", "<img", "](https://"} {
		if strings.Contains(plain, bad) {
			t.Errorf("expected %q to be rendered, not shown raw:\n%s", bad, plain)
		}
	}
	for _, want := range []string{"Tsuzuri", "spelling", "Bubble Tea", "CI", "New note", "┌"} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %q in:\n%s", want, plain)
		}
	}
	for _, l := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(l); w > 40 {
			t.Errorf("line wider than the pane (%d): %q", w, ansi.Strip(l))
		}
	}
	if strings.Contains(plain, "\n\n\n") {
		t.Errorf("expected blank lines collapsed:\n%s", plain)
	}
}

func TestViewBlocks(t *testing.T) {
	th := theme.DefaultTheme()
	cases := map[string][]string{
		"```board\n## Todo\n- A\n## Done\n- B\n```":                                     {"● Todo", "● Done", "A", "B", "+ New page"},
		"```calendar\nmonth: 2026-10\n2026-10-15: Launch\n```":                          {"October 2026", "Sun", "Sat", "Laun", "31"},
		"```timeline\nAlpha: 2026-10-01 -> 2026-10-04\n```":                             {"Alpha", "Oct 2026"},
		"```chart\ntype: bar\ntitle: Sales\nJan: 3\nFeb: 6\n```":                        {"Sales", "Jan", "Feb", "█"},
		"```chart\ntype: hbar\nGo: 5\nRust: 2\n```":                                     {"Go", "Rust", "█"},
		"```chart\ntype: line\nA: 1\nB: 5\nC: 2\n```":                                   {"┤", "A", "C"},
		"```chart\ntype: pie\nYes: 3\nNo: 1\n```":                                       {"Yes", "75%", "No", "25%"},
		"```form\ntitle: Survey\n?* Name\n? Pick (choice): X | Y\n? Rate (rating)\n```": {"Survey", "Name *", "○ X", "☆ ☆"},
	}
	for src, wants := range cases {
		out := preview.Compile(src, th, 60)
		plain := ansi.Strip(out)
		for _, w := range wants {
			if !strings.Contains(plain, w) {
				t.Errorf("%q: missing %q in\n%s", src[:12], w, plain)
			}
		}
		for _, l := range strings.Split(out, "\n") {
			if ansi.StringWidth(l) > 60 {
				t.Errorf("%q: line too wide (%d)", src[:12], ansi.StringWidth(l))
			}
		}
	}
}

func TestBoardCardDescriptions(t *testing.T) {
	src := "```board\n## In progress\n- Card 2\ncant able to add description in cards\n- card 3\n## Done\n- Card 3\n```"
	plain := ansi.Strip(preview.Compile(src, theme.DefaultTheme(), 80))
	for _, want := range []string{"Card 2", "cant able to add", "card 3", "Done"} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %q in\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "● In progress  3") {
		t.Errorf("a description line must not count as a card:\n%s", plain)
	}
}

func TestTimelineFitsLongRanges(t *testing.T) {
	src := "```timeline\nCard 1: 2026-09-30 -> 2026-10-05\nCard 2: 2026-10-03 -> 2026-10-07\nCard 3: 2026-02-01 -> 2026-10-02\n```"
	for _, w := range []int{40, 60, 100} {
		out := preview.Compile(src, theme.DefaultTheme(), w)
		plain := ansi.Strip(out)
		for _, want := range []string{"Card 1", "Card 2", "Card 3", "Feb 2026", "Oct"} {
			if !strings.Contains(plain, want) {
				t.Errorf("width %d: missing %q in\n%s", w, want, plain)
			}
		}
		for _, l := range strings.Split(out, "\n") {
			if ansi.StringWidth(l) > w {
				t.Errorf("width %d: line too wide (%d): %q", w, ansi.StringWidth(l), ansi.Strip(l))
			}
		}
		if strings.Contains(plain, "1011") || strings.Contains(plain, "3031") {
			t.Errorf("width %d: day numbers ran together:\n%s", w, plain)
		}
	}
}

func TestCalendarMonthNavigation(t *testing.T) {
	p := preview.New(theme.DefaultTheme())
	p.SetSize(80, 40)
	p.SetPage(core.Page{ID: "c.md", Content: "```calendar\nmonth: 2026-02\n2026-03-10: Trip\n```"})
	view := func() string { return ansi.Strip(p.View()) }
	if !strings.Contains(view(), "February 2026") {
		t.Fatalf("expected February:\n%s", view())
	}
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'>'}})
	if v := view(); !strings.Contains(v, "March 2026") || !strings.Contains(v, "Trip") {
		t.Fatalf("expected March with its event after '>':\n%s", v)
	}
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'<'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'<'}})
	if !strings.Contains(view(), "January 2026") {
		t.Fatalf("expected January after two '<':\n%s", view())
	}

	// Click the "›" in the header.
	for y, l := range strings.Split(view(), "\n") {
		if i := strings.Index(l, "‹  Today  ›"); i >= 0 {
			x := ansi.StringWidth(l[:i]) + 10
			p, _ = p.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			break
		}
	}
	if !strings.Contains(view(), "February 2026") {
		t.Fatalf("expected February after clicking ›:\n%s", view())
	}
}

func TestFoldHeadingsAndCode(t *testing.T) {
	p := preview.New(theme.DefaultTheme())
	p.SetSize(70, 40)
	p.SetPage(core.Page{ID: "f.md", Content: "# One\nsecret text\n## Sub\nmore\n# Two\n```go\nfmt.Println(1)\n```\nafter"})
	view := func() string { return ansi.Strip(p.View()) }
	click := func(text string) {
		t.Helper()
		for y, l := range strings.Split(view(), "\n") {
			if i := strings.Index(l, text); i >= 0 {
				p, _ = p.Update(tea.MouseMsg{X: ansi.StringWidth(l[:i]) + 1, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
				return
			}
		}
		t.Fatalf("%q not shown:\n%s", text, view())
	}

	click("One")
	v := view()
	if strings.Contains(v, "secret text") || strings.Contains(v, "Sub") || !strings.Contains(v, "▸") || !strings.Contains(v, "… 3 lines") {
		t.Fatalf("heading One should hide its section:\n%s", v)
	}
	if !strings.Contains(v, "Two") || !strings.Contains(v, "fmt.Println") {
		t.Fatalf("the next H1 must stay visible:\n%s", v)
	}

	click("go · 1 line")
	if strings.Contains(view(), "fmt.Println") {
		t.Fatalf("code block should fold:\n%s", view())
	}

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	if v := view(); !strings.Contains(v, "secret text") || !strings.Contains(v, "fmt.Println") {
		t.Fatalf("zR should open everything:\n%s", v)
	}
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}})
	if v := view(); strings.Contains(v, "secret text") || strings.Contains(v, "after") {
		t.Fatalf("zM should fold everything:\n%s", v)
	}
}

func TestTableRowDividers(t *testing.T) {
	out := ansi.Strip(preview.Compile("| A | B |\n| --- | --- |\n| 1 | 2 |\n| 3 | 4 |", theme.DefaultTheme(), 40))
	if n := strings.Count(out, "├"); n != 2 {
		t.Fatalf("expected a divider under the header and between the two rows, got %d:\n%s", n, out)
	}
	if strings.Count(out, "└") != 1 || strings.Contains(out, "├───┼───┤\n└") {
		t.Fatalf("no divider after the last row:\n%s", out)
	}
}

func TestBarChartRowsAlignWithFractionalAxisLabel(t *testing.T) {
	out := preview.Compile("```chart\ntype: bar\nA: 4\nB: 15\n```", theme.DefaultTheme(), 60)
	var axis []int
	for _, l := range strings.Split(ansi.Strip(out), "\n") {
		if i := strings.IndexAny(l, "┤└"); i >= 0 {
			axis = append(axis, ansi.StringWidth(l[:i]))
		}
	}
	if len(axis) < 3 {
		t.Fatalf("no chart axis found:\n%s", ansi.Strip(out))
	}
	for _, x := range axis {
		if x != axis[0] {
			t.Fatalf("axis must be a straight line, got columns %v (the 7.5 label shifted its row)", axis)
		}
	}
}
