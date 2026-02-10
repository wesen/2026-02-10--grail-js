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
      Note: Existing Textual implementation — reference for feature parity
    - Path: reference.jsx
      Note: Original React/SVG source
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Bubbletea Port Analysis — GRaIL Flowchart Editor

## Executive Summary

This document analyzes what it takes to port the GRaIL flowchart editor from
Python/Textual to Go/Bubbletea. Bubbletea gives you less out of the box:
no layout engine, no widget tree, no per-character styling API, no modal
screens. You are handed `View() string` and told to build everything from
raw strings and ANSI escapes. The upside is total control, a single binary,
and Go's type safety.

The core challenge isn't the interpreter (trivial Go port) or the data model
(straightforward structs). It's the **rendering pipeline**: you must build a
character-cell canvas with per-cell styling, composite it into styled strings
line-by-line, and manage layout arithmetic that Textual does automatically.

Estimated effort: ~1500–2000 lines of Go (vs ~700 lines of Python), mostly
because you're writing the layout engine, canvas renderer, and input widget
by hand.

---

## 1. Architecture: Elm vs. Textual's Widget Tree

### Textual model (what we have)

Textual gives you a **widget tree** with independent render cycles:

```
App
├── Static (toolbar)
├── Horizontal
│   ├── FlowCanvas (custom widget, render_line per row)
│   └── Vertical (panel)
│       ├── Static (vars)
│       ├── VerticalScroll > Static (console)
│       ├── Input (prog-input, hidden by default)
│       └── Static (help)
└── Footer
```

Each widget manages its own rendering. The framework handles layout (CSS
flexbox via Yoga), focus, scrolling, and event routing. `FlowCanvas` overrides
`render_line(y) → Strip` for per-character control.

### Bubbletea model (what you'd build)

Bubbletea is **one model, one Update, one View**. There is no widget tree.
The entire UI is a single `View() string`. You compose it from sub-models
manually:

```
Model {
    // Sub-models (each with their own Update/View)
    toolbar     ToolbarModel
    canvas      CanvasModel
    varsPanel   StaticTextModel
    console     ViewportModel      // from bubbles
    helpPanel   StaticTextModel
    inputField  textinput.Model    // from bubbles
    
    // Shared state
    nodes       []FlowNode
    edges       []FlowEdge
    interp      *FlowInterpreter
    
    // UI state
    tool        Tool
    selectedID  *int
    execNodeID  *int
    ...
    
    // Layout (computed on WindowSizeMsg)
    width       int
    height      int
    canvasW     int
    canvasH     int
    panelW      int
}
```

The key difference: **you are the layout engine**. On every `WindowSizeMsg`,
you recalculate `canvasW`, `canvasH`, `panelW`, and pass sizes down to
sub-models.

### Pseudocode: Top-level Elm architecture

```
FUNCTION Init():
    nodes = makeInitialNodes()
    edges = makeInitialEdges()
    return model, WindowSize()    // request initial terminal size

FUNCTION Update(model, msg):
    MATCH msg:
        WindowSizeMsg(w, h):
            model.width = w
            model.height = h
            model.panelW = 34
            model.canvasW = w - model.panelW - 1   // -1 for border
            model.canvasH = h - 4                    // toolbar(3) + footer(1)
            // Resize sub-models
            model.console.Width = model.panelW - 2
            model.console.Height = model.canvasH - varsHeight - helpHeight
            
        KeyMsg(key):
            IF model.mode == EDITING:
                model.inputField, cmd = model.inputField.Update(msg)
            ELSE:
                MATCH key:
                    's': model.tool = SELECT
                    'a': model.tool = ADD
                    'c': model.tool = CONNECT
                    'r': startInterpreter(model)
                    ...
            
        MouseMsg(event):
            // Translate absolute coords to canvas-relative
            canvasX = event.X
            canvasY = event.Y - 3  // subtract toolbar height
            IF canvasY >= 0 AND canvasX < model.canvasW:
                handleCanvasMouse(model, canvasX, canvasY, event)
            
        TickMsg:
            IF model.autoRunning:
                model.interp.Step()
                syncInterpreterState(model)
                return model, tickCmd(model.speed)
    
    return model, cmd

FUNCTION View(model):
    toolbar = renderToolbar(model)
    canvas  = renderCanvas(model)
    panel   = renderPanel(model)
    body    = joinHorizontal(canvas, "│", panel)
    footer  = renderFooter(model)
    return joinVertical(toolbar, body, footer)
```

---

## 2. The Canvas Renderer — The Hard Part

This is where 60% of the effort goes. Textual's `render_line(y) → Strip`
gives you per-character, per-style control. Bubbletea's `View() string`
gives you a flat string with embedded ANSI escapes. You must build an
equivalent system.

### 2.1 The Cell Buffer

You need a 2D grid where each cell holds a character and a style. This
is the foundation of everything.

```
STRUCT Cell:
    char    rune       // the character to display
    fg      Color      // foreground color (#RRGGBB or ANSI)
    bg      Color      // background color
    bold    bool
    
STRUCT CellBuffer:
    width   int
    height  int
    cells   [][]Cell    // [row][col]
    
FUNCTION NewCellBuffer(w, h):
    buf = CellBuffer{width: w, height: h}
    buf.cells = make([][]Cell, h)
    FOR each row:
        buf.cells[row] = make([]Cell, w)
        FOR each col:
            buf.cells[row][col] = Cell{char: ' ', fg: DIM_GREEN, bg: BG_BLACK}
    return buf

FUNCTION (buf) Set(x, y, ch, fg, bg, bold):
    IF 0 <= x < buf.width AND 0 <= y < buf.height:
        buf.cells[y][x] = Cell{ch, fg, bg, bold}

FUNCTION (buf) SetString(x, y, text, fg, bg, bold):
    FOR i, ch IN text:
        buf.Set(x + i, y, ch, fg, bg, bold)
```

### 2.2 Rendering the Buffer to a String

This is the critical function. You must walk the buffer row by row,
and within each row, group consecutive cells with the same style into
runs to minimize ANSI escape overhead.

**Why grouping matters:** If you emit a separate `\033[38;2;R;G;Bm` for
every single character, the output is ~20× larger than necessary. Grouping
consecutive same-styled cells into runs keeps the string compact and
rendering fast.

```
FUNCTION (buf) Render() string:
    var sb StringBuilder
    
    FOR y = 0; y < buf.height; y++:
        IF y > 0:
            sb.WriteString("\n")
        
        // Run-length encode by style
        runStart = 0
        currentStyle = buf.cells[y][0].style()
        
        FOR x = 1; x <= buf.width; x++:
            cellStyle = IF x < buf.width THEN buf.cells[y][x].style() ELSE nil
            
            IF cellStyle != currentStyle:
                // Emit the accumulated run
                ansi = styleToANSI(currentStyle)
                sb.WriteString(ansi)
                FOR i = runStart; i < x; i++:
                    sb.WriteRune(buf.cells[y][i].char)
                sb.WriteString(RESET)
                
                runStart = x
                currentStyle = cellStyle
    
    return sb.String()
```

### 2.3 Alternative: Use Lipgloss for Cell Styling

Instead of raw ANSI escapes, you *can* use Lipgloss per cell, but it's
expensive. A better approach: pre-build Lipgloss styles for your palette
(~10 styles), then use `lipgloss.StyleRanges()` to style substrings
within each row.

