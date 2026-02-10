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
      Note: Existing Textual/Python implementation — source of truth for feature parity
    - Path: reference.jsx
      Note: Original React/SVG browser app
    - Path: ttmp/2026/02/10/GRAIL-001-PORT-TO-INKJS--port-grail-flowchart-interpreter-from-react-jsx-to-ink-js-terminal-app-with-mouse-support/design-doc/01-bubbletea-port-analysis.md
      Note: Previous analysis (v1 CellBuffer approach) — superseded by this document for rendering architecture
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# GRaIL Go Port — Lipgloss v2 Canvas/Layer Architecture

## 1. Executive Summary

This document is a complete blueprint for porting the GRaIL flowchart
editor+interpreter from Python/Textual (`grail.py`, ~1112 lines) to
Go using **Bubble Tea v2 + Lipgloss v2.0.0-beta.2**.

The previous analysis (doc 01) designed a rendering pipeline around a
hand-built CellBuffer — a 2D grid of `(rune, style)` cells that you
paint into character by character. That approach required ~1500 lines
of Go and contained three genuinely hard problems: ANSI string splicing
for modal overlays, manual hit testing that must stay in sync with
rendering, and per-character box drawing for node shapes.

**Lipgloss v2 eliminates all three.** Its Canvas/Layer compositing
system lets you:

- Render each node as a **Lipgloss styled box** → wrap in a `Layer`
  with `X/Y/Z/ID` → the framework composites them
- Place a modal edit dialog as a **high-Z Layer** → it just appears on
  top, no ANSI splicing
- Call `Canvas.Hit(x, y)` → get back the `Layer` (and its `ID`) under
  the mouse → hit testing is consistent with rendering by construction

The architecture becomes structurally similar to the **original
React/SVG** source: absolutely-positioned styled boxes with z-index
layering. Estimated effort: **~1100 lines of Go**, with the hardest
parts being "easy" and the remaining custom code well-scoped.

### Required stack (all v2 betas)

```
github.com/charmbracelet/bubbletea/v2
github.com/charmbracelet/lipgloss/v2   (v2.0.0-beta.2+)
github.com/charmbracelet/bubbles/v2    (textinput, viewport)
github.com/dop251/goja                 (JS interpreter for eval)
```

Lipgloss v2 compositing is **only supported with Bubble Tea v2**. This
is a hard constraint from the Charm team.

---

## 2. Layer Map — What the User Sees

Every visual element maps to a `lipgloss.Layer` with a position (X, Y),
stacking order (Z), and optional ID for hit testing.

```
Full-screen Canvas
│
├─ Z=0  "toolbar"          X=0, Y=0                    Styled toolbar text
├─ Z=0  "edge-canvas"      X=0, Y=TOOLBAR_H            MiniBuffer → string (grid + edges)
├─ Z=0  "panel-border"     X=canvasW, Y=TOOLBAR_H      Vertical │ separator
├─ Z=0  "vars-panel"       X=canvasW+1, Y=TOOLBAR_H    Variables display
├─ Z=0  "console-panel"    X=canvasW+1, Y=consoleTop    Console output (viewport)
├─ Z=0  "help-panel"       X=canvasW+1, Y=helpTop       Key bindings help
├─ Z=0  "footer"           X=0, Y=termH-1               Status line
│
├─ Z=2  "node-1"           X=screenX, Y=screenY         Lipgloss styled box, ID="node-1"
├─ Z=2  "node-2"           ...                           ...
├─ Z=2  ...                                              one Layer per visible node
│
├─ Z=3  "label-0"          X=labelX, Y=labelY           Edge label "Y" or "N"
├─ Z=3  "label-1"          ...
│
├─ Z=5  "connect-preview"  X=0, Y=TOOLBAR_H             Only present during connect-mode
│
├─ Z=10 "input-overlay"    X=canvasW+1, Y=inputTop      Only present when awaiting input
│
└─ Z=100 "edit-modal"      X=centered, Y=centered       Only present when editing a node
```

### Comparison to React/SVG original

| React/SVX (`reference.jsx`) | Lipgloss v2 Canvas/Layer |
|---|---|
| `<div style={{position:'absolute', left, top, zIndex:2}}>` | `NewLayer(content).X(x).Y(y).Z(2)` |
| `<line>` SVG elements below nodes | Edge MiniBuffer at Z=0 |
| CSS `z-index` for stacking | `Layer.Z(int)` |
| `onClick` event on each `<div>` | `Canvas.Hit(x, y)` + `Layer.ID()` |
| Modal overlay with `z-index: 200` | Layer at Z=100 |

The mental model is nearly identical. The terminal port "looks like" the
browser original at the architecture level.

---

## 3. The Model

```go
// ═══════════════════════════════════════════════════
// model.go — Elm architecture root
// ═══════════════════════════════════════════════════

type Tool int
const (
    ToolSelect Tool = iota
    ToolAdd
    ToolConnect
)

type FocusTarget int
const (
    FocusCanvas FocusTarget = iota
    FocusInput                      // program input field
    FocusEditLabel                  // edit modal: label field
    FocusEditCode                   // edit modal: code field
)

type Model struct {
    // ── Data ──
    nodes       []FlowNode
    edges       []FlowEdge
    nextID      int

    // ── Selection / Interaction ──
    tool        Tool
    newNodeType string          // "process", "decision", etc.
    selectedID  *int
    connectID   *int            // source node during connect
    dragging    bool
    dragNodeID  int
    dragOffX    int             // cursor offset within node at drag start
    dragOffY    int

    // ── Camera ──
    camX, camY  int

    // ── Mouse ──
    lastMouseX  int             // for throttling (absolute terminal coords)
    lastMouseY  int

    // ── Interpreter ──
    interp      *FlowInterpreter
    execNodeID  *int
    running     bool
    autoRunning bool
    waitInput   bool
    speed       time.Duration

    // ── Derived state (from interpreter) ──
    consoleLines []string
    variables    map[string]interface{}

    // ── Sub-models (from bubbles/v2) ──
    inputField   textinput.Model
    editLabel    textinput.Model     // only active when editOpen
    editCode     textinput.Model

    // ── Edit modal ──
    editOpen     bool
    editNodeID   int

    // ── Layout (recomputed on WindowSizeMsg) ──
    termW, termH int
    canvasW      int                // canvas area width in columns
    canvasH      int                // canvas area height in rows
    panelW       int                // right panel width

    // ── Focus ──
    focus        FocusTarget

    // ── Compositing (retained for hit testing) ──
    canvas       *lipgloss.Canvas   // last-rendered canvas, for Hit()
}
```

### Layout constants

```go
const (
    ToolbarH = 3   // title line + tool buttons + bottom border
    FooterH  = 1   // status line
    PanelW   = 34  // right panel width
    BorderW  = 1   // vertical separator between canvas and panel
    VarsH    = 6   // variables section height
    HelpH    = 8   // help section height
)
```

---

## 4. Init / Update / View — Top Level

### 4.1 Init

```
FUNCTION Init() (Model, Cmd):
    m = Model{
        nodes:       makeInitialNodes(),
        edges:       makeInitialEdges(),
        nextID:      20,
        tool:        ToolSelect,
        newNodeType: "process",
        speed:       400 * time.Millisecond,
        panelW:      PanelW,
    }
    m.inputField = textinput.New()
    m.inputField.Placeholder = "type input…"
    m.editLabel = textinput.New()
    m.editCode = textinput.New()
    return m, tea.WindowSize()    // request initial terminal size
```

### 4.2 Update (routing)

```
FUNCTION Update(m Model, msg tea.Msg) (Model, Cmd):
    MATCH msg:

    // ── Terminal resize ──
    tea.WindowSizeMsg:
        m.termW = msg.Width
        m.termH = msg.Height
        m.canvasW = msg.Width - PanelW - BorderW
        m.canvasH = msg.Height - ToolbarH - FooterH
        return m, nil

    // ── Mouse ──
    tea.MouseMsg:
        // Throttle: skip if same cell
        IF msg.X == m.lastMouseX AND msg.Y == m.lastMouseY AND isMotion(msg):
            return m, nil
        m.lastMouseX = msg.X
        m.lastMouseY = msg.Y
        return handleMouse(m, msg)

    // ── Keyboard ──
    tea.KeyMsg:
        IF m.editOpen:
            return handleEditKeys(m, msg)
        IF m.focus == FocusInput:
            return handleInputKeys(m, msg)
        return handleCanvasKeys(m, msg)

    // ── Auto-step timer ──
    TickMsg:
        IF m.autoRunning AND m.interp != nil:
            IF NOT m.interp.done AND NOT m.interp.error AND NOT m.interp.waitInput:
                m.interp.Step(nil)
                syncInterpreter(&m)
                return m, tickCmd(m.speed)
            ELSE:
                m.autoRunning = false
        return m, nil

    return m, nil
```

