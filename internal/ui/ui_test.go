package ui_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"
	"github.com/muesli/termenv"
)

func TestPaintAppliesDefaultColours(t *testing.T) {
	r := lipgloss.NewRenderer(nil)
	r.SetColorProfile(termenv.TrueColor)
	lipgloss.SetDefaultRenderer(r)
	paint := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#000000"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("red")
	out := ui.Paint("plain "+red+" after\nline2", paint)
	for _, l := range strings.Split(out, "\n") {
		if !strings.HasPrefix(l, "\x1b[") || !strings.Contains(l, "48;2;0;0;0") {
			t.Fatalf("line not painted: %q", l)
		}
	}
	// After the red span resets, the background must come back.
	if !strings.Contains(out, "\x1b[0m\x1b[") {
		t.Fatalf("expected default colours restored after a reset: %q", out)
	}
}

func TestFitAndOverlayKeepWidth(t *testing.T) {
	plain := lipgloss.NewStyle()
	box := ui.Fit("abc\ndefghijkl", 5, 3, plain)
	lines := strings.Split(box, "\n")
	if len(lines) != 3 || lipgloss.Width(lines[1]) != 5 {
		t.Fatalf("Fit gave %q", box)
	}
	out := ui.Overlay("..........\n..........", "XX", 4, 1)
	if strings.Split(out, "\n")[1] != "....\x1b[0mXX\x1b[0m...." {
		t.Fatalf("Overlay gave %q", out)
	}
}
