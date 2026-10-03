// Package core defines domain entities, state management, and event messages.
// It must NOT import any UI packages.
package core

import "time"

// Page represents a workspace document or folder backed by a real file on disk.
//
// ID is the slash-separated path of the backing file relative to the workspace
// root (e.g. "Parent 1/child 1.md"), which doubles as a stable identity: it is
// recomputed from the filesystem on every List/Get, so it always matches reality.
type Page struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	ParentID  string    `json:"parent_id,omitempty"`
	Icon      string    `json:"icon,omitempty"`
	IsFolder  bool      `json:"is_folder,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}
