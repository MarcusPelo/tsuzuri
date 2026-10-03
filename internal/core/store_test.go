package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jaisuriya-11/tsuzuri/internal/core"
)

func newTestStore(t *testing.T) *core.Store {
	t.Helper()
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error creating store: %v", err)
	}
	return store
}

func TestStoreCreate(t *testing.T) {
	tests := []struct {
		name          string
		inputTitle    string
		expectedTitle string
	}{
		{name: "Create with empty title generates default title", inputTitle: "", expectedTitle: "Untitled"},
		{name: "Create with custom title", inputTitle: "Meeting Notes", expectedTitle: "Meeting Notes"},
		{name: "Create with special characters", inputTitle: "🔑 Credentials & Keys", expectedTitle: "🔑 Credentials & Keys"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t)
			p, err := store.Create(tt.inputTitle, "")
			if err != nil {
				t.Fatalf("unexpected error creating page: %v", err)
			}
			if p.Title != tt.expectedTitle {
				t.Errorf("expected title %q, got %q", tt.expectedTitle, p.Title)
			}
			if p.ID == "" {
				t.Error("expected non-empty page ID")
			}
			if _, err := os.Stat(filepath.Join(store.Root(), filepath.FromSlash(p.ID))); err != nil {
				t.Errorf("expected a real file on disk at %q: %v", p.ID, err)
			}
		})
	}
}

func TestStoreCreateDuplicateTitlesAreDisambiguated(t *testing.T) {
	store := newTestStore(t)

	first, err := store.Create("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := store.Create("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.Title != "Untitled" {
		t.Errorf("expected first page title 'Untitled', got %q", first.Title)
	}
	if second.Title != "Untitled 2" {
		t.Errorf("expected second page title 'Untitled 2', got %q", second.Title)
	}
}

func TestStoreCreateChild(t *testing.T) {
	store := newTestStore(t)

	parent, err := store.Create("Parent 1", "")
	if err != nil {
		t.Fatalf("unexpected error creating parent page: %v", err)
	}

	child, err := store.Create("Nested Tasks", parent.ID)
	if err != nil {
		t.Fatalf("unexpected error creating child page: %v", err)
	}
	if child.ParentID != parent.ID {
		t.Errorf("expected parent ID %q, got %q", parent.ID, child.ParentID)
	}
	if child.Title != "Nested Tasks" {
		t.Errorf("expected title 'Nested Tasks', got %q", child.Title)
	}

	// The sidecar folder should exist alongside the parent file.
	sidecar := filepath.Join(store.Root(), "Parent 1")
	if info, err := os.Stat(sidecar); err != nil || !info.IsDir() {
		t.Errorf("expected sidecar folder %q to exist", sidecar)
	}

	_, err = store.Create("Child", "non-existent.md")
	if err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound for non-existent parent, got %v", err)
	}
}

func TestStorePlainFoldersAreNavigable(t *testing.T) {
	store := newTestStore(t)

	if err := os.MkdirAll(filepath.Join(store.Root(), "assets"), 0755); err != nil {
		t.Fatalf("failed to seed plain folder: %v", err)
	}
	page, err := store.Create("notes", "assets")
	if err != nil {
		t.Fatalf("unexpected error creating page inside plain folder: %v", err)
	}
	if page.ParentID != "assets" {
		t.Errorf("expected parent ID 'assets', got %q", page.ParentID)
	}

	pages := store.List()
	var folder core.Page
	found := false
	for _, p := range pages {
		if p.ID == "assets" {
			folder = p
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'assets' folder in List(), got %v", pages)
	}
	if !folder.IsFolder {
		t.Errorf("expected 'assets' to be marked IsFolder, got %v", folder)
	}
}

func TestStoreUpdateRenamesFileAndSidecarFolder(t *testing.T) {
	store := newTestStore(t)

	parent, err := store.Create("Parent 1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := store.Create("child 1", parent.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parent.Title = "Renamed Parent"
	parent.Content = "# Renamed Parent"
	updated, err := store.Update(parent)
	if err != nil {
		t.Fatalf("unexpected error updating parent: %v", err)
	}
	if updated.ID != "Renamed Parent.md" {
		t.Errorf("expected renamed ID 'Renamed Parent.md', got %q", updated.ID)
	}

	// The sidecar folder (and the child inside it) must have moved too.
	movedChild := filepath.Join(store.Root(), "Renamed Parent", "child 1.md")
	if _, err := os.Stat(movedChild); err != nil {
		t.Errorf("expected child page to move with renamed sidecar folder: %v", err)
	}

	got, err := store.Get(updated.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching renamed page: %v", err)
	}
	if got.Content != "# Renamed Parent" {
		t.Errorf("expected content to persist through rename, got %q", got.Content)
	}

	// Sanity: the child's own ID now reflects the new parent path.
	pages := store.List()
	foundChild := false
	for _, p := range pages {
		if p.Title == "child 1" {
			foundChild = true
			if p.ParentID != updated.ID {
				t.Errorf("expected child ParentID %q, got %q", updated.ID, p.ParentID)
			}
		}
	}
	if !foundChild {
		t.Errorf("expected to find child 1 after rename, got %v", pages)
	}
}

func TestStoreDeleteCascadesToSidecarFolder(t *testing.T) {
	store := newTestStore(t)

	parent, err := store.Create("Parent 1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := store.Create("child 1", parent.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := store.Delete(parent.ID); err != nil {
		t.Fatalf("unexpected error deleting parent: %v", err)
	}

	if pages := store.List(); len(pages) != 0 {
		t.Errorf("expected deleting a parent to cascade to its children, got %v", pages)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "Parent 1")); !os.IsNotExist(err) {
		t.Errorf("expected sidecar folder to be removed, stat err = %v", err)
	}
}

func TestStoreEdgeCases(t *testing.T) {
	store := newTestStore(t)

	if _, err := store.Get("non-existent.md"); err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound, got %v", err)
	}
	if _, err := store.Update(core.Page{ID: "invalid-id.md"}); err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound on update, got %v", err)
	}
	if err := store.Delete("invalid-id.md"); err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound on delete, got %v", err)
	}
}

func TestStoreListReflectsRealDirectoryOrdering(t *testing.T) {
	store := newTestStore(t)

	if _, err := store.Create("Zebra", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := store.Create("Apple", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(store.Root(), "Middle Folder"), 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pages := store.List()
	if len(pages) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(pages), pages)
	}
	// Folders sort before files; both groups are alphabetical.
	if pages[0].Title != "Middle Folder" || !pages[0].IsFolder {
		t.Errorf("expected folder first, got %v", pages[0])
	}
	if pages[1].Title != "Apple" || pages[2].Title != "Zebra" {
		t.Errorf("expected files alphabetically after folders, got %v, %v", pages[1], pages[2])
	}
}

func TestNewStoreCreatesMissingWorkspaceDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist-yet")
	store, err := core.NewStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info, err := os.Stat(store.Root()); err != nil || !info.IsDir() {
		t.Errorf("expected workspace directory to be created, got err=%v", err)
	}
}