### 4.3 View — The Compositing Pipeline

This is the heart of the v2 architecture. Instead of building a single
CellBuffer or joining strings, you build layers and let Canvas composite.

```
FUNCTION View(m Model) string:
    IF m.termW == 0 OR m.termH == 0:
        return ""                    // not sized yet

    layers := []lipgloss.Layer{}

    // ═══ Z=0: Chrome (toolbar, panel, footer) ═══
    layers = append(layers, buildToolbarLayer(m))
    layers = append(layers, buildPanelBorderLayer(m))
    layers = append(layers, buildVarsPanelLayer(m))
    layers = append(layers, buildConsolePanelLayer(m))
    layers = append(layers, buildHelpPanelLayer(m))
    layers = append(layers, buildFooterLayer(m))

    // ═══ Z=0: Edge canvas (grid dots + edge lines) ═══
    layers = append(layers, buildEdgeCanvasLayer(m))

    // ═══ Z=2: Nodes ═══
    FOR each node IN m.nodes:
        IF nodeIsVisible(node, m):
            layers = append(layers, buildNodeLayer(node, m))

    // ═══ Z=3: Edge labels ═══
    FOR each edge IN m.edges:
        IF edge.Label != "":
            layers = append(layers, buildEdgeLabelLayer(edge, m))

    // ═══ Z=5: Connect preview (conditional) ═══
    IF m.connectID != nil:
        layers = append(layers, buildConnectPreviewLayer(m))

    // ═══ Z=10: Input overlay (conditional) ═══
    IF m.waitInput:
        layers = append(layers, buildInputOverlayLayer(m))

    // ═══ Z=100: Edit modal (conditional) ═══
    IF m.editOpen:
        layers = append(layers, buildEditModalLayer(m))

    // ═══ Composite ═══
    m.canvas = lipgloss.NewCanvas(layers...)
    return m.canvas.Render()
```

Note: `m.canvas` is retained on the model so that `handleMouse` can
call `m.canvas.Hit(x, y)` on the **same** canvas that was just rendered.
This is what makes hit testing consistent with rendering by construction.

---

## 5. Node Rendering — Lipgloss Styled Boxes

This is the biggest simplification over the v1 CellBuffer approach.
Instead of manually painting corners, borders, horizontal bars, vertical
bars, clearing interiors, and centering text character by character,
you call Lipgloss and get a styled box.

### 5.1 Style palette

```go
// styles.go

var nodeColors = map[string]struct{ border, text lipgloss.Color }{
    "process":   {lipgloss.Color("#00d4a0"), lipgloss.Color("#00ffc8")},
    "decision":  {lipgloss.Color("#00ccee"), lipgloss.Color("#66ffee")},
    "terminal":  {lipgloss.Color("#44ff88"), lipgloss.Color("#88ffbb")},
    "io":        {lipgloss.Color("#ddaa44"), lipgloss.Color("#ffcc66")},
    "connector": {lipgloss.Color("#1a6a4a"), lipgloss.Color("#00d4a0")},
}

var (
    selBorder = lipgloss.Color("#00ffee")
    selText   = lipgloss.Color("#00ffee")
    selBG     = lipgloss.Color("#0a1a15")
    execBorder= lipgloss.Color("#ffcc00")
    execText  = lipgloss.Color("#ffee66")
    execBG    = lipgloss.Color("#12120a")
    canvasBG  = lipgloss.Color("#080e0b")
)

// Border type per node type
func borderForType(nodeType string) lipgloss.Border {
    switch nodeType {
    case "terminal":
        return lipgloss.RoundedBorder()     // ╭─╮ │ ╰─╯
    case "decision":
        return lipgloss.DoubleBorder()      // ╔═╗ ║ ╚═╝
    default:
        return lipgloss.NormalBorder()      // ┌─┐ │ └─┘
    }
}

// Tag shown in top border
func tagForType(nodeType string) string {
    switch nodeType {
    case "process":   return "[P]"
    case "decision":  return "[?]"
    case "terminal":  return "[T]"
    case "io":        return "[IO]"
    default:          return ""
    }
}
```

### 5.2 buildNodeLayer

```
FUNCTION buildNodeLayer(node FlowNode, m Model) lipgloss.Layer:
    info = NODE_TYPES[node.Type]
    isSelected = (m.selectedID != nil AND *m.selectedID == node.ID)
    isExec     = (m.execNodeID != nil AND *m.execNodeID == node.ID)

    // ── Pick colors ──
    VAR borderColor, textColor lipgloss.Color
    VAR bgColor lipgloss.Color = canvasBG

    IF isExec:
        borderColor = execBorder
        textColor   = execText
        bgColor     = execBG
    ELSE IF isSelected:
        borderColor = selBorder
        textColor   = selText
        bgColor     = selBG
    ELSE:
        colors = nodeColors[node.Type]
        borderColor = colors.border
        textColor   = colors.text

    // ── Build style ──
    border = borderForType(node.Type)
    style := lipgloss.NewStyle().
        Border(border).
        BorderForeground(borderColor).
        Foreground(textColor).
        Background(bgColor).
        Bold(isSelected OR isExec).
        Width(info.W - 2).          // inner width; border adds 2
        Height(info.H - 2).         // inner height; border adds 2
        Align(lipgloss.Center, lipgloss.Center)

    // ── Content ──
    label = node.Text
    IF len(label) > info.W - 4:
        label = label[:info.W-4]

    tag = tagForType(node.Type)
    // NOTE: Lipgloss doesn't natively support text-in-border.
    // Two options:
    //   (a) Prepend tag to the first line of content
    //   (b) Build the border string manually (see §5.3)
    // For simplicity, we'll use option (a):

    content = label
    IF node.Type == "connector" AND node.Text == "":
        content = "○"

    rendered = style.Render(content)

    // If tag needed, overwrite the top-left of the rendered string.
    // This requires ANSI-aware string surgery — see §5.3 for approach.
    IF tag != "":
        rendered = injectTag(rendered, tag, borderColor, bgColor)

    // ── Position ──
    screenX = node.X - m.camX
    screenY = node.Y - m.camY + ToolbarH

    return lipgloss.NewLayer(rendered).
        X(screenX).
        Y(screenY).
        Z(2).
        ID(fmt.Sprintf("node-%d", node.ID))
```

### 5.3 The Tag-in-Border Problem

The Python version renders `[P]`, `[?]`, `[T]`, `[IO]` inside the top
border of each node (e.g., `┌[P]────────────────┐`). Lipgloss's
`Border()` doesn't support injecting text into borders.

**Three approaches, ranked by simplicity:**

**(a) Tag as first content line (recommended for v1):**

Drop the tag into the border entirely. Instead, show it as the first
line inside the box: `[P] ACCUMULATE`. This changes the visual slightly
but simplifies the code enormously. The node height stays at 3 (border +
1 content line + border), and the content becomes `"[P] " + label`.

```
IF tag != "":
    content = tag + " " + label
```

**(b) Custom border rendering:**

Build the top border string manually, then use `lipgloss.NewStyle()`
without a top border, and prepend your custom top row:

```
FUNCTION renderNodeWithTag(node, info, tag, style, borderColor) string:
    border = borderForType(node.Type)
    topLeft  = border.TopLeft
    topRight = border.TopRight
    hBar     = border.Top

    // Build custom top border
    innerW = info.W - 2
    tagLen = len(tag)
    barCount = innerW - tagLen
    topRow = topLeft + tag + strings.Repeat(hBar, barCount) + topRight

    // Style the tag row
    topStyle = lipgloss.NewStyle().
        Foreground(borderColor).
        Background(bgColor)
    styledTop = topStyle.Render(topRow)

    // Render the rest without top border
    bodyStyle = style.
        BorderTop(false)
    styledBody = bodyStyle.Render(content)

    return styledTop + "\n" + styledBody
```

