package grailui

import (
	"fmt"
	"image"

	"charm.land/lipgloss/v2"
	"github.com/wesen/grail/pkg/graphmodel"
)

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
