package grailui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/wesen/grail/pkg/tealayout"
)

var (
	tbStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#0a1510")).
		Foreground(toolbarColor).
		Bold(true)

	ftStyle = lipgloss.NewStyle().
		Foreground(footerColor)

	bgStyle = lipgloss.NewStyle().
		Background(colorBG)
)

// toolNames maps Tool to display name.
var toolNames = map[Tool]string{
	ToolSelect:  "SELECT",
	ToolAdd:     "ADD",
	ToolConnect: "CONNECT",
}

// View implements tea.Model.
func (m Model) View() tea.View {
	if m.Width == 0 || m.Height == 0 {
		return tea.NewView("")
	}

	// Layout: toolbar(1) + canvas(remaining)  + footer(1)
	layout := tealayout.NewLayoutBuilder(m.Width, m.Height).
		TopFixed("toolbar", 1).
		BottomFixed("footer", 1).
		Remaining("canvas").
		Build()

	canvasRegion := layout.Get("canvas")

	// Layers
	var layers []*lipgloss.Layer

	// Background
	layers = append(layers,
		tealayout.FillLayer(layout.Get("toolbar"), tbStyle, "toolbar-bg", 0),
		tealayout.FillLayer(canvasRegion, bgStyle, "canvas-bg", 0),
		tealayout.FillLayer(layout.Get("footer"), ftStyle, "footer-bg", 0),
	)

	// Toolbar content
	toolStr := toolNames[m.CurrentTool]
	tbContent := fmt.Sprintf(
		" GRaIL  │  [s]elect [a]dd [c]onnect  │  Tool: %s  │  ←↑↓→ pan  │  [q]uit",
		toolStr,
	)
	layers = append(layers,
		tealayout.ToolbarLayer(tbContent, m.Width, tbStyle),
	)

	// Footer content
	selStr := "none"
	if m.SelectedID != nil {
		n := m.Graph.Node(*m.SelectedID)
		if n != nil {
			selStr = fmt.Sprintf("%d:%s", n.ID, n.Data.Text)
		}
	}
	ftContent := fmt.Sprintf(
		" Mouse: (%d,%d)  Cam: (%d,%d)  Sel: %s  Nodes: %d",
		m.MouseX, m.MouseY, m.CamX, m.CamY, selStr, len(m.Graph.Nodes()),
	)
	layers = append(layers,
		tealayout.FooterLayer(ftContent, m.Width, m.Height-1, ftStyle),
	)

	// Node layers
	nodeLayers := buildNodeLayers(m.Graph, m.CamX, m.CamY, canvasRegion.Rect, m.SelectedID, m.ExecID)
	layers = append(layers, nodeLayers...)

	// Compose
	comp := lipgloss.NewCompositor(layers...)
	canvas := lipgloss.NewCanvas(m.Width, m.Height)
	canvas.Compose(comp)

	v := tea.NewView(canvas.Render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}
