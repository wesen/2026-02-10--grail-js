---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: grail.py
      Note: Existing Textual implementation — feature parity target
    - Path: ttmp/2026/02/10/GRAIL-001-PORT-TO-INKJS--port-grail-flowchart-interpreter-from-react-jsx-to-ink-js-terminal-app-with-mouse-support/design-doc/02-bubbletea-v2-canvas-architecture.md
      Note: Full architecture blueprint — this plan implements it incrementally
    - Path: ttmp/2026/02/10/GRAIL-001-PORT-TO-INKJS--port-grail-flowchart-interpreter-from-react-jsx-to-ink-js-terminal-app-with-mouse-support/reference/02-lipgloss-rendering-performance-investigation.md
      Note: Rendering performance data — informs MiniBuffer implementation
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# GRaIL Go Port — Component-by-Component Build Plan

## 1. Design Philosophy

### 1.1 Why component-by-component?

GRaIL as a single monolith is ~1100 lines of Go. That's manageable, but
the pieces solve problems that come up in *any* terminal-based spatial
editor: character-level drawing, graph layout, mouse-driven interaction
over a pannable canvas, modal dialogs. Building them as isolated,
tested packages means:

- Each piece can be validated independently before integration
- Other projects (dependency graphs, state machine editors, circuit
  diagrams, roguelikes) can reuse the packages
- The final GRaIL app becomes thin glue between well-tested components

### 1.2 Package boundary principle

A package is worth extracting if it:
1. Has a clear input/output contract that doesn't depend on GRaIL types
2. Could plausibly be used in a different terminal application
3. Has ≥50 lines of non-trivial logic

Things that are NOT worth extracting: the specific GRaIL node types, the
initial flowchart data, the interpreter's node-type dispatch. Those are
application logic, not reusable infrastructure.

### 1.3 Build order principle

Each step produces a **running program** you can see and interact with.
No invisible infrastructure steps. Every step adds something visible.

---

## 2. Package Map

```
grail/
├── pkg/
│   ├── cellbuf/              ← Step 1: 2D character buffer with styled rendering
│   │   ├── buffer.go         # CellBuffer struct, New, Set, SetString
│   │   ├── render.go         # Render() → string via Render-per-run
│   │   └── buffer_test.go
│   │
│   ├── drawutil/             ← Step 2: Terminal drawing primitives
│   │   ├── line.go           # Bresenham, lineChar, arrowChar
│   │   ├── edge.go           # Edge exit point calculation
│   │   ├── grid.go           # Grid dot pattern
│   │   └── line_test.go
│   │
│   ├── graphmodel/           ← Step 3: Generic graph data model
│   │   ├── graph.go          # Graph[N, E], AddNode, AddEdge, RemoveNode
│   │   ├── spatial.go        # HitTest, BoundingBox, CenterOf
│   │   └── graph_test.go
│   │
│   └── tealayout/            ← Step 5: Bubbletea layout helpers
│       ├── regions.go        # RegionLayout: compute named rects from terminal size
│       ├── chrome.go         # Toolbar, SidePanel, Footer layer builders
│       ├── modal.go          # ModalLayer: centered high-Z overlay
│       └── regions_test.go
│
├── internal/
│   ├── flowinterp/           ← Step 10: GRaIL-specific interpreter
│   │   ├── interpreter.go
│   │   └── interpreter_test.go
│   │
│   └── grailui/              ← Steps 4-12: Application glue
│       ├── model.go          # Model struct, Init
│       ├── update.go         # Update routing
│       ├── view.go           # View — layer composition
│       ├── mouse.go          # Mouse handler
│       ├── keys.go           # Keyboard handler
│       ├── layers.go         # buildNodeLayer, buildEdgeCanvas, etc.
│       └── data.go           # Initial flowchart, node type registry
│
├── main.go                   # Entry point
└── go.mod
```

### What's reusable (`pkg/`) vs app-specific (`internal/`)

