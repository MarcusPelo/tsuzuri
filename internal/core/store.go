package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// mdExt is the file extension used for page content files.
const mdExt = ".md"

var (
	// ErrPageNotFound indicates the requested page ID does not exist on disk.
	ErrPageNotFound = errors.New("page not found")
)

// Store is a thin, filesystem-backed workspace: every Page is a real ".md"
// file on disk (plus an optional same-name sidecar folder holding its
// sub-pages), and every plain directory it finds is surfaced as a navigable
// folder node. There is no separate persistence format — the directory tree
// IS the source of truth, so IDs (slash-separated paths relative to the
// workspace root) are recomputed fresh from disk on every List/Get call.
type Store struct {
	mu   sync.RWMutex
	root string
}

// NewStore opens (creating if necessary) a filesystem-backed workspace rooted at dir.
func NewStore(dir string) (*Store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("invalid workspace path: %w", err)
	}

	info, statErr := os.Stat(abs)
	switch {
	case os.IsNotExist(statErr):
		if mkErr := os.MkdirAll(abs, 0755); mkErr != nil {
			return nil, fmt.Errorf("failed to create workspace directory: %w", mkErr)
		}
	case statErr != nil:
		return nil, fmt.Errorf("failed to access workspace directory: %w", statErr)
	case !info.IsDir():
		return nil, fmt.Errorf("workspace path %q is not a directory", abs)
	}

	return &Store{root: abs}, nil
}

// Root returns the absolute workspace directory backing this store.
func (s *Store) Root() string {
	return s.root
}

func (s *Store) idToAbs(id string) string {
	return filepath.Join(s.root, filepath.FromSlash(id))
}

// parentIDFor derives the ID of id's parent page/folder purely from its path.
func parentIDFor(root, id string) string {
	dir := path.Dir(id)
	if dir == "." || dir == "/" {
		return ""
	}
	mdCandidate := dir + mdExt
	if fileExists(filepath.Join(root, filepath.FromSlash(mdCandidate))) {
		return mdCandidate
	}
	return dir
}

func fileExists(abs string) bool {
	_, err := os.Stat(abs)
	return err == nil
}

// List walks the workspace directory tree and returns a flattened list of
// pages and folders, each carrying its real ParentID so callers can rebuild
// the hierarchy. Folders are ordered before files, both alphabetically.
// Folders with no Markdown file anywhere inside them are left out.
func (s *Store) List() []Page {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pages []Page
	s.walk("", "", &pages)
	return pages
}

// walk appends relDir's notes and folders to out and reports whether it
// found any Markdown file in the subtree.
func (s *Store) walk(relDir, parentID string, out *[]Page) bool {
	absDir := filepath.Join(s.root, filepath.FromSlash(relDir))
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return false
	}
	found := false

	mdBases := map[string]bool{}
	dirBases := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			dirBases[name] = true
		} else if strings.HasSuffix(name, mdExt) {
			mdBases[strings.TrimSuffix(name, mdExt)] = true
		}
	}

	var expandable, leaves []string
	seen := map[string]bool{}
	for base := range mdBases {
		seen[base] = true
		if dirBases[base] {
			expandable = append(expandable, base)
		} else {
			leaves = append(leaves, base)
		}
	}
	for base := range dirBases {
		if !seen[base] {
			expandable = append(expandable, base)
		}
	}

	caseInsensitive := func(list []string) func(i, j int) bool {
		return func(i, j int) bool { return strings.ToLower(list[i]) < strings.ToLower(list[j]) }
	}
	sort.Slice(expandable, caseInsensitive(expandable))
	sort.Slice(leaves, caseInsensitive(leaves))

	for _, base := range expandable {
		relChild := path.Join(relDir, base)
		if mdBases[base] {
			relFile := relChild + mdExt
			id := relFile
			*out = append(*out, Page{
				ID:        id,
				Title:     base,
				ParentID:  parentID,
				UpdatedAt: modTime(filepath.Join(s.root, filepath.FromSlash(relFile))),
			})
			s.walk(relChild, id, out)
			found = true
		} else {
			id := relChild
			folder := Page{
				ID:        id,
				Title:     base,
				ParentID:  parentID,
				IsFolder:  true,
				UpdatedAt: modTime(filepath.Join(s.root, filepath.FromSlash(relChild))),
			}
			var children []Page
			if s.walk(relChild, id, &children) {
				*out = append(*out, folder)
				*out = append(*out, children...)
				found = true
			}
		}
	}

	for _, base := range leaves {
		relFile := path.Join(relDir, base) + mdExt
		*out = append(*out, Page{
			ID:        relFile,
			Title:     base,
			ParentID:  parentID,
			UpdatedAt: modTime(filepath.Join(s.root, filepath.FromSlash(relFile))),
		})
		found = true
	}
	return found
}

