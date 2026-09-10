package core_test

import (
	"path/filepath"
	"testing"

	"tsuzuri/internal/core"
)

func TestStoreCreate(t *testing.T) {
	tests := []struct {
		name          string
		inputTitle    string
		expectedTitle string
	}{
		{
			name:          "Create with empty title generates default title",
			inputTitle:    "",
			expectedTitle: "Untitled 2",
		},
		{
			name:          "Create with custom title",
			inputTitle:    "Meeting Notes",
			expectedTitle: "Meeting Notes",
		},
		{
			name:          "Create with special characters",
			inputTitle:    "🔑 Credentials & Keys",
			expectedTitle: "🔑 Credentials & Keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := core.NewStore()
			p, err := store.Create(tt.inputTitle)
			if err != nil {
				t.Fatalf("unexpected error creating page: %v", err)
			}
			if p.Title != tt.expectedTitle {
				t.Errorf("expected title %q, got %q", tt.expectedTitle, p.Title)
			}
			if p.ID == "" {
				t.Error("expected non-empty page ID")
			}
		})
	}
}

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		pages []core.Page
	}{
		{
			name: "Round-trip single default page",
			pages: []core.Page{
				{ID: "p1", Title: "Initial Doc", Content: "Hello world"},
			},
		},
		{
			name: "Round-trip multiple complex pages",
			pages: []core.Page{
				{ID: "p1", Title: "Page 1", Content: "# Header\nSome markdown text"},
				{ID: "p2", Title: "Page 2", Content: "```go\nfmt.Println(\"hi\")\n```"},
				{ID: "p3", Title: "Page 3", Content: "Final page"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "workspace.json")

			store1 := core.NewStore()
			pagesInStore := store1.List()
			for _, p := range pagesInStore {
				_ = store1.Delete(p.ID)
			}

			for _, p := range tt.pages {
				created, err := store1.Create(p.Title)
				if err != nil {
					t.Fatalf("failed to create test page: %v", err)
				}
				created.Content = p.Content
				if err := store1.Update(created); err != nil {
					t.Fatalf("failed to update test page: %v", err)
				}
			}

			if err := store1.SaveToFile(filePath); err != nil {
				t.Fatalf("SaveToFile failed: %v", err)
			}

			store2 := core.NewStore()
			if err := store2.LoadFromFile(filePath); err != nil {
				t.Fatalf("LoadFromFile failed: %v", err)
			}

			loadedPages := store2.List()
			if len(loadedPages) != len(tt.pages) {
				t.Fatalf("expected %d pages loaded, got %d", len(tt.pages), len(loadedPages))
			}

			for i, expected := range tt.pages {
				if loadedPages[i].Title != expected.Title {
					t.Errorf("page [%d] title mismatch: expected %q, got %q", i, expected.Title, loadedPages[i].Title)
				}
				if loadedPages[i].Content != expected.Content {
					t.Errorf("page [%d] content mismatch: expected %q, got %q", i, expected.Content, loadedPages[i].Content)
				}
			}
		})
	}
}

func TestStoreEdgeCases(t *testing.T) {
	store := core.NewStore()

	// Fetch non-existent ID
	_, err := store.Get("non-existent-id")
	if err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound, got %v", err)
	}

	// Update non-existent page
	err = store.Update(core.Page{ID: "invalid-id"})
	if err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound on update, got %v", err)
	}

	// Delete non-existent page
	err = store.Delete("invalid-id")
	if err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound on delete, got %v", err)
	}
}
