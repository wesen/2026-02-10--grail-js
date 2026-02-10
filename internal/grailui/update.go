package grailui

import (
	"image"

	tea "charm.land/bubbletea/v2"
)

const panStep = 3

// nodeTypes for add-mode cycling with number keys.
var nodeTypeKeys = map[string]string{
	"1": "process",
	"2": "decision",
	"3": "terminal",
	"4": "io",
	"5": "connector",
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case tea.KeyMsg:
		return m.handleKeys(msg)

	case tea.MouseMsg:
		canvasRect := m.canvasRect()
		return handleMouse(m, msg, canvasRect)
	}

	return m, nil
}

// handleKeys processes keyboard input.
func (m Model) handleKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit

	// Camera panning
	case "up":
		m.CamY -= panStep
	case "down":
		m.CamY += panStep
	case "left":
		m.CamX -= panStep
	case "right":
		m.CamX += panStep

	// Tool selection
	case "s":
		m.CurrentTool = ToolSelect
		m.ConnectFromID = nil
	case "a":
		m.CurrentTool = ToolAdd
		m.ConnectFromID = nil
	case "c":
		m.CurrentTool = ToolConnect
		m.ConnectFromID = nil

	// Node type in add mode
	case "1", "2", "3", "4", "5":
		if nt, ok := nodeTypeKeys[key]; ok {
			m.AddNodeType = nt
			m.CurrentTool = ToolAdd
		}

	// Delete selected
	case "d", "delete", "backspace":
		if m.SelectedID != nil {
			m.Graph.RemoveNode(*m.SelectedID)
			m.SelectedID = nil
		}

	// Escape — cancel current operation
	case "esc", "escape":
		m.ConnectFromID = nil
		m.SelectedID = nil
		m.CurrentTool = ToolSelect
	}

	return m, nil
}

// canvasRect computes the canvas region rectangle for coordinate transforms.
func (m Model) canvasRect() image.Rectangle {
	// Must match the layout in View
	topH := 1
	bottomH := 1
	rightW := panelWidth
	return image.Rect(0, topH, m.Width-rightW, m.Height-bottomH)
}
