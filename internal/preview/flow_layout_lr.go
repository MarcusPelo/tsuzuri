package preview

import (
	"math"

	"github.com/charmbracelet/x/ansi"
)

// layoutLR draws the ranked graph left to right: layers are columns, edges
// leave a box's right side and enter the next box's left side.
func layoutLR(r *flowRank) *flowCanvas {
	nodes, segs, layers, preds, succs, loops := r.nodes, r.segs, r.layers, r.preds, r.succs, r.loops
	depth := len(layers)
	cy := func(n *flowNode) int { return n.y + n.h/2 }

	colW := make([]int, depth)
	for _, nd := range nodes {
		if !nd.dummy {
			colW[nd.layer] = max(colW[nd.layer], nd.w)
		}
	}
	for l := range colW {
		colW[l] = max(colW[l], 1)
	}
	for _, nd := range nodes {
		if nd.dummy {
			nd.w, nd.h = colW[nd.layer], 1
		}
	}

	// Rows: stack each column beside its neighbours, keeping order and a
	// one-row gap.
	place := func(l int, nbrs [][]int) {
		bottom := math.MinInt / 2
		for _, id := range layers[l] {
			nd := nodes[id]
			want := nd.y
			if ns := nbrs[id]; len(ns) > 0 {
				sum := 0
				for _, o := range ns {
					sum += cy(nodes[o])
				}
				want = sum/len(ns) - nd.h/2
			}
			nd.y = max(want, bottom+1)
			bottom = nd.y + nd.h
		}
	}
	for l := range layers {
		y := 0
		for _, id := range layers[l] {
			nodes[id].y = y
			y += nodes[id].h + 1
		}
	}
	for l := 1; l < depth; l++ {
		place(l, preds)
	}
	for l := depth - 2; l >= 0; l-- {
		place(l, succs)
	}
	for l := 1; l < depth; l++ {
		place(l, preds)
	}
	minY, maxY := math.MaxInt, 0
	for _, nd := range nodes {
		minY = min(minY, nd.y)
	}
	for _, nd := range nodes {
		nd.y -= minY
		maxY = max(maxY, nd.y+nd.h)
	}

	// The gap after each column holds a stub, one column per vertical
	// track, room for labels, and the arrowheads.
	type span struct{ a, b, from, to int }
	tracks := make([][][]span, depth)
	labelW := make([]int, depth)
	for i := range segs {
		s := &segs[i]
		l := nodes[s.from].layer
		labelW[l] = max(labelW[l], ansi.StringWidth(s.label))
		a, b := cy(nodes[s.from]), cy(nodes[s.to])
		s.track = -1
		if a == b {
			continue
		}
		if a > b {
			a, b = b, a
		}
		t := 0
		for ; t < len(tracks[l]); t++ {
			ok := true
			for _, o := range tracks[l][t] {
				shared := o.from == s.from || o.to == s.to
				if !shared && !(b+1 < o.a || o.b+1 < a) {
					ok = false
					break
				}
			}
			if ok {
				break
			}
		}
		if t == len(tracks[l]) {
			tracks[l] = append(tracks[l], nil)
		}
		tracks[l][t] = append(tracks[l][t], span{a, b, s.from, s.to})
		s.track = t
	}
	colX := make([]int, depth)
	for l := 1; l < depth; l++ {
		room := 1
		if labelW[l-1] > 0 {
			room = labelW[l-1] + 2
		}
		colX[l] = colX[l-1] + colW[l-1] + 2 + len(tracks[l-1]) + room
	}
	for _, nd := range nodes {
		nd.x = colX[nd.layer]
		if !nd.dummy {
			nd.x += (colW[nd.layer] - nd.w) / 2
		}
	}
	width := colX[depth-1] + colW[depth-1]

	// Loops run back underneath, one row each.
	height := maxY
	for _, e := range loops {
		height++
		if lw := ansi.StringWidth(e.label); lw > 0 {
			width = max(width, max(nodes[e.from].cx(), nodes[e.to].cx())+2+lw)
		}
	}

	cv := newFlowCanvas(width, height)
	for _, nd := range nodes {
		if !nd.dummy {
			drawFlowBox(cv, nd)
		}
	}
	trackX := func(s flowSeg) int {
		u := nodes[s.from]
		return colX[u.layer] + colW[u.layer] + 1 + s.track
	}
	for _, s := range segs {
		u, v := nodes[s.from], nodes[s.to]
		a, b := cy(u), cy(v)
		start := u.x
		if !u.dummy {
			start = u.x + u.w
			if u.shape == shapeRect || u.shape == shapeRound || u.shape == shapeSubroutine || u.shape == shapeCylinder {
				cv.set(start-1, a, '├', styleOf(u))
			}
		}
		end := v.x
		if !v.dummy {
			end = v.x - 1
		}
		if s.track < 0 {
			cv.hline(a, start, end, s.kind)
		} else {
			tx := trackX(s)
			cv.hline(a, start, tx, s.kind)
			cv.vline(tx, a, b, false, false, s.kind)
			cv.hline(b, tx, end, s.kind)
		}
		if !v.dummy {
			cv.set(end, b, '▶', flowArrow)
		}
	}
	// Labels sit on the line just before the arrowhead, or above it where
	// lines meet.
	for _, s := range segs {
		if s.label == "" {
			continue
		}
		v := nodes[s.to]
		lw := ansi.StringWidth(s.label)
		end := v.x - 1
		if v.dummy {
			end = v.x
		}
		row := cy(v)
		onLine := func(x, y int) bool {
			return cv.in(x, y) && cv.cells[y][x].r == 0 && cv.cells[y][x].mask&(dirUp|dirDown) == 0
		}
		x0 := end - 1 - lw
		fits := x0 >= 0
		for i := 0; fits && i < lw; i++ {
			fits = onLine(x0+i, row)
		}
		if !fits {
			row--
			fits = x0 >= 0 && row >= 0
			for i := 0; fits && i < lw; i++ {
				fits = cv.free(x0+i, row)
			}
		}
		if fits {
			cv.text(x0, row, s.label, flowLabel)
		}
	}

	for k, e := range loops {
		u, v := nodes[e.from], nodes[e.to]
		row := maxY + k
		a, b := u.cx(), v.cx()
		if u.shape == shapeRect || u.shape == shapeSubroutine {
			cv.set(a, u.y+u.h-1, '┬', styleOf(u))
		}
		cv.vline(a, u.y+u.h, row, true, false, e.kind)
		cv.hline(row, b, a, e.kind)
		cv.vline(b, v.y+v.h, row, false, false, e.kind)
		cv.set(b, v.y+v.h, '▲', flowArrow)
		if e.label != "" {
			cv.text(a+2, row, e.label, flowLabel)
		}
	}
	return cv
}