| Package | Reusable for | GRaIL-specific? |
|---|---|---|
| `cellbuf` | Any app needing character-level drawing (roguelikes, diagram renderers, ASCII art tools) | No |
| `drawutil` | Any app drawing lines/arrows between objects in a terminal | No |
| `graphmodel` | Any node+edge graph editor or visualizer | No |
| `tealayout` | Any Bubbletea app with toolbar/panel/modal chrome | No |
| `flowinterp` | GRaIL flowchart interpreter only | **Yes** |
| `grailui` | GRaIL application wiring only | **Yes** |

---

## 3. The Build Steps

### Dependency graph

```
Step 1: cellbuf          (no deps)
Step 2: drawutil          (depends on cellbuf for drawing targets)
Step 3: graphmodel        (no deps — pure data)
Step 4: scaffold          (depends on bubbletea/lipgloss — first running app)
Step 5: tealayout         (depends on lipgloss — layout helpers)
Step 6: nodes on canvas   (depends on 3+4+5 — first visual graph)
Step 7: edge rendering    (depends on 1+2+6 — edges between nodes)
Step 8: side panel        (depends on 5 — chrome layers)
Step 9: mouse interaction (depends on 6+7 — click/drag/connect)
Step 10: interpreter      (depends on 3 — pure logic)
Step 11: interpreter UI   (depends on 8+9+10 — run/step/auto)
Step 12: edit modal       (depends on 5+9 — modal dialog)
```

Each step is described below with: what you build, what you can see/test,
what the demo looks like, and the approximate line count.

---

### Step 1: `cellbuf` — 2D Character Buffer (~80 lines)

**What:** A `CellBuffer` that holds a grid of `(rune, StyleKey)` cells and
renders to a styled string using Lipgloss `Render()`-per-run.

**Why first:** This is the foundation for edge drawing and grid dots. It
has zero dependencies on Bubbletea or application logic. Pure library code
with pure tests.

**Package API:**

```go
package cellbuf

type StyleKey int

type Cell struct {
    Ch    rune
    Style StyleKey
}

type Buffer struct {
    W, H  int
    Cells [][]Cell
}

func New(w, h int, defaultStyle StyleKey) *Buffer
func (b *Buffer) Set(x, y int, ch rune, style StyleKey)
func (b *Buffer) SetString(x, y int, s string, style StyleKey)
func (b *Buffer) Fill(style StyleKey)                          // clear to spaces
func (b *Buffer) Render(styles map[StyleKey]lipgloss.Style) string
func (b *Buffer) InBounds(x, y int) bool
```

**Demo:** A test that creates a 20×5 buffer, writes "Hello" at (2,2),
renders it, and verifies the output contains styled ANSI sequences.

**Tests:**
- `Set`/`SetString` write at correct positions, clamp to bounds
- `Render` produces correct number of lines
- `Render` merges consecutive same-styled cells (verify by checking output
  length is smaller than width × per-cell overhead)
- Benchmark: `Render` on a 200×50 buffer < 2ms

---

### Step 2: `drawutil` — Line Drawing and Edge Geometry (~80 lines)

**What:** Bresenham line algorithm, line/arrow characters, grid dot
pattern, edge exit-point calculation. Functions that draw into a
`cellbuf.Buffer`.

**Why now:** These are pure math functions with well-defined outputs.
They can be tested without any UI.

**Package API:**

```go
package drawutil

import "image"

func Bresenham(x0, y0, x1, y1 int) []image.Point
func LineChar(dx, dy int) rune
func ArrowChar(dx, dy int) rune

// Edge exit point: where an edge exits a rectangle facing a target point
func EdgeExit(rect image.Rectangle, targetCenter image.Point) image.Point

// Draw a grid of dots into a buffer
func DrawGrid(buf *cellbuf.Buffer, camX, camY, spacingX, spacingY int, style cellbuf.StyleKey)

// Draw a Bresenham line with arrowhead into a buffer
func DrawLine(buf *cellbuf.Buffer, x0, y0, x1, y1 int, style cellbuf.StyleKey)
func DrawArrowLine(buf *cellbuf.Buffer, x0, y0, x1, y1 int, lineStyle, arrowStyle cellbuf.StyleKey)

// Draw a dashed line (every 3rd char skipped) for connect preview
func DrawDashedLine(buf *cellbuf.Buffer, x0, y0, x1, y1 int, style cellbuf.StyleKey)
```

