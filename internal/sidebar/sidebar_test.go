package sidebar_test

import (
	"strings"
	"testing"

	"tsuzuri/internal/core"
	"tsuzuri/internal/sidebar"
	"tsuzuri/internal/theme"
)

func TestSidebarComponent(t *testing.T) {
	th := theme.DefaultTheme()
	sb := sidebar.New(th)
	sb.SetSize(28, 20)

	pages := []core.Page{
		{ID: "p1", Title: "Page One"},
		{ID: "p2", Title: "Page Two"},
	}
	sb.SetPages(pages)
	sb.SetFocused(true)

	view := sb.View()
	if !strings.Contains(view, "Page One") {
		t.Errorf("expected view to contain 'Page One', got %q", view)
	}
	if !strings.Contains(view, "Page Two") {
		t.Errorf("expected view to contain 'Page Two', got %q", view)
	}
}
