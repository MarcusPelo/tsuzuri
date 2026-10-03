package preview_test

import (
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
		Title:   "Notion Plan",
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
