package content_test

import (
	"strings"
	"testing"

	"github.com/jaisuriya-11/tsuzuri/internal/content"
	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
)

func TestContentComponent(t *testing.T) {
	th := theme.DefaultTheme()
	c := content.New(th)
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "Notes.md", Title: "Notes", Content: "Hello world"})

	view := c.View()
	if !strings.Contains(view, "Notes") {
		t.Errorf("expected content view to contain title 'Notes', got %q", view)
	}
	if c.ModeString() != "NORMAL" {
		t.Errorf("expected content mode 'NORMAL', got %q", c.ModeString())
	}
}