```
FUNCTION (buf) RenderWithLipgloss() string:
    var lines []string
    
    FOR y = 0; y < buf.height; y++:
        // Build plain-text row
        var rowChars []rune
        FOR x = 0; x < buf.width; x++:
            rowChars = append(rowChars, buf.cells[y][x].char)
        plainRow = string(rowChars)
        
        // Build style ranges
        var ranges []lipgloss.Range
        runStart = 0
        currentStyleKey = buf.cells[y][0].styleKey()
        
        FOR x = 1; x <= buf.width; x++:
            key = IF x < buf.width THEN buf.cells[y][x].styleKey() ELSE ""
            IF key != currentStyleKey:
                ranges = append(ranges, lipgloss.Range{
                    Start: runStart, End: x,
                    Style: styleLookup[currentStyleKey],
                })
                runStart = x
                currentStyleKey = key
        
        styledRow = lipgloss.StyleRanges(plainRow, ranges...)
        lines = append(lines, styledRow)
    
    return strings.Join(lines, "\n")
```

**Performance note:** `StyleRanges` was added in Lipgloss v1.x. For older
versions, you'd need to manually splice styled substrings, which is more
tedious. The raw ANSI approach (§2.2) is simpler and faster if you control
the palette.

### 2.4 Style Palette

Pre-compute all styles up front. The Textual version uses ~15 distinct
style combinations. Map them to an enum:

```
ENUM StyleKey:
    BG, GRID, 
    NODE_PROCESS, NODE_DECISION, NODE_TERMINAL, NODE_IO, NODE_CONNECTOR,
    TEXT_PROCESS, TEXT_DECISION, TEXT_TERMINAL, TEXT_IO, TEXT_CONNECTOR,
    SELECTED_BORDER, SELECTED_TEXT,
    EXEC_BORDER, EXEC_TEXT,
    EDGE_NORMAL, EDGE_ACTIVE, EDGE_LABEL,
    CONNECT_PREVIEW

// Pre-built lipgloss styles
styleLookup = map[StyleKey]lipgloss.Style{
    BG:              lipgloss.NewStyle().Foreground(color("#1a3a2a")).Background(color("#080e0b")),
    NODE_PROCESS:    lipgloss.NewStyle().Foreground(color("#00d4a0")).Background(color("#080e0b")),
    SELECTED_BORDER: lipgloss.NewStyle().Foreground(color("#00ffee")).Background(color("#0a1a15")).Bold(true),
    ...
}
```

---

## 3. Drawing Primitives

These are pure functions that write into the `CellBuffer`. They port
almost directly from the Python version.

### 3.1 Grid Background

```
FUNCTION drawGrid(buf, camX, camY):
    FOR y = 0; y < buf.height; y++:
        FOR x = 0; x < buf.width; x++:
            worldX = x + camX
            worldY = y + camY
            IF worldX % 5 == 0 AND worldY % 3 == 0:
                buf.Set(x, y, '·', GRID)
```

### 3.2 Bresenham Line Drawing

Identical to the Python version. Go doesn't have generators, so return
a slice.

```
FUNCTION bresenham(x0, y0, x1, y1) []Point:
    points = []Point{}
    dx, dy = abs(x1-x0), abs(y1-y0)
    sx = IF x0 < x1 THEN 1 ELSE -1
    sy = IF y0 < y1 THEN 1 ELSE -1
    err = dx - dy
    x, y = x0, y0
    
    LOOP at most (dx+dy+2) times:
        points.append(Point{x, y})
        IF x == x1 AND y == y1: BREAK
        e2 = 2 * err
        IF e2 > -dy: err -= dy; x += sx
        IF e2 < dx:  err += dx; y += sy
    
    return points

FUNCTION lineChar(dx, dy) rune:
    IF dx == 0: return '│'
    IF dy == 0: return '─'
    IF (dx > 0) == (dy > 0): return '\\'
    return '/'

FUNCTION arrowChar(dx, dy) rune:
    IF abs(dy) > abs(dx):
        return IF dy > 0 THEN '▼' ELSE '▲'
    return IF dx > 0 THEN '►' ELSE '◄'
```

### 3.3 Edge Exit Point Calculation

```
FUNCTION getEdgeExit(node, target) Point:
    dx = target.CX() - node.CX()
    dy = target.CY() - node.CY()
    hw = node.Info().W / 2.0
    hh = node.Info().H / 2.0
    
    ndx = dx / hw IF hw > 0 ELSE 0
    ndy = dy / hh IF hh > 0 ELSE 0
    
    IF abs(ndx) > abs(ndy):
        // Exit left or right
        IF dx > 0: return Point{node.X + node.Info().W - 1, round(node.CY())}
        ELSE:      return Point{node.X,                      round(node.CY())}
    ELSE:
        // Exit top or bottom
        IF dy > 0: return Point{round(node.CX()), node.Y + node.Info().H - 1}
        ELSE:      return Point{round(node.CX()), node.Y}
```

### 3.4 Node Rendering

```
FUNCTION drawNode(buf, node, borderStyle, textStyle, camX, camY):
    // Convert world coords to buffer coords
    x = node.X - camX
    y = node.Y - camY
    w = node.Info().W
    h = node.Info().H
    
    // Choose border characters based on node type
    tl, tr, bl, br, hch, vch = borderCharsForType(node.Type)
    //   terminal:  ╭ ╮ ╰ ╯ ─ │
    //   decision:  ╔ ╗ ╚ ╝ ═ ║
    //   default:   ┌ ┐ └ ┘ ─ │
    
    tag = tagForType(node.Type)  // "[P]", "[?]", "[T]", "[IO]", ""
    
    // Draw corners
    buf.Set(x,     y,     tl, borderStyle)
    buf.Set(x+w-1, y,     tr, borderStyle)
    buf.Set(x,     y+h-1, bl, borderStyle)
    buf.Set(x+w-1, y+h-1, br, borderStyle)
    
    // Draw horizontal borders + tag
    FOR c = x+1; c < x+w-1; c++:
        buf.Set(c, y,     hch, borderStyle)
        buf.Set(c, y+h-1, hch, borderStyle)
    IF tag != "":
        buf.SetString(x+2, y, tag, borderStyle)
    
    // Draw vertical borders
    FOR r = y+1; r < y+h-1; r++:
        buf.Set(x,     r, vch, borderStyle)
        buf.Set(x+w-1, r, vch, borderStyle)
    
    // Clear interior + draw centered text
    FOR r = y+1; r < y+h-1; r++:
        FOR c = x+1; c < x+w-1; c++:
            buf.Set(c, r, ' ', textStyle)
    midY = y + h/2
    label = truncate(node.Text, w-4)
    tx = x + (w - len(label)) / 2
    buf.SetString(tx, midY, label, textStyle)
    
    // Connector: draw ○ if no text
    IF node.Type == CONNECTOR AND node.Text == "":
        buf.Set(x + w/2, y + h/2, '○', textStyle)
```

### 3.5 Full Canvas Build

