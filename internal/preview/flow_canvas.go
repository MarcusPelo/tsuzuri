package preview

import ()

type flowCell struct {
	r     rune
	mask  uint8
	kind  flowEdgeKind
	style int
}

type flowCanvas struct {
	w, h  int
	cells [][]flowCell
}

func newFlowCanvas(w, h int) *flowCanvas {
	c := &flowCanvas{w: w, h: h, cells: make([][]flowCell, h)}
	for y := range c.cells {
		c.cells[y] = make([]flowCell, w)
	}
	return c
}

func (c *flowCanvas) in(x, y int) bool { return x >= 0 && y >= 0 && x < c.w && y < c.h }

func (c *flowCanvas) set(x, y int, r rune, style int) {
	if c.in(x, y) {
		c.cells[y][x] = flowCell{r: r, style: style}
	}
}

func (c *flowCanvas) free(x, y int) bool {
	return c.in(x, y) && c.cells[y][x].r == 0 && c.cells[y][x].mask == 0
}

// bit adds a line direction to a cell. Where lines of different kinds meet,
// the cell falls back to a solid line.
func (c *flowCanvas) bit(x, y int, d uint8, k flowEdgeKind) {
	if !c.in(x, y) {
		return
	}
	cell := &c.cells[y][x]
	if cell.mask == 0 {
		cell.kind = k
	} else if cell.kind != k {
		cell.kind = edgeSolid
	}
	cell.mask |= d
}

// vline joins (x, y0) to (x, y1); capTop/capBottom extend the line out of the
// ends so it meets a box or arrow instead of turning a corner.
func (c *flowCanvas) vline(x, y0, y1 int, capTop, capBottom bool, k flowEdgeKind) {
	if y0 > y1 {
		y0, y1 = y1, y0
		capTop, capBottom = capBottom, capTop
	}
	for y := y0; y < y1; y++ {
		c.bit(x, y, dirDown, k)
		c.bit(x, y+1, dirUp, k)
	}
	if capTop {
		c.bit(x, y0, dirUp, k)
	}
	if capBottom {
		c.bit(x, y1, dirDown, k)
	}
}

func (c *flowCanvas) hline(y, x0, x1 int, k flowEdgeKind) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	for x := x0; x < x1; x++ {
		c.bit(x, y, dirRight, k)
		c.bit(x+1, y, dirLeft, k)
	}
}

func (c *flowCanvas) text(x, y int, s string, style int) {
	for _, r := range s {
		c.set(x, y, r, style)
		x++
	}
}

func lineRune(m uint8, k flowEdgeKind) rune {
	switch m {
	case dirUp, dirDown, dirUp | dirDown:
		return [...]rune{'│', '┆', '┃'}[k]
	case dirLeft, dirRight, dirLeft | dirRight:
		return [...]rune{'─', '┄', '━'}[k]
	case dirDown | dirRight:
		return '╭'
	case dirDown | dirLeft:
		return '╮'
	case dirUp | dirRight:
		return '╰'
	case dirUp | dirLeft:
		return '╯'
	case dirUp | dirDown | dirRight:
		return '├'
	case dirUp | dirDown | dirLeft:
		return '┤'
	case dirDown | dirLeft | dirRight:
		return '┬'
	case dirUp | dirLeft | dirRight:
		return '┴'
	}
	return '┼'
}