**Demo:** A test that draws a line from (0,0) to (15,8) into a buffer
and verifies the points match expected Bresenham output.

**Tests:**
- `Bresenham` horizontal, vertical, diagonal, steep, and zero-length cases
- `LineChar` returns `│` `─` `/` `\` for cardinal and diagonal directions
- `ArrowChar` returns `▲▼◄►` for four directions
- `EdgeExit` exits from correct side (right/left/top/bottom) for various
  target positions relative to a rectangle
- `DrawGrid` places dots at correct spacing intervals

---

### Step 3: `graphmodel` — Generic Graph Data Model (~100 lines)

**What:** A generic graph with positioned nodes and labeled edges.
Operations: add/remove nodes, add/remove edges, spatial hit testing.

**Why now:** The data model has no UI dependencies. It defines the core
types that everything else consumes.

**Key design choice:** Use an interface for node position + size, not
concrete GRaIL types. This makes the package reusable for any spatial
graph.

**Package API:**

```go
package graphmodel

import "image"

// Spatial is anything that has a position and size in world coordinates
type Spatial interface {
    Pos() image.Point              // top-left
    Size() image.Point             // width, height
    Center() image.Point           // computed: Pos + Size/2
    Bounds() image.Rectangle       // computed: Rect(Pos, Pos+Size)
}

type Edge[E any] struct {
    FromID, ToID int
    Data         E
}

type Node[N Spatial] struct {
    ID   int
    Data N
}

type Graph[N Spatial, E any] struct {
    nodes  map[int]*Node[N]
    edges  []Edge[E]
    nextID int
}

func New[N Spatial, E any]() *Graph[N, E]
func (g *Graph[N, E]) AddNode(data N) int                     // returns assigned ID
func (g *Graph[N, E]) RemoveNode(id int)                      // also removes connected edges
func (g *Graph[N, E]) AddEdge(fromID, toID int, data E)
func (g *Graph[N, E]) RemoveEdge(fromID, toID int)
func (g *Graph[N, E]) Node(id int) *Node[N]
func (g *Graph[N, E]) Nodes() []*Node[N]                      // stable iteration order
func (g *Graph[N, E]) Edges() []Edge[E]
func (g *Graph[N, E]) OutEdges(nodeID int) []Edge[E]
func (g *Graph[N, E]) MoveNode(id int, pos image.Point)

// Spatial queries
func (g *Graph[N, E]) HitTest(worldPt image.Point) *Node[N]   // topmost node containing point
func (g *Graph[N, E]) NodesInRect(r image.Rectangle) []*Node[N]
```

**Demo:** A test that creates a graph, adds nodes, connects them, hit-tests
a point, removes a node (verifying edges are cleaned up).

**Tests:**
- Add node → ID is assigned and increments
- Remove node → connected edges are removed
- HitTest returns topmost (highest ID) node at point
- HitTest returns nil for empty space
- OutEdges returns correct edges for a given node
- MoveNode updates position

---

### Step 4: Scaffold — Minimal Bubbletea + Lipgloss v2 App (~60 lines)

**What:** The smallest possible running Bubbletea v2 app that uses
Lipgloss v2 Canvas/Layer compositing. A colored rectangle with a
title. Confirms the v2 beta stack works.

**Why now:** Before building anything complex, verify that `bubbletea/v2`
+ `lipgloss/v2` + `tea.WithMouseAllMotion()` actually work together.
Catch dependency issues early.

**What you see:** A terminal fills with a dark green background. A
toolbar at the top says "GRaIL". A footer at the bottom shows "q: quit".
Mouse coordinates are displayed live (proving mouse events work). Press
`q` to exit.

**What this validates:**
- `bubbletea/v2` + `lipgloss/v2` compile and run together
- `tea.WithAltScreen()` works
- `tea.WithMouseAllMotion()` delivers `MouseMsg` events
- `lipgloss.NewCanvas()` + `NewLayer()` + `.Render()` produce visible output
- `WindowSizeMsg` fires and provides correct terminal dimensions

**Pseudocode:**

```
Model:
    termW, termH int
    mouseX, mouseY int