**(c) Post-process the rendered string:**

Render normally, then find the top border in the output string and
splice in the tag. This requires ANSI-aware string manipulation
(using `github.com/muesli/ansi` for truncation). Fragile — not
recommended.

**Recommendation:** Start with (a). It's one line of code. If the visual
matters, graduate to (b) which is ~15 lines.

### 5.4 Visibility Culling

Only create Layers for nodes that are within the visible canvas area.
Layers outside the terminal bounds are harmless (Canvas clips them),
but skipping them saves allocation:

```
FUNCTION nodeIsVisible(node FlowNode, m Model) bool:
    info = NODE_TYPES[node.Type]
    screenX = node.X - m.camX
    screenY = node.Y - m.camY

    // Node's screen rect must overlap the canvas rect [0, canvasW) × [0, canvasH)
    return screenX + info.W > 0 AND screenX < m.canvasW AND
           screenY + info.H > 0 AND screenY < m.canvasH
```

---

## 6. Edge Canvas — The Remaining MiniBuffer

Edges are sparse diagonal lines drawn with Bresenham. They can't be
represented as rectangular Lipgloss boxes. So edges (and grid dots)
still use a character buffer — but it's simpler now because it only
handles 4 style keys instead of 15+.

### 6.1 MiniBuffer

```go
// canvas.go

type StyleKey int
const (
    StyleBG StyleKey = iota
    StyleGrid
    StyleEdge
    StyleEdgeActive
    StyleConnPreview
)

// Pre-built lipgloss styles for the 5 keys
var bufStyles = map[StyleKey]lipgloss.Style{
    StyleBG:           lipgloss.NewStyle().Foreground(lipgloss.Color("#1a3a2a")).Background(lipgloss.Color("#080e0b")),
    StyleGrid:         lipgloss.NewStyle().Foreground(lipgloss.Color("#0e2e20")).Background(lipgloss.Color("#080e0b")),
    StyleEdge:         lipgloss.NewStyle().Foreground(lipgloss.Color("#00d4a0")).Background(lipgloss.Color("#080e0b")),
    StyleEdgeActive:   lipgloss.NewStyle().Foreground(lipgloss.Color("#ffcc00")).Background(lipgloss.Color("#080e0b")).Bold(true),
    StyleConnPreview:  lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffc8")).Background(lipgloss.Color("#080e0b")),
}

type Cell struct {
    Ch    rune
    Style StyleKey
}

type MiniBuffer struct {
    W, H  int
    Cells [][]Cell   // [row][col]
}

FUNCTION NewMiniBuffer(w, h int) *MiniBuffer:
    buf := &MiniBuffer{W: w, H: h}
    buf.Cells = make([][]Cell, h)
    FOR y := 0; y < h; y++:
        buf.Cells[y] = make([]Cell, w)
        FOR x := 0; x < w; x++:
            buf.Cells[y][x] = Cell{' ', StyleBG}
    return buf

FUNCTION (buf *MiniBuffer) Set(x, y int, ch rune, s StyleKey):
    IF x >= 0 AND x < buf.W AND y >= 0 AND y < buf.H:
        buf.Cells[y][x] = Cell{ch, s}

FUNCTION (buf *MiniBuffer) SetString(x, y int, text string, s StyleKey):
    FOR i, ch := range text:
        buf.Set(x+i, y, ch, s)
```

### 6.2 Rendering MiniBuffer to String

Run-length encode consecutive same-styled cells into styled substrings.
Use `lipgloss.StyleRanges` (if available in v2 beta.2) or manual ANSI:

```
FUNCTION (buf *MiniBuffer) Render() string:
    var lines []string

    FOR y := 0; y < buf.H; y++:
        // Build plain-text row
        row := make([]rune, buf.W)
        FOR x := 0; x < buf.W; x++:
            row[x] = buf.Cells[y][x].Ch

        // Build style runs
        var runs []StyledRun   // {start, end int; style StyleKey}
        runStart := 0
        runStyle := buf.Cells[y][0].Style

        FOR x := 1; x <= buf.W; x++:
            curStyle := IF x < buf.W THEN buf.Cells[y][x].Style ELSE StyleKey(-1)
            IF curStyle != runStyle:
                runs = append(runs, StyledRun{runStart, x, runStyle})
                runStart = x
                runStyle = curStyle

        // Render runs into a single styled line
        var sb strings.Builder
        FOR each run IN runs:
            chunk := string(row[run.Start:run.End])
            sb.WriteString(bufStyles[run.Style].Render(chunk))

        lines = append(lines, sb.String())

    return strings.Join(lines, "\n")
```

### 6.3 Drawing Into the MiniBuffer

These are direct ports from the Python `build_buffer()` function, but
they only draw grid dots and edges — **not nodes** (nodes are Layers).

```
FUNCTION drawGrid(buf *MiniBuffer, camX, camY int):
    FOR y := 0; y < buf.H; y++:
        FOR x := 0; x < buf.W; x++:
            worldX := x + camX
            worldY := y + camY
            IF worldX % 5 == 0 AND worldY % 3 == 0:
                buf.Set(x, y, '·', StyleGrid)

FUNCTION drawEdges(buf *MiniBuffer, edges []FlowEdge, nodes []FlowNode,
                   camX, camY int, execNodeID *int):
    nmap := buildNodeMap(nodes)

    FOR each edge IN edges:
        fromNode := nmap[edge.FromID]
        toNode   := nmap[edge.ToID]
        IF fromNode == nil OR toNode == nil: CONTINUE

        active := (execNodeID != nil AND *execNodeID == edge.ToID)
        style  := IF active THEN StyleEdgeActive ELSE StyleEdge

        p1 := getEdgeExit(fromNode, toNode)
        p2 := getEdgeExit(toNode, fromNode)

        // Convert to buffer coords
        bx1, by1 := p1.X - camX, p1.Y - camY
        bx2, by2 := p2.X - camX, p2.Y - camY

        points := bresenham(bx1, by1, bx2, by2)

        FOR i, pt := range points:
            dx, dy := direction(points, i)
            buf.Set(pt.X, pt.Y, lineChar(dx, dy), style)

        // Arrowhead
        IF len(points) >= 2:
            last := points[len(points)-1]
            prev := points[len(points)-2]
            buf.Set(last.X, last.Y,
                    arrowChar(last.X-prev.X, last.Y-prev.Y), style)

FUNCTION drawConnectPreview(buf *MiniBuffer, sourceNode *FlowNode,
                            mouseX, mouseY, camX, camY int):
    cx := int(sourceNode.CX()) - camX
    cy := int(sourceNode.CY()) - camY
    points := bresenham(cx, cy, mouseX, mouseY)
    FOR i, pt := range points:
        IF i % 3 < 2:
            buf.Set(pt.X, pt.Y, '·', StyleConnPreview)
```

### 6.4 Assembling the Edge Canvas Layer

```
FUNCTION buildEdgeCanvasLayer(m Model) lipgloss.Layer:
    buf := NewMiniBuffer(m.canvasW, m.canvasH)

    drawGrid(buf, m.camX, m.camY)
    drawEdges(buf, m.edges, m.nodes, m.camX, m.camY, m.execNodeID)

    // Connect preview goes here too (it's behind nodes at Z=0)
    IF m.connectID != nil:
        sourceNode := nodeByID(m.nodes, *m.connectID)
        IF sourceNode != nil:
            // mouseX/mouseY are already canvas-relative (buffer coords)
            drawConnectPreview(buf, sourceNode, m.mouseX, m.mouseY, m.camX, m.camY)

    return lipgloss.NewLayer(buf.Render()).
        X(0).
        Y(ToolbarH).
        Z(0)
```

**What this replaces from the Python version:** The Python `build_buffer()`
is ~130 lines and draws everything: grid, edges, connect preview, AND all
nodes. The Go MiniBuffer draws only grid + edges + connect preview (~60
lines of drawing code). Nodes are handled entirely by `buildNodeLayer`.

