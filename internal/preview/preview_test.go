package preview_test

import (
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
	if !strings.Contains(compiled, "󰄳") {
		t.Errorf("expected checked checkbox icon '󰄳', got %q", compiled)
	}
	if !strings.Contains(compiled, "•") {
		t.Errorf("expected bullet point icon '•', got %q", compiled)
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