```
FUNCTION buildCanvas(model) CellBuffer:
    buf = NewCellBuffer(model.canvasW, model.canvasH)
    
    // Layer 1: Grid dots
    drawGrid(buf, model.camX, model.camY)
    
    // Layer 2: Edges (drawn first, nodes overwrite)
    FOR each edge IN model.edges:
        fromNode = nodeByID(edge.FromID)
        toNode   = nodeByID(edge.ToID)
        
        active = (model.execNodeID == edge.ToID)
        style  = IF active THEN EDGE_ACTIVE ELSE EDGE_NORMAL
        
        p1 = getEdgeExit(fromNode, toNode)
        p2 = getEdgeExit(toNode, fromNode)
        
        // Convert to buffer coords
        bp1 = Point{p1.X - camX, p1.Y - camY}
        bp2 = Point{p2.X - camX, p2.Y - camY}
        
        points = bresenham(bp1.X, bp1.Y, bp2.X, bp2.Y)
        
        FOR i, pt IN points:
            dx, dy = direction(points, i)
            buf.Set(pt.X, pt.Y, lineChar(dx, dy), style)
        
        // Arrowhead at destination
        IF len(points) >= 2:
            last = points[len-1]
            prev = points[len-2]
            buf.Set(last.X, last.Y, arrowChar(last.X-prev.X, last.Y-prev.Y), style)
        
        // Edge label at midpoint
        IF edge.Label != "":
            mx = (bp1.X + bp2.X) / 2
            my = (bp1.Y + bp2.Y) / 2
            horizontal = abs(bp2.X-bp1.X) >= abs(bp2.Y-bp1.Y)
            IF horizontal:
                buf.SetString(mx, my-1, edge.Label, EDGE_LABEL)
            ELSE:
                buf.SetString(mx+1, my, edge.Label, EDGE_LABEL)
    
    // Layer 3: Connect preview (dashed line to mouse)
    IF model.connectingID != nil:
        cn = nodeByID(*model.connectingID)
        points = bresenham(cn.CX()-camX, cn.CY()-camY, model.mouseX, model.mouseY)
        FOR i, pt IN points:
            IF i % 3 < 2:
                buf.Set(pt.X, pt.Y, '·', CONNECT_PREVIEW)
    
    // Layer 4: Nodes (on top of everything)
    FOR each node IN model.nodes:
        borderStyle, textStyle = stylesForNode(node, model.selectedID, model.execNodeID)
        drawNode(buf, node, borderStyle, textStyle, model.camX, model.camY)
    
    return buf
```

---

## 4. Layout Engine (Manual)

Textual handles this with CSS. In Bubbletea, you compute it yourself on
every `WindowSizeMsg`.

```
CONSTANTS:
    TOOLBAR_H  = 3    // title line + content + border
    FOOTER_H   = 1
    PANEL_W    = 34
    VARS_H     = 6    // max height for variables section
    HELP_H     = 8    // max height for help section
    BORDER_W   = 1    // the │ between canvas and panel

FUNCTION recalcLayout(model, termW, termH):
    model.width   = termW
    model.height  = termH
    model.canvasW = termW - PANEL_W - BORDER_W
    model.canvasH = termH - TOOLBAR_H - FOOTER_H
    model.panelW  = PANEL_W
    
    // Console gets whatever vertical space remains in the panel
    model.consoleH = model.canvasH - VARS_H - HELP_H - inputFieldH(model)
    IF model.consoleH < 3: model.consoleH = 3
    
    // Resize the viewport sub-model for console scrolling
    model.consoleViewport.Width  = PANEL_W - 2
    model.consoleViewport.Height = model.consoleH
```

### Composing the final View

```
FUNCTION View(model) string:
    // 1. Toolbar (full width)
    toolbar = renderToolbar(model)
    toolbarBox = lipgloss.NewStyle().
        Width(model.width).
        Height(TOOLBAR_H).
        BorderBottom(true).
        BorderStyle(lipgloss.NormalBorder()).
        BorderForeground(color("#1a4a3a")).
        Render(toolbar)
    
    // 2. Canvas (rendered from CellBuffer)
    canvasBuf = buildCanvas(model)
    canvasStr = canvasBuf.Render()   // or RenderWithLipgloss()
    
    // 3. Panel sections
    varsStr    = renderVarsPanel(model)
    consoleStr = model.consoleViewport.View()
    inputStr   = IF model.waitingInput THEN model.inputField.View() ELSE ""
    helpStr    = renderHelpPanel(model)
    
    panelStr = lipgloss.JoinVertical(lipgloss.Top,
        varsStr,
        consoleStr,
        inputStr,
        helpStr,
    )
    
    // Ensure panel is exactly the right size
    panelBox = lipgloss.NewStyle().
        Width(PANEL_W).
        Height(model.canvasH).
        BorderLeft(true).
        Render(panelStr)
    
    // 4. Body = canvas + panel
    body = lipgloss.JoinHorizontal(lipgloss.Top, canvasStr, panelBox)
    
    // 5. Footer
    footer = renderFooter(model)
    
    // 6. Stack vertically
    return lipgloss.JoinVertical(lipgloss.Top, toolbarBox, body, footer)
```

**Critical gotcha:** `lipgloss.JoinHorizontal` pads shorter strings with
spaces. If your canvas string has 20 lines and your panel has 25 lines,
the canvas will be padded. You must ensure both sides have exactly the
same number of lines (`model.canvasH`). Pad the canvas or truncate the
panel accordingly.

---

## 5. Mouse Handling

### 5.1 Enabling Mouse

```go
// In main():
p := tea.NewProgram(initialModel(),
    tea.WithAltScreen(),
    tea.WithMouseCellMotion(),  // enables mouse tracking
)
```

`WithMouseCellMotion()` sends `MouseMsg` for presses, releases, motion
(when button held), and scroll. `WithMouseAllMotion()` sends motion even
without button held (needed for connect-mode preview line), but generates
much more traffic.

### 5.2 Coordinate Translation

Mouse coordinates in Bubbletea are **absolute terminal coordinates** (0,0
is top-left of terminal). You must manually subtract the toolbar height
and check bounds:

```
FUNCTION handleMouse(model, msg MouseMsg):
    absX, absY = msg.X, msg.Y
    
    // Is it in the canvas region?
    canvasScreenTop  = TOOLBAR_H
    canvasScreenLeft = 0
    
    relX = absX - canvasScreenLeft
    relY = absY - canvasScreenTop
    
    IF relX < 0 OR relX >= model.canvasW: return   // outside canvas
    IF relY < 0 OR relY >= model.canvasH: return   // outside canvas
    
    // Convert to world (canvas) coordinates
    worldX = relX + model.camX
    worldY = relY + model.camY
    
    MATCH msg.Action:
        MouseActionPress:
            IF msg.Button == MouseButtonLeft:
                hit = hitTestNode(model.nodes, worldX, worldY)
                // ... same logic as Textual version
        
        MouseActionMotion:
            model.mouseX = relX    // for connect preview (buffer coords)
            model.mouseY = relY
            IF model.dragging:
                node = nodeByID(model.dragID)
                node.X = worldX - model.dragOffX
                node.Y = worldY - model.dragOffY
        
        MouseActionRelease:
            model.dragging = false
```

### 5.3 The MouseAllMotion Problem

For the connect-mode preview line (dashed line from source node to cursor),
you need mouse position even when no button is held. This requires
`tea.WithMouseAllMotion()`, which floods `Update()` with motion events on
every cursor movement. This can cause performance issues.

