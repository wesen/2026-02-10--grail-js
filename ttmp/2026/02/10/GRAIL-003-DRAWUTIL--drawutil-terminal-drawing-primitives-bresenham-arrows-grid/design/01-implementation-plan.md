---
title: "Implementation Plan — drawutil"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-003-DRAWUTIL
topics:
  - rendering
  - go
  - cellbuf
---

# Implementation Plan — drawutil

## Overview

Build `pkg/drawutil`, a collection of terminal drawing primitives: Bresenham
line algorithm, line/arrow character selection, edge exit-point geometry,
grid patterns, and convenience functions that draw into a `cellbuf.Buffer`.

## Dependencies

- `pkg/cellbuf` (GRAIL-002-CELLBUF) — drawing target
- `image` (stdlib) — `image.Point`, `image.Rectangle`

## Blocked by

- GRAIL-002-CELLBUF

## Blocks

- GRAIL-008-EDGES (uses drawutil for edge rendering)
- GRAIL-010-MOUSE (uses DrawDashedLine for connect preview)

## File plan

```
pkg/drawutil/
├── line.go          # Bresenham, LineChar, ArrowChar
├── edge.go          # EdgeExit calculation
├── grid.go          # DrawGrid
├── draw.go          # DrawLine, DrawArrowLine, DrawDashedLine
└── line_test.go     # Unit tests
```

## Implementation details

### Bresenham

Port directly from Python `bresenham()` in `grail.py:303`. Return
`[]image.Point` instead of Python's `list[tuple[int,int]]`. Go doesn't
have generators, so return the full slice.

Safety: cap the loop at `dx + dy + 2` iterations to prevent infinite loops
on degenerate inputs.

### LineChar / ArrowChar

Pure lookup tables:
- `LineChar(dx, dy)`: `│` (vertical), `─` (horizontal), `/` or `\` (diagonal)
- `ArrowChar(dx, dy)`: `▲▼◄►` based on dominant direction

### EdgeExit

Port from Python `get_edge_exit()` in `grail.py:349`. Takes an
`image.Rectangle` (the node bounds) and a target `image.Point` (the
other node's center). Returns the point on the rectangle's border
closest to the target direction.

Key: compare normalized dx/dy against half-width/half-height to determine
which side (left/right/top/bottom) the edge exits from.

### Draw functions

All draw into a `*cellbuf.Buffer` with a given `cellbuf.StyleKey`:
- `DrawGrid(buf, camX, camY, spacingX, spacingY, style)` — place `·` at grid intersections
- `DrawLine(buf, x0, y0, x1, y1, style)` — Bresenham line with LineChar per point
- `DrawArrowLine(buf, x0, y0, x1, y1, lineStyle, arrowStyle)` — line + arrowhead at end
- `DrawDashedLine(buf, x0, y0, x1, y1, style)` — every 3rd point skipped (for connect preview)

## Test cases

- Bresenham horizontal: (0,0)→(5,0) = 6 points at y=0
- Bresenham vertical: (0,0)→(0,5) = 6 points at x=0
- Bresenham diagonal: (0,0)→(5,5) = 6 points on y=x
- Bresenham steep: (0,0)→(2,8) — more vertical steps than horizontal
- Bresenham zero-length: (3,3)→(3,3) = single point
- EdgeExit: target to the right → exit from right edge
- EdgeExit: target below → exit from bottom edge
- EdgeExit: target at 45° → exit depends on aspect ratio
- LineChar: (1,0)→`─`, (0,1)→`│`, (1,1)→`\`, (-1,1)→`/`
- ArrowChar: (0,1)→`▼`, (0,-1)→`▲`, (1,0)→`►`, (-1,0)→`◄`

## Estimated effort

~80 lines of code, ~30 minutes.
