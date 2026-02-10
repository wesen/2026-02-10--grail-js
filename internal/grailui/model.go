package grailui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"github.com/wesen/grail/internal/flowinterp"
)

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

	// Interpreter state
	Interp      *flowinterp.Interpreter
	Running     bool
	AutoRunning bool
	AutoSpeed   time.Duration
	InputMode   bool   // waiting for user input
	InputBuf    string // typed input text

	// Edit modal state
	EditOpen    bool
	EditNodeID  int
	EditLabel   textinput.Model
	EditCode    textinput.Model
	EditFocus   int // 0=label, 1=code
}

// NewModel creates the initial model with the demo flowchart.
func NewModel() Model {
	return Model{
		Graph:       MakeInitialGraph(),
		AddNodeType: "process",
		DragNodeID:  -1,
		AutoSpeed:   400 * time.Millisecond,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}