---

## 7. Edge Labels as Layers

Edge labels ("Y", "N") are positioned at the midpoint of each edge.
Making them separate layers at Z=3 ensures they appear above both edges
(Z=0) and nodes (Z=2).

```
FUNCTION buildEdgeLabelLayer(edge FlowEdge, m Model) lipgloss.Layer:
    fromNode := nodeByID(m.nodes, edge.FromID)
    toNode   := nodeByID(m.nodes, edge.ToID)
    IF fromNode == nil OR toNode == nil:
        return emptyLayer()     // shouldn't happen

    p1 := getEdgeExit(fromNode, toNode)
    p2 := getEdgeExit(toNode, fromNode)

    mx := (p1.X + p2.X) / 2
    my := (p1.Y + p2.Y) / 2

    horizontal := abs(p2.X - p1.X) >= abs(p2.Y - p1.Y)

    // Position offset: above the line if horizontal, right of line if vertical
    labelX := mx - len(edge.Label)/2
    labelY := my - 1
    IF NOT horizontal:
        labelX = mx + 1
        labelY = my

    // Convert to screen coords
    screenX := labelX - m.camX
    screenY := labelY - m.camY + ToolbarH

    labelStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#00ffc8")).
        Background(lipgloss.Color("#080e0b")).
        Bold(true)

    return lipgloss.NewLayer(labelStyle.Render(edge.Label)).
        X(screenX).
        Y(screenY).
        Z(3)
```

---

## 8. Hit Testing — Canvas.Hit()

### 8.1 The Mechanism

After `View()` builds the canvas and stores it on `m.canvas`, mouse
events use `m.canvas.Hit(x, y)` to find which layer is under the cursor.
Because layers have IDs (e.g., `"node-7"`, `"btn-run"`, `"edit-modal"`),
you can identify exactly what was clicked.

### 8.2 Mouse Handling

```
FUNCTION handleMouse(m Model, msg tea.MouseMsg) (Model, Cmd):
    // ── Drag in progress ──
    IF m.dragging AND isMotion(msg):
        worldX := msg.X + m.camX
        worldY := msg.Y - ToolbarH + m.camY
        node := nodeByID(m.nodes, m.dragNodeID)
        IF node != nil:
            node.X = worldX - m.dragOffX
            node.Y = worldY - m.dragOffY
        return m, nil

    IF isRelease(msg):
        m.dragging = false
        return m, nil

    // ── Track mouse position (for connect preview) ──
    IF isMotion(msg):
        // Store canvas-relative coords for connect preview drawing
        m.mouseX = msg.X              // these are already buffer-relative
        m.mouseY = msg.Y - ToolbarH   // if edge canvas starts at ToolbarH
        return m, nil

    // ── Press: use hit testing ──
    IF NOT isPress(msg):
        return m, nil

    // ── Hit test against the rendered canvas ──
    VAR hitID string
    IF m.canvas != nil:
        hit := m.canvas.Hit(msg.X, msg.Y)
        IF hit != nil:
            hitID = hit.GetID()

    // ── Route by tool and hit result ──

    // Click in edit modal → don't deselect
    IF strings.HasPrefix(hitID, "edit-"):
        return m, nil

    // Click on toolbar buttons
    IF hitID == "btn-select":
        m.tool = ToolSelect; m.connectID = nil; return m, nil
    IF hitID == "btn-add":
        m.tool = ToolAdd; m.connectID = nil; return m, nil
    IF hitID == "btn-connect":
        m.tool = ToolConnect; m.connectID = nil; return m, nil
    IF hitID == "btn-run":
        return startProgram(m)
    IF hitID == "btn-step":
        return stepProgram(m)

    // ── Canvas area interactions ──
    // Check if click is in canvas region
    IF msg.Y < ToolbarH OR msg.Y >= ToolbarH + m.canvasH OR msg.X >= m.canvasW:
        return m, nil     // click outside canvas, ignore

    worldX := msg.X + m.camX
    worldY := msg.Y - ToolbarH + m.camY

    // ── ADD mode: place new node ──
    IF m.tool == ToolAdd:
        info := NODE_TYPES[m.newNodeType]
        m = addNode(m, m.newNodeType, worldX - info.W/2, worldY - info.H/2)
        m.tool = ToolSelect
        return m, nil

    // ── Hit a node? ──
    nodeID := extractNodeID(hitID)   // "node-7" → 7, or -1 if not a node

    // ── CONNECT mode ──
    IF m.tool == ToolConnect:
        IF nodeID >= 0:
            IF m.connectID == nil:
                m.connectID = &nodeID
            ELSE:
                IF nodeID != *m.connectID:
                    m = addEdge(m, *m.connectID, nodeID)
                m.connectID = nil
                m.tool = ToolSelect
        return m, nil

    // ── SELECT mode ──
    IF nodeID >= 0:
        m.selectedID = &nodeID
        m.dragging = true
        m.dragNodeID = nodeID
        node := nodeByID(m.nodes, nodeID)
        m.dragOffX = worldX - node.X
        m.dragOffY = worldY - node.Y
    ELSE:
        m.selectedID = nil
        m.dragging = false

    return m, nil

// Helper
FUNCTION extractNodeID(layerID string) int:
    IF strings.HasPrefix(layerID, "node-"):
        id, err := strconv.Atoi(layerID[5:])
        IF err == nil: return id
    return -1
```

### 8.3 What This Eliminates

The Python version has a `_hit()` method that loops through all nodes in
reverse order, checking `node.x <= cx < node.x + info.w AND node.y <= cy
< node.y + info.h`. This is ~10 lines of coordinate math that must stay
in sync with how nodes are drawn.

With `Canvas.Hit()`, there is **no separate hit test function**. The
same Layer that is rendered is the same Layer that is hit-tested. If you
move a node's Layer to a new position, the hit test automatically
reflects the new position. Desync is impossible.

### 8.4 Hit Testing for Drag

During drag, mouse motion events change the node's world position. But
you don't use `Canvas.Hit()` during drag — you already know which node
is being dragged (`m.dragNodeID`). You just update its position from the
mouse delta. `Canvas.Hit()` is only used on press events.

---

## 9. Toolbar — Clickable Buttons via Nested Layers

### 9.1 Structure

The toolbar is a parent Layer at Z=0 with child Layers for each button.
Each button has an ID for hit testing.

