package preview

import (
	"math"

	"github.com/charmbracelet/x/ansi"
)

// layoutTD draws the ranked graph top to bottom.
func layoutTD(r *flowRank) *flowCanvas {
	nodes, segs, layers, preds, succs, loops := r.nodes, r.segs, r.layers, r.preds, r.succs, r.loops
	depth := len(layers)

	// Columns: place each layer under (or over) its neighbours, keeping the
	// order and the gaps.
	place := func(l int, nbrs [][]int) {
		right := math.MinInt / 2
		for _, id := range layers[l] {
			nd := nodes[id]
			want := nd.x
			if ns := nbrs[id]; len(ns) > 0 {
				sum := 0
				for _, o := range ns {
					sum += nodes[o].cx()
				}
				want = sum/len(ns) - nd.w/2
			}
			nd.x = max(want, right+flowGap)
			right = nd.x + nd.w
		}
	}
	for l := range layers {
		x := 0
		for _, id := range layers[l] {
			nodes[id].x = x
			x += nodes[id].w + flowGap
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
	minX, maxX := math.MaxInt, 0
	for _, nd := range nodes {
		minX = min(minX, nd.x)
	}
	for _, nd := range nodes {
		nd.x -= minX
		maxX = max(maxX, nd.x+nd.w)
	}

	// Rows: each layer is as tall as its tallest box; the gap below it
	// holds one row per horizontal track plus a row for the arrowheads.
	band := make([]int, depth)
	for _, nd := range nodes {
		band[nd.layer] = max(band[nd.layer], nd.h, 1)
	}
	type span struct{ a, b, from, to int }
	tracks := make([][][]span, depth)
	for i := range segs {
		s := &segs[i]
		a, b := nodes[s.from].cx(), nodes[s.to].cx()
		s.track = -1
		if a == b {
			continue
		}
		if a > b {
			a, b = b, a
		}
		l := nodes[s.from].layer
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
	top := make([]int, depth)
	for l := 1; l < depth; l++ {
		top[l] = top[l-1] + band[l-1] + 2 + len(tracks[l-1])
	}
	for _, nd := range nodes {
		nd.y = top[nd.layer]
	}
	height := top[depth-1] + band[depth-1]

	// Loops run up the right-hand side, one column each.
	labelW := 0
	for _, e := range loops {
		labelW = max(labelW, ansi.StringWidth(e.label))
	}
	width := maxX
	if len(loops) > 0 {
		width = maxX + 2*len(loops) + 1
		if labelW > 0 {
			width += labelW + 1
		}
	}

	cv := newFlowCanvas(width, height)
	for _, nd := range nodes {
		if !nd.dummy {
			drawFlowBox(cv, nd)
		}
	}

	for _, s := range segs {
		u, v := nodes[s.from], nodes[s.to]
		a, b := u.cx(), v.cx()
		start := u.y
		if !u.dummy {
			start = u.y + u.h
			if u.shape == shapeRect || u.shape == shapeSubroutine {
				cv.set(a, start-1, '┬', styleOf(u))
			}
		}
		end := v.y
		if !v.dummy {
			end = v.y - 1
		}
		if s.track < 0 {
			cv.vline(a, start, end, true, true, s.kind)
		} else {
			ty := u.y + band[u.layer] + 1 + s.track
			cv.vline(a, start, ty, true, false, s.kind)
			cv.hline(ty, a, b, s.kind)
			cv.vline(b, ty, end, false, true, s.kind)
		}
		if !v.dummy {
			cv.set(b, end, '▼', flowArrow)
		}
	}
	// Labels after all lines, so they sit on top.
	for _, s := range segs {
		if s.label == "" {
			continue
		}
		u, v := nodes[s.from], nodes[s.to]
		a, b := u.cx(), v.cx()
		lw := ansi.StringWidth(s.label)
		if s.track >= 0 && abs(b-a)-1 >= lw+2 {
			ty := u.y + band[u.layer] + 1 + s.track
			cv.text(min(a, b)+(abs(b-a)-lw)/2+1, ty, s.label, flowLabel)
			continue
		}
		row := v.y - 1
		if v.dummy {
			row = v.y
		}
		for _, x := range []int{b + 2, b - 1 - lw} {
			fits := true
			for i := range lw {
				if !cv.free(x+i, row) {
					fits = false
					break
				}
			}
			if fits {
				cv.text(x, row, s.label, flowLabel)
				break
			}
		}
	}

	for k, e := range loops {
		u, v := nodes[e.from], nodes[e.to]
		col := maxX + 1 + 2*k
		ru, rv := u.y+u.h/2, v.y+v.h/2
		if u.shape == shapeRect || u.shape == shapeRound || u.shape == shapeCylinder {
			cv.set(u.x+u.w-1, ru, '├', styleOf(u))
		}
		cv.hline(ru, u.x+u.w, col, e.kind)
		cv.bit(u.x+u.w, ru, dirLeft, e.kind)
		cv.vline(col, rv, ru, false, false, e.kind)
		cv.hline(rv, v.x+v.w, col, e.kind)
		cv.set(v.x+v.w, rv, '◀', flowArrow)
		if e.label != "" {
			cv.text(width-labelW, (ru+rv)/2, e.label, flowLabel)
		}
	}
	return cv
}