View():
    bg := lipgloss.NewStyle().Width(m.termW).Height(m.termH).
        Background(color("#080e0b")).Render("")
    bgLayer := NewLayer(bg).X(0).Y(0).Z(0)

    title := NewLayer(titleStyle.Render("  GRaIL  ")).X(0).Y(0).Z(1)
    mouse := NewLayer(fmt.Sprintf("Mouse: %d,%d", m.mouseX, m.mouseY)).
        X(0).Y(m.termH-1).Z(1)

    canvas := NewCanvas(bgLayer, title, mouse)
    return canvas.Render()
```

**~60 lines. Should take 15 minutes. If it doesn't work, you know before
investing further.**

---

### Step 5: `tealayout` — Layout Regions and Chrome (~120 lines)

**What:** Helpers for computing layout regions from terminal size, and
building common UI chrome (toolbar, side panel, footer) as Lipgloss
layers.

**Why now:** Steps 6+ need layout math. Extracting it into a package
means every Bubbletea app you build can use it.

**Package API:**

```go
package tealayout

import (
    "image"
    "github.com/charmbracelet/lipgloss/v2"
)

// Region is a named rectangular area of the terminal
type Region struct {
    Name   string
    Rect   image.Rectangle   // in terminal coordinates
}

// Layout computes named regions from terminal size and constraints
type Layout struct {
    TermW, TermH int
    Regions      map[string]Region
}

// LayoutBuilder declaratively defines regions
type LayoutBuilder struct { ... }

func NewLayoutBuilder(termW, termH int) *LayoutBuilder
func (b *LayoutBuilder) TopFixed(name string, height int) *LayoutBuilder
func (b *LayoutBuilder) BottomFixed(name string, height int) *LayoutBuilder
func (b *LayoutBuilder) RightFixed(name string, width int) *LayoutBuilder
func (b *LayoutBuilder) Remaining(name string) *LayoutBuilder   // fills remaining space
func (b *LayoutBuilder) Build() *Layout

// Convenience: make a layer that fills a region
func (l *Layout) FillLayer(name, id string, style lipgloss.Style, content string, z int) lipgloss.Layer

// Chrome builders
func ToolbarLayer(content string, width int, style lipgloss.Style) lipgloss.Layer
func FooterLayer(content string, width int, y int, style lipgloss.Style) lipgloss.Layer
func VerticalSeparator(x, y, height int, style lipgloss.Style) lipgloss.Layer
func ModalLayer(content string, termW, termH int, boxStyle lipgloss.Style) lipgloss.Layer
```

**Usage in GRaIL:**

```go
layout := tealayout.NewLayoutBuilder(m.termW, m.termH).
    TopFixed("toolbar", 3).
    BottomFixed("footer", 1).
    RightFixed("panel", 34).
    Remaining("canvas").
    Build()

