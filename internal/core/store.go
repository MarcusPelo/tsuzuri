package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	// ErrPageNotFound indicates the requested page ID does not exist in the store.
	ErrPageNotFound = errors.New("page not found")
	// ErrNoPages indicates the store contains no pages.
	ErrNoPages = errors.New("no pages available in store")
)

// Store manages in-memory and workspace persistence for Pages.
type Store struct {
	mu    sync.RWMutex
	pages map[string]Page
	order []string
}

// NewStore creates a Store populated with default workspace pages.
func NewStore() *Store {
	s := &Store{
		pages: make(map[string]Page),
		order: make([]string, 0),
	}

	initialPage := Page{
		ID:        "page-1",
		Title:     "Solutions Architect",
		Content:   "# Solutions Architect\n\nWelcome to your Notion-style workspace in Tsuzuri.\n\n- [ ] Design cloud microservices\n- [x] Review architecture proposal\n\n> 💡 Connect your calendar and start meeting notes.",
		Icon:      "",
		UpdatedAt: time.Now(),
	}

	s.pages[initialPage.ID] = initialPage
	s.order = append(s.order, initialPage.ID)

	return s
}

// List returns all pages in workspace ordering.
func (s *Store) List() []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Page, 0, len(s.order))
	for _, id := range s.order {
		if p, ok := s.pages[id]; ok {
			result = append(result, p)
		}
	}
	return result
}

// Get fetches a page by ID, returning ErrPageNotFound if not found.
func (s *Store) Get(id string) (Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.pages[id]
	if !ok {
		return Page{}, ErrPageNotFound
	}
	return p, nil
}

// Create generates and stores a new Page.
func (s *Store) Create(title string) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nextNum := len(s.pages) + 1
	if title == "" {
		title = fmt.Sprintf("Untitled %d", nextNum)
	}

	p := Page{
		ID:        fmt.Sprintf("page-%d", nextNum),
		Title:     title,
		Content:   "",
		UpdatedAt: time.Now(),
	}

	s.pages[p.ID] = p
	s.order = append(s.order, p.ID)
	return p, nil
}

// CreateChild generates and stores a new Page nested under parentID.
func (s *Store) CreateChild(parentID, title string) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pages[parentID]; !ok {
		return Page{}, ErrPageNotFound
	}

	nextNum := len(s.pages) + 1
	if title == "" {
		title = fmt.Sprintf("Untitled %d", nextNum)
	}

	p := Page{
		ID:        fmt.Sprintf("page-%d", nextNum),
		Title:     title,
		Content:   "",
		ParentID:  parentID,
		UpdatedAt: time.Now(),
	}

	s.pages[p.ID] = p
	s.order = append(s.order, p.ID)
	return p, nil
}

// Update updates an existing page's title or content.
func (s *Store) Update(p Page) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pages[p.ID]; !ok {
		return ErrPageNotFound
	}
	p.UpdatedAt = time.Now()
	s.pages[p.ID] = p
	return nil
}

// Delete removes a page by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.pages[id]; !ok {
		return ErrPageNotFound
	}
	delete(s.pages, id)

	newOrder := make([]string, 0, len(s.order)-1)
	for _, pageID := range s.order {
		if pageID != id {
			newOrder = append(newOrder, pageID)
		}
	}
	s.order = newOrder
	return nil
}

// WorkspaceData defines the serializable JSON schema for store save/load persistence.
type WorkspaceData struct {
	Pages []Page `json:"pages"`
}

// SaveToFile serializes the store contents to a JSON file.
func (s *Store) SaveToFile(filePath string) error {
	s.mu.RLock()
	data := WorkspaceData{Pages: s.List()}
	s.mu.RUnlock()

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workspace data: %w", err)
	}

	if err := os.WriteFile(filePath, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write workspace file: %w", err)
	}
	return nil
}

// LoadFromFile reads and populates the store from a JSON workspace file.
func (s *Store) LoadFromFile(filePath string) error {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read workspace file: %w", err)
	}

	var data WorkspaceData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return fmt.Errorf("failed to unmarshal workspace file: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pages = make(map[string]Page)
	s.order = make([]string, 0, len(data.Pages))

	for _, p := range data.Pages {
		s.pages[p.ID] = p
		s.order = append(s.order, p.ID)
	}

	return nil
}

// SeedHierarchy populates the store with the multi-level Notion structure if only the initial page is present.
func (s *Store) SeedHierarchy() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pages) > 1 {
		return
	}

	p1 := Page{
		ID:        "page-parent-1",
		Title:     "Parent 1",
		Content:   "# Parent 1\n\nTop-level workspace document.",
		UpdatedAt: time.Now(),
	}
	c1 := Page{
		ID:        "page-child-1",
		Title:     "child 1",
		ParentID:  "page-parent-1",
		Content:   "## child 1\n\nNested sub-document.",
		UpdatedAt: time.Now(),
	}
	gc1 := Page{
		ID:        "page-child-1-child",
		Title:     "child 1's child",
		ParentID:  "page-child-1",
		Content:   "### child 1's child\n\nGrandchild note at depth 3.",
		UpdatedAt: time.Now(),
	}
	p2 := Page{
		ID:        "page-parent-2",
		Title:     "Parent 2",
		Content:   "# Parent 2\n\nSecond root document.",
		UpdatedAt: time.Now(),
	}
	c2 := Page{
		ID:        "page-child-2",
		Title:     "child 2",
		ParentID:  "page-parent-2",
		Content:   "## child 2\n\nNested document under Parent 2.",
		UpdatedAt: time.Now(),
	}

	for _, p := range []Page{p1, c1, gc1, p2, c2} {
		s.pages[p.ID] = p
		s.order = append(s.order, p.ID)
	}
}
