---
title: "Implementation Plan — Edge Rendering"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-008-EDGES
topics:
  - rendering
  - cellbuf
  - go
  - graph
---

# Implementation Plan — Edge Rendering

## Overview

Build the MiniBuffer background layer: grid dots, Bresenham edge lines
with arrowheads, and edge labels as separate Z=3 layers. After this step,
the app displays a complete flowchart.

## Dependencies

- `pkg/cellbuf` (GRAIL-002) — the MiniBuffer
- `pkg/drawutil` (GRAIL-003) — Bresenham, EdgeExit, DrawGrid, DrawArrowLine
- Nodes on canvas (GRAIL-007) — node data + layer composition

## Blocked by

- GRAIL-002-CELLBUF, GRAIL-003-DRAWUTIL, GRAIL-007-NODES

## Blocks

- GRAIL-010-MOUSE (connect preview uses same drawing infrastructure)

## What this adds to `internal/grailui/`

```
internal/grailui/
└── layers.go    # ADD: buildEdgeCanvasLayer, buildEdgeLabelLayer
```

## Implementation details

### buildEdgeCanvasLayer

1. Create `cellbuf.New(canvasW, canvasH, StyleBG)`
2. Call `drawutil.DrawGrid(buf, camX, camY, 5, 3, StyleGrid)`
3. For each edge:
   a. Look up from/to nodes
   b. Compute exit points with `drawutil.EdgeExit(fromBounds, toCenter)`
   c. Convert world coords to buffer coords (subtract camX, camY)
   d. Call `drawutil.DrawArrowLine(buf, bx1, by1, bx2, by2, edgeStyle, edgeStyle)`
   e. Determine active style if `execNodeID == edge.ToID`
4. Call `buf.Render(bufStyles)` → string
5. Wrap in `NewLayer(rendered).X(0).Y(ToolbarH).Z(0)`

### Edge labels

Each edge with a non-empty label ("Y", "N") becomes a separate Layer at Z=3:
- Position: midpoint of the edge line
- Offset: above the line if horizontal, right of line if vertical
- Style: bold bright green on canvas background

### Style palette (4 keys)

```go
var bufStyles = map[cellbuf.StyleKey]lipgloss.Style{
    StyleBG:         lipgloss.NewStyle().Foreground(color("#1a3a2a")).Background(color("#080e0b")),
    StyleGrid:       lipgloss.NewStyle().Foreground(color("#0e2e20")).Background(color("#080e0b")),
    StyleEdge:       lipgloss.NewStyle().Foreground(color("#00d4a0")).Background(color("#080e0b")),
    StyleEdgeActive: lipgloss.NewStyle().Foreground(color("#ffcc00")).Background(color("#080e0b")).Bold(true),
}
```

### Z-ordering validation

Edge canvas at Z=0, nodes at Z=2, labels at Z=3. Nodes should visually
cover edge lines passing through them. This is the correct behavior
(spaces in node layers are opaque).

## Visual validation

- All 7 edges visible with green lines
- Arrowheads (▼►◄▲) at destination ends
- "Y" and "N" labels on decision's outgoing edges
- Grid dots (·) visible in background
- Nodes cleanly cover edges underneath

## Estimated effort

~80 lines of code, ~30 minutes.