canvasRect := layout.Regions["canvas"].Rect
m.canvasW = canvasRect.Dx()
m.canvasH = canvasRect.Dy()
```

**Tests:**
- Layout computes correct region sizes
- Regions don't overlap
- `Remaining` fills leftover space correctly
- `ModalLayer` centers content correctly

---

### Step 6: Nodes on Canvas — First Visual Graph (~100 lines)

**What:** Wire `graphmodel` to `tealayout` + Lipgloss v2 layers. Each
node in the graph becomes a styled Lipgloss box layer with Z=2 and an
ID. Camera panning with arrow keys.

**Why now:** This is the first step where you see something that looks
like a flowchart. It validates the core architecture: data model →
layers → Canvas.Render().

**What you see:** The initial 7-node flowchart rendered as styled boxes
on a dark background. Each node has the correct border style (rounded
for terminal, double for decision, normal for process). Arrow keys pan
the camera. Nodes that scroll off-screen disappear (visibility culling).

**What this builds (in `grailui/`):**
- GRaIL-specific `FlowNodeData` type implementing `graphmodel.Spatial`
- `FlowEdgeData` type with `Label string`
- `buildNodeLayer(node, isSelected, isExec, cam, layout)` function
- Node type → border style mapping
- Node type → color mapping
- View pipeline: toolbar + nodes + footer (no edges yet)

**Demo validation:**
- All 7 nodes visible at default camera position
- Arrow keys pan smoothly
- Connector node (ID 5) is smaller (7×3) than others (22×3)
- Different border styles visible: `╭╮╰╯` (terminal), `╔╗╚╝` (decision), `┌┐└┘` (process/io)

---

### Step 7: Edge Rendering — Lines Between Nodes (~80 lines)

**What:** Build the MiniBuffer background layer with grid dots, edge
lines (Bresenham), arrowheads, and edge labels at Z=3.

**Why now:** Edges are what make a graph a graph. After this step, the
app looks like a real flowchart.

**What you see:** Green lines connecting all 7 nodes. Arrowheads (▼►◄▲)
at destination ends. "Y" and "N" labels on the decision's outgoing edges.
Grid dots (·) in the background.

**What this builds:**
- `buildEdgeCanvasLayer()`: creates MiniBuffer, draws grid + all edges,
  returns as Layer at Z=0
- `buildEdgeLabelLayer()`: each label as a Layer at Z=3
- Uses `drawutil.DrawArrowLine` for each edge
- Uses `drawutil.EdgeExit` for edge start/end points
- Uses `drawutil.DrawGrid` for background dots

**Integration with Step 6:** Edge layer at Z=0 composites below node
layers at Z=2. Nodes naturally occlude edge lines passing through them.

**Tests:**
- Edge from node 1→2 produces a vertical line with ▼ at bottom
- Edge exit points are on the correct side of nodes
- Labels "Y" and "N" positioned near edge midpoints

---

### Step 8: Side Panel — Variables, Console, Help (~80 lines)

**What:** Three panel sections as layers to the right of the canvas,
separated from the canvas by a vertical `│` border. Static content
for now (will be wired to interpreter in Step 11).

**Why now:** The UI shape needs to be complete before adding interaction.
After this step, the app has the same visual layout as the Python version.

**What you see:** Right panel with three sections:
- 📦 VARIABLES: "(none)" in dim text
- 🖥️ CONSOLE: empty
- HELP: keybinding list

**What this builds:**
- `buildVarsPanelLayer(vars map, layout)` → Layer
- `buildConsolePanelLayer(lines []string, layout)` → Layer
- `buildHelpPanelLayer(layout)` → Layer
- `tealayout.VerticalSeparator()` for the `│` column

---

### Step 9: Mouse Interaction — Select, Drag, Connect (~120 lines)

**What:** Mouse handling: click to select, drag to move, connect mode
to create edges, add mode to place new nodes, delete selected.

**Why now:** This is where the app becomes *interactive*. It requires
nodes (Step 6) and edges (Step 7) to already be rendering, because you
need visual feedback for selection highlighting and connect preview.

**What you see:**
- Click a node → it highlights (cyan border)
- Drag a node → it follows the mouse, edges update in real-time
- Press `c`, click source node, click target → edge created
- Press `a`, click empty space → new node placed
- Press `d` with node selected → node deleted (edges cleaned up)
- Connect mode: dashed preview line from source node to cursor

**What this builds:**
- `handleMouse(m, msg)`: uses `m.canvas.Hit(msg.X, msg.Y)` for
  hit testing, routes to tool-specific handlers
- `handleCanvasKeys(m, msg)`: tool switching (s/a/c), node type
  (1-5), edit (e), delete (d), panning (arrows)
- Selection state: `m.selectedID`
- Drag state machine: press→drag→release
- Connect state machine: click source→track mouse→click target
- `buildConnectPreviewLayer()`: dashed line using `drawutil.DrawDashedLine`
  into a MiniBuffer layer

**Critical correctness check:** Verify that `Canvas.Hit()` coordinates
match `MouseMsg.X/Y` coordinates. Write a small standalone test first:

```go
// Does canvas.Hit(10, 5) find a layer at X=10, Y=5?
layer := lipgloss.NewLayer("test").X(10).Y(5).Z(0).ID("target")
canvas := lipgloss.NewCanvas(layer)
hit := canvas.Hit(10, 5)
assert hit.GetID() == "target"
```

**If this fails, the entire hit testing architecture needs revision.
Do this test BEFORE building Step 9.**

---

### Step 10: Flow Interpreter — Pure Logic (~200 lines)

**What:** The flowchart interpreter using Goja (embedded JS runtime).
Step-through execution with support for process, decision, terminal,
I/O, and connector node types.

**Why now:** The interpreter is pure logic with no UI dependencies.
It can be built and fully tested in isolation.

**Why `internal/` not `pkg/`:** The interpreter's node-type dispatch
(process/decision/terminal/io/connector) is GRaIL-specific. A generic
"graph interpreter" would need a plugin/callback architecture that
isn't worth the complexity for this project.

**Package API:**

```go
package flowinterp

