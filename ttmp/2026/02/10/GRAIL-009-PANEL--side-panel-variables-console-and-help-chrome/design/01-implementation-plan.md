---
title: "Implementation Plan — Side Panel"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-009-PANEL
topics:
  - layout
  - bubbletea
  - lipgloss-v2
  - go
---

# Implementation Plan — Side Panel

## Overview

Build three panel sections as layers to the right of the canvas: variables,
console, and help. Add a vertical separator. Static placeholder content
for now — wired to interpreter in GRAIL-012-INTERP-UI.

## Dependencies

- `pkg/tealayout` (GRAIL-006) — VerticalSeparator, region layout
- Scaffold with layout (GRAIL-005, GRAIL-007)

## Blocked by

- GRAIL-006-TEALAYOUT, GRAIL-007-NODES (need the canvas to exist)

## Blocks

- GRAIL-012-INTERP-UI (wires live data into these panels)

## What this adds

```
internal/grailui/
└── layers.go    # ADD: buildVarsPanelLayer, buildConsolePanelLayer, buildHelpPanelLayer
```

## Implementation details

### Panel layout

Right panel is 34 columns wide. Divided vertically:
- Variables: 6 rows (top)
- Console: remaining space (middle, stretches with terminal height)
- Help: 8 rows (bottom)

Positions computed from `tealayout.Layout`:
- vars: `X=canvasW+1, Y=ToolbarH`
- console: `X=canvasW+1, Y=ToolbarH+VarsH`
- help: `X=canvasW+1, Y=ToolbarH+VarsH+consoleH`

### Variables panel (placeholder)

```
📦 VARIABLES
  (none)
```

Accepts `map[string]interface{}` — for now always empty. Styled with
gold for names, green for numeric values, blue for string values.

### Console panel (placeholder)

```
🖥️  CONSOLE
  (empty)
```

Accepts `[]string` — for now always empty. Each line styled by prefix:
`⚠`=red, `──`=dim, `>`=yellow, else green.

### Help panel (static)

```
HELP
  Mouse: click=select, drag=move
  [s]Select [a]Add [c]Connect
  [e]Edit  [d]Delete selected
  [r]Run [n]Step [g]Auto [x]Stop
  Add mode: [1-5] node type
  Arrows: pan canvas
```

### Vertical separator

`tealayout.VerticalSeparator(canvasW, ToolbarH, canvasH, borderStyle)`
— a column of `│` characters.

## Visual validation

- Right panel visible with three sections
- Help text shows all keybindings
- Vertical `│` separator between canvas and panel
- Panel resizes correctly on terminal resize

## Estimated effort

~80 lines of code, ~30 minutes.
