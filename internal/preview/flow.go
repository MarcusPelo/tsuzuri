package preview

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
)

// ---------------------------------------------------------------------------
// Flowchart
//
// A ```flow block (or a ```mermaid block that starts with "graph" or
// "flowchart") is drawn top to bottom as boxes joined by arrows. The syntax is
// the Mermaid flowchart subset:
//
//	start([Start]) --> check{Logged in?}
//	check -->|yes| home[Home]
//	check -- no --> login(Login)
//	login --> check
//
// Nodes are laid out in layers (longest path), ordered to reduce crossings,
// and edges are routed through the gaps between layers. Edges that point back
// up (loops) run along the right-hand side.

type flowShape int

const (
	shapeRect       flowShape = iota // [text]
	shapeRound                       // (text)
	shapeStadium                     // ([text])
	shapeDecision                    // {text}
	shapeHexagon                     // {{text}}
	shapeCircle                      // ((text))
	shapeCylinder                    // [(text)]
	shapeSubroutine                  // [[text]]
)

// flowEdgeKind is how an edge's line is drawn.
type flowEdgeKind uint8

const (
	edgeSolid  flowEdgeKind = iota // -->
	edgeDotted                     // -.->
	edgeThick                      // ==>
)

type flowNode struct {
	id, text string
	shape    flowShape
	dummy    bool // a bend point of an edge that spans several layers
	layer    int
	lines    []string
	x, y     int
	w, h     int
}

func (n *flowNode) cx() int { return n.x + n.w/2 }

type flowEdge struct {
	from, to int
	label    string
	kind     flowEdgeKind
}

type flowGraph struct {
	nodes []*flowNode
	index map[string]int
	edges []flowEdge
	lr    bool // "graph LR": layers run left to right
}

var (
	flowNodeRe = regexp.MustCompile(`^\s*([\p{L}\p{N}_]+)\s*(\(\[[^\]]*\]\)|\(\([^)]*\)\)|\[\([^)]*\)\]|\[\[[^\]]*\]\]|\{\{[^}]*\}\}|\[[^\]]*\]|\([^)]*\)|\{[^}]*\}|>[^\]]*\])?`)
	flowEdgeRe = regexp.MustCompile(`^\s*(?:(?:-->|---|-\.->|-\.-|==>|===|->)\s*(?:\|([^|]*)\|)?|--\s*([^-|>][^>]*?)\s*-->|==\s*([^=>][^>]*?)\s*==>)`)
)

// isMermaidFlow reports whether a ```mermaid block is a flowchart.
func isMermaidFlow(body []string) bool {
	for _, l := range body {
		t := strings.ToLower(strings.TrimSpace(l))
		if t == "" || strings.HasPrefix(t, "%%") {
			continue
		}
		return strings.HasPrefix(t, "graph") || strings.HasPrefix(t, "flowchart")
	}
	return false
}

