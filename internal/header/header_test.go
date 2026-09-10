package header_test

import (
	"strings"
	"testing"

	"tsuzuri/internal/core"
	"tsuzuri/internal/header"
	"tsuzuri/internal/theme"
)

func TestHeaderComponent(t *testing.T) {
	th := theme.DefaultTheme()
	h := header.New(th)
	h.SetSize(100, 1)
	h.SetPage(core.Page{Title: "Design Doc"})
	h.SetMode("INSERT", false)

	view := h.View()
	if !strings.Contains(view, "Design Doc") {
		t.Errorf("expected header view to contain title 'Design Doc', got %q", view)
	}
	if !strings.Contains(view, "INSERT") {
		t.Errorf("expected header view to contain mode 'INSERT', got %q", view)
	}
}
