package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jaisuriya-11/tsuzuri/internal/app"
	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const (
	termW = 140
	termH = 40
)

func newTestStore(t *testing.T) *core.Store {
	t.Helper()
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error creating store: %v", err)
	}
	return store
}

// harness drives the model like the Bubble Tea runtime: every returned
// command is executed and its message fed back in. Timer-based commands
// (cursor blinks) are dropped.
type harness struct {
	t     *testing.T
	m     tea.Model
	quit  bool
	store *core.Store
}

func newHarness(t *testing.T, store *core.Store, opts ...app.Option) *harness {
	h := &harness{t: t, m: app.New(store, opts...), store: store}
	h.send(tea.WindowSizeMsg{Width: termW, Height: termH})
	return h
}

func runCmd(c tea.Cmd) tea.Msg {
	ch := make(chan tea.Msg, 1)
	go func() { ch <- c() }()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(30 * time.Millisecond):
		return nil
	}
}

func (h *harness) send(msgs ...tea.Msg) {
	h.t.Helper()
	for _, msg := range msgs {
		var cmd tea.Cmd
		h.m, cmd = h.m.Update(msg)
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			out := runCmd(c)
			switch out := out.(type) {
			case nil:
			case tea.BatchMsg:
				queue = append(queue, out...)
			case tea.QuitMsg:
				h.quit = true
			default:
				if name := fmt.Sprintf("%T", out); strings.Contains(name, "cursor.") || strings.Contains(name, "Blink") {
					continue
				}
				var next tea.Cmd
				h.m, next = h.m.Update(out)
				queue = append(queue, next)
			}
		}
	}
}