```
FUNCTION buildToolbarLayer(m Model) lipgloss.Layer:
    // ── Background ──
    bgStyle := lipgloss.NewStyle().
        Width(m.termW).
        Height(ToolbarH).
        Background(lipgloss.Color("#0a1510")).
        BorderBottom(true).
        BorderStyle(lipgloss.NormalBorder()).
        BorderBottomForeground(lipgloss.Color("#1a4a3a"))

    // Title text
    titleStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#00ffc8"))
    subtitleStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#1a4a3a"))

    bgContent := titleStyle.Render("  GRaIL ") +
                 subtitleStyle.Render("FLOWCHART INTERPRETER")

    toolbar := lipgloss.NewLayer(bgStyle.Render(bgContent)).
        X(0).Y(0).Z(0).ID("toolbar")

    // ── Tool buttons (child layers) ──
    activeBtn := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#080e0b")).
        Background(lipgloss.Color("#00d4a0"))
    inactiveBtn := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#00d4a0")).
        Background(lipgloss.Color("#0a1510"))

    btnY := 0   // within toolbar
    col := 26   // after title

    buttons := []struct{ label, id string; active bool }{
        {"[s]SEL",  "btn-select",  m.tool == ToolSelect},
        {"[a]ADD",  "btn-add",     m.tool == ToolAdd},
        {"[c]LINK", "btn-connect", m.tool == ToolConnect},
    }

    FOR each btn IN buttons:
        style := IF btn.active THEN activeBtn ELSE inactiveBtn
        child := lipgloss.NewLayer(style.Render(" "+btn.label+" ")).
            X(col).Y(btnY).ID(btn.id)
        toolbar = toolbar.AddLayers(child)
        col += len(btn.label) + 4

    // ── Run/control buttons ──
    col += 3    // gap
    IF NOT m.running:
        runBtn := lipgloss.NewLayer(
            lipgloss.NewStyle().
                Bold(true).
                Foreground(lipgloss.Color("#44ff88")).
                Background(lipgloss.Color("#0a1510")).
                Render(" [r]▶ RUN "),
        ).X(col).Y(btnY).ID("btn-run")
        toolbar = toolbar.AddLayers(runBtn)
    ELSE:
        // Show step/go/pause/stop buttons
        tag := IF m.autoRunning THEN "▶ AUTO" ELSE "⏸ READY"
        tagStyle := lipgloss.NewStyle().
            Bold(true).
            Foreground(lipgloss.Color("#ffcc00")).
            Background(lipgloss.Color("#0a1510"))
        toolbar = toolbar.AddLayers(
            lipgloss.NewLayer(tagStyle.Render(" "+tag+" ")).X(col).Y(btnY),
        )
        col += len(tag) + 4

        ctrlStyle := lipgloss.NewStyle().
            Foreground(lipgloss.Color("#44ddff")).
            Background(lipgloss.Color("#0a1510"))
        FOR each ctrl IN []struct{ label, id string }{
            {"[n]STEP", "btn-step"},
            {"[g]GO",   "btn-go"},
            {"[p]PAUSE","btn-pause"},
            {"[x]STOP", "btn-stop"},
        }:
            child := lipgloss.NewLayer(ctrlStyle.Render(" "+ctrl.label+" ")).
                X(col).Y(btnY).ID(ctrl.id)
            toolbar = toolbar.AddLayers(child)
            col += len(ctrl.label) + 4

    // ── Selected node indicator ──
    IF m.selectedID != nil:
        node := nodeByID(m.nodes, *m.selectedID)
        IF node != nil:
            selStyle := lipgloss.NewStyle().
                Foreground(lipgloss.Color("#ff8866")).
                Background(lipgloss.Color("#0a1510"))
            indicator := fmt.Sprintf(" [%s] [e]EDIT [d]DEL ",
                strings.ToUpper(node.Type[:1]))
            toolbar = toolbar.AddLayers(
                lipgloss.NewLayer(selStyle.Render(indicator)).X(col+3).Y(btnY),
            )

    return toolbar
```

### 9.2 What This Enables

Toolbar buttons are now **clickable** via `Canvas.Hit()`. In the Python
version, toolbar buttons are just styled text — they only respond to
keyboard shortcuts. In the Go/v2 version, you get mouse-clickable
toolbar buttons for free by giving each button an ID.

---

## 10. Side Panel — Variables, Console, Help

The panel sections are each a Layer at Z=0, positioned to the right of
the canvas.

### 10.1 Variables Panel

```
FUNCTION buildVarsPanelLayer(m Model) lipgloss.Layer:
    headerStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#1a6a4a"))
    nameStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#ffcc66"))
    eqStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#1a4a3a"))
    numStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#44ff88"))
    strStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#88ddff"))

    var sb strings.Builder
    sb.WriteString(headerStyle.Render("📦 VARIABLES") + "\n")

    IF len(m.variables) == 0:
        dimStyle := lipgloss.NewStyle().
            Foreground(lipgloss.Color("#1a4a3a")).Italic(true)
        sb.WriteString(dimStyle.Render("  (none)") + "\n")
    ELSE:
        FOR k, v := range m.variables:
            vs, vc := formatVar(v)
            style := IF vc == "str" THEN strStyle ELSE numStyle
            sb.WriteString("  " + nameStyle.Render(k) +
                          eqStyle.Render("=") +
                          style.Render(vs) + "\n")

    panelStyle := lipgloss.NewStyle().
        Width(PanelW - 1).
        Height(VarsH).
        Background(lipgloss.Color("#050c0a"))

    return lipgloss.NewLayer(panelStyle.Render(sb.String())).
        X(m.canvasW + BorderW).
        Y(ToolbarH).
        Z(0).
        ID("vars-panel")
```

### 10.2 Variables Panel — Alternative: Lipgloss Table

The v2 table API with `BaseStyle` and `StyleFunc` provides a cleaner way
to display variables:

```
FUNCTION buildVarsTable(m Model) string:
    IF len(m.variables) == 0:
        return "(none)"

    var names, values []string
    FOR k, v := range m.variables:
        names = append(names, k)
        values = append(values, formatValue(v))

    rows := make([][]string, len(names))
    FOR i := range names:
        rows[i] = []string{names[i], values[i]}

    t := table.New().
        BaseStyle(lipgloss.NewStyle().Background(lipgloss.Color("#050c0a"))).
        Border(lipgloss.HiddenBorder()).
        Headers("Var", "Value").
        Rows(rows...).
        StyleFunc(func(row, col int) lipgloss.Style {
            IF row == table.HeaderRow:
                return lipgloss.NewStyle().
                    Foreground(lipgloss.Color("#1a6a4a")).Bold(true)
            IF col == 0:
                return lipgloss.NewStyle().Foreground(lipgloss.Color("#ffcc66"))
            return lipgloss.NewStyle().Foreground(lipgloss.Color("#44ff88"))
        })

    return t.Render()
```

This uses `BaseStyle` (new in beta.2) to set the panel background on
the entire table, not just individual cells. It uses `StyleFunc` for
per-cell coloring. The `HiddenBorder()` keeps the table compact.

### 10.3 Console Panel

```
FUNCTION buildConsolePanelLayer(m Model) lipgloss.Layer:
    consoleH := m.canvasH - VarsH - HelpH
    IF consoleH < 3: consoleH = 3

    headerStyle := lipgloss.NewStyle().
        Bold(true).Foreground(lipgloss.Color("#1a6a4a"))

    var sb strings.Builder
    sb.WriteString(headerStyle.Render("🖥️  CONSOLE") + "\n")

    // Show last N lines that fit
    maxLines := consoleH - 2
    start := 0
    IF len(m.consoleLines) > maxLines:
        start = len(m.consoleLines) - maxLines

    FOR i := start; i < len(m.consoleLines); i++:
        line := m.consoleLines[i]
        styled := styleConsoleLine(line)    // color by prefix: ⚠=red, ──=dim, >=yellow, else green
        sb.WriteString("  " + styled + "\n")

    panelStyle := lipgloss.NewStyle().
        Width(PanelW - 1).
        Height(consoleH).
        Background(lipgloss.Color("#050c0a"))

    return lipgloss.NewLayer(panelStyle.Render(sb.String())).
        X(m.canvasW + BorderW).
        Y(ToolbarH + VarsH).
        Z(0).
        ID("console-panel")
```

### 10.4 Help Panel

```
FUNCTION buildHelpPanelLayer(m Model) lipgloss.Layer:
    headerStyle := lipgloss.NewStyle().
        Bold(true).Foreground(lipgloss.Color("#1a6a4a"))
    lineStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#0e4e30"))

    helpLines := []string{
        "Mouse: click=select, drag=move",
        "[s]Select [a]Add [c]Connect",
        "[e]Edit  [d]Delete selected",
        "[r]Run [n]Step [g]Auto [x]Stop",
        "Add mode: [1-5] node type",
        "Arrows: pan canvas",
    }

    var sb strings.Builder
    sb.WriteString(headerStyle.Render("HELP") + "\n")
    FOR each line IN helpLines:
        sb.WriteString("  " + lineStyle.Render(line) + "\n")

    panelStyle := lipgloss.NewStyle().
        Width(PanelW - 1).
        Height(HelpH).
        Background(lipgloss.Color("#050c0a"))

    consoleH := m.canvasH - VarsH - HelpH
    return lipgloss.NewLayer(panelStyle.Render(sb.String())).
        X(m.canvasW + BorderW).
        Y(ToolbarH + VarsH + consoleH).
        Z(0).
        ID("help-panel")
```

---

## 11. Edit Modal — A High-Z Layer

### 11.1 The Problem That Disappeared

In the v1 analysis (§6.3), the modal overlay was identified as "the
hardest problem in Bubbletea rendering" because overlaying styled text
on styled text requires ANSI-aware string splicing.

With v2 Canvas/Layer: you create a Layer at Z=100, positioned in the
center of the screen. Done.

### 11.2 Implementation

