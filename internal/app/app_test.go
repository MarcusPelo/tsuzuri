package app_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"tsuzuri/internal/app"
	"tsuzuri/internal/core"
)

func TestAppOrchestrator(t *testing.T) {
	store := core.NewStore()
	m := app.New(store)

	if m.Init() == nil {
		t.Error("expected non-nil Init command")
	}

	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	viewOutput := updatedModel.View()

	if viewOutput == "loading…" {
		t.Error("expected rendered layout view after WindowSizeMsg")
	}
}
