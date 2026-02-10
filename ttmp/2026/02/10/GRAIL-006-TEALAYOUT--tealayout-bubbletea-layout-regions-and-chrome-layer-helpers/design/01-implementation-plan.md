---
title: "Implementation Plan — tealayout"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-006-TEALAYOUT
topics:
  - layout
  - bubbletea
  - lipgloss-v2
  - go
---

# Implementation Plan — tealayout

## Overview

Build `pkg/tealayout`, a reusable package for declarative layout
computation and common chrome layer builders (toolbar, footer, separator,
modal) for Bubbletea v2 + Lipgloss v2 apps.

## Dependencies

- `github.com/charmbracelet/lipgloss/v2` — for `NewLayer`, `NewStyle`
- `image` (stdlib) — `image.Rectangle`

## Blocked by

- GRAIL-005-SCAFFOLD (validates that Lipgloss v2 layers work)

## Blocks

- GRAIL-007-NODES (uses layout for canvas region)
- GRAIL-009-PANEL (uses layout for panel regions)
- GRAIL-013-EDIT-MODAL (uses ModalLayer)

## File plan

```
pkg/tealayout/
├── regions.go       # LayoutBuilder, Layout, Region
├── chrome.go        # ToolbarLayer, FooterLayer, VerticalSeparator, ModalLayer
└── regions_test.go  # Unit tests
```

## Implementation details

### LayoutBuilder

Declarative region allocation. Subtracts fixed regions from the terminal
rectangle, then assigns the remainder.

```go
type Region struct {
    Name string
    Rect image.Rectangle
}

type Layout struct {
    TermW, TermH int
    Regions      map[string]Region
}

type LayoutBuilder struct {
    termW, termH int
    top, bottom  int       // consumed from top/bottom
    right        int       // consumed from right
    regions      []Region
}
```

Example: `TopFixed("toolbar", 3)` reserves rows 0..2. `BottomFixed("footer", 1)`
reserves the last row. `RightFixed("panel", 34)` reserves 34 columns from the
right. `Remaining("canvas")` gets whatever's left.

### Chrome helpers

- `ToolbarLayer(content, width, style)` → Layer at Y=0, Z=0
- `FooterLayer(content, width, y, style)` → Layer at Y=y, Z=0
- `VerticalSeparator(x, y, height, style)` → Layer with `strings.Repeat("│\n", height)`
- `ModalLayer(content, termW, termH, boxStyle)` → Layer at centered X/Y, Z=100

### ModalLayer details

```go
func ModalLayer(content string, termW, termH int, boxStyle lipgloss.Style) lipgloss.Layer {
    rendered := boxStyle.Render(content)
    w := lipgloss.Width(rendered)
    h := lipgloss.Height(rendered)
    return lipgloss.NewLayer(rendered).
        X((termW - w) / 2).
        Y((termH - h) / 2).
        Z(100).
        ID("modal")
}
```

## Test cases

- Layout 80×24 with top=3, bottom=1, right=34 → canvas = 45×20 at (0,3)
- Layout with only Remaining → fills entire terminal
- Layout with zero terminal size → all regions empty
- Regions don't overlap (check all pairs)

## Estimated effort

~120 lines of code, ~45 minutes.
