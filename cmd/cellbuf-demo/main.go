// cellbuf-demo renders a sample flowchart to the terminal, exercising
// all 3 foundation packages: graphmodel → drawutil → cellbuf.
//
// Run: GOWORK=off go run ./cmd/cellbuf-demo/
package main

import (
	"fmt"
	"image"

	"charm.land/lipgloss/v2"
	"github.com/wesen/grail/pkg/cellbuf"
	"github.com/wesen/grail/pkg/drawutil"
	"github.com/wesen/grail/pkg/graphmodel"
)

// Style keys
const (
	BG       cellbuf.StyleKey = 0
	Grid     cellbuf.StyleKey = 1
	Edge     cellbuf.StyleKey = 2
	EdgeHot  cellbuf.StyleKey = 3
	NodeBox  cellbuf.StyleKey = 4
	NodeText cellbuf.StyleKey = 5
	Label    cellbuf.StyleKey = 6
)

// FlowNode is our concrete Spatial type for the graph.
type FlowNode struct {
	X, Y  int
	W, H  int
	Label string
	Kind  string // "terminal", "process", "decision"
}

func (n FlowNode) Pos() image.Point  { return image.Pt(n.X, n.Y) }
func (n FlowNode) Size() image.Point { return image.Pt(n.W, n.H) }

// FlowEdge holds an optional label for display.
type FlowEdge struct {
	Label string
}

func main() {
	styles := map[cellbuf.StyleKey]lipgloss.Style{
		BG:       lipgloss.NewStyle().Foreground(lipgloss.Color("#333333")).Background(lipgloss.Color("#0a0a0a")),
		Grid:     lipgloss.NewStyle().Foreground(lipgloss.Color("#1a3a1a")).Background(lipgloss.Color("#0a0a0a")),
		Edge:     lipgloss.NewStyle().Foreground(lipgloss.Color("#00d4a0")).Background(lipgloss.Color("#0a0a0a")),
		EdgeHot:  lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6600")).Background(lipgloss.Color("#0a0a0a")).Bold(true),
		NodeBox:  lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff88")).Background(lipgloss.Color("#0a1510")),
		NodeText: lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffcc")).Background(lipgloss.Color("#0a1510")).Bold(true),
		Label:    lipgloss.NewStyle().Foreground(lipgloss.Color("#ffcc00")).Background(lipgloss.Color("#0a0a0a")),
	}

	// ── Build graph (graphmodel) ──
	g := graphmodel.New[FlowNode, FlowEdge]()

	start := g.AddNode(FlowNode{X: 5, Y: 2, W: 12, H: 3, Label: "START", Kind: "terminal"})
	process := g.AddNode(FlowNode{X: 3, Y: 14, W: 16, H: 3, Label: "x = x + 1", Kind: "process"})
	decision := g.AddNode(FlowNode{X: 30, Y: 7, W: 14, H: 3, Label: "x > 10 ?", Kind: "decision"})
	end := g.AddNode(FlowNode{X: 32, Y: 21, W: 10, H: 3, Label: "END", Kind: "terminal"})

	g.AddEdge(start, decision, FlowEdge{})
	g.AddEdge(decision, process, FlowEdge{Label: "YES"})
	g.AddEdge(decision, end, FlowEdge{Label: "NO"})
	g.AddEdge(process, start, FlowEdge{})

	// ── Render into buffer (cellbuf + drawutil) ──
	buf := cellbuf.New(55, 26, BG)

	// Grid
	drawutil.DrawGrid(buf, 0, 0, 4, 2, Grid)

	// Draw edges first (behind nodes)
	for _, edge := range g.Edges() {
		fromNode := g.Node(edge.FromID)
		toNode := g.Node(edge.ToID)
		fromRect := graphmodel.BoundsOf(fromNode.Data)
		toRect := graphmodel.BoundsOf(toNode.Data)
		a := drawutil.EdgeExit(fromRect, graphmodel.CenterOf(toNode.Data))
		b := drawutil.EdgeExit(toRect, graphmodel.CenterOf(fromNode.Data))

		style := Edge
		if edge.Data.Label == "YES" {
			style = EdgeHot
		}
		drawutil.DrawArrowLine(buf, a.X, a.Y, b.X, b.Y, style, style)

		// Draw edge label at midpoint
		if edge.Data.Label != "" {
			mx := (a.X + b.X) / 2
			my := (a.Y + b.Y) / 2
			buf.SetString(mx, my-1, edge.Data.Label, Label)
		}
	}

	// Draw nodes on top
	borders := map[string][6]rune{
		"terminal": {'╭', '╮', '╰', '╯', '─', '│'},
		"process":  {'┌', '┐', '└', '┘', '─', '│'},
		"decision": {'◇', '◇', '◇', '◇', '─', '│'},
	}
	for _, node := range g.Nodes() {
		d := node.Data
		b := borders[d.Kind]
		drawBox(buf, d.X, d.Y, d.W, d.H, NodeBox, b[0], b[1], b[2], b[3], b[4], b[5])
		// Center the label
		pad := (d.W - 2 - len(d.Label)) / 2
		if pad < 0 {
			pad = 0
		}
		buf.SetString(d.X+1+pad, d.Y+1, d.Label, NodeText)
	}

	// ── Print ──
	fmt.Println()
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffcc")).Bold(true).Underline(true)
	fmt.Println(title.Render("  cellbuf + drawutil + graphmodel demo"))
	fmt.Println()
	fmt.Println(buf.Render(styles))
	fmt.Println()

	// Stats
	info := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	fmt.Println(info.Render(fmt.Sprintf("  Graph: %d nodes, %d edges | Buffer: %dx%d | HitTest(15,3): %s",
		len(g.Nodes()), len(g.Edges()), buf.W, buf.H,
		hitLabel(g, 15, 3))))
	fmt.Println(info.Render(fmt.Sprintf("  HitTest(35,8): %s | HitTest(0,0): %s",
		hitLabel(g, 35, 8), hitLabel(g, 0, 0))))
	fmt.Println()
}

func hitLabel(g *graphmodel.Graph[FlowNode, FlowEdge], x, y int) string {
	n := g.HitTest(image.Pt(x, y))
	if n == nil {
		return "(miss)"
	}
	return fmt.Sprintf("%q [id=%d]", n.Data.Label, n.ID)
}

func drawBox(buf *cellbuf.Buffer, x, y, w, h int, style cellbuf.StyleKey,
	tl, tr, bl, br, horiz, vert rune) {
	buf.Set(x, y, tl, style)
	for i := 1; i < w-1; i++ {
		buf.Set(x+i, y, horiz, style)
	}
	buf.Set(x+w-1, y, tr, style)
	buf.Set(x, y+h-1, bl, style)
	for i := 1; i < w-1; i++ {
		buf.Set(x+i, y+h-1, horiz, style)
	}
	buf.Set(x+w-1, y+h-1, br, style)
	for j := 1; j < h-1; j++ {
		buf.Set(x, y+j, vert, style)
		buf.Set(x+w-1, y+j, vert, style)
		for i := 1; i < w-1; i++ {
			buf.Set(x+i, y+j, ' ', style)
		}
	}
}
