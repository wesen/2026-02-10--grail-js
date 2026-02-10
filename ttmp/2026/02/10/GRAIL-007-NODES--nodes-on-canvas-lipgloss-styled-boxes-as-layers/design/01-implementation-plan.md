---
title: "Implementation Plan — Nodes on Canvas"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-007-NODES
topics:
  - rendering
  - bubbletea
  - lipgloss-v2
  - go
  - graph
---

# Implementation Plan — Nodes on Canvas

## Overview

Wire `graphmodel` to the Bubbletea scaffold. Each node becomes a Lipgloss
styled box wrapped in a Layer with X/Y/Z/ID. Camera panning with arrow keys.
This is the first step where the app looks like a flowchart.

## Dependencies

- `pkg/graphmodel` (GRAIL-004)
- `pkg/tealayout` (GRAIL-006)
- Scaffold (GRAIL-005)

## Blocked by

- GRAIL-004-GRAPHMODEL, GRAIL-005-SCAFFOLD, GRAIL-006-TEALAYOUT

## Blocks

- GRAIL-008-EDGES (adds edges between these nodes)
- GRAIL-010-MOUSE (interacts with these nodes)

## File plan (in `internal/grailui/`)

```
internal/grailui/
├── model.go     # Model struct, Init, tool/focus enums
├── update.go    # Update routing (keyboard only at this step)
├── view.go      # View — builds layers, returns canvas.Render()
├── layers.go    # buildNodeLayer
├── data.go      # FlowNodeData, FlowEdgeData, node type registry, initial data
└── styles.go    # Color palette, borderForType
```

## Implementation details

### FlowNodeData

```go
type FlowNodeData struct {
    Type string   // "process", "decision", "terminal", "io", "connector"
    X, Y int
    Text string
    Code string
}

func (n FlowNodeData) Pos() image.Point  { return image.Pt(n.X, n.Y) }
func (n FlowNodeData) Size() image.Point { return image.Pt(nodeTypeInfo[n.Type].W, nodeTypeInfo[n.Type].H) }
```

### buildNodeLayer

For each visible node:
1. Pick border style: `RoundedBorder` (terminal), `DoubleBorder` (decision), `NormalBorder` (others)
2. Pick colors: green (process), cyan (decision), bright green (terminal), gold (io), dim green (connector)
3. Override colors if selected (cyan) or executing (yellow)
4. Build `lipgloss.NewStyle().Border(...).Width(w-2).Height(h-2).Align(Center).Render(text)`
5. Wrap in `NewLayer(rendered).X(screenX).Y(screenY).Z(2).ID("node-N")`

### Camera panning

Arrow keys adjust `m.camX` / `m.camY`. `screenX = node.X - camX`,
`screenY = node.Y - camY + ToolbarH`.

### Visibility culling

Only create layers for nodes whose screen rect overlaps the canvas region.
Saves allocation for large graphs with many off-screen nodes.

### Initial data

Port the 7-node flowchart from `grail.py:make_initial_nodes()` and
`make_initial_edges()`. Edges stored but not rendered yet (Step 7).

## Visual validation

After this step, running the app shows:
- 7 styled boxes on a dark background
- Different border styles visible per node type
- Arrow keys pan the camera smoothly
- Connector node (7×3) visually smaller than others (22×3)

## Estimated effort

~100 lines of code, ~45 minutes.
