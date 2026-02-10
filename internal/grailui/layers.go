package grailui

import (
	"fmt"
	"image"
	"math"

	"charm.land/lipgloss/v2"
	"github.com/wesen/grail/pkg/cellbuf"
	"github.com/wesen/grail/pkg/drawutil"
	"github.com/wesen/grail/pkg/graphmodel"
)

// cellbuf style keys for the edge/grid background layer.
const (
	styleBG         cellbuf.StyleKey = 0
	styleGrid       cellbuf.StyleKey = 1
	styleEdge       cellbuf.StyleKey = 2
	styleEdgeActive cellbuf.StyleKey = 3
)

// bufStyles maps cellbuf StyleKeys to lipgloss styles for rendering.
var bufStyles = map[cellbuf.StyleKey]lipgloss.Style{
	styleBG:         lipgloss.NewStyle().Foreground(c("#1a3a2a")).Background(c("#080e0b")),
	styleGrid:       lipgloss.NewStyle().Foreground(c("#0e2e20")).Background(c("#080e0b")),
	styleEdge:       lipgloss.NewStyle().Foreground(c("#00d4a0")).Background(c("#080e0b")),
	styleEdgeActive: lipgloss.NewStyle().Foreground(c("#ffcc00")).Background(c("#080e0b")).Bold(true),
}

// buildEdgeCanvasLayer renders the grid + edge lines into a cellbuf and
// returns it as a single background Layer at Z=0.
func buildEdgeCanvasLayer(g *FlowGraph, camX, camY int, viewport image.Rectangle,
	execID *int) *lipgloss.Layer {

	w := viewport.Dx()
	h := viewport.Dy()
	if w <= 0 || h <= 0 {
		return lipgloss.NewLayer("").X(viewport.Min.X).Y(viewport.Min.Y).Z(0)
	}

	buf := cellbuf.New(w, h, styleBG)

	// Grid dots
	drawutil.DrawGrid(buf, camX, camY, 5, 3, styleGrid)

	// Edge lines
	for _, edge := range g.Edges() {
		fromNode := g.Node(edge.FromID)
		toNode := g.Node(edge.ToID)
		if fromNode == nil || toNode == nil {
			continue
		}

		fromBounds := graphmodel.BoundsOf(fromNode.Data)
		toBounds := graphmodel.BoundsOf(toNode.Data)
		fromCenter := graphmodel.CenterOf(toNode.Data)
		toCenter := graphmodel.CenterOf(fromNode.Data)

		p1 := drawutil.EdgeExit(fromBounds, fromCenter)
		p2 := drawutil.EdgeExit(toBounds, toCenter)

		// World → buffer coords
		bx1 := p1.X - camX
		by1 := p1.Y - camY
		bx2 := p2.X - camX
		by2 := p2.Y - camY

		// Style: active if executing node is the destination
		es := styleEdge
		if execID != nil && edge.ToID == *execID {
			es = styleEdgeActive
		}

		drawutil.DrawArrowLine(buf, bx1, by1, bx2, by2, es, es)
	}

	rendered := buf.Render(bufStyles)
	return lipgloss.NewLayer(rendered).X(viewport.Min.X).Y(viewport.Min.Y).Z(0).ID("edge-canvas")
}

