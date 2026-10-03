package app_test

import (
	"strings"
	"testing"

	"github.com/jaisuriya-11/tsuzuri/internal/app"
	"github.com/jaisuriya-11/tsuzuri/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestStore(t *testing.T) *core.Store {
	t.Helper()
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error creating store: %v", err)
	}
	return store
}

func TestAppOrchestrator(t *testing.T) {
	store := newTestStore(t)
	m := app.New(store)

	if m.Init() == nil {
		t.Error("expected non-nil Init command")
	}

	const termWidth = 120
	const termHeight = 40

	// 1. Initial size msg -> enters dashboard
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: termWidth, Height: termHeight})
	dashboardView := updatedModel.View()

	if dashboardView == "loading…" {
		t.Fatal("expected rendered layout view after WindowSizeMsg")
	}
	if !strings.Contains(dashboardView, "New Document") {
		t.Errorf("expected dashboard view to contain 'New Document', got %q", dashboardView)
	}

	// Verify line height never exceeds terminal height in dashboard
	dashLines := strings.Count(dashboardView, "\n") + 1
	if dashLines > termHeight {
		t.Errorf("dashboard lines (%d) exceeded terminal height (%d) - terminal will scroll!", dashLines, termHeight)
	}

	// 2. Press 'n' to create new document -> transitions to workspace
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	workspaceView := updatedModel.View()

	if !strings.Contains(workspaceView, "SIDEBAR") && !strings.Contains(workspaceView, "NORMAL") {
		t.Errorf("expected workspace view to contain editor or sidebar badges, got %q", workspaceView)
	}

	// Verify line height never exceeds terminal height in workspace
	workLines := strings.Count(workspaceView, "\n") + 1
	if workLines > termHeight {
		t.Errorf("workspace lines (%d) exceeded terminal height (%d) - terminal will scroll!", workLines, termHeight)
	}

	// Verify top bar [Tab] Switch Pane line is removed
	if strings.Contains(workspaceView, "[Tab] Switch Pane") {
		t.Errorf("expected top header '[Tab] Switch Pane' to be removed from workspace view, got %q", workspaceView)
	}

	// Verify status bar has no brackets
	if strings.Contains(workspaceView, "[SPC h]") || strings.Contains(workspaceView, "[Top]") {
		t.Errorf("expected bottom status bar not to have brackets like [SPC h] or [Top], got %q", workspaceView)
	}

	// A brand-new page is an unsaved draft — nothing touches disk until the
	// user explicitly saves it (vim semantics: no file until :w).
	if pages := store.List(); len(pages) != 0 {
		t.Fatalf("expected no file on disk before an explicit save, got %d: %v", len(pages), pages)
	}
	if !strings.Contains(workspaceView, "(unsaved)") {
		t.Errorf("expected the status bar to flag the new page as unsaved, got %q", workspaceView)
	}

	// Typing content and running :wq must materialize exactly one real file.
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	var saveCmd tea.Cmd
	updatedModel, saveCmd = updatedModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if saveCmd == nil {
		t.Fatal("expected a VimSaveMsg command from ':w<enter>'")
	}
	updatedModel, _ = updatedModel.Update(saveCmd())

	pages := store.List()
	if len(pages) != 1 {
		t.Fatalf("expected 1 real page on disk after :w, got %d: %v", len(pages), pages)
	}
	saved, err := store.Get(pages[0].ID)
	if err != nil {
		t.Fatalf("unexpected error reading back saved page: %v", err)
	}
	if saved.Content != "hi" {
		t.Errorf("expected saved content 'hi', got %q", saved.Content)
	}

	// 3. Press 'ctrl+d' to return to dashboard
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	returnDashView := updatedModel.View()

	if !strings.Contains(returnDashView, "New Document") {
		t.Errorf("expected return to dashboard after ctrl+d, got %q", returnDashView)
	}
	returnLines := strings.Count(returnDashView, "\n") + 1
	if returnLines > termHeight {
		t.Errorf("returned dashboard lines (%d) exceeded terminal height (%d) - terminal will scroll!", returnLines, termHeight)
	}

	// 4. Test Quit
	_, quitCmd := updatedModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if quitCmd == nil {
		t.Error("expected non-nil quit command on ctrl+c")
	}
}

func TestAppStartsEmptyWithNoSeedData(t *testing.T) {
	store := newTestStore(t)
	m := app.New(store)

	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := updatedModel.View()

	if strings.Contains(view, "Solutions Architect") {
		t.Errorf("expected no seeded demo content, got %q", view)
	}
	if pages := store.List(); len(pages) != 0 {
		t.Errorf("expected a brand new workspace to start empty, got %v", pages)
	}
}

func TestAppKeymapModalAndBufferCycle(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Create("Parent 1", ""); err != nil {
		t.Fatalf("unexpected error seeding page: %v", err)
	}
	if _, err := store.Create("Parent 2", ""); err != nil {
		t.Fatalf("unexpected error seeding page: %v", err)
	}

	m := app.New(store)

	const termWidth = 120
	const termHeight = 40
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: termWidth, Height: termHeight})

	// Enter workspace
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	view := updatedModel.View()

	// Verify the tab strip shows the open page and only reflects buffers the
	// user actually opened (VSCode-style), not every file in the workspace.
	if !strings.Contains(view, "Parent 1.md") {
		t.Errorf("expected tab strip to show the open page 'Parent 1.md', got %q", view)
	}
	if strings.Contains(view, "Parent 2.md") {
		t.Errorf("expected tab strip NOT to show 'Parent 2.md' (never opened), got %q", view)
	}

	// Verify Cheatsheet Modal opens on '?'
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	modalView := updatedModel.View()
	if !strings.Contains(modalView, "CHEATSHEET") {
		t.Errorf("expected Keymap modal to show 'CHEATSHEET', got %q", modalView)
	}

	// Dismiss modal with esc
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	viewAfterDismiss := updatedModel.View()
	if strings.Contains(viewAfterDismiss, "CHEATSHEET") {
		t.Errorf("expected Keymap modal to be dismissed after Esc")
	}

	// Test buffer cycling with ']'
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	viewAfterCycle := updatedModel.View()
	if viewAfterCycle == "" {
		t.Errorf("expected valid view after cycling buffer")
	}
}

func TestAppDeletingLastPageReturnsToDashboard(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Create("Only Page", ""); err != nil {
		t.Fatalf("unexpected error seeding page: %v", err)
	}

	m := app.New(store)
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Enter workspace, focus sidebar
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	// Delete the only page from the sidebar. Sidebar emits PageDeletedMsg as a
	// tea.Cmd, which the real runtime executes and feeds back into Update —
	// do the same here.
	var cmd tea.Cmd
	updatedModel, cmd = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("expected a PageDeletedMsg command from pressing 'd' in the sidebar")
	}
	updatedModel, _ = updatedModel.Update(cmd())

	if pages := store.List(); len(pages) != 0 {
		t.Errorf("expected the page to be deleted from disk, got %v", pages)
	}

	view := updatedModel.View()
	if !strings.Contains(view, "New Document") {
		t.Errorf("expected app to bounce back to the dashboard once the workspace is empty, got %q", view)
	}
}
