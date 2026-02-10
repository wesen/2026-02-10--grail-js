package grailui

import tea "charm.land/bubbletea/v2"

// Tool is the current interaction mode.
type Tool int

const (
	ToolSelect  Tool = iota
	ToolAdd
	ToolConnect
)

// Model is the main application state.
type Model struct {
	Width, Height  int
	MouseX, MouseY int
	CamX, CamY    int
	Graph          *FlowGraph
	SelectedID     *int
	ExecID         *int
	CurrentTool    Tool
	AddNodeType    string // node type for add tool

	// Drag state
	Dragging   bool
	DragNodeID int
	DragOffX   int
	DragOffY   int

	// Connect state
	ConnectFromID *int


}

// NewModel creates the initial model with the demo flowchart.
func NewModel() Model {
	return Model{
		Graph:       MakeInitialGraph(),
		AddNodeType: "process",
		DragNodeID:  -1,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}