// buildEdgeLabelLayers creates a Layer for each edge that has a label,
// positioned at the edge midpoint.
func buildEdgeLabelLayers(g *FlowGraph, camX, camY int, viewport image.Rectangle) []*lipgloss.Layer {
	labelStyle := lipgloss.NewStyle().
		Foreground(c("#00ffc8")).
		Background(c("#080e0b")).
		Bold(true)

	var layers []*lipgloss.Layer

	for _, edge := range g.Edges() {
		if edge.Data.Label == "" {
			continue
		}
		fromNode := g.Node(edge.FromID)
		toNode := g.Node(edge.ToID)
		if fromNode == nil || toNode == nil {
			continue
		}

		fromBounds := graphmodel.BoundsOf(fromNode.Data)
		toBounds := graphmodel.BoundsOf(toNode.Data)
		p1 := drawutil.EdgeExit(fromBounds, graphmodel.CenterOf(toNode.Data))
		p2 := drawutil.EdgeExit(toBounds, graphmodel.CenterOf(fromNode.Data))

		// Midpoint in screen coords
		mx := (p1.X+p2.X)/2 - camX + viewport.Min.X
		my := (p1.Y+p2.Y)/2 - camY + viewport.Min.Y

		// Offset: above if mostly horizontal, right if mostly vertical
		dx := math.Abs(float64(p2.X - p1.X))
		dy := math.Abs(float64(p2.Y - p1.Y))
		if dx >= dy {
			mx -= len(edge.Data.Label) / 2
			my -= 1
		} else {
			mx += 1
		}

		rendered := labelStyle.Render(edge.Data.Label)
		layer := lipgloss.NewLayer(rendered).
			X(mx).Y(my).Z(3).
			ID(fmt.Sprintf("elbl-%d-%d", edge.FromID, edge.ToID))
		layers = append(layers, layer)
	}

	return layers
}

// buildNodeLayers creates a Layer for each visible node.
// screenX = node.X - camX, screenY = node.Y - camY + offsetY.
func buildNodeLayers(g *FlowGraph, camX, camY int, viewport image.Rectangle,
	selectedID, execID *int) []*lipgloss.Layer {

	var layers []*lipgloss.Layer

	for _, node := range g.Nodes() {
		d := node.Data
		info := nodeTypeInfo[d.Type]

		// Screen position
		sx := d.X - camX + viewport.Min.X
		sy := d.Y - camY + viewport.Min.Y

		// Visibility culling
		nodeRect := image.Rect(sx, sy, sx+info.W, sy+info.H)
		if !nodeRect.Overlaps(viewport) {
			continue
		}

		// Pick colors
		bc, tc, bg := nodeColors[d.Type].border, nodeColors[d.Type].text, colorBG
		if selectedID != nil && node.ID == *selectedID {
			bc, tc, bg = selBorder, selText, selBG
		}
		if execID != nil && node.ID == *execID {
			bc, tc, bg = execBorder, execText, execBG
		}

		// Build the styled box
		border := borderForType(d.Type)
		boxStyle := lipgloss.NewStyle().
			Border(border).
			BorderForeground(bc).
			Background(bg).
			Width(info.W - 2).   // inner width (minus border columns)
			AlignHorizontal(lipgloss.Center)

		// Node label (truncate if too long)
		label := d.Text
		maxLen := info.W - 4
		if maxLen < 0 {
			maxLen = 0
		}
		if len(label) > maxLen {
			label = label[:maxLen]
		}

		textStyle := lipgloss.NewStyle().
			Foreground(tc).
			Background(bg).
			Bold(true)

		content := textStyle.Render(label)

		// Add type tag in border if present
		rendered := boxStyle.Render(content)

		// Tag overlay (e.g. [P], [?], [IO]) — rendered separately above the box
		if info.Tag != "" {
			tag := lipgloss.NewStyle().
				Foreground(bc).
				Background(bg).
				Render(fmt.Sprintf("[%s]", info.Tag))
			tagLayer := lipgloss.NewLayer(tag).
				X(sx + 2).Y(sy).Z(3).
				ID(fmt.Sprintf("tag-%d", node.ID))
			layers = append(layers, tagLayer)
		}

		layer := lipgloss.NewLayer(rendered).
			X(sx).Y(sy).Z(2).
			ID(fmt.Sprintf("node-%d", node.ID))
		layers = append(layers, layer)
	}

	return layers
}

// nodeCenter returns the screen-space center of a node.
func nodeCenter(d FlowNodeData, camX, camY int, viewport image.Rectangle) image.Point {
	c := graphmodel.CenterOf(d)
	return image.Pt(
		c.X-camX+viewport.Min.X,
		c.Y-camY+viewport.Min.Y,
	)
}