func modTime(abs string) time.Time {
	info, err := os.Stat(abs)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// Get fetches a single page (or folder) by ID, reading its content from disk.
func (s *Store) Get(id string) (Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id == "" {
		return Page{}, ErrPageNotFound
	}

	abs := s.idToAbs(id)
	info, err := os.Stat(abs)
	if err != nil {
		return Page{}, ErrPageNotFound
	}

	parentID := parentIDFor(s.root, id)
	title := strings.TrimSuffix(path.Base(id), mdExt)

	if info.IsDir() {
		return Page{ID: id, Title: title, ParentID: parentID, IsFolder: true, UpdatedAt: info.ModTime()}, nil
	}

	if !strings.HasSuffix(id, mdExt) {
		return Page{}, ErrPageNotFound
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return Page{}, ErrPageNotFound
	}

	return Page{ID: id, Title: title, Content: string(data), ParentID: parentID, UpdatedAt: info.ModTime()}, nil
}

// childDirFor resolves the relative directory new children of parentID should
// be written into: the workspace root, an existing plain folder, or the
// sidecar folder belonging to a page.
func (s *Store) childDirFor(parentID string) (string, error) {
	if parentID == "" {
		return "", nil
	}

	abs := s.idToAbs(parentID)
	info, err := os.Stat(abs)
	if err != nil {
		return "", ErrPageNotFound
	}
	if info.IsDir() {
		return parentID, nil
	}
	if !strings.HasSuffix(parentID, mdExt) {
		return "", ErrPageNotFound
	}
	return strings.TrimSuffix(parentID, mdExt), nil
}

// sanitizeTitle strips path separators and surrounding whitespace so a title
// can never escape its intended directory.
func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", "\x00", "")
	cleaned := strings.TrimSpace(replacer.Replace(title))
	if cleaned == "" {
		cleaned = "Untitled"
	}
	return cleaned
}

func nextUntitled(absDir string) string {
	entries, _ := os.ReadDir(absDir)
	existing := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() {
			name = strings.TrimSuffix(name, mdExt)
		}
		existing[name] = true
	}
	if !existing["Untitled"] {
		return "Untitled"
	}
	for i := 2; ; i++ {
		cand := fmt.Sprintf("Untitled %d", i)
		if !existing[cand] {
			return cand
		}
	}
}

// Create writes a new page file to disk under parentID ("" for the workspace
// root) and returns the resulting Page. An empty title generates "Untitled"
// (or "Untitled N" if that's taken); a colliding title is disambiguated the
// same way.
func (s *Store) Create(title, parentID string) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	relDir, err := s.childDirFor(parentID)
	if err != nil {
		return Page{}, err
	}

	absDir := filepath.Join(s.root, filepath.FromSlash(relDir))
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return Page{}, fmt.Errorf("failed to create folder: %w", err)
	}

	title = strings.TrimSpace(title)
	if title == "" {
		title = nextUntitled(absDir)
	} else {
		title = sanitizeTitle(title)
	}

	finalTitle := title
	absFile := filepath.Join(absDir, finalTitle+mdExt)
	for n := 2; fileExists(absFile); n++ {
		finalTitle = fmt.Sprintf("%s %d", title, n)
		absFile = filepath.Join(absDir, finalTitle+mdExt)
	}

	if err := os.WriteFile(absFile, []byte(""), 0644); err != nil {
		return Page{}, fmt.Errorf("failed to create page: %w", err)
	}

	id := path.Join(relDir, finalTitle+mdExt)
	return Page{ID: id, Title: finalTitle, ParentID: parentID, UpdatedAt: modTime(absFile)}, nil
}

