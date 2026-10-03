package content_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jaisuriya-11/tsuzuri/internal/content"
	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

func keys(c content.Model, s string) content.Model {
	for _, r := range s {
		c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return c
}

func TestContentComponent(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "notes/Notes.md", Title: "Notes", Content: "Hello world"})

	view := c.View()
	if strings.Contains(view, "Notes.md") || !strings.Contains(view, "Hello world") {
		t.Errorf("expected only the text (no file name row), got %q", view)
	}
	if c.ModeString() != "NORMAL" {
		t.Errorf("expected NORMAL mode, got %q", c.ModeString())
	}
	if got := strings.Count(view, "\n") + 1; got != 20 {
		t.Errorf("expected exactly 20 rows, got %d", got)
	}
}

func TestVimEdits(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "a.md", Content: "one\ntwo\nthree"})

	c = keys(c, "jdd")
	if c.Value() != "one\nthree" {
		t.Fatalf("dd: got %q", c.Value())
	}
	c = keys(c, "x")
	if c.Value() != "one\nhree" {
		t.Fatalf("x: got %q", c.Value())
	}
	c = keys(c, "ozero")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if c.Value() != "one\nhree\nzero" || c.Mode() != content.ModeNormal {
		t.Fatalf("o: got %q mode %v", c.Value(), c.Mode())
	}
	c = keys(c, "ggA!")
	if !strings.HasPrefix(c.Value(), "one!") {
		t.Fatalf("gg A: got %q", c.Value())
	}
}

func TestWheelScrollsWithoutFocus(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 10)
	var b strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&b, "row %d\n", i)
	}
	c.SetPage(core.Page{ID: "a.md", Content: b.String()})
	for i := 0; i < 10; i++ {
		c, _ = c.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	}
	v := c.View()
	if strings.Contains(v, "row 1\n") || !strings.Contains(v, "row 31") {
		t.Fatalf("expected view scrolled by 30 lines, got:\n%s", v)
	}
	if line, _ := c.CursorPosition(); line != 31 {
		t.Fatalf("cursor should be dragged to the first visible line, got %d", line)
	}
}

func TestCommandParsing(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 10)
	c.SetPage(core.Page{ID: "a.md", Content: "x"})
	c = keys(c, ":w notes/new")
	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	var found bool
	for _, msg := range collect(cmd) {
		if s, ok := msg.(core.VimSaveMsg); ok && s.Path == "notes/new" && s.Content == "x" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected VimSaveMsg with path from ':w notes/new'")
	}
}

func collect(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, collect(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func TestSlashMenuInsertsBlocks(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "a.md", Content: ""})
	c.SetFocused(true)
	c = keys(c, "i/")
	if !c.SlashOpen() {
		t.Fatal("expected '/' at line start to open the block menu")
	}
	if box, _, _, ok := c.SlashView(); !ok || !strings.Contains(box, "Heading 1") || !strings.Contains(box, "Basic blocks") {
		t.Fatalf("expected menu with sections, got %q", box)
	}
	c = keys(c, "todo")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	c = keys(c, "buy milk")
	if c.Value() != "- [ ] buy milk" || c.SlashOpen() {
		t.Fatalf("expected to-do inserted, got %q", c.Value())
	}

	// Code block puts the cursor inside the fence.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	c = keys(c, "/code")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	c = keys(c, "x := 1")
	if !strings.HasSuffix(c.Value(), "```\nx := 1\n```") {
		t.Fatalf("expected cursor inside code fence, got %q", c.Value())
	}

	// A slash inside a word is just a slash; Esc closes without changes.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	c = keys(c, "Goa/b")
	if c.SlashOpen() {
		t.Fatal("slash inside a word must not open the menu")
	}
	c = keys(c, " /")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if c.SlashOpen() || c.Mode() != content.ModeInsert {
		t.Fatal("Esc should only close the menu")
	}
}

func TestTabIndents(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 10)
	c.SetPage(core.Page{ID: "a.md", Content: ""})
	c.SetFocused(true)
	c = keys(c, "i- item")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	if c.Value() != "  - item" {
		t.Fatalf("Tab on a list line should nest it, got %q", c.Value())
	}
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if c.Value() != "- item" {
		t.Fatalf("Shift+Tab should outdent, got %q", c.Value())
	}
	c = keys(c, " x")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	c = keys(c, "a")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	c = keys(c, "b")
	if !strings.HasSuffix(c.Value(), "\na  b") {
		t.Fatalf("Tab in text should insert two spaces, got %q", c.Value())
	}
}
