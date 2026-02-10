---
title: "Implementation Plan — cellbuf"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-002-CELLBUF
topics:
  - cellbuf
  - rendering
  - go
  - lipgloss-v2
---

# Implementation Plan — cellbuf

## Overview

Build `pkg/cellbuf`, a reusable 2D character buffer with per-cell styling
and efficient Lipgloss-based rendering. This is the foundation for edge
and grid drawing in GRaIL, but is reusable for any terminal app needing
character-level control.

## Dependencies

- `github.com/charmbracelet/lipgloss/v2` — for `Style.Render()`
- No Bubbletea dependency (pure rendering library)

## Blocked by

Nothing — this is Step 1, no dependencies on other GRAIL tickets.

## Blocks

- GRAIL-003-DRAWUTIL (draws into cellbuf)
- GRAIL-008-EDGES (uses cellbuf for edge background layer)

## File plan

```
pkg/cellbuf/
├── buffer.go        # Buffer struct, New, Set, SetString, Fill, InBounds
├── render.go        # Render() → string via Render-per-run
└── buffer_test.go   # Unit tests + benchmark
```

## Implementation details

### Core types

```go
type StyleKey int

type Cell struct {
    Ch    rune
    Style StyleKey
}

type Buffer struct {
    W, H  int
    Cells [][]Cell
}
```

`StyleKey` is an int enum. The caller provides a `map[StyleKey]lipgloss.Style`
at render time. This decouples the buffer from specific styles — GRaIL uses
4 keys (BG, Grid, Edge, EdgeActive), a roguelike might use 20.

### Render algorithm

Walk each row. Run-length encode consecutive cells with the same `StyleKey`.
For each run, extract the rune slice, convert to string, call
`styles[key].Render(chunk)`. Concatenate runs for the row. Join rows with `\n`.

**Critical: use `Style.Render()` per run, NOT `StyleRanges()`.** Per the
performance investigation (GRAIL-001 reference doc), `Render`-per-run is
~4× faster than `StyleRanges` on merged runs (1.9ms vs 7.4ms for a 150×40
buffer).

### Edge cases

- `Set(x, y, ...)` with out-of-bounds coords → silently ignored (clamped)
- `New(0, 0, ...)` → valid empty buffer, `Render()` returns `""`
- Unicode runes with width > 1 (e.g., CJK) — not handled in v1; document
  as limitation. All GRaIL characters are single-width.

## Acceptance criteria

- [ ] `Set`/`SetString` write at correct positions
- [ ] Out-of-bounds writes are silently ignored
- [ ] `Render` produces correct number of `\n`-separated lines
- [ ] Consecutive same-styled cells produce a single `Render` call (verify by output size)
- [ ] Benchmark: `Render` on 200×50 buffer < 2ms

## Estimated effort

~80 lines of code, ~30 minutes.
