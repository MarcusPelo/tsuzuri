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

func TestStoreCreateChild(t *testing.T) {
	store := core.NewStore()
	child, err := store.CreateChild("page-1", "Nested Tasks")
	if err != nil {
		t.Fatalf("unexpected error creating child page: %v", err)
	}
	if child.ParentID != "page-1" {
		t.Errorf("expected parent ID 'page-1', got %q", child.ParentID)
	}
	if child.Title != "Nested Tasks" {
		t.Errorf("expected title 'Nested Tasks', got %q", child.Title)
	}

	_, err = store.CreateChild("non-existent", "Child")
	if err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound for non-existent parent, got %v", err)
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

func TestStoreSeedHierarchy(t *testing.T) {
	store := core.NewStore()
	store.SeedHierarchy()

	pages := store.List()
	// Should have initial page + 5 hierarchy pages = 6 pages
	if len(pages) != 6 {
		t.Fatalf("expected 6 pages after SeedHierarchy, got %d", len(pages))
	}

	pageMap := make(map[string]core.Page)
	for _, p := range pages {
		pageMap[p.ID] = p
	}

	// Verify child 1's parent is Parent 1
	c1, ok := pageMap["page-child-1"]
	if !ok || c1.ParentID != "page-parent-1" {
		t.Errorf("expected page-child-1 to have ParentID page-parent-1, got %v", c1)
	}

	// Verify grandchild's parent is child 1
	gc1, ok := pageMap["page-child-1-child"]
	if !ok || gc1.ParentID != "page-child-1" {
		t.Errorf("expected page-child-1-child to have ParentID page-child-1, got %v", gc1)
	}

	// Verify child 2's parent is Parent 2
	c2, ok := pageMap["page-child-2"]
	if !ok || c2.ParentID != "page-parent-2" {
		t.Errorf("expected page-child-2 to have ParentID page-parent-2, got %v", c2)
	}
}