**Mitigation strategies:**
1. Only use `WithMouseAllMotion()` — accept the traffic, keep `Update` cheap
2. Throttle: skip re-rendering if the mouse hasn't moved enough (e.g., same cell)
3. Only track motion when in connect mode (can't change the option dynamically,
   so you'd filter in `Update`)

```
FUNCTION Update(model, msg):
    MATCH msg:
        MouseMsg:
            // Throttle: skip if same cell as last event
            IF msg.X == model.lastMouseX AND msg.Y == model.lastMouseY:
                return model, nil
            model.lastMouseX = msg.X
            model.lastMouseY = msg.Y
            handleMouse(model, msg)
```

---

## 6. Input Handling and Focus

### 6.1 The Problem

In Textual, focus routing is automatic: the focused widget gets key events.
In Bubbletea, **all keys go to one Update function**. You must manually
route them based on app state.

```
ENUM FocusTarget:
    CANVAS      // arrow keys pan, letters are tool shortcuts
    INPUT       // typing goes to textinput.Model
    EDIT_LABEL  // typing goes to edit field 1
    EDIT_CODE   // typing goes to edit field 2

FUNCTION Update(model, msg):
    MATCH msg:
        KeyMsg:
            MATCH model.focus:
                CANVAS:
                    // Handle tool shortcuts, movement, etc.
                    MATCH key:
                        's': model.tool = SELECT
                        'e': IF model.selectedID != nil: openEditModal(model)
                        'enter': IF model.focus == INPUT: submitInput(model)
                        ...
                
                INPUT:
                    // Forward to textinput
                    IF key == 'escape':
                        model.focus = CANVAS
                    ELSE IF key == 'enter':
                        submitInput(model)
                    ELSE:
                        model.inputField, cmd = model.inputField.Update(msg)
                
                EDIT_LABEL, EDIT_CODE:
                    // Forward to the active edit field
                    IF key == 'escape':
                        closeEditModal(model)
                    ELSE IF key == 'tab':
                        toggleEditField(model)
                    ELSE IF key == 'enter' AND model.focus == EDIT_CODE:
                        commitEdit(model)
                    ELSE:
                        activeField(model).Update(msg)
```

### 6.2 The Edit Modal

Textual has `ModalScreen`. Bubbletea has nothing built-in. You implement
modals as a **state flag** that changes what `View()` renders.

```
FUNCTION View(model) string:
    base = renderNormalView(model)    // toolbar + canvas + panel
    
    IF model.editModalOpen:
        overlay = renderEditModal(model)
        return overlayOnTop(base, overlay, model.width, model.height)
    
    return base

FUNCTION overlayOnTop(base, overlay, termW, termH) string:
    // Split base into lines
    baseLines = strings.Split(base, "\n")
    // Pad to termH
    FOR len(baseLines) < termH:
        baseLines = append(baseLines, strings.Repeat(" ", termW))
    
    // Calculate overlay position (centered)
    overlayLines = strings.Split(overlay, "\n")
    overlayW = maxLineWidth(overlayLines)
    overlayH = len(overlayLines)
    startX = (termW - overlayW) / 2
    startY = (termH - overlayH) / 2
    
    // Splice overlay into base
    // NOTE: This is character-position-based splicing within ANSI-escaped
    // strings. This is HARD. See §6.3 for why.
    FOR i, overlayLine IN overlayLines:
        y = startY + i
        IF y >= 0 AND y < termH:
            baseLines[y] = spliceANSI(baseLines[y], startX, overlayLine)
    
    return strings.Join(baseLines, "\n")
```

### 6.3 The ANSI Splicing Problem

Overlaying styled text onto styled text is the **hardest problem** in
Bubbletea rendering. When your base string is `\033[32mHello World\033[0m`
and you want to insert an overlay at column 6, you can't just do
`base[:6] + overlay + base[6+overlayLen:]` because byte position 6 is in
the middle of an ANSI escape sequence, not at character column 6.

**Solutions (from simplest to best):**

1. **Don't overlay.** Render the modal as a full-screen replacement. When
   the edit modal is open, `View()` returns only the modal, not the
   canvas underneath. Simple, but you lose the visual context.

2. **Use the CellBuffer for everything.** Render the modal into the same
   CellBuffer as the canvas (as a rectangle of cells in the center). Then
   render the entire buffer once. This is the cleanest approach and avoids
   ANSI splicing entirely.

3. **Use a library.** `github.com/muesli/ansi` has functions for
   ANSI-aware string cutting. Or use `lipgloss.Place()` which handles
   some of this.

**Recommended approach:** Option 2 — extend the CellBuffer to cover the
full terminal, not just the canvas. Render toolbar, panel, and modal all
into the same buffer. Then `Render()` once.

```
FUNCTION View(model) string:
    // Full-screen buffer
    buf = NewCellBuffer(model.width, model.height)
    
    // Draw toolbar region (rows 0..2)
    drawToolbar(buf, model)
    
    // Draw canvas region (rows 3..3+canvasH-1, cols 0..canvasW-1)
    drawCanvasInto(buf, model, 0, TOOLBAR_H, model.canvasW, model.canvasH)
    
    // Draw panel border (col canvasW, rows 3..end)
    FOR y = TOOLBAR_H; y < TOOLBAR_H + model.canvasH; y++:
        buf.Set(model.canvasW, y, '│', BORDER_STYLE)
    
    // Draw panel sections (cols canvasW+1..end, rows 3..end)
    drawPanelInto(buf, model, model.canvasW + 1, TOOLBAR_H)
    
    // Draw footer (last row)
    drawFooter(buf, model)
    
    // Overlay: edit modal (if open)
    IF model.editModalOpen:
        drawEditModal(buf, model)   // writes directly into the buffer
    
    return buf.Render()
```

This "full-screen CellBuffer" approach eliminates all layout-joining
issues, all ANSI-splicing issues, and makes overlays trivial. The cost
is that **everything** goes through the buffer — even text panels. But
for a fixed-size terminal app, this is the simplest and most robust
approach.

---

## 7. The Interpreter

This is the easiest part of the port. The interpreter is pure logic with
no UI dependencies.

### 7.1 Go vs Python Execution

The Python version uses `eval()` and `exec()` for code execution. Go
has no built-in eval. Options:

| Approach | Pros | Cons |
|---|---|---|
| **Goja** (JS engine in Go) | Full JS eval, closest to original | External dependency, ~5MB binary increase |
| **Yaegi** (Go interpreter) | Eval Go code | Unusual UX (users write Go?) |
| **Custom expression parser** | Zero deps, small | Must implement yourself |
| **Expr** (`github.com/expr-lang/expr`) | Safe, fast, typed | Not a full language; no assignment |
| **Tengo** (scripting language for Go) | Designed for embedding | Users learn a new syntax |

**Recommended: Goja** (JS runtime). This preserves the original's semantics
exactly — users write the same `i = 1; sum = 0` syntax as the React version.

```
STRUCT FlowInterpreter:
    nodes       []FlowNode
    edges       []FlowEdge
    vars        map[string]interface{}
    output      []string
    currentID   *int
    done        bool
    error       string
    waitInput   bool
    inputPrompt string
    inputVar    string
    stepCount   int
    runtime     *goja.Runtime      // JS runtime instance

FUNCTION NewInterpreter(nodes, edges):
    interp = &FlowInterpreter{...}
    interp.runtime = goja.New()
    // Register print function
    interp.runtime.Set("print", func(call goja.FunctionCall) goja.Value {
        args = formatArgs(call.Arguments)
        interp.output = append(interp.output, args)
        return goja.Undefined()
    })
    return interp

FUNCTION (interp) Step(inputValue *string):
    IF interp.done OR interp.error != "": return
    interp.stepCount++
    IF interp.stepCount > 500:
        interp.error = "MAX STEPS EXCEEDED"
        interp.done = true
        return
    
    // Handle waiting input
    IF interp.waitInput:
        IF inputValue == nil: return
        interp.runtime.Set(interp.inputVar, parseValue(*inputValue))
        interp.vars[interp.inputVar] = parseValue(*inputValue)
        interp.output = append(interp.output, "> " + *inputValue)
        interp.waitInput = false
        interp.advance()
        return
    
    // ... rest is identical logic to Python version,
    // but using interp.runtime.RunString(code) instead of eval()

FUNCTION (interp) execStatements(code string):
    // Sync Go vars into JS runtime
    FOR k, v IN interp.vars:
        interp.runtime.Set(k, v)
    
    // Execute in JS runtime
    FOR each statement IN splitSemicolon(code):
        IF isAssignment(statement):
            varName, expr = parseAssignment(statement)
            result, err = interp.runtime.RunString(expr)
            IF err: THROW
            interp.vars[varName] = result.Export()
            interp.runtime.Set(varName, result)
        ELSE:
            interp.runtime.RunString(statement)
    
FUNCTION (interp) evalExpr(code string) bool:
    // Sync vars
    FOR k, v IN interp.vars:
        interp.runtime.Set(k, v)
    result, err = interp.runtime.RunString(code)
    IF err: THROW
    return result.ToBoolean()
```

### 7.2 Simpler Alternative: Custom Mini-Language

If you don't want the Goja dependency, implement a tiny expression evaluator:

```
STRUCT Expr:
    // Supports: numbers, strings, variables, +, -, *, /, %, ==, !=, <, <=, >, >=
    // Built-in functions: print(), str(), int()

FUNCTION evalMiniExpr(code string, vars map[string]interface{}) interface{}:
    tokens = tokenize(code)
    ast = parseExpression(tokens)
    return ast.Eval(vars)

// This is ~200 lines of Go for a Pratt parser.
// Sufficient for the demo flowchart but limited for real use.
```

---

## 8. Timer / Auto-Step

Textual has `set_interval()`. Bubbletea uses `tea.Tick()`:

```
FUNCTION tickCmd(interval time.Duration) Cmd:
    return tea.Tick(interval, func(t time.Time) Msg {
        return TickMsg{t}
    })

FUNCTION Update(model, msg):
    MATCH msg:
        TickMsg:
            IF model.autoRunning AND model.interp != nil:
                IF NOT interp.done AND NOT interp.waitInput:
                    interp.Step(nil)
                    syncState(model)
                    return model, tickCmd(model.speed)
                ELSE:
                    model.autoRunning = false
            return model, nil
```

The pattern: each tick returns the *next* tick command. To stop, simply
don't return another tick. This is idiomatic Bubbletea.

---

## 9. Data Model (Go Structs)

```
TYPE NodeType STRUCT:
    Label  string
    W, H   int

TYPE FlowNode STRUCT:
    ID     int
    Type   string    // "process", "decision", "terminal", "io", "connector"
    X, Y   int       // world position (top-left)
    Text   string
    Code   string

    METHODS:
        Info() NodeType      // lookup in NODE_TYPES map
        CX() float64         // center X = X + Info().W/2
        CY() float64         // center Y = Y + Info().H/2

TYPE FlowEdge STRUCT:
    FromID int
    ToID   int
    Label  string    // "Y", "N", or ""

TYPE Tool INT ENUM:
    SELECT, ADD, CONNECT

TYPE Model STRUCT:
    // Data
    nodes       []FlowNode
    edges       []FlowEdge
    nextID      int
    
    // Selection / interaction
    tool        Tool
    newNodeType string
    selectedID  *int
    connectID   *int
    dragging    bool
    dragID      int
    dragOffX    int
    dragOffY    int
    
    // Camera
    camX, camY  int
    
    // Mouse tracking
    mouseX, mouseY  int    // buffer-relative
    lastMouseX, lastMouseY int  // for throttling
    
    // Interpreter
    interp      *FlowInterpreter
    execNodeID  *int
    running     bool
    autoRunning bool
    waitInput   bool
    speed       time.Duration
    
    // Console / variables (derived from interp)
    consoleLines []string
    variables    map[string]interface{}
    
    // Sub-models (from bubbles)
    inputField   textinput.Model
    
    // Edit modal
    editOpen     bool
    editNodeID   int
    editLabel    textinput.Model
    editCode     textinput.Model
    editFocus    int   // 0=label, 1=code
    
    // Layout (computed on resize)
    width, height    int
    canvasW, canvasH int
    panelW           int
    
    // Focus
    focus       FocusTarget
```

---

## 10. Effort Breakdown

| Component | Textual (Python) | Bubbletea (Go) | Why the difference |
|---|---|---|---|
| **CellBuffer + Render** | 0 lines (framework) | ~150 lines | Must build from scratch |
| **Layout engine** | 0 lines (CSS) | ~50 lines | Manual arithmetic |
| **Drawing primitives** | ~200 lines | ~200 lines | Direct port |
| **Canvas View** | ~50 lines (render_line) | ~100 lines (buffer composition) | Full-screen buffer approach |
| **Mouse handling** | ~60 lines | ~100 lines | Manual coord translation |
| **Modal / overlay** | ~50 lines (ModalScreen) | ~80 lines | Buffer-based overlay |
| **Input focus routing** | 0 lines (framework) | ~60 lines | Manual routing |
| **Interpreter** | ~200 lines | ~250 lines | Go verbosity + Goja setup |
| **Toolbar / panel rendering** | ~100 lines | ~150 lines | Manual styled strings |
| **Elm architecture glue** | ~150 lines | ~200 lines | Single Update function |
| **Data model** | ~50 lines | ~80 lines | Go struct verbosity |
| **Total** | **~700 lines** | **~1400–1600 lines** | ~2× due to missing framework |

---

## 11. Recommended File Structure

```
grail/
├── main.go              // tea.NewProgram, entry point
├── model.go             // Model struct, Init, Update, View
├── canvas.go            // CellBuffer, Render, drawing primitives
├── draw.go              // drawNode, drawEdge, drawGrid, buildCanvas
├── layout.go            // recalcLayout, renderToolbar, renderPanel
├── mouse.go             // handleMouse, hitTestNode, coordinate translation
├── interpreter.go       // FlowInterpreter, Step, eval/exec via Goja
├── modal.go             // drawEditModal, edit state management
├── data.go              // FlowNode, FlowEdge, NodeType, initial data
├── styles.go            // StyleKey enum, palette, styleLookup map
└── go.mod
```

---

## 12. Key Risks and Recommendations

### Risk 1: Rendering performance

**Problem:** `View()` is called on every `Update`. Building a full-screen
CellBuffer + rendering to string on every mouse move could be slow.

**Mitigation:** 
- Cache the buffer. Only rebuild when state changes (dirty flag).
- Throttle mouse motion events (skip if same cell).
- The canvas is small (typically < 200×50 = 10K cells). Rendering 10K cells
  to a string takes <1ms in Go.

### Risk 2: ANSI string width calculation

**Problem:** When using `lipgloss.JoinHorizontal`, the library calculates
visual width by counting characters, but ANSI escapes are invisible. If
your CellBuffer renderer emits raw ANSI, lipgloss may miscalculate widths.

**Mitigation:** Use the full-screen CellBuffer approach (§6.3, option 2).
Bypass lipgloss layout entirely for the main canvas. Use lipgloss only for
self-contained panels and toolbar.

### Risk 3: Flickering

**Problem:** Full-screen redraws on every event can cause flicker.

**Mitigation:** Bubbletea uses an internal renderer that diffs the
previous and current `View()` output and only updates changed regions.
This works well as long as you return consistent-length strings. Don't
change the number of lines between frames.

### Risk 4: Eval in Go

**Problem:** Go has no `eval()`. The interpreter needs a runtime.

**Mitigation:** Use Goja. It's well-maintained, fast, and preserves the
original JS semantics exactly. Binary size increase is ~5MB — acceptable
for a CLI tool. If binary size matters, use a custom Pratt parser (~200
lines) for a simpler expression language.

---

## 13. Summary: What You Must Build vs What You Get Free

### You must build from scratch (Textual gives these for free):
1. **CellBuffer** — 2D grid of (char, style) cells
2. **Buffer-to-string renderer** — run-length encoded ANSI output
3. **Layout arithmetic** — manual width/height calculation on resize
4. **Focus routing** — manual key event dispatch based on app state
5. **Modal overlay** — rendered into the buffer, no framework support
6. **Coordinate translation** — mouse absolute → canvas-relative → world

### You get for free (from Bubbletea/Lipgloss/Bubbles):
1. **Elm architecture** — clean Model/Update/View separation
2. **Mouse events** — `tea.WithMouseCellMotion()` / `tea.WithMouseAllMotion()`
3. **Key events** — full keyboard with modifiers
4. **Terminal management** — alt screen, raw mode, resize events
5. **Timer/tick** — `tea.Tick()` for auto-step
6. **Text input** — `bubbles/textinput` for edit fields and program input
7. **Viewport** — `bubbles/viewport` for scrollable console output
8. **Lipgloss styling** — styled strings for toolbar, panel text
9. **Diff-based rendering** — minimal terminal updates (no flicker)

### Ports almost directly (logic unchanged):
1. **FlowInterpreter** — same algorithm, different eval backend
2. **Bresenham line drawing** — pure math
3. **Edge exit calculation** — pure math
4. **Node data model** — struct translation
5. **Hit testing** — point-in-rect check
6. **Edge auto-labeling** — Y/N assignment logic

---
---

# ADDENDUM: How Lipgloss v2 Compositing Changes Everything

## 14. The Paradigm Shift

Lipgloss v2.0.0-beta.2 introduces `Canvas` and `Layer` — a compositing
system that directly addresses the three hardest problems identified above:

| Problem (from §2, §6) | v1 Bubbletea approach | v2 Canvas/Layer approach |
|---|---|---|
| Per-character canvas rendering | Build CellBuffer from scratch (~150 lines) | **Nodes become Layers with Lipgloss styling** |
| Modal overlay (ANSI splicing) | Splice ANSI strings or full-screen buffer | **High-Z Layer placed on Canvas** |
| Mouse hit testing | Manual coordinate translation + point-in-rect | **`Canvas.Hit(x, y)` + `Layer.ID()`** |

The mental model shifts from "I own a 2D character grid and paint into it"
to "I have styled content blocks positioned on a canvas with Z-ordering,
and the framework composites them."

This is closer to how the **original React/SVG** version works (absolutely-
positioned divs with z-index) than the Python/Textual version is.

---

## 15. Revised Architecture with v2 Compositing

### 15.1 The Layer Map

Every visual element becomes a Layer with a position, Z-index, and
optional ID for hit testing:

```
Full-screen Canvas:
    Z=0  Layer "toolbar"        X=0, Y=0
    Z=0  Layer "grid-and-edges" X=0, Y=3          // character buffer (only for this!)
    Z=0  Layer "panel-border"   X=canvasW, Y=3
    Z=0  Layer "vars-panel"     X=canvasW+1, Y=3
    Z=0  Layer "console-panel"  X=canvasW+1, Y=3+varsH
    Z=0  Layer "help-panel"     X=canvasW+1, Y=3+varsH+consoleH
    Z=0  Layer "footer"         X=0, Y=termH-1
    
    Z=2  Layer "node-1"         X=node1.screenX, Y=node1.screenY, ID="node-1"
    Z=2  Layer "node-2"         X=node2.screenX, Y=node2.screenY, ID="node-2"
    Z=2  ...one per node...
    
    Z=3  Layer "label-Y"        X=labelX, Y=labelY
    Z=3  Layer "label-N"        X=labelX, Y=labelY
    
    Z=5  Layer "connect-preview" X=..., Y=...     // only when connecting
    
    Z=100 Layer "edit-modal"    X=centered, Y=centered  // only when editing
```

### 15.2 What This Eliminates

**The entire CellBuffer for nodes is gone.** Instead of maintaining a
`[][]Cell` grid and manually painting corners, borders, and text character
by character, each node is a Lipgloss styled box:

```
FUNCTION renderNodeLayer(node, isSelected, isExecuting) Layer:
    // Choose style based on state
    style = lipgloss.NewStyle()
    
    IF node.Type == "decision":
        style = style.Border(DoubleBorder())
    ELSE IF node.Type == "terminal":
        style = style.Border(RoundedBorder())
    ELSE:
        style = style.Border(NormalBorder())
    
    IF isExecuting:
        style = style.
            BorderForeground(color("#ffcc00")).
            Foreground(color("#ffee66")).
            Bold(true)
    ELSE IF isSelected:
        style = style.
            BorderForeground(color("#00ffee")).
            Foreground(color("#00ffee")).
            Bold(true)
    ELSE:
        colors = NODE_PALETTE[node.Type]
        style = style.
            BorderForeground(colors.border).
            Foreground(colors.text)
    
    style = style.
        Width(node.Info().W - 2).    // inner width (border adds 2)
        Height(node.Info().H - 2).   // inner height
        Align(lipgloss.Center, lipgloss.Center)
    
    content = style.Render(node.Text)
    
    // Convert world coords to screen coords
    screenX = node.X - camX
    screenY = node.Y - camY + TOOLBAR_H
    
    return lipgloss.NewLayer(content).
        X(screenX).
        Y(screenY).
        Z(2).
        ID(fmt.Sprintf("node-%d", node.ID))
```

**Compare this to the CellBuffer approach (§3.4):** that was 40+ lines of
manual corner-placing, border-drawing, interior-clearing, text-centering.
The Lipgloss version is ~25 lines that read like a style sheet. And Lipgloss
handles the box-drawing characters, padding, and alignment for you.

**The modal overlay problem (§6.3) vanishes entirely:**

```
FUNCTION renderEditModal(model) Layer:
    // Build the modal content with Lipgloss styles
    titleStyle = lipgloss.NewStyle().Bold(true).Foreground(color("#00d4a0"))
    labelStyle = lipgloss.NewStyle().Foreground(color("#00d4a0"))
    boxStyle = lipgloss.NewStyle().
        Border(NormalBorder()).
        BorderForeground(color("#00d4a0")).
        Background(color("#0a1510")).
        Width(50).
        Padding(1, 2)
    
    content = lipgloss.JoinVertical(lipgloss.Left,
        titleStyle.Render("✏️  EDIT — " + nodeTypeName),
        labelStyle.Render("Label:"),
        model.editLabel.View(),
        labelStyle.Render("Code:"),
        model.editCode.View(),
        "[enter] Save  [esc] Cancel",
    )
    
    return lipgloss.NewLayer(boxStyle.Render(content)).
        X((model.width - 54) / 2).    // centered
        Y((model.height - 12) / 2).
        Z(100)                          // on top of everything
```

No ANSI splicing. No buffer overlay math. Just a layer at Z=100.

### 15.3 What Still Needs a Character Buffer

**Edges and grid dots.** These are sparse, non-rectangular content that
can't be expressed as Lipgloss styled boxes. An edge is a diagonal line
drawn with Bresenham — it needs character-by-character placement.

The solution: keep a **small CellBuffer just for the background layer**.
This buffer covers only the canvas area (not the full terminal), and it
only draws grid dots and edge lines. Nodes are NOT drawn into it — they're
separate layers that composite on top.

```
FUNCTION renderBackgroundLayer(model) Layer:
    buf = NewCellBuffer(model.canvasW, model.canvasH)
    
    // Grid dots
    drawGrid(buf, model.camX, model.camY)
    
    // Edges (just the lines, no nodes!)
    FOR each edge IN model.edges:
        drawEdgeLine(buf, edge, model.nodes, model.camX, model.camY,
                     model.execNodeID)
    
    // Connect preview
    IF model.connectingID != nil:
        drawConnectPreview(buf, ...)
    
    // Render to string — this IS still needed
    bgString = buf.Render()
    
    return lipgloss.NewLayer(bgString).X(0).Y(TOOLBAR_H).Z(0)
```

The CellBuffer is now ~60% smaller because it only handles:
- Grid dots (simple)
- Edge lines (Bresenham)
- Edge arrowheads

It does NOT handle:
- Node shapes (eliminated — Lipgloss boxes)
- Node text centering (eliminated — Lipgloss alignment)
- Node selection/execution highlighting (eliminated — Lipgloss styles)
- Modal overlay rendering (eliminated — Layer Z-ordering)

---

## 16. Hit Testing: The Biggest Win

### 16.1 Before (Manual)

In the v1 approach, mouse handling requires:
1. Subtract toolbar height from absolute Y
2. Add camera offset to get world coordinates
3. Loop through all nodes in reverse order
4. Check if point is inside node's bounding rectangle
5. Return the first hit

That's ~20 lines of coordinate math per click, and you must keep it in
sync with the rendering.

### 16.2 After (Canvas.Hit)

```
FUNCTION handleMousePress(model, msg MouseMsg):
    // Ask the canvas which layer was clicked
    hit = model.canvas.Hit(msg.X, msg.Y)
    
    IF hit == nil:
        // Clicked empty space
        IF model.tool == ADD:
            // Place new node (still need coord translation for world pos)
            worldX = msg.X + model.camX
            worldY = msg.Y - TOOLBAR_H + model.camY
            addNode(model, worldX, worldY)
        ELSE:
            model.selectedID = nil
        return
    
    id = hit.GetID()
    
    IF strings.HasPrefix(id, "node-"):
        nodeID = parseIntAfterPrefix(id, "node-")
        
        IF model.tool == CONNECT:
            IF model.connectingID == nil:
                model.connectingID = &nodeID
            ELSE:
                addEdge(model, *model.connectingID, nodeID)
                model.connectingID = nil
                model.tool = SELECT
        ELSE:
            model.selectedID = &nodeID
            startDrag(model, nodeID, msg.X, msg.Y)
    
    ELSE IF id == "edit-modal":
        // Click inside modal — don't deselect
        pass
    
    ELSE IF id == "toolbar":
        // Could add clickable toolbar buttons with nested layers/IDs
        pass
```

The key insight: **hit testing is now consistent with rendering by
construction**. In the CellBuffer approach, the hit test math and the
drawing math are separate codepaths that must agree. With Canvas.Hit,
the same Layer that is rendered is the same Layer that is hit-tested.
There's no way for them to desync.

### 16.3 Nested Hit Testing for Toolbar Buttons

Lipgloss v2 supports nested layers via `Layer.AddLayers(...)`. This means
you could make toolbar buttons individually clickable:

```
FUNCTION renderToolbar(model) Layer:
    // Each button is a child layer with an ID
    selBtn = lipgloss.NewLayer(renderButton("SEL", model.tool == SELECT)).
        X(10).Y(0).ID("btn-select")
    addBtn = lipgloss.NewLayer(renderButton("ADD", model.tool == ADD)).
        X(20).Y(0).ID("btn-add")
    linkBtn = lipgloss.NewLayer(renderButton("LINK", model.tool == CONNECT)).
        X(30).Y(0).ID("btn-connect")
    runBtn = lipgloss.NewLayer(renderButton("▶ RUN", false)).
        X(50).Y(0).ID("btn-run")
    
    toolbarBg = lipgloss.NewStyle().
        Width(model.width).
        Height(TOOLBAR_H).
        Background(color("#0a1510")).
        Render("  GRaIL FLOWCHART INTERPRETER")
    
    toolbar = lipgloss.NewLayer(toolbarBg).X(0).Y(0).Z(0)
    toolbar.AddLayers(selBtn, addBtn, linkBtn, runBtn)
    
    return toolbar
```

Then in `handleMousePress`:
```
    ELSE IF id == "btn-select": model.tool = SELECT
    ELSE IF id == "btn-add":    model.tool = ADD
    ELSE IF id == "btn-connect": model.tool = CONNECT
    ELSE IF id == "btn-run":    startProgram(model)
```

This is clickable toolbar buttons for free — something that would be
extremely tedious with manual coordinate checking.

---

## 17. Revised View() Function

```
FUNCTION View(model) string:
    var layers []lipgloss.Layer
    
    // ── Layer 0: Toolbar ──
    layers = append(layers, renderToolbar(model))
    
    // ── Layer 0: Grid + Edges background ──
    layers = append(layers, renderBackgroundLayer(model))
    
    // ── Layer 0: Panel sections ──
    layers = append(layers,
        renderVarsPanel(model),
        renderConsolePanel(model),
        renderHelpPanel(model),
        renderFooter(model),
    )
    
    // ── Layer 0: Panel border ──
    borderStr = strings.Repeat("│\n", model.canvasH)
    layers = append(layers,
        lipgloss.NewLayer(borderStr).X(model.canvasW).Y(TOOLBAR_H).Z(0),
    )
    
    // ── Layer 2: Nodes (one layer per node) ──
    FOR each node IN model.nodes:
        isSelected = (model.selectedID != nil && *model.selectedID == node.ID)
        isExec     = (model.execNodeID != nil && *model.execNodeID == node.ID)
        
        // Only add if visible (screen coords within canvas bounds)
        screenX = node.X - model.camX
        screenY = node.Y - model.camY + TOOLBAR_H
        IF screenX + node.Info().W >= 0 AND screenX < model.canvasW AND
           screenY + node.Info().H >= 0 AND screenY < model.canvasH + TOOLBAR_H:
            layers = append(layers,
                renderNodeLayer(node, isSelected, isExec, model.camX, model.camY))
    
    // ── Layer 3: Edge labels ──
    FOR each edge IN model.edges:
        IF edge.Label != "":
            layers = append(layers,
                renderEdgeLabel(edge, model.nodes, model.camX, model.camY))
    
    // ── Layer 5: Connect preview (only in connect mode) ──
    IF model.connectingID != nil:
        layers = append(layers, renderConnectPreview(model))
    
    // ── Layer 100: Edit modal (only when editing) ──
    IF model.editOpen:
        layers = append(layers, renderEditModal(model))
    
    // ── Compose and render ──
    model.canvas = lipgloss.NewCanvas(layers...)
    return model.canvas.Render()
```

This is remarkably clean. The `View()` function reads like a description
of the visual hierarchy, not a rendering algorithm. Each layer is
independently styled, positioned, and z-ordered.

---

## 18. Revised Effort Estimate

| Component | v1 Bubbletea (CellBuffer) | v2 Bubbletea (Canvas/Layer) | Delta |
|---|---|---|---|
| CellBuffer + Render | ~150 lines | ~80 lines (edges/grid only) | **-47%** |
| Node rendering | ~80 lines (manual box drawing) | ~30 lines (Lipgloss styled boxes) | **-63%** |
| Modal overlay | ~80 lines (ANSI splice or buffer) | ~20 lines (high-Z Layer) | **-75%** |
| Mouse hit testing | ~40 lines (manual coord math) | ~15 lines (Canvas.Hit) | **-63%** |
| Layout composition | ~50 lines (manual JoinH/V) | ~40 lines (Layer positioning) | -20% |
| Drawing primitives (edges) | ~200 lines | ~200 lines (unchanged) | 0% |
| Focus routing | ~60 lines | ~60 lines (unchanged) | 0% |
| Interpreter | ~250 lines | ~250 lines (unchanged) | 0% |
| Toolbar rendering | ~60 lines | ~40 lines (clickable child layers) | -33% |
| Panel rendering | ~90 lines | ~80 lines (slight simplification) | -11% |
| Elm glue (Update) | ~200 lines | ~180 lines (simpler hit handling) | -10% |
| Data model | ~80 lines | ~80 lines (unchanged) | 0% |
| **Total** | **~1400–1600 lines** | **~1050–1200 lines** | **~25-30% less** |

The savings come primarily from three areas:
1. **Nodes as Lipgloss boxes** instead of manual character painting
2. **Modal as a Layer** instead of ANSI splicing
3. **Canvas.Hit** instead of manual hit testing

---

## 19. The Background Layer: Remaining CellBuffer

Even with v2, you still need a character buffer for edges and grid dots.
But it's simpler because it only draws sparse line characters onto a
blank background — no boxes, no text, no styling of rectangular regions.

```
STRUCT MiniBuffer:
    width, height  int
    chars          [][]rune       // just characters
    styles         [][]StyleKey   // enum index into palette

FUNCTION (buf) Render() string:
    // Same run-length encoding as before (§2.2),
    // but the buffer is simpler: only 3-4 style keys
    // (BG, GRID, EDGE_NORMAL, EDGE_ACTIVE)
    // instead of 15+
    ...

FUNCTION drawEdgeLine(buf, edge, nodes, camX, camY, execNodeID):
    from = nodeByID(edge.FromID)
    to   = nodeByID(edge.ToID)
    active = (execNodeID != nil && *execNodeID == edge.ToID)
    style = IF active THEN EDGE_ACTIVE ELSE EDGE_NORMAL
    
    p1 = getEdgeExit(from, to)
    p2 = getEdgeExit(to, from)
    
    // Buffer-relative coords
    bx1, by1 = p1.X - camX, p1.Y - camY
    bx2, by2 = p2.X - camX, p2.Y - camY
    
    points = bresenham(bx1, by1, bx2, by2)
    FOR i, pt IN points:
        dx, dy = direction(points, i)
        buf.Set(pt.X, pt.Y, lineChar(dx, dy), style)
    
    // Arrowhead
    IF len(points) >= 2:
        last, prev = points[len-1], points[len-2]
        buf.Set(last.X, last.Y,
                arrowChar(last.X-prev.X, last.Y-prev.Y), style)
```

The buffer palette shrinks from ~15 styles to ~4 styles. Node-related
styles (6 border styles, 6 text styles, selected, executing) are all
handled by Lipgloss on the Layer side. The buffer only needs:
- `BG` (background)
- `GRID` (dim dot)
- `EDGE_NORMAL` (green line)
- `EDGE_ACTIVE` (yellow line, bold)

---

## 20. Transparency and Layering: The Subtlety

### 20.1 The Opacity Question

When Layer A (Z=0, background with edges) and Layer B (Z=2, a node box)
overlap, the compositing system must decide: does Layer B's content fully
replace Layer A's content in the overlapping region?

**Yes.** In terminal compositing, every character cell within a Layer's
bounds is opaque. A space character in a higher-Z layer overwrites whatever
is in the lower-Z layer at that position. This is correct behavior for
GRaIL: node boxes should fully obscure any edge lines that pass through
them (which is exactly what our Python CellBuffer approach does by drawing
nodes last).

### 20.2 Why This Is Perfect for Flowcharts

The original React/SVG version draws edges as `<line>` elements with
`z-index: 1` and nodes as `<div>` elements with `z-index: 2`. Nodes
naturally occlude edges. The Lipgloss v2 layer model replicates this
exactly: edge background at Z=0, nodes at Z=2, labels at Z=3, modal at
Z=100.

### 20.3 Where Transparency Would Help But Doesn't Exist

If edges could be semi-transparent layers (only the line characters are
opaque, spaces are see-through), you could eliminate the background
CellBuffer entirely — each edge would be its own Layer. But since spaces
are opaque in terminal compositing, edges must share a single background
layer to avoid occluding each other and the grid dots.

This is a minor limitation. The background CellBuffer for edges is simple
(~80 lines) and well-understood.

---

## 21. Bubble Tea v2 Integration Note

The Lipgloss v2 compositing system is **designed to work with Bubble Tea
v2** (not v1). Specifically:

- `lipgloss.Canvas` implements Bubble Tea v2's `tea.Layer` interface
- Bubble Tea v2 has its own view/layer concepts that align with
  the compositing model
- The Charm team has explicitly stated v2 compositing + Bubble Tea v1
  is not a supported combination

This means if you adopt this architecture, you commit to the **v2 beta
stack**: `lipgloss/v2` + `bubbletea/v2`. This is pre-release software.
The API may change before final release. But for a new project (not
migrating an existing app), this is acceptable.

### Dependency chain:
```
go.mod:
    github.com/charmbracelet/bubbletea/v2  (latest v2 beta)
    github.com/charmbracelet/lipgloss/v2   (v2.0.0-beta.2+)
    github.com/charmbracelet/bubbles/v2    (for textinput, viewport)
    github.com/dop251/goja                 (JS interpreter)
```

---

## 22. Revised Summary: What You Build vs What The Framework Gives You

### You must still build:
1. **Edge/grid CellBuffer** — simplified (~80 lines), only for sparse line drawing
2. **Layout arithmetic** — still manual on WindowSizeMsg (~50 lines)
3. **Focus routing** — still manual key dispatch (~60 lines)
4. **Coordinate translation for drag/add** — world↔screen math for node placement (~20 lines)

### Now provided by Lipgloss v2 Canvas/Layer:
1. ~~**Full CellBuffer**~~ → **Nodes are Lipgloss styled boxes** (Layer)
2. ~~**Buffer-to-string renderer for nodes**~~ → **Lipgloss.Render()** handles it
3. ~~**Modal overlay / ANSI splicing**~~ → **High-Z Layer on Canvas**
4. ~~**Manual hit testing**~~ → **Canvas.Hit(x, y) + Layer.ID()**
5. ~~**Manual Z-ordering (draw edges then nodes)**~~ → **Layer.Z()**
6. **NEW: Clickable toolbar buttons** via nested layers + hit testing

### Still ports directly:
1. FlowInterpreter (Goja)
2. Bresenham line drawing
3. Edge exit calculation
4. Data model
5. Edge auto-labeling

### Bottom line

With Lipgloss v2, the port drops from ~1500 lines to ~1100 lines, and
more importantly, the **hardest problems disappear**:

- Node rendering goes from "manual character painting" to "Lipgloss style sheet"
- Modal overlay goes from "the hardest problem in Bubbletea" to "one line of Z=100"
- Hit testing goes from "keep two codepaths in sync" to "consistent by construction"

The remaining custom code is well-scoped: edge drawing (pure math),
layout arithmetic (simple), focus routing (tedious but straightforward).

The architecture becomes recognizably similar to the original React/SVG
version: absolutely-positioned styled boxes with z-index layering, which
is exactly what Lipgloss v2 Canvas/Layer provides for the terminal.
