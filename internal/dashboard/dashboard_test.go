package dashboard_test

import (
	"strings"
	"testing"

	"tsuzuri/internal/core"
	"tsuzuri/internal/dashboard"
	"tsuzuri/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardComponent(t *testing.T) {
	th := theme.DefaultTheme()
	db := dashboard.New(th)
	db.SetSize(100, 30)

	pages := []core.Page{
		{ID: "p1", Title: "Intro Document"},
		{ID: "p2", Title: "Todo List"},
	}
	db.SetRecentPages(pages)

	view := db.View()
	if !strings.Contains(view, "New Document") {
		t.Errorf("expected view to contain 'New Document', got %q", view)
	}
	if !strings.Contains(view, "Intro Document") {
		t.Errorf("expected view to contain recent page 'Intro Document', got %q", view)
	}
	if !strings.Contains(view, "Todo List") {
		t.Errorf("expected view to contain recent page 'Todo List', got %q", view)
	}

	// Default selection is New Document
	if db.SelectedAction() != dashboard.ActionNewPage {
		t.Errorf("expected default action to be ActionNewPage, got %v", db.SelectedAction())
	}

	// Move cursor down
	updated, _ := db.Update(tea.KeyMsg{Type: tea.KeyDown})
	db = updated
	if db.SelectedAction() != dashboard.ActionBrowse {
		t.Errorf("expected action after down to be ActionBrowse, got %v", db.SelectedAction())
	}

	// Move cursor down again
	updated, _ = db.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	db = updated
	if db.SelectedAction() != dashboard.ActionQuit {
		t.Errorf("expected action after j to be ActionQuit, got %v", db.SelectedAction())
	}

	// Move cursor down at bottom (clamping check)
	updated, _ = db.Update(tea.KeyMsg{Type: tea.KeyDown})
	db = updated
	if db.SelectedAction() != dashboard.ActionQuit {
		t.Errorf("expected action to stay ActionQuit at bottom clamp, got %v", db.SelectedAction())
	}

	// Move cursor up
	updated, _ = db.Update(tea.KeyMsg{Type: tea.KeyUp})
	db = updated
	if db.SelectedAction() != dashboard.ActionBrowse {
		t.Errorf("expected action after up to be ActionBrowse, got %v", db.SelectedAction())
	}
}

func TestDashboardZeroSize(t *testing.T) {
	th := theme.DefaultTheme()
	db := dashboard.New(th)
	db.SetSize(0, 0)

	if db.View() != "" {
		t.Errorf("expected empty string for zero size, got %q", db.View())
	}
}
