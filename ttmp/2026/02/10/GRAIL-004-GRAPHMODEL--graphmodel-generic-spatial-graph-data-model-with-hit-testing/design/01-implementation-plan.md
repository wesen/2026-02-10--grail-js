---
title: "Implementation Plan — graphmodel"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-004-GRAPHMODEL
topics:
  - graph
  - go
---

# Implementation Plan — graphmodel

## Overview

Build `pkg/graphmodel`, a generic spatial graph with positioned nodes and
labeled edges. Supports add/remove operations, spatial hit testing, and
stable iteration order. Uses Go generics so the package doesn't depend
on GRaIL-specific types.

## Dependencies

- `image` (stdlib) — `image.Point`, `image.Rectangle`
- No external dependencies

## Blocked by

Nothing — pure data model, no UI deps.

## Blocks

- GRAIL-007-NODES (FlowNodeData implements Spatial)
- GRAIL-008-EDGES (iterates edges for drawing)
- GRAIL-010-MOUSE (AddNode, RemoveNode, AddEdge, MoveNode)
- GRAIL-011-INTERPRETER (reads nodes/edges for execution)

## File plan

```
pkg/graphmodel/
├── graph.go          # Graph[N, E] struct, CRUD operations
├── spatial.go        # Spatial interface, HitTest, NodesInRect
└── graph_test.go     # Unit tests
```

## Implementation details

### The Spatial interface

```go
type Spatial interface {
    Pos() image.Point
    Size() image.Point
}
```

Minimal: just position and size. `Center()` and `Bounds()` are computed
as free functions rather than interface methods — this avoids forcing
implementors to write boilerplate:

```go
func CenterOf(s Spatial) image.Point {
    return s.Pos().Add(s.Size().Div(2))
}
func BoundsOf(s Spatial) image.Rectangle {
    p := s.Pos()
    return image.Rect(p.X, p.Y, p.X+s.Size().X, p.Y+s.Size().Y)
}
```

### Graph struct

```go
type Node[N Spatial] struct {
    ID   int
    Data N
}

type Edge[E any] struct {
    FromID, ToID int
    Data         E
}

type Graph[N Spatial, E any] struct {
    nodes    map[int]*Node[N]
    edges    []Edge[E]
    nextID   int
    orderIDs []int    // maintain insertion order for stable iteration
}
```

`orderIDs` tracks insertion order so `Nodes()` returns a deterministic
slice. This matters for rendering (draw order) and hit testing (topmost =
last inserted).

### Operations

- `AddNode(data N) int` — assigns `nextID`, increments, appends to `orderIDs`
- `RemoveNode(id int)` — deletes from `nodes` map, removes from `orderIDs`,
  removes all edges where `FromID == id || ToID == id`
- `AddEdge(fromID, toID int, data E)` — appends to `edges` slice.
  Duplicate check: skip if identical `(fromID, toID)` already exists.
- `RemoveEdge(fromID, toID int)` — removes first matching edge
- `MoveNode(id int, pos image.Point)` — the caller must provide a way
  to set position. Since `N` is a type parameter, use `Node[N].Data`
  access + caller updates position. Alternative: `MoveNode` takes an
  `image.Point` and the `Spatial` implementor must expose a `SetPos`.

**Design choice on MoveNode:** Since Go interfaces don't support setters
elegantly, expose `Node(id)` which returns `*Node[N]`, and let the caller
mutate `Data` directly. This is simpler than a `Movable` interface.

### HitTest

```go
func (g *Graph[N, E]) HitTest(pt image.Point) *Node[N] {
    // Iterate in reverse order (topmost = last in orderIDs)
    for i := len(g.orderIDs) - 1; i >= 0; i-- {
        n := g.nodes[g.orderIDs[i]]
        if pt.In(BoundsOf(n.Data)) {
            return n
        }
    }
    return nil
}
```

Note: in the Lipgloss v2 architecture, Canvas.Hit() replaces this for
mouse interaction. But `graphmodel.HitTest` is still useful for non-UI
queries (e.g., "which node is at this world coordinate?") and for testing.

## Test cases

- AddNode → ID assigned and increments
- AddNode × 3 → Nodes() returns in insertion order
- RemoveNode → node gone, connected edges gone, other edges intact
- RemoveNode with non-existent ID → no-op
- AddEdge duplicate → ignored
- OutEdges → returns correct subset
- HitTest inside node → returns that node
- HitTest outside all nodes → returns nil
- HitTest overlapping nodes → returns topmost (last added)
- MoveNode → Pos() reflects new position

## Estimated effort

~100 lines of code, ~45 minutes.
