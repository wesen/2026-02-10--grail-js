package grailui

import (
	"image"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/wesen/grail/internal/flowinterp"
)

const panStep = 3

// nodeTypeKeys maps number keys to node types for add mode.
var nodeTypeKeys = map[string]string{
	"1": "process",
	"2": "decision",
	"3": "terminal",
	"4": "io",
	"5": "connector",
}

// TickMsg drives auto-stepping.
type TickMsg time.Time

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case tea.KeyMsg:
		if m.InputMode {
			return m.handleInputKeys(msg)
		}
		return m.handleKeys(msg)

	case tea.MouseMsg:
		if m.InputMode {
			return m, nil
		}
		canvasRect := m.canvasRect()
		return handleMouse(m, msg, canvasRect)

	case TickMsg:
		if m.AutoRunning && m.Interp != nil && !m.Interp.Done && m.Interp.Err == "" && !m.Interp.WaitInput {
			m.Interp.Step(nil)
			syncInterpreter(&m)
			if m.AutoRunning {
				return m, tickCmd(m.AutoSpeed)
			}
		} else {
			m.AutoRunning = false
		}
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

	// Interpreter controls
	case "r":
		return m.startProgram()
	case "n":
		return m.stepProgram()
	case "g":
		return m.autoRun()
	case "p":
		m.AutoRunning = false
	case "x":
		m.stopProgram()
	}

	return m, nil
}

// handleInputKeys processes keys when waiting for user input.
func (m Model) handleInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "enter":
		val := m.InputBuf
		m.InputBuf = ""
		m.InputMode = false
		if m.Interp != nil {
			m.Interp.Step(&val)
			syncInterpreter(&m)
		}
	case "backspace":
		if len(m.InputBuf) > 0 {
			m.InputBuf = m.InputBuf[:len(m.InputBuf)-1]
		}
	case "esc", "escape":
		m.InputMode = false
		m.InputBuf = ""
	default:
		if len(key) == 1 {
			m.InputBuf += key
		}
	}
	return m, nil
}

// startProgram creates the interpreter and takes the first step.
func (m Model) startProgram() (tea.Model, tea.Cmd) {
	if m.Running {
		return m, nil
	}

	// Convert graph nodes/edges to interpreter types
	nodes := make([]flowinterp.FlowNode, 0)
	for _, n := range m.Graph.Nodes() {
		nodes = append(nodes, flowinterp.FlowNode{
			ID:   n.ID,
			Type: n.Data.Type,
			Text: n.Data.Text,
			Code: n.Data.Code,
		})
	}
	edges := make([]flowinterp.FlowEdge, 0)
	for _, e := range m.Graph.Edges() {
		edges = append(edges, flowinterp.FlowEdge{
			FromID: e.FromID,
			ToID:   e.ToID,
			Label:  e.Data.Label,
		})
	}

	m.Interp = flowinterp.New(nodes, edges)
	m.Running = true
	m.Interp.Step(nil)
	syncInterpreter(&m)
	return m, nil
}

// stepProgram executes one interpreter step.
func (m Model) stepProgram() (tea.Model, tea.Cmd) {
	if !m.Running || m.Interp == nil || m.Interp.Done || m.InputMode {
		return m, nil
	}
	m.Interp.Step(nil)
	syncInterpreter(&m)
	return m, nil
}

// autoRun starts auto-stepping.
func (m Model) autoRun() (tea.Model, tea.Cmd) {
	if !m.Running || m.AutoRunning || m.Interp == nil {
		return m, nil
	}
	m.AutoRunning = true
	return m, tickCmd(m.AutoSpeed)
}

// stopProgram clears all interpreter state.
func (m *Model) stopProgram() {
	m.Interp = nil
	m.Running = false
	m.AutoRunning = false
	m.ExecID = nil
	m.InputMode = false
	m.InputBuf = ""
}

// syncInterpreter copies interpreter state to the model.
func syncInterpreter(m *Model) {
	if m.Interp == nil {
		return
	}
	m.ExecID = m.Interp.Current

	// Check for input wait
	if m.Interp.WaitInput {
		m.InputMode = true
		m.AutoRunning = false
	}

	// Check for completion/error
	if m.Interp.Done || m.Interp.Err != "" {
		m.AutoRunning = false
		if m.Interp.Err != "" {
			m.Interp.Output = append(m.Interp.Output, "⚠ "+m.Interp.Err)
		}
	}
}

// canvasRect computes the canvas region rectangle for coordinate transforms.
func (m Model) canvasRect() image.Rectangle {
	topH := 1
	bottomH := 1
	rightW := panelWidth
	return image.Rect(0, topH, m.Width-rightW, m.Height-bottomH)
}