func (h *harness) keys(s string) {
	h.t.Helper()
	for _, r := range s {
		h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func (h *harness) key(t tea.KeyType) { h.t.Helper(); h.send(tea.KeyMsg{Type: t}) }

func (h *harness) click(x, y int) {
	h.t.Helper()
	h.send(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
}

func (h *harness) view() string {
	h.t.Helper()
	v := h.m.View()
	assertFrame(h.t, v)
	return ansi.Strip(v)
}

// assertFrame checks the frame is exactly termW × termH cells, which is what
// keeps pane dividers straight.
func assertFrame(t *testing.T, v string) {
	t.Helper()
	lines := strings.Split(v, "\n")
	if len(lines) != termH {
		t.Fatalf("frame has %d rows, want %d", len(lines), termH)
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w != termW {
			t.Fatalf("row %d is %d cells wide, want %d: %q", i, w, termW, ansi.Strip(l))
		}
	}
}

// find returns the screen position of the first occurrence of text.
func stripped(s string) string { return ansi.Strip(s) }

func find(t *testing.T, view, text string) (int, int) {
	t.Helper()
	for y, l := range strings.Split(view, "\n") {
		if i := strings.Index(l, text); i >= 0 {
			return ansi.StringWidth(l[:i]), y
		}
	}
	t.Fatalf("%q not on screen:\n%s", text, view)
	return 0, 0
}

func TestDashboardAndNewNoteSaveAs(t *testing.T) {
	store := newTestStore(t)
	if err := os.MkdirAll(filepath.Join(store.Root(), "journal"), 0755); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)

	v := h.view()
	for _, want := range []string{"New Note", "Find Note", "HOME"} {
		if !strings.Contains(v, want) {
			t.Fatalf("dashboard missing %q:\n%s", want, v)
		}
	}

	// n opens an in-memory Untitled-1 tab, straight into INSERT mode.
	h.keys("n")
	v = h.view()
	if !strings.Contains(v, "Untitled-1") || !strings.Contains(v, "INSERT") {
		t.Fatalf("expected Untitled-1 tab in INSERT mode:\n%s", v)
	}
	if pages := store.List(); len(pages) != 1 { // just the journal folder
		t.Fatalf("nothing may be written before saving, got %v", pages)
	}

	h.keys("hello")
	h.key(tea.KeyEsc)
	h.keys(":w")
	h.key(tea.KeyEnter)
	v = h.view()
	if !strings.Contains(v, "Save As") || !strings.Contains(v, "workspace root") {
		t.Fatalf("expected the Save As dialog:\n%s", v)
	}

	// Pick the journal folder in the dialog with the mouse, rename, save.
	_, top := find(t, v, "Folder")
	x, y := find(t, strings.Join(strings.Split(v, "\n")[top:], "\n"), "journal")
	h.click(x, y+top)
	for range "Untitled-1" {
		h.key(tea.KeyBackspace)
	}
	h.keys("day one.txt")
	if v = h.view(); !strings.Contains(v, "journal/day one.txt.md") {
		t.Fatalf("expected forced .md target path in dialog:\n%s", v)
	}
	h.key(tea.KeyEnter)

	got, err := store.Get("journal/day one.txt.md")
	if err != nil || got.Content != "hello" {
		t.Fatalf("expected saved note, got %+v, %v", got, err)
	}
	v = h.view()
	if strings.Contains(v, "Save As") || !strings.Contains(v, "day one.txt.md") {
		t.Fatalf("dialog should close and the tab show the file name:\n%s", v)
	}
}

func TestUnsavedEditsSurviveTabSwitchAndQuitAsks(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"alpha", "beta"} {
		if _, err := store.SaveAs("", n, n+" body"); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e") // explorer
	h.key(tea.KeyEnter)
	h.keys("A!") // append in alpha
	h.key(tea.KeyEsc)
	h.keys("[") // back to the explorer? no: [ cycles tabs; only one tab yet

	// Open beta from the explorer via Ctrl+P search.
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("beta")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "beta.md") || !strings.Contains(v, "alpha.md") {
		t.Fatalf("expected two tabs:\n%s", v)
	}

	h.keys("[")
	v := h.view()
	if !strings.Contains(v, "alpha body!") {
		t.Fatalf("unsaved edit lost after switching tabs:\n%s", v)
	}
	if got, _ := store.Get("alpha.md"); got.Content != "alpha body" {
		t.Fatalf("unsaved edit must not touch disk, got %q", got.Content)
	}

	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	if h.quit {
		t.Fatal("quit with unsaved changes must ask first")
	}
	if v = h.view(); !strings.Contains(v, "Save All") {
		t.Fatalf("expected quit confirmation:\n%s", v)
	}
	h.keys("s") // Save All
	if !h.quit {
		t.Fatal("expected quit after Save All")
	}
	if got, _ := store.Get("alpha.md"); got.Content != "alpha body!" {
		t.Fatalf("Save All did not write alpha, got %q", got.Content)
	}
}

func TestCloseTabAndCtrlS(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "x"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1") // open the recent note
	h.keys("A2")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got, _ := store.Get("doc.md"); got.Content != "x2" {
		t.Fatalf("Ctrl+S in INSERT mode should save, got %q", got.Content)
	}
	h.key(tea.KeyEsc)
	h.keys("A3")
	h.key(tea.KeyEsc)
	h.keys(" x") // leader x closes the tab
	if v := h.view(); !strings.Contains(v, "Don't Save") {
		t.Fatalf("closing a dirty tab must ask:\n%s", v)
	}
	h.keys("d")
	v := h.view()
	if strings.Contains(v, "doc.md  ") && strings.Contains(v, "●") {
		t.Fatalf("tab should be closed:\n%s", v)
	}
	if got, _ := store.Get("doc.md"); got.Content != "x2" {
		t.Fatalf("Don't Save must keep disk content, got %q", got.Content)
	}
}

func TestMouseTabsAndPanes(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"one", "two"} {
		if _, err := store.SaveAs("", n, n); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e")
	v := h.view()
	x, y := find(t, v, "one.md")
	h.click(x, y)
	v = h.view()
	x, y = find(t, v, "two.md")
	h.click(x, y)
	v = h.view()

	// Clicking a tab switches to it.
	x, _ = find(t, strings.Split(v, "\n")[0], "one.md")
	h.click(x, 0)
	if v = h.view(); !strings.Contains(strings.Split(v, "\n")[1], "1 one") {
		t.Fatalf("expected one.md active after clicking its tab:\n%s", v)
	}

	// The "+" button opens a new draft.
	x, _ = find(t, strings.Split(v, "\n")[0], "\uf067")
	h.click(x+1, 0)
	if v = h.view(); !strings.Contains(v, "Untitled-1") {
		t.Fatalf("expected new draft from + button:\n%s", v)
	}

	// Wheel over the explorer must not steal focus or crash.
	h.send(tea.MouseMsg{X: 3, Y: 5, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	h.view()
}

func TestRenameKeepsContentAndDeleteAsks(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "Only Page", "precious"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("e")
	h.keys("r")
	for range "Only Page" {
		h.key(tea.KeyBackspace)
	}
	h.keys("Renamed")
	h.key(tea.KeyEnter)
	got, err := store.Get("Renamed.md")
	if err != nil || got.Content != "precious" {
		t.Fatalf("rename must keep content, got %+v %v", got, err)
	}

	h.keys("d")
	if len(store.List()) != 1 {
		t.Fatal("delete must wait for confirmation")
	}
	if v := h.view(); !strings.Contains(v, "Delete Renamed.md?") {
		t.Fatalf("expected delete confirmation:\n%s", v)
	}
	h.key(tea.KeyEnter)
	if len(store.List()) != 0 {
		t.Fatalf("expected file deleted, got %v", store.List())
	}
	if v := h.view(); !strings.Contains(v, "New Note") {
		t.Fatalf("empty workspace should return home:\n%s", v)
	}
}

func TestLongFileLoadsCompletely(t *testing.T) {
	store := newTestStore(t)
	var b strings.Builder
	for i := 1; i <= 300; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	if _, err := store.SaveAs("", "long", b.String()); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.keys("G")
	v := h.view()
	if !strings.Contains(v, "line 300") {
		t.Fatalf("expected the end of a 300-line file to be reachable:\n%s", v)
	}
	// Mouse wheel scrolls the editor back up.
	for i := 0; i < 120; i++ {
		h.send(tea.MouseMsg{X: 60, Y: 10, Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	}
	if v = h.view(); !strings.Contains(v, "line 1\n") && !strings.Contains(v, "line 1 ") {
		t.Fatalf("expected wheel to scroll to the top:\n%s", v)
	}
}

func TestKeymapModalAndSmallScreens(t *testing.T) {
	h := newHarness(t, newTestStore(t))
	h.keys("?")
	if v := h.view(); !strings.Contains(v, "TSUZURI KEYMAPS") {
		t.Fatalf("expected keymap modal:\n%s", v)
	}
	h.key(tea.KeyEsc)
	if v := h.view(); strings.Contains(v, "TSUZURI KEYMAPS") {
		t.Fatal("modal should close on Esc")
	}

	for _, size := range [][2]int{{60, 12}, {80, 24}, {200, 60}} {
		m, _ := app.New(newTestStore(t)).Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		lines := strings.Split(m.View(), "\n")
		if len(lines) != size[1] {
			t.Errorf("%v: %d rows", size, len(lines))
		}
		for i, l := range lines {
			if w := ansi.StringWidth(l); w != size[0] {
				t.Errorf("%v: row %d width %d", size, i, w)
				break
			}
		}
	}
}

func TestFinderFromHomeOpensInCurrentBuffer(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"alpha", "beta", "gamma"} {
		if _, err := store.SaveAs("notes", n, n+" text"); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)

	// Find from the home screen is a modal over the home screen.
	h.keys("f")
	v := h.view()
	if !strings.Contains(v, "Find Note") || !strings.Contains(v, "HOME") {
		t.Fatalf("expected finder over the home screen:\n%s", v)
	}
	h.keys("bta") // fuzzy: b-e-t-a
	h.key(tea.KeyEnter)
	v = h.view()
	if !strings.Contains(strings.Split(v, "\n")[0], "beta.md") || strings.Contains(v, "Find Note") {
		t.Fatalf("expected beta.md open and the finder closed:\n%s", v)
	}

	// Ctrl+P from the editor replaces the (clean) current tab.
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("gamma")
	h.key(tea.KeyEnter)
	tabs := strings.Split(h.view(), "\n")[0]
	if !strings.Contains(tabs, "gamma.md") || strings.Contains(tabs, "beta.md") {
		t.Fatalf("expected gamma.md to replace beta.md in the same tab: %q", tabs)
	}

	// A dirty tab is never replaced.
	h.keys("A!")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("alpha")
	h.key(tea.KeyEnter)
	tabs = strings.Split(h.view(), "\n")[0]
	if !strings.Contains(tabs, "gamma.md") || !strings.Contains(tabs, "alpha.md") {
		t.Fatalf("expected dirty gamma.md kept next to alpha.md: %q", tabs)
	}

	// Esc closes the finder without changing anything.
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.key(tea.KeyEsc)
	if strings.Contains(h.view(), "Find Note") {
		t.Fatal("Esc should close the finder")
	}
}

func TestCtrlBFocusesExplorerFromInsertMode(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"a", "b"} {
		if _, err := store.SaveAs("", n, n); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("n") // draft in INSERT mode
	h.send(tea.KeyMsg{Type: tea.KeyCtrlB})
	if v := h.view(); !strings.Contains(v, "EXPLORER") || strings.Contains(v, "INSERT") {
		t.Fatalf("expected explorer focus after Ctrl+B:\n%s", v)
	}
	h.keys("j")
	h.key(tea.KeyEnter)
	if tabs := strings.Split(h.view(), "\n")[0]; !strings.Contains(tabs, "b.md") {
		t.Fatalf("expected tree keys to work after Ctrl+B: %q", tabs)
	}
}

func TestThemePickerPreviewRevertAndSave(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	h := newHarness(t, newTestStore(t), app.WithConfigPath(cfgPath))

	h.keys("t")
	if v := h.view(); !strings.Contains(v, "Themes") || !strings.Contains(v, "onedark") {
		t.Fatalf("expected theme picker:\n%s", v)
	}
	h.keys("gruvbox_light")
	h.key(tea.KeyEsc) // revert
	if _, err := os.Stat(cfgPath); err == nil {
		t.Fatal("Esc must not save a theme")
	}

	h.keys("t")
	h.keys("gruvbox_light")
	h.key(tea.KeyEnter)
	data, err := os.ReadFile(cfgPath)
	if err != nil || !strings.Contains(string(data), "gruvbox_light") {
		t.Fatalf("expected theme saved, got %q %v", data, err)
	}

	// :colorscheme from the editor.
	h.keys("n")
	h.key(tea.KeyEsc)
	h.keys(":colorscheme nord")
	h.key(tea.KeyEnter)
	if data, _ := os.ReadFile(cfgPath); !strings.Contains(string(data), "nord") {
		t.Fatalf("expected :colorscheme to save, got %q", data)
	}
	h.keys(":colo nosuch")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Cannot find color scheme") {
		t.Fatalf("expected error for unknown theme:\n%s", v)
	}
}

func TestEveryThemeRendersFullFrames(t *testing.T) {
	for _, name := range theme.Names() {
		h := newHarness(t, newTestStore(t), app.WithTheme(name))
		h.view()
		h.keys("n")
		h.view()
	}
}

func TestFinderSearchesTextAcrossAllFolders(t *testing.T) {
	store := newTestStore(t)
	body := "# Trip\n\nline two\npack the passport\n"
	if _, err := store.SaveAs("travel/2026/japan", "itinerary", body); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "inbox", "nothing here"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("f")
	h.keys("passport")
	v := h.view()
	if !strings.Contains(v, "travel/2026/japan/itinerary.md:4") {
		t.Fatalf("expected a text hit with path and line from a nested folder:\n%s", v)
	}
	h.key(tea.KeyEnter)
	v = h.view()
	if !strings.Contains(strings.Split(v, "\n")[0], "itinerary.md") || !strings.Contains(v, "Ln 4, Col 1") {
		t.Fatalf("expected itinerary.md open with the cursor on line 4:\n%s", v)
	}
}