// Update persists p.Content to disk and, if p.Title no longer matches the
// page's current filename, renames the file (and its sidecar sub-pages
// folder, if any) to match. It returns the page as it now exists on disk,
// since a rename changes its ID.
func (s *Store) Update(p Page) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	abs := s.idToAbs(p.ID)
	info, err := os.Stat(abs)
	if err != nil {
		return Page{}, ErrPageNotFound
	}

	if info.IsDir() {
		return s.renameFolder(p, abs)
	}

	currentTitle := strings.TrimSuffix(path.Base(p.ID), mdExt)
	newTitle := strings.TrimSpace(p.Title)
	if newTitle == "" {
		newTitle = currentTitle
	} else {
		newTitle = sanitizeTitle(newTitle)
	}

	id := p.ID
	if newTitle != currentTitle {
		dir := path.Dir(p.ID)
		if dir == "." {
			dir = ""
		}
		absDir := filepath.Join(s.root, filepath.FromSlash(dir))
		newAbs := filepath.Join(absDir, newTitle+mdExt)

		if fileExists(newAbs) {
			return Page{}, fmt.Errorf("a page named %q already exists here", newTitle)
		}
		if err := os.Rename(abs, newAbs); err != nil {
			return Page{}, fmt.Errorf("failed to rename page: %w", err)
		}

		oldSidecar := strings.TrimSuffix(abs, mdExt)
		if info, err := os.Stat(oldSidecar); err == nil && info.IsDir() {
			newSidecar := strings.TrimSuffix(newAbs, mdExt)
			if err := os.Rename(oldSidecar, newSidecar); err != nil {
				return Page{}, fmt.Errorf("failed to rename sub-pages: %w", err)
			}
		}

		id = path.Join(dir, newTitle+mdExt)
		abs = newAbs
	}

	if err := os.WriteFile(abs, []byte(p.Content), 0644); err != nil {
		return Page{}, fmt.Errorf("failed to save page: %w", err)
	}

	parentID := parentIDFor(s.root, id)
	return Page{ID: id, Title: newTitle, Content: p.Content, ParentID: parentID, UpdatedAt: modTime(abs)}, nil
}

// renameFolder handles Update for a plain folder node (no backing .md file,
// so there's no content to persist — only a possible rename). Caller must
// already hold s.mu.
func (s *Store) renameFolder(p Page, abs string) (Page, error) {
	currentTitle := path.Base(p.ID)
	newTitle := strings.TrimSpace(p.Title)
	if newTitle == "" {
		newTitle = currentTitle
	} else {
		newTitle = sanitizeTitle(newTitle)
	}

	id := p.ID
	if newTitle != currentTitle {
		dir := path.Dir(p.ID)
		if dir == "." {
			dir = ""
		}
		absDir := filepath.Join(s.root, filepath.FromSlash(dir))
		newAbs := filepath.Join(absDir, newTitle)

		if fileExists(newAbs) {
			return Page{}, fmt.Errorf("a folder named %q already exists here", newTitle)
		}
		if err := os.Rename(abs, newAbs); err != nil {
			return Page{}, fmt.Errorf("failed to rename folder: %w", err)
		}

		id = path.Join(dir, newTitle)
		abs = newAbs
	}

	parentID := parentIDFor(s.root, id)
	return Page{ID: id, Title: newTitle, ParentID: parentID, IsFolder: true, UpdatedAt: modTime(abs)}, nil
}

// Delete removes a page (or folder) by ID. Deleting a page cascades to its
// sidecar sub-pages folder, if any; deleting a folder removes everything
// nested inside it.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	abs := s.idToAbs(id)
	info, err := os.Stat(abs)
	if err != nil {
		return ErrPageNotFound
	}

	if info.IsDir() {
		if err := os.RemoveAll(abs); err != nil {
			return fmt.Errorf("failed to delete folder: %w", err)
		}
		return nil
	}

	if err := os.Remove(abs); err != nil {
		return fmt.Errorf("failed to delete page: %w", err)
	}

	sidecar := strings.TrimSuffix(abs, mdExt)
	if sInfo, err := os.Stat(sidecar); err == nil && sInfo.IsDir() {
		_ = os.RemoveAll(sidecar)
	}
	return nil
}

