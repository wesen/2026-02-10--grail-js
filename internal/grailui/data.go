package grailui

import (
	"image"

	"github.com/wesen/grail/pkg/graphmodel"
)

// NodeTypeInfo describes the fixed geometry and display metadata for a node type.
type NodeTypeInfo struct {
	Label string // human-readable label (e.g. "Process")
	Tag   string // short tag in top border (e.g. "P", "?", "IO")
	W, H  int    // fixed width and height in terminal cells
}

// nodeTypeInfo maps node type strings to their geometry.
var nodeTypeInfo = map[string]NodeTypeInfo{
	"process":   {Label: "Process", Tag: "P", W: 22, H: 3},
	"decision":  {Label: "Decision", Tag: "?", W: 22, H: 3},
	"terminal":  {Label: "Terminal", Tag: "T", W: 22, H: 3},
	"io":        {Label: "I/O", Tag: "IO", W: 22, H: 3},
	"connector": {Label: "Connector", Tag: "", W: 7, H: 3},
}

// FlowNodeData is the concrete node type stored in the graph.
type FlowNodeData struct {
	Type string
	X, Y int
	Text string
	Code string
}

// Pos implements graphmodel.Spatial.
func (n FlowNodeData) Pos() image.Point { return image.Pt(n.X, n.Y) }

// Size implements graphmodel.Spatial.
func (n FlowNodeData) Size() image.Point {
	info := nodeTypeInfo[n.Type]
	return image.Pt(info.W, info.H)
}

// SetPos is the setter for graphmodel.MoveNode.
func SetPos(n *FlowNodeData, p image.Point) {
	n.X = p.X
	n.Y = p.Y
}

// FlowEdgeData holds optional edge metadata.
type FlowEdgeData struct {
	Label string
}

// FlowGraph is the concrete graph type for GRaIL.
type FlowGraph = graphmodel.Graph[FlowNodeData, FlowEdgeData]

// NewFlowGraph creates an empty flow graph.
func NewFlowGraph() *FlowGraph {
	return graphmodel.New[FlowNodeData, FlowEdgeData]()
}

// MakeInitialGraph creates the demo flowchart (sum 1..5).
func MakeInitialGraph() *FlowGraph {
	g := NewFlowGraph()

	start := g.AddNode(FlowNodeData{Type: "terminal", X: 5, Y: 1, Text: "START"})
	init := g.AddNode(FlowNodeData{Type: "process", X: 4, Y: 5, Text: "INIT", Code: "i = 1; sum = 0"})
	cond := g.AddNode(FlowNodeData{Type: "decision", X: 4, Y: 9, Text: "i <= 5?", Code: "i <= 5"})
	accum := g.AddNode(FlowNodeData{Type: "process", X: 4, Y: 17, Text: "ACCUMULATE", Code: "sum = sum + i; i = i + 1"})
	conn := g.AddNode(FlowNodeData{Type: "connector", X: 32, Y: 13, Text: ""})
	printN := g.AddNode(FlowNodeData{Type: "io", X: 44, Y: 9, Text: "PRINT SUM", Code: `print("Sum 1..5 = " + str(sum))`})
	end := g.AddNode(FlowNodeData{Type: "terminal", X: 46, Y: 14, Text: "END"})

	g.AddEdge(start, init, FlowEdgeData{})
	g.AddEdge(init, cond, FlowEdgeData{})
	g.AddEdge(cond, accum, FlowEdgeData{Label: "Y"})
	g.AddEdge(accum, conn, FlowEdgeData{})
	g.AddEdge(conn, cond, FlowEdgeData{})
	g.AddEdge(cond, printN, FlowEdgeData{Label: "N"})
	g.AddEdge(printN, end, FlowEdgeData{})

	return g
}