```
FUNCTION buildEditModalLayer(m Model) lipgloss.Layer:
    node := nodeByID(m.nodes, m.editNodeID)
    IF node == nil: return emptyLayer()

    info := NODE_TYPES[node.Type]

    titleStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#00d4a0"))
    labelStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#00d4a0"))
    hintStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#1a6a4a")).
        Italic(true)

    hints := map[string]string{
        "process":  "statements separated by ;",
        "decision": "boolean expression",
        "io":       `print("...") or input("prompt", var)`,
    }

    hintText := ""
    IF h, ok := hints[node.Type]; ok:
        hintText = " (" + h + ")"

    // Build modal content using the active textinput sub-models
    content := lipgloss.JoinVertical(lipgloss.Left,
        titleStyle.Render("✏️  EDIT — " + strings.ToUpper(info.Label)),
        "",
        labelStyle.Render("Label:"),
        m.editLabel.View(),
        "",
        labelStyle.Render("Code" + hintText + ":"),
        m.editCode.View(),
        "",
        hintStyle.Render("[tab] switch field  [enter] save  [esc] cancel"),
    )

    boxStyle := lipgloss.NewStyle().
        Border(lipgloss.NormalBorder()).
        BorderForeground(lipgloss.Color("#00d4a0")).
        Background(lipgloss.Color("#0a1510")).
        Width(50).
        Padding(1, 2)

    rendered := boxStyle.Render(content)

    // Center on screen
    // lipgloss.Height/Width count visual dimensions of rendered string
    modalW := lipgloss.Width(rendered)
    modalH := lipgloss.Height(rendered)
    centerX := (m.termW - modalW) / 2
    centerY := (m.termH - modalH) / 2

    return lipgloss.NewLayer(rendered).
        X(centerX).
        Y(centerY).
        Z(100).
        ID("edit-modal")
```

### 11.3 Focus Routing for the Modal

When `m.editOpen` is true, all key events go to the edit modal handler:

```
FUNCTION handleEditKeys(m Model, msg tea.KeyMsg) (Model, Cmd):
    MATCH msg.String():

    "esc":
        m.editOpen = false
        m.focus = FocusCanvas
        return m, nil

    "enter":
        // Save
        node := nodeByID(m.nodes, m.editNodeID)
        IF node != nil:
            node.Text = strings.ToUpper(m.editLabel.Value())
            node.Code = m.editCode.Value()
        m.editOpen = false
        m.focus = FocusCanvas
        return m, nil

    "tab", "shift+tab":
        // Toggle focus between label and code fields
        IF m.focus == FocusEditLabel:
            m.focus = FocusEditCode
            m.editLabel.Blur()
            m.editCode.Focus()
        ELSE:
            m.focus = FocusEditLabel
            m.editCode.Blur()
            m.editLabel.Focus()
        return m, nil

    DEFAULT:
        // Forward to active textinput
        var cmd tea.Cmd
        IF m.focus == FocusEditLabel:
            m.editLabel, cmd = m.editLabel.Update(msg)
        ELSE:
            m.editCode, cmd = m.editCode.Update(msg)
        return m, cmd
```

---

## 12. Interpreter — Goja (JS Runtime)

The interpreter logic is identical to the Python version. The only
change is the eval/exec backend: Python's `eval()`/`exec()` →
Go's Goja JS runtime.

### 12.1 Data Structures

```go
type FlowInterpreter struct {
    nodes       []FlowNode
    edges       []FlowEdge
    vars        map[string]interface{}
    output      []string
    currentID   *int
    done        bool
    err         string
    waitInput   bool
    inputPrompt string
    inputVar    string
    stepCount   int
    maxSteps    int
    runtime     *goja.Runtime
}
```

### 12.2 Construction

```
FUNCTION NewInterpreter(nodes []FlowNode, edges []FlowEdge) *FlowInterpreter:
    interp := &FlowInterpreter{
        nodes:    cloneNodes(nodes),
        edges:    cloneEdges(edges),
        vars:     make(map[string]interface{}),
        maxSteps: 500,
        runtime:  goja.New(),
    }

    // Register print() in the JS runtime
    interp.runtime.Set("print", func(call goja.FunctionCall) goja.Value {
        parts := make([]string, len(call.Arguments))
        for i, arg := range call.Arguments {
            parts[i] = arg.String()
        }
        interp.output = append(interp.output, strings.Join(parts, " "))
        return goja.Undefined()
    })

    // Register str(), int() helpers
    interp.runtime.Set("str", func(call goja.FunctionCall) goja.Value {
        if len(call.Arguments) > 0 {
            return interp.runtime.ToValue(call.Arguments[0].String())
        }
        return interp.runtime.ToValue("")
    })

    return interp
```

### 12.3 Step Function

```
FUNCTION (interp *FlowInterpreter) Step(inputValue *string):
    IF interp.done OR interp.err != "": return

    interp.stepCount++
    IF interp.stepCount > interp.maxSteps:
        interp.err = "MAX STEPS EXCEEDED"
        interp.done = true
        return

    // Handle pending input
    IF interp.waitInput:
        IF inputValue == nil: return
        parsed := parseValue(*inputValue)
        interp.vars[interp.inputVar] = parsed
        interp.runtime.Set(interp.inputVar, parsed)
        interp.output = append(interp.output, "> " + *inputValue)
        interp.waitInput = false
        interp.advance()
        return

    // First step — find START
    IF interp.currentID == nil:
        start := interp.findStart()
        IF start == nil:
            interp.err = "NO START NODE"
            interp.done = true
            return
        interp.currentID = &start.ID
        interp.output = append(interp.output, "── PROGRAM START ──")
        interp.advance()
        return

    node := interp.nodeByID(*interp.currentID)
    IF node == nil:
        interp.err = "BROKEN LINK"
        interp.done = true
        return

    // Execute based on node type
    MATCH node.Type:

    "terminal":
        interp.output = append(interp.output, "── PROGRAM END ──")
        interp.done = true

    "connector":
        interp.advance()

    "process":
        IF node.Code != "":
            err := interp.execStatements(node.Code)
            IF err != nil:
                interp.err = fmt.Sprintf("ERROR at \"%s\": %v", node.Text, err)
                return
        interp.advance()

    "decision":
        result := false
        IF node.Code != "":
            val, err := interp.evalExpr(node.Code)
            IF err != nil:
                interp.err = fmt.Sprintf("ERROR at \"%s\": %v", node.Text, err)
                return
            result = val

        outs := interp.outEdges(*interp.currentID)
        yEdge := findEdgeByLabel(outs, "Y")
        nEdge := findEdgeByLabel(outs, "N")
        var next *FlowEdge
        IF result:
            next = IF yEdge != nil THEN yEdge ELSE firstOrNil(outs)
        ELSE:
            next = IF nEdge != nil THEN nEdge ELSE firstOrNil(outs)

        IF next != nil:
            interp.currentID = &next.ToID
        ELSE:
            interp.currentID = nil
            interp.done = true

    "io":
        code := strings.TrimSpace(node.Code)
        IF isInputCall(code):
            prompt, varName := parseInputCall(code)
            interp.inputPrompt = prompt
            interp.inputVar = varName
            interp.waitInput = true
            interp.output = append(interp.output, prompt)
        ELSE:
            IF code != "":
                err := interp.execStatements(code)
                IF err != nil:
                    interp.err = fmt.Sprintf("ERROR at \"%s\": %v", node.Text, err)
                    return
            interp.advance()
```

### 12.4 Eval/Exec via Goja

```
FUNCTION (interp *FlowInterpreter) syncVarsToRuntime():
    FOR k, v := range interp.vars:
        interp.runtime.Set(k, v)

FUNCTION (interp *FlowInterpreter) execStatements(code string) error:
    interp.syncVarsToRuntime()

    FOR each stmt IN splitSemicolon(code):
        stmt = strings.TrimSpace(stmt)
        IF stmt == "": CONTINUE

        // Check for assignment: varName = expression
        IF m := assignmentRegex.FindStringSubmatch(stmt); m != nil:
            varName := m[1]
            expr := m[2]
            result, err := interp.runtime.RunString(expr)
            IF err != nil: return err
            interp.vars[varName] = result.Export()
            interp.runtime.Set(varName, result)
        ELSE:
            _, err := interp.runtime.RunString(stmt)
            IF err != nil: return err

    return nil

FUNCTION (interp *FlowInterpreter) evalExpr(code string) (bool, error):
    interp.syncVarsToRuntime()
    result, err := interp.runtime.RunString(code)
    IF err != nil: return false, err
    return result.ToBoolean(), nil
```