// Dirs returns every non-hidden directory in the workspace as a
// slash-separated path relative to the root, sorted case-insensitively. The
// root itself is represented by "".
func (s *Store) Dirs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dirs := []string{""}
	_ = filepath.WalkDir(s.root, func(p string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || p == s.root {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		rel, relErr := filepath.Rel(s.root, p)
		if relErr != nil {
			return nil
		}
		dirs = append(dirs, filepath.ToSlash(rel))
		return nil
	})
	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i]) < strings.ToLower(dirs[j]) })
	return dirs
}

// ChildDir returns the directory (relative to the root) where new children of
// parentID live: the root, a plain folder, or a page's sidecar folder. Unknown
// parents resolve to the root.
func (s *Store) ChildDir(parentID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dir, err := s.childDirFor(parentID)
	if err != nil {
		return ""
	}
	return dir
}

// NormalizeNotePath cleans a user-supplied relative note path and forces the
// ".md" extension (Tsuzuri only ever writes Markdown). It rejects absolute
// paths, hidden segments and anything that would escape the workspace.
func NormalizeNotePath(dir, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("file name is empty")
	}
	if strings.ContainsAny(name, "/\\\x00") {
		return "", errors.New("file name cannot contain / or \\")
	}
	if !strings.HasSuffix(strings.ToLower(name), mdExt) {
		name += mdExt
	} else {
		name = name[:len(name)-len(mdExt)] + mdExt
	}
	if strings.TrimSuffix(name, mdExt) == "" {
		return "", errors.New("file name is empty")
	}

	dir = strings.Trim(strings.TrimSpace(filepath.ToSlash(dir)), "/")
	rel := path.Clean(path.Join(dir, name))
	if path.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", errors.New("location must be inside the workspace")
	}
	for _, seg := range strings.Split(rel, "/") {
		if strings.HasPrefix(seg, ".") {
			return "", errors.New("hidden files and folders are not allowed")
		}
	}
	return rel, nil
}

// SaveAs writes content to a new note at dir/name (".md" is added if
// missing), creating any missing folders. It refuses to overwrite an
// existing file.
func (s *Store) SaveAs(dir, name, content string) (Page, error) {
	rel, err := NormalizeNotePath(dir, name)
	if err != nil {
		return Page{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	abs := s.idToAbs(rel)
	if fileExists(abs) {
		return Page{}, fmt.Errorf("%s already exists", rel)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return Page{}, fmt.Errorf("failed to create folder: %w", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		return Page{}, fmt.Errorf("failed to save page: %w", err)
	}
	return Page{
		ID:        rel,
		Title:     strings.TrimSuffix(path.Base(rel), mdExt),
		Content:   content,
		ParentID:  parentIDFor(s.root, rel),
		UpdatedAt: modTime(abs),
	}, nil
}

// Rename changes a page's or folder's name on disk without touching its
// content (Update would overwrite the file with p.Content).
func (s *Store) Rename(id, newTitle string) (Page, error) {
	p, err := s.Get(id)
	if err != nil {
		return Page{}, err
	}
	p.Title = strings.TrimSuffix(strings.TrimSpace(newTitle), mdExt)
	return s.Update(p)
}

// AttachFile makes src available to a note living in noteDir (relative to
// the root) and returns the link target to write in the note, relative to
// noteDir. Files already inside the workspace are linked in place; anything
// else is copied into noteDir/assets/ (renamed if the name is taken).
func (s *Store) AttachFile(noteDir, src string) (string, error) {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absSrc)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a folder", filepath.Base(absSrc))
	}
	noteAbs := s.idToAbs(noteDir)

	if rel, err := filepath.Rel(s.root, absSrc); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		link, err := filepath.Rel(noteAbs, absSrc)
		if err != nil {
			return "", err
		}
		return filepath.ToSlash(link), nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	assets := filepath.Join(noteAbs, "assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		return "", fmt.Errorf("failed to create assets folder: %w", err)
	}
	ext := filepath.Ext(absSrc)
	stem := strings.TrimSuffix(filepath.Base(absSrc), ext)
	dst := filepath.Join(assets, stem+ext)
	for n := 2; fileExists(dst); n++ {
		dst = filepath.Join(assets, fmt.Sprintf("%s-%d%s", stem, n, ext))
	}
	if err := copyFile(absSrc, dst); err != nil {
		return "", err
	}
	return "assets/" + filepath.Base(dst), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return fmt.Errorf("failed to copy %s: %w", filepath.Base(src), err)
	}
	return out.Close()
}
