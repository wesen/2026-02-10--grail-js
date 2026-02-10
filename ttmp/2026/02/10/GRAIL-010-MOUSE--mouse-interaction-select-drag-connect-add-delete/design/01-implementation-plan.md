---
title: "Implementation Plan — Mouse Interaction"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-010-MOUSE
topics:
  - mouse
  - bubbletea
  - go
  - graph
---

# Implementation Plan — Mouse Interaction

## Overview

Implement all mouse-driven interactions: click-to-select, drag-to-move,
connect mode (edge creation), add mode (node placement), and delete.
Uses `Canvas.Hit()` for hit testing. This step contains **Checkpoint B**:
validate that Canvas.Hit coordinates match MouseMsg coordinates.

## Dependencies

- `pkg/graphmodel` (GRAIL-004) — AddNode, RemoveNode, AddEdge, MoveNode
- `pkg/drawutil` (GRAIL-003) — DrawDashedLine for connect preview
- `pkg/cellbuf` (GRAIL-002) — MiniBuffer for connect preview layer
- Nodes + Edges on canvas (GRAIL-007, GRAIL-008)

## Blocked by

- GRAIL-007-NODES, GRAIL-008-EDGES

## Blocks

- GRAIL-012-INTERP-UI (selection highlighting for executing node)
- GRAIL-013-EDIT-MODAL (edit requires selected node)

## Risk checkpoint (BEFORE implementation)

**Checkpoint B:** Write a standalone test program:

```go
layer := lipgloss.NewLayer("test").X(10).Y(5).Z(0).ID("target")
canvas := lipgloss.NewCanvas(layer)
hit := canvas.Hit(10, 5)
// MUST return layer with GetID() == "target"
```

If this fails, Canvas.Hit coordinates don't match expectations. Options:
(a) apply an offset, (b) fall back to `graphmodel.HitTest` (~30 extra lines),
(c) file upstream bug.

**Store this test in `scripts/test-canvas-hit.go`.**

## What this adds

```
internal/grailui/
├── mouse.go      # handleMouse, extractNodeID, drag state machine
├── keys.go       # handleCanvasKeys (tool switching, node type, panning)
└── layers.go     # ADD: buildConnectPreviewLayer
```

## Implementation details

### Mouse handler structure

```
handleMouse(m, msg):
  IF dragging AND motion → update node position
  IF release → stop drag
  IF motion (no drag) → update mouseX/mouseY for connect preview
  IF press:
    hit = m.canvas.Hit(msg.X, msg.Y)
    nodeID = extractNodeID(hit.GetID())
    
    MATCH tool:
      ADD → place new node at world coords
      CONNECT → click source, then click target
      SELECT → select node, start drag
```

### State machines

**Drag:** press on node → set `m.dragging=true, m.dragNodeID, m.dragOffX/Y`
→ motion updates `node.X/Y` → release clears drag state.

**Connect:** first click sets `m.connectID` → mouse motion shows dashed
preview line → second click creates edge → clear connect state, switch
to select tool.

### Connect preview layer

Build a MiniBuffer the size of the canvas. Call
`drawutil.DrawDashedLine(buf, sourceCX, sourceCY, mouseX, mouseY, StyleConnPreview)`.
Wrap in Layer at Z=5.

### Keyboard routing

`handleCanvasKeys` handles: `s`/`a`/`c` (tools), `1-5` (node types in add
mode), `d`/`delete` (delete selected), `e` (edit — wired in GRAIL-013),
arrow keys (pan), `esc` (cancel), `q` (quit).

### Edge auto-labeling (from Python)

When adding an edge from a decision node:
- If no "Y" edge exists → label "Y"
- Else if no "N" edge exists → label "N"
- Else → empty label

## Visual validation

- Click node → cyan border highlight
- Drag node → follows mouse, edges update
- `c` + click source + click target → edge appears
- `a` + click empty space → new node appears
- `d` with selection → node + edges disappear
- Connect mode → dashed line from source to cursor

## Estimated effort

~120 lines of code, ~60 minutes.