---

## 13. Timer / Auto-Step

Identical to v1 Bubbletea pattern. Each tick returns the next tick Cmd:

```go
type TickMsg time.Time

func tickCmd(d time.Duration) tea.Cmd {
    return tea.Tick(d, func(t time.Time) tea.Msg {
        return TickMsg(t)
    })
}
```

In Update:
```
TickMsg:
    IF m.autoRunning AND m.interp != nil:
        IF NOT m.interp.done AND m.interp.err == "" AND NOT m.interp.waitInput:
            m.interp.Step(nil)
            syncInterpreter(&m)
            return m, tickCmd(m.speed)
        ELSE:
            m.autoRunning = false
    return m, nil
```

---

## 14. Keyboard Handling

### 14.1 Canvas Keys (default focus)

```
FUNCTION handleCanvasKeys(m Model, msg tea.KeyMsg) (Model, Cmd):
    MATCH msg.String():

    // Tool selection
    "s": m.tool = ToolSelect; m.connectID = nil
    "a": m.tool = ToolAdd;    m.connectID = nil
    "c": m.tool = ToolConnect; m.connectID = nil

    // Node type selection (add mode)
    "1": m.newNodeType = "process"
    "2": m.newNodeType = "decision"
    "3": m.newNodeType = "terminal"
    "4": m.newNodeType = "io"
    "5": m.newNodeType = "connector"

    // Edit selected node
    "e":
        IF m.selectedID != nil:
            node := nodeByID(m.nodes, *m.selectedID)
            IF node != nil:
                m.editOpen = true
                m.editNodeID = node.ID
                m.editLabel.SetValue(node.Text)
                m.editCode.SetValue(node.Code)
                m.editLabel.Focus()
                m.focus = FocusEditLabel

    // Delete selected node
    "d", "delete":
        IF m.selectedID != nil:
            m = deleteNode(m, *m.selectedID)
            m.selectedID = nil

    // Interpreter controls
    "r": return startProgram(m)
    "n": return stepProgram(m)
    "g": return autoRun(m)
    "p": m.autoRunning = false
    "x": return stopProgram(m)

    // Canvas panning
    "up":    m.camY = max(0, m.camY - 2)
    "down":  m.camY += 2
    "left":  m.camX = max(0, m.camX - 3)
    "right": m.camX += 3

    // Escape
    "esc":
        m.tool = ToolSelect
        m.connectID = nil
        m.selectedID = nil

    // Quit
    "q", "ctrl+c":
        return m, tea.Quit

    return m, nil
```

### 14.2 Input Keys (program input mode)

```
FUNCTION handleInputKeys(m Model, msg tea.KeyMsg) (Model, Cmd):
    MATCH msg.String():

    "enter":
        IF m.interp != nil AND m.interp.waitInput:
            val := m.inputField.Value()
            m.inputField.SetValue("")
            m.interp.Step(&val)
            syncInterpreter(&m)
            IF NOT m.waitInput:
                m.focus = FocusCanvas
        return m, nil

    "esc":
        m.focus = FocusCanvas
        return m, nil

    DEFAULT:
        var cmd tea.Cmd
        m.inputField, cmd = m.inputField.Update(msg)
        return m, cmd
```

---

## 15. Background-Aware Theming

Lipgloss v2 provides explicit background detection. For GRaIL, this
enables automatic theme selection: CRT-green on dark terminals, adjusted
colors on light terminals.

```go
// In Init() or main():
func main() {
    hasDarkBG := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
    choose := lipgloss.LightDark(hasDarkBG)

    // Adjust key colors for light/dark
    theme := Theme{
        canvasBG: choose(
            lipgloss.Color("#f0f0f0"),    // light terminal
            lipgloss.Color("#080e0b"),    // dark terminal (CRT green)
        ),
        nodeProcess: choose(
            lipgloss.Color("#008060"),
            lipgloss.Color("#00d4a0"),
        ),
        // ... etc
    }

    m := initialModel(theme)
    p := tea.NewProgram(m,
        tea.WithAltScreen(),
        tea.WithMouseAllMotion(),
    )
    if _, err := p.Run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

In practice, GRaIL's CRT-green aesthetic assumes a dark terminal. You
could skip `LightDark` and hardcode the dark theme. But having the
detection plumbed through means a future light-theme PR is trivial.

For Bubble Tea apps, the v2 docs recommend reacting to
`tea.BackgroundColorMsg` instead of calling `HasDarkBackground` yourself.
This lets Bubble Tea detect the background through its own I/O channel:

```
FUNCTION Update(m Model, msg tea.Msg):
    MATCH msg:
    tea.BackgroundColorMsg:
        m.hasDarkBG = lipgloss.IsDarkColor(msg.Color)
        m.theme = buildTheme(m.hasDarkBG)
        return m, nil
```

---

## 16. Drawing Primitives (Unchanged)

These are pure math — identical across v1 CellBuffer and v2 Canvas
architectures. They're used in the MiniBuffer edge drawing.

### 16.1 Bresenham

```go
func bresenham(x0, y0, x1, y1 int) []image.Point {
    var pts []image.Point
    dx, dy := abs(x1-x0), abs(y1-y0)
    sx, sy := 1, 1
    if x0 > x1 { sx = -1 }
    if y0 > y1 { sy = -1 }
    err := dx - dy
    x, y := x0, y0
    for i := 0; i < dx+dy+2; i++ {
        pts = append(pts, image.Pt(x, y))
        if x == x1 && y == y1 { break }
        e2 := 2 * err
        if e2 > -dy { err -= dy; x += sx }
        if e2 < dx  { err += dx; y += sy }
    }
    return pts
}
```

### 16.2 Line / Arrow Characters

```go
func lineChar(dx, dy int) rune {
    if dx == 0 { return '│' }
    if dy == 0 { return '─' }
    if (dx > 0) == (dy > 0) { return '\\' }
    return '/'
}

func arrowChar(dx, dy int) rune {
    if abs(dy) > abs(dx) {
        if dy > 0 { return '▼' }
        return '▲'
    }
    if dx > 0 { return '►' }
    return '◄'
}
```

### 16.3 Edge Exit Point

```go
func getEdgeExit(node, target *FlowNode) image.Point {
    info := node.Info()
    dx := target.CX() - node.CX()
    dy := target.CY() - node.CY()
    hw, hh := float64(info.W)/2, float64(info.H)/2

    if math.Abs(dx) < 0.01 && math.Abs(dy) < 0.01 {
        return image.Pt(int(node.CX()), int(node.CY()))
    }

    ndx, ndy := 0.0, 0.0
    if hw > 0 { ndx = dx / hw }
    if hh > 0 { ndy = dy / hh }

    if math.Abs(ndx) > math.Abs(ndy) {
        if dx > 0 {
            return image.Pt(node.X+info.W-1, int(math.Round(node.CY())))
        }
        return image.Pt(node.X, int(math.Round(node.CY())))
    }
    if dy > 0 {
        return image.Pt(int(math.Round(node.CX())), node.Y+info.H-1)
    }
    return image.Pt(int(math.Round(node.CX())), node.Y)
}
```

---

## 17. Data Model

```go
// data.go

type NodeTypeInfo struct {
    Label string
    W, H  int
}

var NODE_TYPES = map[string]NodeTypeInfo{
    "process":   {"Process", 22, 3},
    "decision":  {"Decision", 22, 3},
    "terminal":  {"Terminal", 22, 3},
    "io":        {"I/O", 22, 3},
    "connector": {"Connector", 7, 3},
}

type FlowNode struct {
    ID   int
    Type string
    X, Y int
    Text string
    Code string
}

func (n *FlowNode) Info() NodeTypeInfo { return NODE_TYPES[n.Type] }
func (n *FlowNode) CX() float64       { return float64(n.X) + float64(n.Info().W)/2 }
func (n *FlowNode) CY() float64       { return float64(n.Y) + float64(n.Info().H)/2 }