func parseFlow(body []string) *flowGraph {
	g := &flowGraph{index: map[string]int{}}
	for _, raw := range body {
		line := raw
		if i := strings.Index(line, "%%"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		low := strings.ToLower(line)
		if f := strings.Fields(low); len(f) > 1 && (f[0] == "graph" || f[0] == "flowchart" || f[0] == "direction") {
			g.lr = f[1] == "lr" || f[1] == "rl"
		}
		if line == "" || flowIgnored(low) {
			continue
		}
		for _, stmt := range strings.Split(line, ";") {
			g.statement(stmt)
		}
	}
	return g
}

// flowIgnored reports Mermaid lines that don't add nodes or edges.
func flowIgnored(l string) bool {
	if l == "end" {
		return true
	}
	for _, p := range []string{"graph", "flowchart", "subgraph", "direction ", "classdef ", "class ", "style ", "linkstyle ", "click "} {
		if strings.HasPrefix(l, p) {
			return true
		}
	}
	return false
}

// statement parses one "a --> b --> c" chain.
func (g *flowGraph) statement(s string) {
	prev, rest, ok := g.parseNode(s)
	if !ok {
		return
	}
	for {
		m := flowEdgeRe.FindStringSubmatch(rest)
		if m == nil {
			return
		}
		label := strings.Trim(strings.TrimSpace(m[1]+m[2]+m[3]), `"`)
		kind := edgeSolid
		switch arrow := strings.TrimSpace(m[0]); {
		case strings.HasPrefix(arrow, "-."):
			kind = edgeDotted
		case strings.HasPrefix(arrow, "=="):
			kind = edgeThick
		}
		next, r, ok := g.parseNode(rest[len(m[0]):])
		if !ok {
			return
		}
		g.edges = append(g.edges, flowEdge{from: prev, to: next, label: label, kind: kind})
		prev, rest = next, r
	}
}

func (g *flowGraph) parseNode(s string) (int, string, bool) {
	m := flowNodeRe.FindStringSubmatch(s)
	if m == nil {
		return 0, s, false
	}
	id, shape, label := m[1], shapeRect, m[1]
	if t := m[2]; t != "" {
		switch {
		case strings.HasPrefix(t, "(["):
			shape, label = shapeStadium, t[2:len(t)-2]
		case strings.HasPrefix(t, "(("):
			shape, label = shapeCircle, t[2:len(t)-2]
		case strings.HasPrefix(t, "[("):
			shape, label = shapeCylinder, t[2:len(t)-2]
		case strings.HasPrefix(t, "[["):
			shape, label = shapeSubroutine, t[2:len(t)-2]
		case strings.HasPrefix(t, "{{"):
			shape, label = shapeHexagon, t[2:len(t)-2]
		case t[0] == '(':
			shape, label = shapeRound, t[1:len(t)-1]
		case t[0] == '{':
			shape, label = shapeDecision, t[1:len(t)-1]
		default: // [text], [/text/], >text]
			label = strings.Trim(t[1:len(t)-1], `/\`)
		}
		label = strings.Trim(strings.TrimSpace(label), `"`)
	}
	i, ok := g.index[id]
	switch {
	case !ok:
		g.nodes = append(g.nodes, &flowNode{id: id, text: label, shape: shape})
		i = len(g.nodes) - 1
		g.index[id] = i
	case m[2] != "":
		g.nodes[i].text, g.nodes[i].shape = label, shape
	}
	return i, s[len(m[0]):], true
}

// Canvas cells: a rune drawn as is, or line directions merged into a
// box-drawing character.
const (
	dirUp uint8 = 1 << iota
	dirDown
	dirLeft
	dirRight
)

const (
	flowLine = iota
	flowArrow
	flowLabel
	flowText
	flowRect
	flowRound
	flowDecision
	flowHexagon
	flowCircle
	flowCylinder
)

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

// flowSeg is one hop of an edge between neighbouring layers.
type flowSeg struct {
	from, to int
	label    string
	kind     flowEdgeKind
	track    int // row in the gap below the source layer; -1 = straight down
}

const flowGap = 3 // columns between neighbouring boxes

// flowRank is a graph split into layers, ready to be placed.
type flowRank struct {
	nodes        []*flowNode // real nodes, then dummies
	segs         []flowSeg
	layers       [][]int
	preds, succs [][]int
	loops        []flowEdge // edges that point back up a layer
}

// rankFlow layers and orders the graph, with box text wrapped at wrapW.
func rankFlow(g *flowGraph, wrapW int) *flowRank {
	n := len(g.nodes)
	nodes := make([]*flowNode, n)
	for i, nd := range g.nodes {
		cp := *nd
		nodes[i] = &cp
	}

	// Edges that close a loop (found by DFS) are drawn separately, so the
	// rest form a DAG.
	out := make([][]int, n)
	for i, e := range g.edges {
		out[e.from] = append(out[e.from], i)
	}
	state := make([]int, n)
	back := make([]bool, len(g.edges))
	var post []int
	var visit func(u int)
	visit = func(u int) {
		state[u] = 1
		for _, ei := range out[u] {
			switch v := g.edges[ei].to; state[v] {
			case 0:
				visit(v)
			case 1:
				back[ei] = true
			}
		}
		state[u] = 2
		post = append(post, u)
	}
	for u := range n {
		if state[u] == 0 {
			visit(u)
		}
	}

	// Longest-path layering in topological (reverse post) order.
	for i := len(post) - 1; i >= 0; i-- {
		u := post[i]
		for _, ei := range out[u] {
			if !back[ei] {
				v := g.edges[ei].to
				nodes[v].layer = max(nodes[v].layer, nodes[u].layer+1)
			}
		}
	}

	// Split long edges into one-layer hops through dummy nodes.
	var segs []flowSeg
	for i, e := range g.edges {
		if back[i] {
			continue
		}
		u, label := e.from, e.label
		for l := nodes[e.from].layer + 1; l < nodes[e.to].layer; l++ {
			nodes = append(nodes, &flowNode{dummy: true, layer: l})
			d := len(nodes) - 1
			segs = append(segs, flowSeg{from: u, to: d, label: label, kind: e.kind})
			u, label = d, ""
		}
		segs = append(segs, flowSeg{from: u, to: e.to, label: label, kind: e.kind})
	}

	depth := 0
	for _, nd := range nodes {
		depth = max(depth, nd.layer+1)
	}
	layers := make([][]int, depth)
	for i, nd := range nodes {
		layers[nd.layer] = append(layers[nd.layer], i)
	}
	preds := make([][]int, len(nodes))
	succs := make([][]int, len(nodes))
	for _, s := range segs {
		preds[s.to] = append(preds[s.to], s.from)
		succs[s.from] = append(succs[s.from], s.to)
	}

	// Barycenter sweeps to reduce crossings.
	order := make([]float64, len(nodes))
	renumber := func(l int) {
		for i, id := range layers[l] {
			order[id] = float64(i)
		}
	}
	for l := range layers {
		renumber(l)
	}
	sortBy := func(l int, nbrs [][]int) {
		key := map[int]float64{}
		for _, id := range layers[l] {
			key[id] = order[id]
			if ns := nbrs[id]; len(ns) > 0 {
				sum := 0.0
				for _, o := range ns {
					sum += order[o]
				}
				key[id] = sum / float64(len(ns))
			}
		}
		sort.SliceStable(layers[l], func(a, b int) bool { return key[layers[l][a]] < key[layers[l][b]] })
		renumber(l)
	}
	for range 4 {
		for l := 1; l < depth; l++ {
			sortBy(l, preds)
		}
		for l := depth - 2; l >= 0; l-- {
			sortBy(l, succs)
		}
	}

	// Box sizes.
	for _, nd := range nodes {
		if nd.dummy {
			nd.w = 1
			continue
		}
		nd.lines = strings.Split(ansi.Wordwrap(nd.text, wrapW, ""), "\n")
		tw := 0
		for i, l := range nd.lines {
			if ansi.StringWidth(l) > wrapW {
				l = ansi.Truncate(l, wrapW, "…")
				nd.lines[i] = l
			}
			tw = max(tw, ansi.StringWidth(l))
		}
		nd.w, nd.h = flowBoxSize(nd.shape, tw, len(nd.lines))
	}

	var loops []flowEdge
	for i, e := range g.edges {
		if back[i] && e.from != e.to {
			loops = append(loops, e)
		}
	}
	return &flowRank{nodes: nodes, segs: segs, layers: layers, preds: preds, succs: succs, loops: loops}
}

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

func styleOf(n *flowNode) int {
	switch n.shape {
	case shapeRound, shapeStadium:
		return flowRound
	case shapeDecision:
		return flowDecision
	case shapeHexagon:
		return flowHexagon
	case shapeCircle:
		return flowCircle
	case shapeCylinder:
		return flowCylinder
	}
	return flowRect
}

// flowBoxSize is the outer size of a shape holding n lines of text tw wide.
func flowBoxSize(shape flowShape, tw, n int) (w, h int) {
	switch shape {
	case shapeStadium, shapeDecision, shapeHexagon:
		return tw + 6, n + 2
	case shapeCircle:
		return tw + 6, n + 4
	case shapeCylinder:
		return tw + 4, n + 3
	case shapeSubroutine:
		return tw + 8, n + 2
	}
	return tw + 4, n + 2
}

// drawFlowBox draws a node's outline and centres its text inside.
//
//	┌──────┐  ╭──────╮   ╭──────╮   ╱──────╲   ╭──────╮  ┌─┬──────┬─┐
//	│ rect │  │ round│  ( stadium)  < choice >  │╰────╯│  │ │ sub  │ │
//	└──────┘  ╰──────╯   ╰──────╯   ╲──────╱   │ db   │  └─┴──────┴─┘
//	                                           ╰──────╯
func drawFlowBox(cv *flowCanvas, n *flowNode) {
	st := styleOf(n)
	x0, y0, x1, y1 := n.x, n.y, n.x+n.w-1, n.y+n.h-1
	row := func(y, a, b int, left, fill, right rune) {
		cv.set(a, y, left, st)
		for x := a + 1; x < b; x++ {
			cv.set(x, y, fill, st)
		}
		cv.set(b, y, right, st)
	}
	clear := func(y, a, b int) {
		for x := a; x <= b; x++ {
			cv.set(x, y, ' ', flowText)
		}
	}
	// sides picks the left/right edge of text row i of rows for the slanted
	// shapes: a point for one row, a wedge for more.
	sides := func(i, rows int, one [2]rune) (rune, rune) {
		switch {
		case rows == 1:
			return one[0], one[1]
		case i == 0:
			return '╱', '╲'
		case i == rows-1:
			return '╲', '╱'
		}
		return '│', '│'
	}

	textTop, rows := y0+1, len(n.lines)
	switch n.shape {
	case shapeStadium, shapeDecision, shapeHexagon:
		top, bottom := [2]rune{'╭', '╮'}, [2]rune{'╰', '╯'}
		one := [2]rune{'(', ')'}
		if n.shape != shapeStadium {
			top, bottom, one = [2]rune{'╱', '╲'}, [2]rune{'╲', '╱'}, [2]rune{'<', '>'}
		}
		row(y0, x0+1, x1-1, top[0], '─', top[1])
		row(y1, x0+1, x1-1, bottom[0], '─', bottom[1])
		for i := range rows {
			l, r := sides(i, rows, one)
			clear(y0+1+i, x0+1, x1-1)
			cv.set(x0, y0+1+i, l, st)
			cv.set(x1, y0+1+i, r, st)
		}
	case shapeCircle:
		row(y0, x0+2, x1-2, '╭', '─', '╮')
		row(y1, x0+2, x1-2, '╰', '─', '╯')
		clear(y0+1, x0+2, x1-2)
		clear(y1-1, x0+2, x1-2)
		cv.set(x0+1, y0+1, '╱', st)
		cv.set(x1-1, y0+1, '╲', st)
		cv.set(x0+1, y1-1, '╲', st)
		cv.set(x1-1, y1-1, '╱', st)
		for i := range rows {
			clear(y0+2+i, x0+1, x1-1)
			cv.set(x0, y0+2+i, '│', st)
			cv.set(x1, y0+2+i, '│', st)
		}
		textTop = y0 + 2
	case shapeCylinder:
		row(y0, x0, x1, '╭', '─', '╮')
		row(y0+1, x0+1, x1-1, '╰', '─', '╯')
		cv.set(x0, y0+1, '│', st)
		cv.set(x1, y0+1, '│', st)
		for i := range rows {
			clear(y0+2+i, x0+1, x1-1)
			cv.set(x0, y0+2+i, '│', st)
			cv.set(x1, y0+2+i, '│', st)
		}
		row(y1, x0, x1, '╰', '─', '╯')
		textTop = y0 + 2
	case shapeSubroutine:
		row(y0, x0, x1, '┌', '─', '┐')
		row(y1, x0, x1, '└', '─', '┘')
		cv.set(x0+2, y0, '┬', st)
		cv.set(x1-2, y0, '┬', st)
		cv.set(x0+2, y1, '┴', st)
		cv.set(x1-2, y1, '┴', st)
		for i := range rows {
			clear(y0+1+i, x0+1, x1-1)
			for _, x := range []int{x0, x0 + 2, x1 - 2, x1} {
				cv.set(x, y0+1+i, '│', st)
			}
		}
	default:
		corners := [4]rune{'┌', '┐', '└', '┘'}
		if n.shape == shapeRound {
			corners = [4]rune{'╭', '╮', '╰', '╯'}
		}
		row(y0, x0, x1, corners[0], '─', corners[1])
		row(y1, x0, x1, corners[2], '─', corners[3])
		for i := range rows {
			clear(y0+1+i, x0+1, x1-1)
			cv.set(x0, y0+1+i, '│', st)
			cv.set(x1, y0+1+i, '│', st)
		}
	}

	for i, l := range n.lines {
		lw := ansi.StringWidth(l)
		cv.text(x0+(n.w-lw)/2, textTop+i, l, flowText)
	}
}

func renderFlow(body []string, th theme.Theme, width int, _ *[]Hit) []string {
	g := parseFlow(body)
	if len(g.nodes) == 0 {
		return []string{lipgloss.NewStyle().Foreground(th.GreyFg).Render("Empty flowchart: write lines like  a[Start] --> b[Next]")}
	}
	// Shrink the boxes until the diagram fits; a left-to-right diagram
	// that still doesn't fit is drawn top to bottom instead.
	var cv *flowCanvas
	for _, lr := range []bool{g.lr, false} {
		for _, wrapW := range []int{20, 14, 10, 6} {
			r := rankFlow(g, wrapW)
			if lr {
				cv = layoutLR(r)
			} else {
				cv = layoutTD(r)
			}
			if cv.w <= width {
				break
			}
		}
		if cv.w <= width || !lr {
			break
		}
	}

	styles := map[int]lipgloss.Style{
		flowLine:     lipgloss.NewStyle().Foreground(th.GreyFg2),
		flowArrow:    lipgloss.NewStyle().Foreground(th.GreyFg2),
		flowLabel:    lipgloss.NewStyle().Foreground(th.Cyan).Italic(true),
		flowText:     lipgloss.NewStyle().Foreground(th.Fg),
		flowRect:     lipgloss.NewStyle().Foreground(th.Blue),
		flowRound:    lipgloss.NewStyle().Foreground(th.Green),
		flowDecision: lipgloss.NewStyle().Foreground(th.Yellow),
		flowHexagon:  lipgloss.NewStyle().Foreground(th.Orange),
		flowCircle:   lipgloss.NewStyle().Foreground(th.Purple),
		flowCylinder: lipgloss.NewStyle().Foreground(th.Cyan),
	}
	lines := make([]string, cv.h)
	for y, row := range cv.cells {
		var b strings.Builder
		var run []rune
		cur := -1
		flush := func() {
			if len(run) > 0 {
				if cur < 0 {
					b.WriteString(string(run))
				} else {
					b.WriteString(styles[cur].Render(string(run)))
				}
			}
			run = run[:0]
		}
		end := len(row)
		for end > 0 && row[end-1].r == 0 && row[end-1].mask == 0 {
			end--
		}
		for _, c := range row[:end] {
			r, st := c.r, c.style
			switch {
			case r == 0 && c.mask != 0:
				r, st = lineRune(c.mask, c.kind), flowLine
			case r == 0:
				r, st = ' ', -1
			}
			if st != cur {
				flush()
				cur = st
			}
			run = append(run, r)
		}
		flush()
		lines[y] = b.String()
	}
	return lines
}
