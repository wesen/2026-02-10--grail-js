package grailui

import (
	"fmt"
	"image"

	tea "charm.land/bubbletea/v2"
)

// handleMouse processes mouse events and returns updated model + command.
func handleMouse(m Model, msg tea.MouseMsg, canvasRect image.Rectangle) (Model, tea.Cmd) {
	mouse := msg.Mouse()
	m.MouseX = mouse.X
	m.MouseY = mouse.Y

	// Only process mouse events inside the canvas region
	if !image.Pt(mouse.X, mouse.Y).In(canvasRect) {
		return m, nil
	}

	// World coordinates from screen position
	worldX := mouse.X - canvasRect.Min.X + m.CamX
	worldY := mouse.Y - canvasRect.Min.Y + m.CamY

	switch msg.(type) {
	case tea.MouseMotionMsg:
		if m.Dragging && m.DragNodeID >= 0 {
			newX := worldX - m.DragOffX
			newY := worldY - m.DragOffY
			m.Graph.MoveNode(m.DragNodeID, image.Pt(newX, newY), SetPos)
		}

	case tea.MouseClickMsg:
		if mouse.Button == tea.MouseLeft {
			m = handleLeftClick(m, worldX, worldY)
		}

	case tea.MouseReleaseMsg:
		if m.Dragging {
			m.Dragging = false
			m.DragNodeID = -1
		}
	}

	return m, nil
}

// handleLeftClick dispatches based on current tool, using graphmodel.HitTest.
func handleLeftClick(m Model, worldX, worldY int) Model {
	// Hit test using graphmodel (world coordinates)
	hitNode := m.Graph.HitTest(image.Pt(worldX, worldY))
	hitNodeID := -1
	if hitNode != nil {
		hitNodeID = hitNode.ID
	}

	switch m.CurrentTool {
	case ToolSelect:
		if hitNodeID >= 0 {
			m.SelectedID = &hitNodeID
			// Start drag
			node := m.Graph.Node(hitNodeID)
			if node != nil {
				m.Dragging = true
				m.DragNodeID = hitNodeID
				m.DragOffX = worldX - node.Data.X
				m.DragOffY = worldY - node.Data.Y
			}
		} else {
			m.SelectedID = nil
		}

	case ToolAdd:
		info := nodeTypeInfo[m.AddNodeType]
		nx := worldX - info.W/2
		ny := worldY - info.H/2
		newText := fmt.Sprintf("NEW")
		id := m.Graph.AddNode(FlowNodeData{
			Type: m.AddNodeType,
			X:    nx,
			Y:    ny,
			Text: newText,
		})
		m.SelectedID = &id
		m.CurrentTool = ToolSelect

	case ToolConnect:
		if m.ConnectFromID == nil {
			if hitNodeID >= 0 {
				m.ConnectFromID = &hitNodeID
			}
		} else {
			if hitNodeID >= 0 && hitNodeID != *m.ConnectFromID {
				label := autoEdgeLabel(m.Graph, *m.ConnectFromID)
				m.Graph.AddEdge(*m.ConnectFromID, hitNodeID, FlowEdgeData{Label: label})
			}
			m.ConnectFromID = nil
			m.CurrentTool = ToolSelect
		}
	}

	return m
}

// autoEdgeLabel assigns "Y"/"N" labels for decision node edges.
func autoEdgeLabel(g *FlowGraph, fromID int) string {
	node := g.Node(fromID)
	if node == nil || node.Data.Type != "decision" {
		return ""
	}
	edges := g.OutEdges(fromID)
	hasY, hasN := false, false
	for _, e := range edges {
		if e.Data.Label == "Y" {
			hasY = true
		}
		if e.Data.Label == "N" {
			hasN = true
		}
	}
	if !hasY {
		return "Y"
	}
	if !hasN {
		return "N"
	}
	return ""
}

// hitTestWorld returns the node ID at world coordinates, or -1.
func hitTestWorld(g *FlowGraph, worldX, worldY int) int {
	hit := g.HitTest(image.Pt(worldX, worldY))
	if hit == nil {
		return -1
	}
	return hit.ID
}
