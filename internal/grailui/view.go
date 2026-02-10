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

	// Layout: toolbar(1) + footer(1) + panel(panelWidth) + canvas(remaining)
	layout := tealayout.NewLayoutBuilder(m.Width, m.Height).
		TopFixed("toolbar", 1).
		BottomFixed("footer", 1).
		RightFixed("panel", panelWidth).
		Remaining("canvas").
		Build()

	canvasRegion := layout.Get("canvas")
	panelRegion := layout.Get("panel")

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
	if m.CurrentTool == ToolAdd {
		toolStr = fmt.Sprintf("ADD [%s]", m.AddNodeType)
	}
	if m.ConnectFromID != nil {
		toolStr = fmt.Sprintf("CONNECT from #%d → click target", *m.ConnectFromID)
	}

	// Run state indicator
	runState := ""
	if m.Running {
		if m.AutoRunning {
			runState = " │ ▶ AUTO"
		} else if m.InputMode {
			runState = " │ ⌨ INPUT"
		} else if m.Interp != nil && m.Interp.Done {
			runState = " │ ✓ DONE"
		} else {
			runState = " │ ⏸ READY"
		}
		runState += " [n]step [g]go [p]ause [x]stop"
	}

	tbContent := fmt.Sprintf(
		" GRaIL  │  [s]elect [a]dd [c]onnect  │  %s%s  │  [q]uit",
		toolStr, runState,
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

	// Edge canvas layer (grid + edge lines + connect preview at Z=0)
	layers = append(layers,
		buildEdgeCanvasLayer(m.Graph, m.CamX, m.CamY, canvasRegion.Rect,
			m.ExecID, m.ConnectFromID, m.MouseX, m.MouseY),
	)

	// Node layers (Z=2, on top of edges)
	nodeLayers := buildNodeLayers(m.Graph, m.CamX, m.CamY, canvasRegion.Rect, m.SelectedID, m.ExecID)
	layers = append(layers, nodeLayers...)

	// Edge labels (Z=3, on top of nodes)
	labelLayers := buildEdgeLabelLayers(m.Graph, m.CamX, m.CamY, canvasRegion.Rect)
	layers = append(layers, labelLayers...)

	// Side panel
	pr := panelRegion.Rect
	pw := pr.Dx()
	ph := pr.Dy()
	if pw > 0 && ph > 0 {
		varsH := 6
		helpH := 8
		consoleH := ph - varsH - helpH
		if consoleH < 3 {
			consoleH = 3
		}

		// Separator
		layers = append(layers, buildSeparatorLayer(pr.Min.X-1, pr.Min.Y, ph))

		// Panel background
		layers = append(layers, tealayout.FillLayer(panelRegion, bgStyle, "panel-bg", 0))

		// Variables (live from interpreter)
		var vars map[string]any
		if m.Interp != nil {
			vars = m.Interp.Vars
		}
		layers = append(layers, buildVarsPanelLayer(vars, pr.Min.X+1, pr.Min.Y, pw-2, varsH))

		// Console (live from interpreter)
		var output []string
		if m.Interp != nil {
			output = m.Interp.Output
		}
		layers = append(layers, buildConsolePanelLayer(output, pr.Min.X+1, pr.Min.Y+varsH, pw-2, consoleH))

		// Input overlay (when waiting for input)
		if m.InputMode && m.Interp != nil {
			prompt := m.Interp.InputPrompt
			inputStr := fmt.Sprintf(" %s %s▌", prompt, m.InputBuf)
			inputStyle := lipgloss.NewStyle().
				Foreground(c("#ffcc00")).
				Background(c("#1a3a2a")).
				Bold(true)
			rendered := inputStyle.Render(inputStr)
			inputLayer := lipgloss.NewLayer(rendered).
				X(pr.Min.X+1).
				Y(pr.Min.Y+varsH+consoleH-2).
				Z(10).
				ID("input-overlay")
			layers = append(layers, inputLayer)
		}

		// Help
		layers = append(layers, buildHelpPanelLayer(pr.Min.X+1, pr.Min.Y+varsH+consoleH, pw-2, helpH))
	}

	// Edit modal (Z=100, on top of everything)
	if m.EditOpen {
		if modal := buildEditModalLayer(m, m.Width, m.Height); modal != nil {
			layers = append(layers, modal)
		}
	}

	// Compose
	comp := lipgloss.NewCompositor(layers...)
	canvas := lipgloss.NewCanvas(m.Width, m.Height)
	canvas.Compose(comp)

	v := tea.NewView(canvas.Render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}