type FlowEdge struct {
    FromID int
    ToID   int
    Label  string
}

func makeInitialNodes() []FlowNode {
    return []FlowNode{
        {1, "terminal",  5, 1,  "START",       ""},
        {2, "process",   4, 5,  "INIT",        "i = 1; sum = 0"},
        {3, "decision",  4, 9,  "i <= 5?",     "i <= 5"},
        {4, "process",   4, 17, "ACCUMULATE",  "sum = sum + i; i = i + 1"},
        {5, "connector", 32, 13, "",           ""},
        {6, "io",        44, 9, "PRINT SUM",   `print("Sum 1..5 = " + str(sum))`},
        {7, "terminal",  46, 14, "END",        ""},
    }
}

func makeInitialEdges() []FlowEdge {
    return []FlowEdge{
        {1, 2, ""},
        {2, 3, ""},
        {3, 4, "Y"},
        {4, 5, ""},
        {5, 3, ""},
        {3, 6, "N"},
        {6, 7, ""},
    }
}
```

---

## 18. File Structure

```
grail-go/
├── main.go              // tea.NewProgram entry point, mouse+altscreen opts
├── model.go             // Model struct, Tool/Focus enums, Init()
├── update.go            // Update() routing, handleMouse, handleCanvasKeys,
│                        //   handleEditKeys, handleInputKeys
├── view.go              // View() — builds all layers, returns canvas.Render()
├── layers.go            // buildToolbarLayer, buildNodeLayer, buildEdgeCanvasLayer,
│                        //   buildVarsPanelLayer, buildConsolePanelLayer,
│                        //   buildHelpPanelLayer, buildEditModalLayer, etc.
├── canvas.go            // MiniBuffer, drawGrid, drawEdges, drawConnectPreview
├── draw.go              // bresenham, lineChar, arrowChar, getEdgeExit
├── interpreter.go       // FlowInterpreter, Step, execStatements, evalExpr (Goja)
├── data.go              // FlowNode, FlowEdge, NodeTypeInfo, initial data
├── styles.go            // Color palette, borderForType, tagForType, bufStyles
└── go.mod               // bubbletea/v2, lipgloss/v2, bubbles/v2, goja
```

---

## 19. Effort Estimate

| Component | Lines | Notes |
|---|---|---|
| **MiniBuffer + edge/grid rendering** | ~80 | Simplified: only edges/grid, 4 styles |
| **Node layers** | ~50 | Lipgloss styled boxes, one function |
| **Toolbar layer** | ~60 | Nested child layers for clickable buttons |
| **Panel layers** (vars + console + help) | ~80 | Lipgloss styled text, optional table |
| **Edge label layers** | ~30 | Small positioned text layers |
| **Edit modal layer** | ~40 | Lipgloss box at Z=100 |
| **View() composition** | ~40 | Build layers, NewCanvas, Render |
| **Mouse handling** | ~80 | Canvas.Hit + tool routing |
| **Keyboard handling** | ~80 | Three focus targets |
| **Interpreter** | ~250 | Goja setup + step logic |
| **Drawing primitives** | ~60 | Bresenham, edge exit, line/arrow chars |
| **Data model + initial data** | ~60 | Structs, constructors |
| **Model + Init + Update routing** | ~80 | Elm architecture glue |
| **Styles** | ~40 | Color palette, border lookup |
| **main.go** | ~20 | Entry point |
| **Total** | **~1050** | |

Compared to the v1 CellBuffer approach (~1500 lines), this is a **30%
reduction**. More importantly, the reduction is concentrated in the
hardest code:

- Node rendering: 80 lines → 50 lines (and much simpler)
- Modal overlay: 80 lines → 40 lines (trivially simple)
- Hit testing: 40 lines → 15 lines (built-in)
- Buffer composition: 100 lines → 40 lines (Canvas does it)

---

## 20. Risks and Mitigations

### Risk 1: Beta API instability

**Problem:** Lipgloss v2.0.0-beta.2 and Bubble Tea v2 are pre-release.
APIs may change before final release.

**Mitigation:** The compositing API (`NewLayer`, `NewCanvas`, `Hit`) is
the headline feature of v2 — unlikely to be removed. Method signatures
may change. Pin exact beta versions in `go.mod`. The app is small enough
that API adjustments are a ~30 minute fix.

### Risk 2: Layer opacity (spaces occlude)

**Problem:** Every character within a Layer's bounds is opaque, including
spaces. Node layers (Z=2) will occlude edge lines that pass through
node areas.

**Mitigation:** This is **desired behavior** — it's exactly how the
Python version works (nodes drawn last, overwriting edge characters).
The React/SVG original also has nodes occluding edges via z-index.

### Risk 3: Performance with many layers

**Problem:** Each node is a separate Layer. With 50+ nodes, you're
creating 50+ layers per frame.

**Mitigation:** Canvas compositing is string concatenation, not pixel
blending. 50 layers × ~66 characters each is ~3300 characters to
composite — trivial. Real-world flowcharts rarely exceed 20-30 nodes.
Visibility culling (§5.4) skips off-screen nodes.

### Risk 4: Bubble Tea v2 required

**Problem:** Lip Gloss v2 compositing only works with Bubble Tea v2.
You're committing to the v2 beta stack.

**Mitigation:** For a new project, this is fine. There's no v1 code to
migrate. If the betas regress badly, you can fall back to the CellBuffer
approach from doc 01 — the data model and interpreter are stack-agnostic.

### Risk 5: Mouse motion flooding

**Problem:** `tea.WithMouseAllMotion()` generates events on every cursor
movement, even without buttons held. Needed for connect-mode preview.

**Mitigation:** Throttle in Update: skip if same cell as last event.
The MiniBuffer + layer composition is fast enough (~1ms) that even
unthrottled motion at 60Hz is fine.

### Risk 6: Canvas.Hit coordinate system

**Problem:** `Canvas.Hit(x, y)` uses the Canvas's coordinate system.
If your Canvas starts at (0,0) of the terminal (which it does — it's
the full-screen canvas), then `msg.X` and `msg.Y` from Bubble Tea
mouse events map directly. But you need to verify this assumption.

**Mitigation:** Write a small test program first:

```go
func main() {
    a := lipgloss.NewLayer("CLICK ME").X(10).Y(5).Z(0).ID("target")
    c := lipgloss.NewCanvas(a)
    // Verify: c.Hit(10, 5) should return the "target" layer
    // Verify: c.Hit(0, 0) should return nil
}
```

If Bubble Tea mouse coordinates are 0-indexed (they are), and Canvas
layers use 0-indexed positions (they do per the API), this should work
directly. The negative-coordinate canvas fix in beta.2 suggests the
coordinate system is robust.

---

## 21. Summary: The Three Columns

| Aspect | React/SVG (`reference.jsx`) | Textual (`grail.py`) | **Go + Lipgloss v2 Canvas** |
|---|---|---|---|
| **Node rendering** | CSS-styled `<div>` with absolute position | CellBuffer: paint chars one by one | **Lipgloss styled box → Layer at X/Y** |
| **Edge rendering** | SVG `<line>` elements | CellBuffer: Bresenham chars | MiniBuffer: Bresenham chars (same) |
| **Stacking** | CSS `z-index` | Draw order (edges first, nodes last) | **Layer.Z() — explicit z-index** |
| **Hit testing** | DOM `onClick` per element | Manual point-in-rect loop | **Canvas.Hit(x,y) + Layer.ID()** |
| **Modal** | React overlay component | Textual ModalScreen | **Layer at Z=100** |
| **Layout** | CSS flexbox | Textual CSS (Yoga) | Manual arithmetic + Layer.X/Y |
| **Theming** | CSS variables | Rich.Style objects | Lipgloss styles + LightDark() |
| **Eval** | `new Function(code)` | `eval()`/`exec()` | Goja JS runtime |

The Go + Lipgloss v2 column is architecturally **closer to the React
original** than the Python/Textual port is. The compositing model
(positioned layers with z-index, hit testing by ID) maps almost 1:1
to the DOM model. The only area where it falls back to low-level
character work is edge rendering, which is inherently character-level
in a terminal.
