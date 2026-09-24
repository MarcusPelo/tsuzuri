package app_test

import (
	"strings"
	"testing"

	"tsuzuri/internal/app"
	"tsuzuri/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAppOrchestrator(t *testing.T) {
	store := core.NewStore()
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

func TestAppKeymapModalAndBufferCycle(t *testing.T) {
	store := core.NewStore()
	store.SeedHierarchy()
	m := app.New(store)

	const termWidth = 120
	const termHeight = 40
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: termWidth, Height: termHeight})

	// Enter workspace
	updatedModel, _ = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	view := updatedModel.View()

	// Verify Tabufline has no 'x' close buttons
	if strings.Contains(view, ".md    x") || strings.Contains(view, ".md  x") {
		t.Errorf("expected Tabufline to have no 'x' close buttons, got %q", view)
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
