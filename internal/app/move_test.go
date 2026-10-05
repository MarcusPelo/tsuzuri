package app

import (
	"strings"
	"testing"
)

func TestMoveBlock(t *testing.T) {
	for _, c := range []struct {
		doc, want     string
		from, end, to int
	}{
		{"First\n\nSecond\n\nThird", "Third\n\nFirst\n\nSecond", 4, 5, 0},
		{"First\n\nSecond\n\nThird", "Second\n\nThird\n\nFirst", 0, 1, 5},
		{"- a\n- b\n- c", "- c\n- a\n- b", 2, 3, 0},
		{"# H\nPara\n\n- a", "# H\n\n- a\n\nPara", 3, 4, 1},
		{"A\n\nB", "A\n\nB", 0, 1, 1},
	} {
		if got := strings.Join(moveBlock(strings.Split(c.doc, "\n"), c.from, c.end, c.to), "\n"); got != c.want {
			t.Errorf("move %q [%d,%d)->%d = %q, want %q", c.doc, c.from, c.end, c.to, got, c.want)
		}
	}
}
