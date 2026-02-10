---
title: "Implementation Plan — Scaffold"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-005-SCAFFOLD
topics:
  - bubbletea
  - lipgloss-v2
  - go
---

# Implementation Plan — Scaffold

## Overview

Build the smallest possible running Bubbletea v2 + Lipgloss v2 app that
uses Canvas/Layer compositing with mouse support. This is **Checkpoint A**:
if the v2 beta stack doesn't work, we know before investing further.

## Dependencies

- `github.com/charmbracelet/bubbletea/v2`
- `github.com/charmbracelet/lipgloss/v2` (v2.0.0-beta.2+)

## Blocked by

Nothing — first running app.

## Blocks

- Everything from GRAIL-006 onward (all build on this scaffold)

## File plan

```
grail-go/
├── main.go     # Entry point + Model/Init/Update/View
└── go.mod      # bubbletea/v2 + lipgloss/v2
```

## Implementation details

### What this validates

1. `bubbletea/v2` compiles and runs with `lipgloss/v2`
2. `tea.WithAltScreen()` enters alternate screen buffer
3. `tea.WithMouseAllMotion()` delivers `tea.MouseMsg` on every cursor move
4. `tea.WindowSizeMsg` fires on startup with correct terminal dimensions
5. `lipgloss.NewCanvas()` + `lipgloss.NewLayer()` + `.Render()` produce output
6. Layers at different Z-values composite correctly

### The minimal app

```go
type Model struct {
    width, height int
    mouseX, mouseY int
}

func (m Model) View() string {
    if m.width == 0 { return "" }
    
    bg := lipgloss.NewStyle().Width(m.width).Height(m.height).
        Background(lipgloss.Color("#080e0b")).Render("")
    bgLayer := lipgloss.NewLayer(bg).X(0).Y(0).Z(0)
    
    title := lipgloss.NewStyle().Bold(true).
        Foreground(lipgloss.Color("#00ffc8")).
        Render("  GRaIL — Scaffold  ")
    titleLayer := lipgloss.NewLayer(title).X(0).Y(0).Z(1)
    
    info := fmt.Sprintf("Mouse: (%d, %d)  Term: %dx%d  [q]uit",
        m.mouseX, m.mouseY, m.width, m.height)
    footerLayer := lipgloss.NewLayer(info).X(0).Y(m.height-1).Z(1)
    
    canvas := lipgloss.NewCanvas(bgLayer, titleLayer, footerLayer)
    return canvas.Render()
}
```

### What to look for when running

- Terminal clears and shows dark green background
- Title "GRaIL — Scaffold" in bright green at top-left
- Footer shows live mouse coordinates as you move the cursor
- Terminal dimensions update when you resize the window
- Press `q` to exit cleanly back to normal terminal

### Failure modes and fallbacks

- **Compile error on import**: v2 module paths changed. Check that
  `go.mod` uses `/v2` suffix: `github.com/charmbracelet/bubbletea/v2`
- **MouseMsg not firing**: Verify `tea.WithMouseAllMotion()` is passed
  to `tea.NewProgram()`. Some terminal emulators disable mouse reporting.
- **Canvas.Render() panics**: Check for nil layers or zero-size content.
- **Blank screen**: WindowSizeMsg may not have fired yet. Guard with
  `if m.width == 0 { return "" }`.

## Acceptance criteria

- [ ] App compiles and runs with `go run main.go`
- [ ] Dark green background fills terminal
- [ ] Mouse coordinates update live in footer
- [ ] Terminal resize updates dimensions
- [ ] `q` exits cleanly
- [ ] No panics on startup or resize

## Estimated effort

~60 lines of code, ~15 minutes.