type Interpreter struct { ... }

func New(nodes []FlowNode, edges []FlowEdge) *Interpreter
func (i *Interpreter) Step(inputValue *string)
func (i *Interpreter) CurrentID() *int
func (i *Interpreter) Vars() map[string]interface{}
func (i *Interpreter) Output() []string
func (i *Interpreter) Done() bool
func (i *Interpreter) Err() string
func (i *Interpreter) WaitingInput() bool
func (i *Interpreter) InputPrompt() string
```

**Tests (port from Python smoke test):**
- Run the initial flowchart to completion → `sum == 15`
- Step count doesn't exceed 500
- Output contains "PROGRAM START", "Sum 1..5 = 15", "PROGRAM END"
- Decision node branches correctly on Y/N
- I/O node with `print()` appends to output
- I/O node with `input()` pauses and waits for value
- Error handling: broken link, missing start node, max steps exceeded

---

### Step 11: Interpreter UI — Run, Step, Auto, Pause, Stop (~100 lines)

**What:** Wire the interpreter to the UI. Toolbar shows run state.
Console panel shows output. Variables panel shows current values.
Current execution node highlights (yellow border). Auto-run with
timer. Program input field.

**Why now:** With interpreter (Step 10) and mouse (Step 9) done, this
is the integration step that makes the app actually *do* something.

**What you see:**
- Press `r` → "PROGRAM START" in console, START node highlights yellow
- Press `n` → interpreter steps, next node highlights, variables update
- Press `g` → auto-run at 400ms intervals, nodes highlight in sequence
- Press `p` → pause auto-run
- Press `x` → stop, clear highlighting
- I/O node: input field appears, type value, press enter
- Console shows all output in real-time

**What this builds:**
- `startProgram`, `stepProgram`, `autoRun`, `pauseProgram`, `stopProgram`
- `syncInterpreter()`: copy interpreter state to model (vars, output,
  currentID, waitInput)
- `TickMsg` handling for auto-step
- `buildInputOverlayLayer()` for program input
- `FocusInput` handling for keyboard routing to textinput
- Toolbar state display (▶ AUTO / ⏸ READY / ▶ RUN)

---

### Step 12: Edit Modal — Node Editing Dialog (~60 lines)

**What:** Press `e` on a selected node → modal dialog appears (Z=100)
with label and code fields. Tab switches fields. Enter saves. Esc
cancels.

**Why now:** Last interactive feature. Builds on `tealayout.ModalLayer`
from Step 5 and focus routing from Step 9.

**What you see:** Centered bordered box over the canvas with:
- Title: "✏️ EDIT — PROCESS"
- Label field (textinput)
- Code field (textinput) with hint for node type
- [tab] switch / [enter] save / [esc] cancel

**What this builds:**
- `buildEditModalLayer(m)` using `tealayout.ModalLayer`
- `handleEditKeys(m, msg)` with FocusEditLabel/FocusEditCode routing
- `openEditModal(m)` / `commitEdit(m)` / `cancelEdit(m)`

---

## 4. Step-by-Step Visual Progression

| After step | What the user sees |
|---|---|
| 4 (scaffold) | Dark green screen, title, mouse coords in footer |
| 6 (nodes) | 7 styled boxes on a dark canvas, arrow keys pan |
| 7 (edges) | Lines connecting nodes, arrowheads, Y/N labels, grid dots |
| 8 (panel) | Right panel with help text, vertical separator |
| 9 (mouse) | Click/drag/connect nodes, selection highlighting |
| 11 (interp) | Run flowchart, watch execution highlight walk through nodes |
| 12 (modal) | Edit node properties in a centered dialog |

Every step from 4 onward produces a program you can run and see
progress. There are no "invisible infrastructure" steps.

---

## 5. Testing Strategy

| Package | Test approach | Coverage target |
|---|---|---|
| `cellbuf` | Unit tests: Set/Render correctness + benchmark | 100% of public API |
| `drawutil` | Unit tests: Bresenham output, edge exit geometry | Golden-value tests |
| `graphmodel` | Unit tests: add/remove/hit-test operations | 100% of public API |
| `tealayout` | Unit tests: layout computation, region math | Boundary conditions |
| `flowinterp` | Integration test: run initial flowchart → sum=15 | Happy path + errors |
| `grailui` | Manual testing: run app, exercise all features | Visual verification |

The key insight: packages `cellbuf`, `drawutil`, `graphmodel`, and
`flowinterp` are **pure logic** — no terminal, no UI, no Bubbletea.
They can be tested with standard `go test` in CI. The UI (`grailui`)
is tested manually.

---

## 6. Estimated Effort per Step

| Step | Package | New lines | Cumulative | Time |
|---|---|---|---|---|
| 1 | `cellbuf` | ~80 | 80 | 30 min |
| 2 | `drawutil` | ~80 | 160 | 30 min |
| 3 | `graphmodel` | ~100 | 260 | 45 min |
| 4 | scaffold | ~60 | 320 | 15 min |
| 5 | `tealayout` | ~120 | 440 | 45 min |
| 6 | nodes on canvas | ~100 | 540 | 45 min |
| 7 | edge rendering | ~80 | 620 | 30 min |
| 8 | side panel | ~80 | 700 | 30 min |
| 9 | mouse interaction | ~120 | 820 | 60 min |
| 10 | interpreter | ~200 | 1020 | 60 min |
| 11 | interpreter UI | ~100 | 1120 | 45 min |
| 12 | edit modal | ~60 | 1180 | 30 min |
| | **Total** | **~1180** | | **~7.5 hours** |

---

## 7. Risk Checkpoints

Certain steps validate architectural assumptions. If they fail, you
need to change approach before investing further.

### Checkpoint A: After Step 4 (scaffold)

**Validate:** Bubbletea v2 + Lipgloss v2 beta.2 + Canvas/Layer +
MouseAllMotion all work together.

**If it fails:** The v2 beta stack has breaking issues. Fall back to
Bubbletea v1 + Lipgloss v1 + CellBuffer approach (doc 01).

### Checkpoint B: Before Step 9 (mouse interaction)

**Validate:** `Canvas.Hit(x, y)` returns the correct Layer for
coordinates matching `MouseMsg.X/Y`. Write the standalone test.

**If it fails:** Canvas.Hit coordinates don't match mouse coordinates.
Options: (a) apply an offset, (b) fall back to manual hit testing
(~30 lines extra), (c) file a bug upstream.

### Checkpoint C: After Step 7 (edges)

**Validate:** The full flowchart renders correctly — all 7 nodes, all
7 edges, Y/N labels, arrowheads. Compare visually with the Python
version running side-by-side.

**If it fails:** Edge exit calculation or Bresenham is wrong. Debug
with `drawutil` unit tests.

### Checkpoint D: After Step 10 (interpreter)

**Validate:** `go test ./internal/flowinterp/` passes with `sum == 15`.

**If it fails:** Goja eval semantics differ from Python eval. Debug
with individual statement tests.

---

## 8. What You Can Reuse Tomorrow

After building GRaIL, you have four reusable packages:

### `cellbuf` — Use it for:
- Roguelike game rendering
- Terminal-based diagram/schematic viewers
- ASCII art generators
- Any app that needs character-level control within a Bubbletea View

### `drawutil` — Use it for:
- Any terminal graph/diagram renderer
- Circuit diagram editors
- Tree visualization tools
- Any app drawing lines between boxes

### `graphmodel` — Use it for:
- Dependency graph visualizers
- State machine editors
- Entity-relationship diagram editors
- Any spatial node-and-edge graph tool

### `tealayout` — Use it for:
- Any Bubbletea app with a toolbar + main area + side panel layout
- Apps needing centered modal dialogs
- Dashboard-style terminal UIs
- Any app that needs declarative region-based layout
