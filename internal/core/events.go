package core

// PageSelectedMsg is emitted by Sidebar when the user selects a page.
type PageSelectedMsg struct {
	ID string
}

// PageCreatedMsg is emitted when a new page is created.
type PageCreatedMsg struct {
	Page Page
}

// PageUpdatedMsg is emitted when a page title or content is updated.
type PageUpdatedMsg struct {
	Page Page
}

// PageDeletedMsg is emitted when a page is deleted.
type PageDeletedMsg struct {
	ID string
}

// VimSaveMsg is emitted when the user executes :w in the editor.
type VimSaveMsg struct {
	Content string
}

// VimQuitMsg is emitted when the user executes :q or :wq in the editor.
type VimQuitMsg struct {
	Save bool
}
