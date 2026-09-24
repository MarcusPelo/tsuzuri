package sidebar_test

import (
	"strings"
	"testing"

	"tsuzuri/internal/core"
	"tsuzuri/internal/sidebar"
	"tsuzuri/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSidebarComponent(t *testing.T) {
	th := theme.DefaultTheme()
	sb := sidebar.New(th)
	sb.SetSize(28, 25)

	pages := []core.Page{
		{ID: "p1", Title: "Code Turtle"},
		{ID: "p2", Title: "Tasks (1)", ParentID: "p1"},
		{ID: "p3", Title: "Project Manager"},
	}
	sb.SetPages(pages)
	sb.SetFocused(true)

	view := sb.View()
	if !strings.Contains(view, "Code Turtle") {
		t.Errorf("expected view to contain parent 'Code Turtle', got %q", view)
	}
	if !strings.Contains(view, "Tasks (1)") {
		t.Errorf("expected view to contain child 'Tasks (1)', got %q", view)
	}
	if strings.Contains(view, "RECENTS") {
		t.Errorf("expected view NOT to contain RECENTS header, got %q", view)
	}
	if strings.Contains(view, "WORKSPACE") {
		t.Errorf("expected view NOT to contain WORKSPACE header, got %q", view)
	}
	if !strings.Contains(view, "Search") {
		t.Errorf("expected view to contain Search box, got %q", view)
	}

	// Test collapsing folder
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	itemsAfterCollapse := sb.VisibleItems()
	for _, it := range itemsAfterCollapse {
		if it.Page.Title == "Tasks (1)" {
			t.Errorf("expected collapsed folder to hide child item from visible tree, but found %v", it)
		}
	}

	// Test expanding folder again
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	itemsAfterExpand := sb.VisibleItems()
	found := false
	for _, it := range itemsAfterExpand {
		if it.Page.Title == "Tasks (1)" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected re-expanded folder to show child 'Tasks (1)' in visible tree, but not found")
	}
}

func TestSidebarEditableElements(t *testing.T) {
	th := theme.DefaultTheme()
	sb := sidebar.New(th)
	sb.SetSize(28, 25)

	pages := []core.Page{
		{ID: "p1", Title: "Original Title"},
	}
	sb.SetPages(pages)

	// Press 'r' to edit/rename selected element
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !sb.IsRenaming() {
		t.Fatal("expected sidebar to be in renaming mode after 'r'")
	}

	// Simulate pressing 'enter' or typing and saving
	sb, cmd := sb.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if sb.IsRenaming() {
		t.Errorf("expected renaming to finish after enter")
	}
	if cmd == nil {
		t.Errorf("expected cmd emitting PageUpdatedMsg on rename commit")
	}
	msg := cmd()
	updatedMsg, ok := msg.(core.PageUpdatedMsg)
	if !ok || updatedMsg.Page.Title != "Original Title" {
		t.Errorf("expected PageUpdatedMsg with title 'Original Title', got %v", msg)
	}
}

func TestSidebarSearch(t *testing.T) {
	th := theme.DefaultTheme()
	sb := sidebar.New(th)
	sb.SetSize(28, 25)

	pages := []core.Page{
		{ID: "p1", Title: "Code Turtle"},
		{ID: "p2", Title: "Credentials"},
	}
	sb.SetPages(pages)

	// Press '/' to search
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !sb.IsSearching() {
		t.Error("expected sidebar to be in searching mode after '/'")
	}

	// Type 'cred'
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	items := sb.VisibleItems()
	if len(items) != 1 || items[0].Page.Title != "Credentials" {
		t.Errorf("expected 1 search result 'Credentials', got %v", items)
	}
}
