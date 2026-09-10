// Package core defines domain entities, state management, and event messages.
// It must NOT import any UI packages.
package core

import "time"

// Block represents an individual content block within a Page.
type Block struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Page represents a workspace document.
type Page struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Blocks    []Block   `json:"blocks"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}
