package preview

import (
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

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
