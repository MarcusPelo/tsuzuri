package content_test

import (
	"strings"
	"testing"

	"tsuzuri/internal/content"
	"tsuzuri/internal/core"
	"tsuzuri/internal/theme"
)

func TestContentComponent(t *testing.T) {
	th := theme.DefaultTheme()
	c := content.New(th)
	c.SetSize(80, 20)
	c.SetPage(core.Page{Title: "Notes", Content: "Hello world"})

	view := c.View()
	if !strings.Contains(view, "Notes") {
		t.Errorf("expected content view to contain title 'Notes', got %q", view)
	}
	if !strings.Contains(view, "NORMAL") {
		t.Errorf("expected content view to contain mode 'NORMAL', got %q", view)
	}
}
