---
title: Diary
doc_type: reference
status: active
intent: long-term
ticket: GRAIL-001-PORT-TO-INKJS
topics:
  - cli
  - terminal-ui
  - inkjs
  - port
related_files:
  - path: /home/manuel/code/wesen/corporate-headquarters/2026-02-10--grail-js/grail.py
    note: "Complete Textual port of the GRaIL flowchart editor+interpreter (commit b3e4646)"
  - path: /home/manuel/code/wesen/corporate-headquarters/2026-02-10--grail-js/reference.jsx
    note: "Original React/JSX source — source of truth for feature parity"
---

# Diary

## Goal

Capture the step-by-step port of GRaIL (Graphical Representation and Interpretation Language) from a React/JSX browser app to a Python Textual terminal application with mouse support.

## Step 1: Project Setup, Library Evaluation, and Technology Decision

The initial request was to port `reference.jsx` to an ink.js (Node.js) terminal app. After examining the reference — a full flowchart editor with SVG canvas, node shapes, drag/connect, and a step-through interpreter — it became clear that ink.js is a poor fit: it has no native mouse support, no canvas drawing, and is optimized for text-flow CLIs (spinners, prompts, logs), not spatial/graphical UIs.

After evaluating Textual (Python), Bubbletea (Go), Ratatui (Rust), and terminal-kit (Node.js), the user chose **Textual** for its first-class mouse events, CSS-like layouts, Rich rendering, and active maintenance. We initialized a Node.js project (npm, ink, react) before pivoting — those artifacts remain in the repo but are unused.

### Prompt Context

**User prompt (verbatim):** "Use docmgr to create a new ticket GRAIL-001-PORT-TO-INKJS and port @reference.jsx to a inkjs terminal application with mouse support."

**Assistant interpretation:** Create a docmgr ticket, then port the React flowchart editor to a terminal app (originally ink.js, later Textual).

**Inferred user intent:** Get a working terminal-native version of the GRaIL flowchart editor that supports mouse interaction.

**Commit (code):** b3e4646 — "Port GRaIL flowchart editor+interpreter to Textual (Python TUI)"

### What I did
- Initialized docmgr ticket GRAIL-001-PORT-TO-INKJS with 8 tasks
- Evaluated ink.js vs Textual vs Bubbletea vs Ratatui for terminal flowchart editing
- Created docmgr diary document and related files
- npm init + installed ink/react (ultimately unused after pivot to Python)
- Verified Textual 2.1.2 and Python 3.11.3 available

### Why
- ink.js lacks mouse events and canvas primitives needed for a spatial flowchart editor
- Textual provides `on_mouse_down`/`on_mouse_move`/`on_mouse_up`, CSS-like layout, and Rich text rendering — all needed for this port

### What worked
- Textual's API (`render_line`, `Strip`, `Segment`, `ModalScreen`) maps well to the requirements
- The `FlowInterpreter` logic is essentially language-agnostic (JS `new Function` ↔ Python `eval`/`exec`)

### What didn't work
- ink.js was initially installed but has no mouse support or drawing canvas — dead end for this use case
- Initial plan to render complex ASCII art shapes (diamonds with `╱╲`, parallelograms) was over-engineered; user suggested "use blocks with different styling" which dramatically simplified rendering

### What I learned
- For terminal UIs that need spatial/mouse interaction, Textual (Python) is the clear winner over ink.js
- Terminal characters are ~2:1 aspect ratio; node dimensions must account for this
- Box-drawing characters (`┌─┐│└─┘`, `╭╮╰╯`, `╔═╗║╚═╝`) are more effective than trying to draw pixel-perfect shapes in ASCII

### What was tricky to build
- **Edge routing**: Bresenham line drawing works for straight edges but produces awkward diagonal characters (`/`, `\`) when the slope is steep. The key insight was to compute edge exit points on node borders (using normalized direction comparison `|ndx| vs |ndy|`) so edges exit cleanly from sides/top/bottom.
- **Node layout**: The initial layout had decision's Y and N edges both going straight down, causing overlap. Fixed by placing the N-branch target (PRINT SUM) to the right of the main column, making the N edge exit from the decision's right side (horizontal).
- **Canvas rendering**: Used a buffer-based approach (`build_buffer` returns `list[list[tuple[str, Style]]]`) with edges drawn first and nodes drawn on top (overwriting overlap). This avoids z-ordering complexity.
- **Interpreter Python port**: JS `new Function(...keys, body)` maps to Python `eval(expr, globals, locals)`. The tricky part was building a safe environment with `__builtins__: {}` while still providing `print`, `str`, `int`, etc.

### What warrants a second pair of eyes
- **Security of eval/exec**: The interpreter uses `eval()` and `exec()` with `{"__builtins__": {}}` and a curated env. This is safer than raw eval but not sandbox-safe. For a toy flowchart interpreter this is acceptable, but should not be used with untrusted input.
- **Edge overlap at connector loop**: Edges 4→5 and 5→3 share the connector's left border point. The visual result is acceptable but could be improved with Manhattan (orthogonal) routing.
- **Mouse coordinate mapping**: Canvas uses a manual camera offset (`cam_x`, `cam_y`). Mouse events give widget-local coordinates; these are translated to canvas coords. Off-by-one errors here would cause misaligned hit detection.

### What should be done in the future
- Add Manhattan (orthogonal) edge routing for cleaner edge rendering
- Add edge deletion (click on edge to select, then delete)
- Add undo/redo
- Add save/load (JSON serialization of nodes + edges)
- Consider adding a scrollbar or minimap for large flowcharts
- Add edge label editing (currently auto-assigned Y/N for decisions)

### Code review instructions
- **Start at**: `grail.py` — single file, ~700 lines
- **Key sections**: `FlowInterpreter` (line ~150), `build_buffer` (line ~300), `FlowCanvas` (line ~480), `GRaILApp` (line ~560)
- **Validate**: `python3 grail.py` — should show CRT-themed flowchart. Press `r` to run, `g` for auto-step. Click nodes to select, drag to move. Press `e` to edit.
- **Smoke test**: `python3 -c "import grail; i = grail.FlowInterpreter(grail.make_initial_nodes(), grail.make_initial_edges()); [i.step() for _ in range(100) if not i.done]; print(i.vars, i.output)"`

### Technical details

**Architecture mapping (React → Textual):**

| React concept | Textual equivalent |
|---|---|
| `useState` + `setNodes` | Mutable `self.nodes` list on App |
| SVG `<line>`, `<rect>` | `build_buffer` → 2D char array with `Segment`/`Style` |
| `onMouseDown`/`onMouseMove` | `on_mouse_down`/`on_mouse_move` on Widget |
| `useCallback` event handlers | Direct method handlers on `FlowCanvas` |
| `useEffect` for auto-timer | `self.set_interval()` |
| React `render()` | `Widget.render_line(y) → Strip` |
| Modal overlay `<div>` | `ModalScreen` subclass |
| CSS-in-JS styles | Textual CSS (`App.CSS` string) + `Rich.Style` objects |

**Node types — visual distinction:**

| Type | Border | Color | Tag |
|---|---|---|---|
| Process | `┌─┐│└─┘` | Green `#00d4a0` | `[P]` |
| Decision | `╔═╗║╚═╝` | Cyan `#00ccee` | `[?]` |
| Terminal | `╭─╮│╰─╯` | Bright green `#44ff88` | `[T]` |
| I/O | `┌─┐│└─┘` | Gold `#ddaa44` | `[IO]` |
| Connector | `┌─┐│└─┘` | Dim green `#1a6a4a` | (none, shows `○`) |
