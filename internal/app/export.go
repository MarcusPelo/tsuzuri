package app

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jaisuriya-11/tsuzuri/internal/export"
)

// exportPDF writes the open note as a PDF (":export [file]"), next to the
// note unless a path is given.
func (m *Model) exportPDF(arg string) tea.Cmd {
	b := m.activeBuffer()
	if b == nil {
		m.setError("Open a note to export it")
		return nil
	}
	dir := m.noteDir(b)
	out := export.Target(arg, dir, b.title)
	if err := export.WriteFile(m.liveText(b), export.Options{Title: b.title, BaseDir: dir}, out); err != nil {
		m.setError("Export failed: " + err.Error())
		return nil
	}
	shown := tildePath(out)
	if rel, err := filepath.Rel(m.store.Root(), out); err == nil && !strings.HasPrefix(rel, "..") {
		shown = filepath.ToSlash(rel)
	}
	m.setStatus("\"" + shown + "\" exported")
	return m.showToast("Exported " + shown)
}
