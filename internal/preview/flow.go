package preview

import (
	"regexp"
	"strings"
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
