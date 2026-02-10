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
  - bubbletea
  - lipgloss-v2
  - port
related_files:
  - path: /home/manuel/code/wesen/corporate-headquarters/2026-02-10--grail-js/grail.py
    note: "Complete Textual port of the GRaIL flowchart editor+interpreter (commit b3e4646)"
  - path: /home/manuel/code/wesen/corporate-headquarters/2026-02-10--grail-js/reference.jsx
    note: "Original React/JSX source — source of truth for feature parity"
  - path: /home/manuel/code/wesen/corporate-headquarters/2026-02-10--grail-js/ttmp/2026/02/10/GRAIL-001-PORT-TO-INKJS--port-grail-flowchart-interpreter-from-react-jsx-to-ink-js-terminal-app-with-mouse-support/design-doc/01-bubbletea-port-analysis.md
    note: "Bubbletea v1 CellBuffer port analysis (commit 9d75680, addendum a0b502d)"
  - path: /home/manuel/code/wesen/corporate-headquarters/2026-02-10--grail-js/ttmp/2026/02/10/GRAIL-001-PORT-TO-INKJS--port-grail-flowchart-interpreter-from-react-jsx-to-ink-js-terminal-app-with-mouse-support/design-doc/02-bubbletea-v2-canvas-architecture.md
    note: "Lipgloss v2 Canvas/Layer architecture doc — complete Go port blueprint (commit c29b5c0)"
  - path: /home/manuel/code/wesen/corporate-headquarters/2026-02-10--grail-js/ttmp/2026/02/10/GRAIL-001-PORT-TO-INKJS--port-grail-flowchart-interpreter-from-react-jsx-to-ink-js-terminal-app-with-mouse-support/scripts/bench-lipgloss-percell-vs-ranges.go
    note: "Benchmark: per-cell Render vs StyleRanges vs raw ANSI"
  - path: /home/manuel/code/wesen/corporate-headquarters/2026-02-10--grail-js/ttmp/2026/02/10/GRAIL-001-PORT-TO-INKJS--port-grail-flowchart-interpreter-from-react-jsx-to-ink-js-terminal-app-with-mouse-support/scripts/bench-lipgloss-realistic-buffer.go
    note: "Benchmark: realistic edge buffer with StyleRanges vs Render-per-run"
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

---

## Step 2: Bubbletea v1 Port Analysis (CellBuffer Approach)

The user asked for a deep analysis of porting GRaIL from Textual to Go/Bubbletea. This was a pure design-doc exercise — no code was written. The focus was on understanding what Textual gives you for free (CSS layout, widget tree, `render_line`, `ModalScreen`, focus routing) and what you'd have to build from scratch in Bubbletea's `View() string` model.

The analysis identified three genuinely hard problems: (1) building a CellBuffer with per-cell styling and run-length-encoded ANSI output, (2) modal overlays requiring ANSI string splicing, and (3) keeping manual hit-testing math in sync with rendering. Estimated effort: ~1500 lines of Go vs ~700 Python.

### Prompt Context

**User prompt (verbatim):** "The user asked for a detailed analysis document about porting GRaIL (a Textual/Python TUI application for flowchart/graph visualization and interaction) to Go using the Bubbletea framework. They wanted deep analysis of rendering differences and pseudocode for all system components."

**Assistant interpretation:** Produce a comprehensive design document covering architecture, rendering, mouse handling, and pseudocode for all subsystems of a Go/Bubbletea port.

**Inferred user intent:** Understand the full scope and difficulty of a Go port before committing to implementation.

**Commit (code):** 9d75680 — "Add Bubbletea port analysis design doc with pseudocode"

### What I did
- Researched Bubbletea docs: `Model` interface, `MouseMsg`/`MouseEvent`, `Cmd` pattern
- Researched Lipgloss v1: `Style`, `JoinHorizontal`/`JoinVertical`, `Place`, border types
- Wrote `01-bubbletea-port-analysis.md` (~1130 lines) covering:
  - Architecture mapping (Textual widget tree → Elm Model/Update/View)
  - CellBuffer design with run-length encoded ANSI rendering
  - Drawing primitives (Bresenham, edge exit, node boxes)
  - Mouse coordinate translation (absolute → canvas-relative → world)
  - Modal overlay problem (ANSI splicing)
  - Interpreter port (Goja JS runtime recommendation)
  - Effort breakdown table

### Why
- The user wanted to evaluate feasibility and understand the "framework tax" before writing Go code
- Bubbletea's single-string `View()` model is fundamentally different from Textual's widget tree — the gap needed documenting

### What worked
- The CellBuffer approach is well-understood (it's what ncurses does)
- Pseudocode for all subsystems makes the port mechanically straightforward
- Identifying the three hard problems (CellBuffer, ANSI splicing, hit-test sync) focused attention correctly

### What didn't work
- N/A — design doc only, no implementation

### What I learned
- Bubbletea's `View()` returns ONE string for the entire terminal — there is no widget-level rendering
- Mouse events in Bubbletea give absolute terminal coordinates (0,0 = top-left) — you must manually subtract layout offsets
- `tea.WithMouseAllMotion()` floods Update with events on every cursor move (needed for connect preview)

### What was tricky to build
- **The ANSI splicing problem (§6.3)**: Overlaying a modal on existing styled content requires inserting text at a visual column position within an ANSI-escaped string. Byte position ≠ visual column because escape sequences are invisible. Three solutions identified: (a) don't overlay (full-screen replace), (b) render everything into a single CellBuffer, (c) use `muesli/ansi` for ANSI-aware string cutting. Recommended (b).
- **Effort estimation**: Hard to estimate accurately without implementation. Used line-count comparison with Textual version as proxy. The 2× multiplier (~1500 vs ~700) accounts for building infrastructure that Textual provides.

### What warrants a second pair of eyes
- The effort estimate (§10) is based on pseudocode complexity, not actual Go implementation. Real effort could be higher if Lipgloss edge cases arise (ANSI width miscalculation, etc.)
- The Goja recommendation for eval — binary size impact (~5MB) and whether a simpler expression parser would suffice

### What should be done in the future
- Evaluate Lipgloss v2 Canvas/Layer compositing as an alternative to the CellBuffer approach (done in Step 3)

### Code review instructions
- **Document**: `design-doc/01-bubbletea-port-analysis.md`
- **Key sections**: §2 (CellBuffer), §5 (Mouse), §6 (Modal/ANSI splicing), §10 (Effort)
- **Validate**: Read sections 1-6 alongside `grail.py` — each section maps to a specific part of the Python code

---

## Step 3: Lipgloss v2 Canvas/Layer Architecture (Revised Blueprint)

The user provided detailed documentation about Lipgloss v2.0.0-beta.2's new compositing system (Canvas + Layer) and asked how it changes the port analysis. This was a major revision: the Canvas/Layer model eliminates the three hardest problems from Step 2 — CellBuffer for nodes (Lipgloss styled boxes as Layers), modal overlays (high-Z Layer), and hit testing (Canvas.Hit + Layer.ID).

First an addendum was added to doc 01, then a complete standalone architecture document (doc 02) was written. Doc 02 is a full implementation blueprint with pseudocode for every subsystem, organized around the Layer compositing model.

### Prompt Context

**User prompt (verbatim):** "how do lipgloss v2 layers help you?" (followed by extensive Lipgloss v2 beta.2 documentation)

**Assistant interpretation:** Analyze how the new Canvas/Layer compositing API changes the Bubbletea port architecture.

**Inferred user intent:** Determine whether Lipgloss v2 simplifies the port enough to change the approach.

**Commit (code):** a0b502d — "Addendum: Lipgloss v2 Canvas/Layer analysis — eliminates 3 hardest problems"
**Commit (code):** c29b5c0 — "Add Lipgloss v2 Canvas/Layer architecture doc — complete Go port blueprint"

### What I did
- Added §14-§22 addendum to doc 01 analyzing Canvas/Layer impact
- Wrote complete `02-bubbletea-v2-canvas-architecture.md` (~1700 lines):
  - Full layer map (Z=0 chrome, Z=2 nodes, Z=3 labels, Z=100 modal)
  - Node rendering as Lipgloss styled boxes → Layers with IDs
  - Simplified MiniBuffer (edges/grid only, 4 styles instead of 15)
  - Hit testing via Canvas.Hit() replacing manual coordinate math
  - Clickable toolbar buttons via nested child layers
  - Variables table using Lipgloss v2 table API with BaseStyle
  - Background-aware theming via HasDarkBackground + LightDark
  - Complete View() pipeline: build layers → NewCanvas → Render
- Uploaded both docs to reMarkable as bundled PDF

### Why
- Lipgloss v2 Canvas/Layer is architecturally closer to the React/SVG original (positioned layers with z-index) than either the Textual version or the CellBuffer approach
- It eliminates ~400 lines of the hardest code (CellBuffer for nodes, ANSI splicing, manual hit testing)
- The resulting architecture is cleaner and more maintainable

### What worked
- The layer model maps 1:1 to how the React original works: `<div style={{position:'absolute', zIndex:2}}>` ↔ `NewLayer(content).X(x).Y(y).Z(2).ID("node-7")`
- Canvas.Hit(x, y) makes hit testing consistent with rendering by construction — desync is impossible
- Modal overlay is trivial: one line (`Z(100)`) instead of §6.3's three-page analysis

### What didn't work
- **Tag-in-border problem**: Lipgloss's `Border()` doesn't support injecting text like `┌[P]──────┐` into the top border. Identified three solutions: (a) tag as first content line (simplest), (b) custom top border string (15 lines), (c) post-process ANSI string (fragile). Recommended (a).

### What I learned
- Lipgloss v2 compositing requires Bubble Tea v2 — this is a hard constraint from the Charm team, not optional
- Spaces within a Layer are opaque (they occlude lower layers) — this is correct for our use case (nodes should hide edges underneath)
- Edges must stay in a shared background buffer because they're sparse diagonal lines that can't be rectangular Layers without occluding each other

### What was tricky to build
- **Retained canvas for hit testing**: `m.canvas` must be stored on the Model after `View()` so that `handleMouse()` can call `m.canvas.Hit()` on the same canvas that was rendered. This couples View and Update in a way that feels unusual for Elm architecture, but is necessary.
- **Coordinate systems**: Layer X/Y are in canvas coordinates, mouse events are in terminal-absolute coordinates, nodes are in world coordinates. Three coordinate spaces with two transforms (cam offset, toolbar offset). Documented the mapping clearly to prevent bugs.
- **MiniBuffer scope reduction**: Deciding what stays in the buffer (edges, grid, connect preview) vs what becomes a Layer (nodes, labels, modal, toolbar, panels) required careful analysis of which content is rectangular vs sparse.

### What warrants a second pair of eyes
- The assumption that Canvas.Hit() coordinates match Bubble Tea MouseMsg coordinates (both 0-indexed from terminal top-left). A small test program is recommended before building the full app.
- Whether Lipgloss v2 beta.2 APIs will change before final release — we're building on pre-release software.
- The `AddLayers` API for nested toolbar buttons — needs verification that child layer coordinates are parent-relative vs canvas-absolute.

### What should be done in the future
- Implement the Go port using doc 02 as blueprint
- Write the small coordinate-system verification program (§20, Risk 6)
- Track Lipgloss v2 / Bubble Tea v2 releases for API changes

### Code review instructions
- **Document**: `design-doc/02-bubbletea-v2-canvas-architecture.md`
- **Key sections**: §4.3 (View pipeline), §5 (Node layers), §6 (MiniBuffer), §8 (Hit testing), §11 (Edit modal)
- **Compare with**: doc 01 §2 (CellBuffer) and §6 (Modal) to see what was eliminated
- **Validate**: Cross-reference pseudocode with `grail.py` — every function in the Python code has a corresponding pseudocode block in doc 02

---

## Step 4: Lipgloss StyleRanges Performance Investigation

The user asked about the claim in doc 01 that "styling with Lipgloss is expensive per-cell" and what `StyleRanges` does. This prompted a hands-on investigation: we wrote two benchmark programs to measure the actual performance difference between per-cell `Style.Render()`, per-cell `StyleRanges`, merged-run `StyleRanges`, `Render`-per-run, and raw ANSI.

### Prompt Context

**User prompt (verbatim):** "you say styling with lipgloss is expensive and to use StyledRanges. Can you explain a bit more and tell me where that comes from and what styledranges does?"

**Assistant interpretation:** Verify and explain the performance claim about per-cell Lipgloss styling, explain what StyleRanges actually is, and back it up with data.

**Inferred user intent:** Understand whether the performance advice is real or hallucinated, and make an informed decision about the rendering approach.

### What I did
- Verified `StyleRanges` exists in both Lipgloss v1.1.0 and v2.0.0-beta.2 via `go doc`
- Wrote `bench-lipgloss-percell-vs-ranges.go`: 4 methods on a 200×50 grid
- Wrote `bench-lipgloss-realistic-buffer.go`: realistic edge buffer (90% bg, ~23 runs/row)
- Saved scripts to `scripts/` in the ticket directory

### Why
- The doc 01 claim ("per-cell Lipgloss is expensive, use StyleRanges") needed verification
- Understanding the actual cost is critical for choosing the MiniBuffer rendering strategy

### What worked
- Benchmarks produced clear, actionable numbers

### What didn't work
- The doc 01 claim was **partially wrong**: `StyleRanges` with per-cell ranges (200 ranges/row) is actually 3.5× *slower* than per-cell `Render()`, not faster. The win comes from **merging runs first**, then using either approach.

### What I learned

**Benchmark results (200×50 grid, 100 iterations):**

| Method | µs/frame | Notes |
|---|---|---|
| Per-cell `Style.Render()` | 18,866 | One Render call per character |
| Per-cell `StyleRanges()` | 65,101 | 200 ranges/row — much worse! |
| Merged `StyleRanges()` (5 runs/row) | 1,002 | Run-length encoded first |
| Raw ANSI (5 runs/row) | 24 | Manual escape sequences |

**Output size for one 200-col row:**

| Method | Bytes |
|---|---|
| Per-cell `Render()` | 3,800 |
| Merged `StyleRanges()` | 290 |
| Raw ANSI | 279 |

**Realistic edge buffer (150×40, ~23 runs/row, 200 iterations):**

| Method | µs/frame |
|---|---|
| Merged `StyleRanges()` | 7,374 |
| `Render()`-per-run | 1,885 |

**Key findings:**
1. **Run-length encoding is what matters**, not the styling API you use. Per-cell anything is slow because of overhead per call.
2. `Style.Render()` on merged runs is ~4× faster than `StyleRanges()` on merged runs (1,885 vs 7,374 µs in realistic scenario).
3. Raw ANSI is ~80× faster than any Lipgloss approach, but loses Lipgloss's color profile downsampling.
4. The doc 01 recommendation should have been "merge consecutive same-styled cells into runs, then render each run" — NOT "use StyleRanges instead of Render".

**What `StyleRanges` actually does:** Given a plain string and a list of `Range{Start, End int; Style Style}`, it applies each Style to the corresponding substring while preserving existing ANSI escapes in the base string. It's designed for syntax-highlighting use cases where you have a pre-rendered string and want to colorize portions of it. It's slower than `Render`-per-run because it must do ANSI-aware string surgery on the base string.

### What was tricky to build
- Understanding *why* `StyleRanges` is slower: it must parse existing ANSI escapes in the base string to correctly splice in new styles. `Render()` on a plain substring has no existing escapes to parse — it just wraps with open/close sequences. This makes `Render`-per-run fundamentally cheaper.

### What warrants a second pair of eyes
- The doc 01 and doc 02 MiniBuffer rendering sections recommend `StyleRanges` — this should be updated to recommend `Render`-per-run instead.
- At ~1.9ms/frame for a 150×40 buffer with Render-per-run, the MiniBuffer is fast enough (well under 16ms frame budget). But if the canvas grows significantly, raw ANSI (~24µs) would be the fallback.

### What should be done in the future
- Update doc 02 §6.2 to recommend `Render`-per-run instead of `StyleRanges`
- Consider whether raw ANSI is worth the loss of color profile downsampling for the edge buffer (probably not — 1.9ms is fine)

### Code review instructions
- **Scripts**: `scripts/bench-lipgloss-percell-vs-ranges.go`, `scripts/bench-lipgloss-realistic-buffer.go`
- **Run**: `cd scripts && go run bench-lipgloss-percell-vs-ranges.go` and `go run bench-lipgloss-realistic-buffer.go`
- **Cross-reference**: doc 01 §2.3 and doc 02 §6.2 — both need correction

### Technical details

**`lipgloss.StyleRanges` API (exists in v1.1.0 and v2.0.0-beta.2):**
```go
func StyleRanges(s string, ranges ...Range) string
type Range struct { Start, End int; Style Style }
func NewRange(start, end int, style Style) Range
```

"StyleRanges allows to, given a string, style ranges of it differently. The function will take into account existing styles. Ranges should not overlap."

**`lipgloss.Style.Render` API:**
```go
func (s Style) Render(strs ...string) string
```

"Render applies the defined style formatting to a given string."

The performance difference comes from `StyleRanges` needing to parse/preserve existing ANSI in the base string, while `Render` on a plain substring just wraps it.

---

## Step 5: Component-by-Component Build Plan

The user asked how to build the Go/Bubbletea port incrementally, with reusable components that can serve other projects. This required decomposing the ~1100-line monolith into packages with clear boundaries, ordering them by dependency, and ensuring every step produces a running program with visible progress.

The key design decision was identifying four genuinely reusable packages (`cellbuf`, `drawutil`, `graphmodel`, `tealayout`) vs two GRaIL-specific packages (`flowinterp`, `grailui`). The build order follows the dependency graph: pure libraries first (no UI), then scaffold, then visual layers, then interaction, then interpreter.

### Prompt Context

**User prompt (verbatim):** "Ok, now if we were to build a grails version bit bubbletea and lipgloss v2, because it's so complex, we would like to build it up component by component, especially so we can reuse them in other contexts. How would you approach this, step by step ?"

**Assistant interpretation:** Design a build plan that decomposes GRaIL into reusable packages, ordered by dependency, with each step producing something testable.

**Inferred user intent:** Get a concrete implementation roadmap where (a) each step is independently validatable, (b) the reusable pieces are cleanly separated for use in future projects, and (c) architectural risks are caught early.

**Commit (code):** (pending — doc only, no code changes)

### What I did
- Analyzed the dependency graph of all GRaIL subsystems
- Identified 4 reusable packages + 2 app-specific packages
- Designed 12 build steps ordered by dependency
- Defined package APIs with Go type signatures
- Placed 4 risk checkpoints at critical junctures
- Estimated effort: ~1180 lines, ~7.5 hours total

### Why
- A monolithic build risks discovering architectural problems late (e.g., Canvas.Hit coordinate mismatch)
- Reusable packages (`cellbuf`, `drawutil`, `graphmodel`, `tealayout`) solve problems common to any terminal-based spatial editor
- Each step from 4 onward produces a visible, runnable program

### What worked
- The dependency graph has a clean layering: pure math → data model → UI scaffold → visual layers → interaction → interpreter → integration
- The `graphmodel` package naturally uses Go generics (`Graph[N Spatial, E any]`) to avoid coupling to GRaIL node types
- The `tealayout.LayoutBuilder` declarative API (`TopFixed → RightFixed → Remaining → Build`) maps cleanly to GRaIL's layout needs

### What didn't work
- N/A — planning only, no implementation yet

### What I learned
- The package boundary test ("could another app use this?") cleanly separates infrastructure from application logic
- The interpreter does NOT belong in `pkg/` — its node-type dispatch is GRaIL-specific, and making it generic would require a plugin architecture that isn't worth the complexity
- Step 4 (scaffold) is the most important step despite being the smallest: it validates that the entire v2 beta stack works before investing further

### What was tricky to build
- **Package boundary for `graphmodel`**: The `Spatial` interface (`Pos()`, `Size()`, `Center()`, `Bounds()`) is the key abstraction that decouples the graph from GRaIL node types. Finding the right interface — not too specific (would couple to GRaIL) and not too abstract (would be useless) — required iterating on what operations the graph actually needs. `image.Point` and `image.Rectangle` from the standard library provide the right vocabulary types.
- **Step ordering around mouse interaction**: Step 9 (mouse) depends on both Step 6 (nodes) and Step 7 (edges) because selection highlighting and connect preview need visual feedback. But it also depends on the Canvas.Hit coordinate validation (Checkpoint B), which should happen before committing to the hit-testing approach. Placed Checkpoint B explicitly before Step 9.

### What warrants a second pair of eyes
- The `graphmodel.Graph` generic API — whether `Graph[N Spatial, E any]` is the right level of genericity, or whether simpler concrete types would be more practical
- Whether `tealayout.LayoutBuilder` is over-engineered for what's essentially 5 lines of arithmetic — might be simpler as a plain function
- The time estimates are optimistic (assume no v2 beta surprises)

### What should be done in the future
- Execute the 12-step plan
- After Step 4 (scaffold), evaluate whether the v2 beta stack is stable enough to continue
- After Step 9 (mouse), evaluate whether Canvas.Hit performance is acceptable with ~50+ layers

### Code review instructions
- **Document**: `design-doc/03-component-build-plan.md`
- **Key sections**: §2 (Package Map), §3 (Build Steps), §7 (Risk Checkpoints)
- **Validate**: Check that the dependency graph in §3 has no cycles and each step's "depends on" list is correct

---

## Step 6: Implement GRAIL-002 — cellbuf Package

Built the `pkg/cellbuf` package: a reusable 2D character buffer with per-cell styling and efficient Lipgloss-based rendering. This is the first real Go code in the port — the foundation for edge and grid drawing.

Hit a significant dependency issue: Lipgloss v2.0.0-beta.3 is broken against newer `x/ansi` (charmbracelet/lipgloss#599). The parent `go.work` workspace pulls in incompatible dependency versions. Resolved by pinning Lipgloss v2.0.0-beta.2 + x/ansi v0.8.0 and building with `GOWORK=off`.

### Prompt Context

**User prompt (verbatim):** "add tasks to implement GRAIL-002, then work through the tasks one by one, ocmmitting at each point, checking the task off updating the diary. Make sure to test things (with tmux if you need interactivity) as you go."

**Assistant interpretation:** Implement the cellbuf package end-to-end, committing after each task, with tests.

**Inferred user intent:** Start actual code implementation following the component plan, with proper process discipline.

**Commit (code):** 9e6a6db — "GRAIL-002: Define Cell, StyleKey, Buffer types"
**Commit (code):** d869ac9 — "GRAIL-002: Implement Render() with run-length encoded Style.Render()-per-run"
**Commit (code):** b25d36b — "GRAIL-002: Unit tests (13 pass) + benchmarks"

### What I did
- Created `pkg/cellbuf/buffer.go`: `Cell`, `StyleKey`, `Buffer` structs + `New`, `Set`, `SetString`, `Fill`, `InBounds`
- Created `pkg/cellbuf/render.go`: `Render()` with run-length encoded `Style.Render()`-per-run
- Created `pkg/cellbuf/buffer_test.go`: 13 unit tests + 2 benchmarks
- Set up `go.mod` with lipgloss/v2 v2.0.0-beta.2
- Created `Makefile` with `GOWORK=off` to avoid workspace dep conflicts

### Why
- First package in the 12-step build plan — foundation for edge rendering
- Zero dependencies on Bubbletea or app logic — pure library

### What worked
- All 13 tests pass on first run
- Run-length encoding correctly merges consecutive same-styled cells
- Out-of-bounds writes silently ignored (no panics)
- Missing style keys fall back to plain text

### What didn't work
- **Lipgloss v2.0.0-beta.3 broken**: `x/ansi` API changed (methods like `Italic()` now require `bool` argument). Beta.3 was compiled against old API. See github.com/charmbracelet/lipgloss/issues/599.
- **Workspace dep conflicts**: Parent `go.work` pulls `x/ansi` v0.11.3 from other modules, which is incompatible with lipgloss beta.2's `x/ansi` v0.8.0. Had to comment out this module from `go.work` and use `GOWORK=off`.
- **Benchmark target missed**: Target was <2ms for 200×50. Got 3.9ms. The allocations (12K) are overwhelmingly inside `lipgloss.Style.Render()`, not our code. Realistic 150×40 buffer: 2.5ms — acceptable (15% of 16ms frame budget).

### What I learned
- Lipgloss `Style.Render()` allocates heavily internally (~50 allocs per call). With ~240 runs per frame, that's ~12K allocs. This is the cost of using Lipgloss over raw ANSI. For our buffer sizes, it's acceptable.
- Go workspace (`go.work`) dependency resolution can force incompatible versions when modules in the workspace have conflicting transitive deps. `GOWORK=off` is the escape hatch.
- Lipgloss v2 is in active development with breaking changes between beta releases. Pin exact versions.

### What was tricky to build
- **The dependency maze**: Three attempts to fix the build — (1) upgrade to beta.3 (still broken), (2) upgrade all x/* deps (different error), (3) downgrade to beta.2 with exact x/ansi v0.8.0. The root cause is that the Go module system's MVS picks the *highest* version of any transitive dep, and the workspace amplifies this by including all modules' deps.
- **Render optimization**: Initial implementation allocated a `[]rune` per run. Changed to reuse a single `[]rune` buffer sized to `b.W`. This saved ~200 allocs/frame but the dominant cost remains inside Lipgloss.

### What warrants a second pair of eyes
- Whether `GOWORK=off` will cause problems as we add more packages that depend on each other within this module
- Whether 2.5ms/frame for the edge buffer is acceptable when combined with node layer rendering and Canvas compositing overhead

### What should be done in the future
- ~~Monitor lipgloss#599 for a fix — upgrade to beta.3+ when available~~ **RESOLVED** — see Step 7
- If Render performance becomes a bottleneck, consider a raw-ANSI fast path (24µs vs 2.5ms per the perf investigation)

### Code review instructions
- **Files**: `pkg/cellbuf/buffer.go`, `pkg/cellbuf/render.go`, `pkg/cellbuf/buffer_test.go`
- **Run**: `make test` (13 tests), `make bench` (2 benchmarks)
- **Key**: The `Render` function in `render.go` — verify run-length encoding correctness and that the sentinel style flush works for the last run in each row

---

## Step 7: Fix Lipgloss v2 Dependency — Switch to charm.land Module Path

The Charm team has moved their v2 development to `charm.land/*` module paths.
Switching from `github.com/charmbracelet/lipgloss/v2` to `charm.land/lipgloss/v2@v2-exp`
resolves the beta.3 x/ansi API breakage that forced us to pin beta.2 in Step 6.

### Prompt Context

**User prompt (verbatim):** User pointed to https://github.com/charmbracelet/lipgloss/issues/599 where a Charm team member posted: "We've moved the latest v2 versions to our charm.land domains... `go get charm.land/lipgloss/v2@v2-exp`". User asked: "Did this not work to fix the lipgloss beta 3 issue?"

**Assistant interpretation:** Re-attempt the charm.land import (which had failed earlier with a module path mismatch) now that the user confirmed it's the official fix.

**Inferred user intent:** Fix the dependency pinning workaround so we use the latest v2 code.

**Commit:** 82d52e2 — "Switch from github.com/charmbracelet/lipgloss/v2 to charm.land/lipgloss/v2@v2-exp"

### What I did
1. Ran `go get charm.land/lipgloss/v2@v2-exp` — this time it resolved correctly to `v2.0.0-beta.3.0.20260210014823-2f36a2f1ba17`
2. Updated imports in `pkg/cellbuf/render.go` and `pkg/cellbuf/buffer_test.go` from `github.com/charmbracelet/lipgloss/v2` to `charm.land/lipgloss/v2`
3. Ran `go mod tidy` — the old `github.com/charmbracelet/lipgloss/v2` dependency was removed entirely
4. Upgraded transitive deps: `x/ansi` v0.8.0 → v0.11.2, `x/cellbuf` v0.0.13 → v0.0.15, `colorprofile` v0.3.1 → v0.3.3
5. Verified all 13 tests pass, benchmarks unchanged (~2.7ms realistic, ~3.4ms worst-case)

### Why
The `github.com/charmbracelet/lipgloss/v2` beta releases (both beta.2 and beta.3) were compiled against `x/ansi` v0.8.0. The `x/ansi` package changed its API (methods like `Italic()`, `Underline()`, `Reverse()` now take `bool` args; `SlowBlink` was removed). Any upgrade of `x/ansi` via transitive deps or Go workspace would break the build.

The `charm.land/lipgloss/v2` module is the new canonical import path for v2 development. The v2-exp branch tip is compiled against `x/ansi` v0.11.2, which matches the new API.

### What worked
- `charm.land/lipgloss/v2@v2-exp` resolved successfully (it had failed earlier with a module path mismatch — apparently that was fixed server-side between attempts)
- Clean drop of the old module: `go mod tidy` removed `github.com/charmbracelet/lipgloss/v2` and all its pinned-old transitive deps
- No code changes needed beyond the import path — the `lipgloss.NewStyle()`, `Style.Render()`, `lipgloss.Color()` APIs are identical
- Benchmarks are the same — no performance regression from the upgrade

### What didn't work
- The initial attempt earlier (in Step 6) to use `charm.land/lipgloss/v2@v2-exp` failed with: `module declares its path as: github.com/charmbracelet/lipgloss/v2 but was required as: charm.land/lipgloss/v2`. This was likely a transient issue where the tagged module hadn't fully propagated, or the v2-exp branch was updated between our attempts.
- Upgrading `clipperhouse/displaywidth` to v0.10.0 then broke `x/cellbuf` v0.0.13 (still compiled against old x/ansi). The dependency chain was: upgrading ANY one dep triggers MVS to pull newer versions of shared deps, which breaks other packages compiled against old versions. The only clean fix was to switch everything to the new `charm.land` module path in one step.

### What I learned
- **Module path migration is the real fix, not version pinning.** When a library has breaking transitive deps, the maintainers need to release under a new module path that declares the correct dependency versions. That's what `charm.land/lipgloss/v2` does.
- **Go's MVS (Minimum Version Selection) amplifies breakage.** When module A requires `x/ansi@v0.8.0` and module B requires `x/ansi@v0.11.0`, Go picks v0.11.0 — which may break A. In a workspace with many modules, this cascading version bumping is almost guaranteed to hit incompatibilities during breaking API changes.
- **`GOWORK=off` is no longer needed** for this specific issue, but is still needed to avoid the workspace pulling in different versions of shared deps from other modules. The `Makefile` still sets it.

### What was tricky to build
- **The cascading upgrade problem.** When we tried to fix beta.3 by upgrading x/ansi, that broke x/cellbuf. Upgrading x/cellbuf then re-broke lipgloss because Go MVS pulled x/ansi even higher. Every "fix" created a new break. The only clean solution was to use a module that was compiled against the newest versions of everything — which is exactly what `charm.land/lipgloss/v2@v2-exp` provides.

### What warrants a second pair of eyes
- We're now on a pseudo-version (`v2.0.0-beta.3.0.20260210014823-2f36a2f1ba17`) rather than a tagged release. This is a commit from the v2-exp branch tip as of Feb 10, 2026. It should be stable enough for development but may need updating as new tags are cut.
- The `charm.land` domain is new — worth verifying it stays the canonical path and doesn't get changed again.

### What should be done in the future
- When `charm.land/lipgloss/v2` gets a proper tagged release (v2.0.0-beta.4 or v2.0.0), switch to the tag instead of the pseudo-version
- Update the design docs (01, 02, 03) to reference `charm.land/lipgloss/v2` instead of `github.com/charmbracelet/lipgloss/v2` in import examples
- When `charm.land/bubbletea/v2` is needed (Step 4: scaffold), use the same `@v2-exp` pattern

### Code review instructions
- **Commit**: 82d52e2
- **Files changed**: `pkg/cellbuf/render.go`, `pkg/cellbuf/buffer_test.go`, `go.mod`, `go.sum`
- **Run**: `make test` (all 13 pass), `make bench` (comparable to before)
- **Verify**: `grep lipgloss go.mod` shows only `charm.land/lipgloss/v2`, no `github.com/charmbracelet`

---

## Step 8: Implement GRAIL-003 — drawutil Package

Built `pkg/drawutil`: terminal drawing primitives ported from `grail.py`. Bresenham line algorithm, directional line/arrow character lookup, edge exit-point geometry, and convenience functions that draw into a `cellbuf.Buffer`. Straightforward port — no surprises.

### Prompt Context

**User prompt (verbatim):** "ok, let's move on. close the ticket, start with the next one, process task by task, keeping a diary and committing to git and checking the task off as you go."

**Assistant interpretation:** Close GRAIL-002, implement GRAIL-003 end-to-end with incremental commits.

**Inferred user intent:** Continue systematic execution of the build plan.

**Commits:** ef63887 (Bresenham+LineChar+ArrowChar), 158f018 (EdgeExit), 590cae0 (Draw functions), f8d83f8 (18 tests), 5c58842 (close+demo update)

### What I did
- `pkg/drawutil/line.go`: `Bresenham()` returning `[]image.Point`, `LineChar()`, `ArrowChar()`, local `abs()` helper
- `pkg/drawutil/edge.go`: `EdgeExit()` — border exit point from rectangle toward target using normalized dx/dy
- `pkg/drawutil/grid.go`: `DrawGrid()` with camera offset and non-negative modulus
- `pkg/drawutil/draw.go`: `DrawLine()`, `DrawArrowLine()`, `DrawDashedLine()` — all draw into `*cellbuf.Buffer`
- `pkg/drawutil/line_test.go`: 18 unit tests covering Bresenham (horizontal, vertical, diagonal, steep, reverse, zero-length), LineChar, ArrowChar, EdgeExit (4 sides + same-center), Draw functions, grid with camera offset
- Updated `cmd/cellbuf-demo/` to use drawutil for edges and grid

### Why
- Direct port of Python drawing code from `grail.py:303-370`
- Foundation for GRAIL-008 (edge rendering) and GRAIL-010 (connect preview)

### What worked
- All 18 tests passed on first run (31 total across cellbuf+drawutil)
- Bresenham algorithm is a direct translation — Go's integer arithmetic makes it cleaner than Python
- `DrawGrid` with camera offset handles negative world coordinates correctly via non-negative modulus
- The demo visually confirms: diagonal Bresenham lines, EdgeExit choosing correct sides, arrowheads pointing right direction

### What didn't work
- Nothing — this was a clean port with no issues

### What I learned
- Go's `%` operator can return negative values for negative operands (unlike Python). Needed a `mod()` helper for grid calculations.
- Returning `[]image.Point` from Bresenham (instead of a channel/iterator) is simpler and the slices are small enough that allocation is negligible.

### What was tricky to build
- Nothing particularly tricky — the Python reference was clear and the port was mechanical.

### What warrants a second pair of eyes
- `EdgeExit` uses float64 for normalized comparison. The integer truncation in the comparison (`abs(int(ndx*1000))`) is slightly ugly — might be cleaner with a direct float comparison.

### What should be done in the future
- Consider adding orthogonal-only line routing (horizontal-then-vertical segments) for cleaner flowchart edges — the current Bresenham produces diagonal lines.

### Code review instructions
- **Files**: `pkg/drawutil/line.go`, `pkg/drawutil/edge.go`, `pkg/drawutil/grid.go`, `pkg/drawutil/draw.go`, `pkg/drawutil/line_test.go`
- **Run**: `make test` (31 tests across both packages)
- **Visual**: `GOWORK=off go run ./cmd/cellbuf-demo/` — shows edges using EdgeExit + DrawArrowLine

---

## Step 9: Implement GRAIL-004 — graphmodel Package

Built `pkg/graphmodel`: a generic spatial graph with positioned nodes, labeled edges, stable insertion-order iteration, and hit testing. This is a pure data model with no UI dependencies — fully reusable.

### Prompt Context

Continuing from the same "close and move on" prompt. GRAIL-003 was clean, GRAIL-004 equally so.

**Commits:** a07d06b (Graph struct + all operations), c40575d (20 tests)

### What I did
- `pkg/graphmodel/spatial.go`: `Spatial` interface (Pos, Size), `CenterOf()`, `BoundsOf()` free functions
- `pkg/graphmodel/graph.go`: `Graph[N Spatial, E any]` with `AddNode`, `RemoveNode`, `MoveNode`, `AddEdge`, `RemoveEdge`, `OutEdges`, `InEdges`, `HitTest`, `NodesInRect`
- `pkg/graphmodel/graph_test.go`: 20 unit tests covering all operations

### Why
- Generic graph is the core data model for GRaIL — stores flowchart nodes and edges
- Reusable for other graph-based TUI apps (dependency viewers, network diagrams, etc.)

### What worked
- All 20 tests passed on first run (51 total across 3 packages)
- `MoveNode` with setter function callback cleanly solves the "Go interfaces don't have setters" problem
- `HitTest` reverse iteration correctly returns topmost (last-inserted) node
- `RemoveNode` cascade-deletes connected edges — verified by test

### What didn't work
- Nothing — clean implementation

### What I learned
- The `MoveNode(id, pos, setPos func(*N, image.Point))` pattern is idiomatic for generic types where the concrete type needs mutation. The caller provides the setter, the generic code provides the lookup.
- `orderIDs []int` for stable iteration is simpler than a linked list and has negligible overhead for <100 nodes.

### What was tricky to build
- Nothing — the plan was detailed enough that implementation was mechanical.

### What warrants a second pair of eyes
- The `AddEdge` duplicate check is O(n) over all edges. For GRaIL's ~20 edges this is fine, but a production graph library would want an adjacency set.
- `RemoveNode` does `g.edges[:0]` filter-in-place which reuses the underlying array — correct but subtle.

### What should be done in the future
- Consider adding `NodeCount()`, `EdgeCount()` convenience methods
- If GRaIL needs undo/redo, the graph operations need to return undo closures

### Code review instructions
- **Files**: `pkg/graphmodel/spatial.go`, `pkg/graphmodel/graph.go`, `pkg/graphmodel/graph_test.go`
- **Run**: `make test` (51 tests across 3 packages)
- **Key**: `HitTest` reverse iteration and `RemoveNode` edge cleanup

---

## Step 10: Fix Edge Rendering — Per-Point Line Characters

User pointed out that diagonal edges looked ugly (rows of `\\\\`) and that edge lines were overwriting node borders. Reviewed the Python reference in `grail.py:410-420` and found the root cause: our `DrawLine`/`DrawArrowLine`/`DrawDashedLine` were using a single `LineChar(dx, dy)` for the entire line based on the overall direction, but the Python version computes the character **per point** based on the direction to the *next* point.

Also fixed `.gitignore` (`cellbuf-demo` was matching the `cmd/cellbuf-demo/` directory, blocking commits). Updated the visual demo to use all 3 packages together (graphmodel stores the flowchart, drawutil renders edges, cellbuf outputs).

### Prompt Context

**User prompt (verbatim):** "Lines should not overwrite the borders, except for arrow ports, it's ugly. Also, should END be closed?" → then "END should be closed, not closer, on the right, I think." → then "what were the textual primitives used for the lines, maybe the textual algorithms would help?"

**Assistant interpretation:** Review how the Python reference draws edges and apply the same per-segment technique.

**Inferred user intent:** Fix visual quality of edge rendering to match the Python version.

**Commit:** 09698b6 — "Fix edge rendering: per-point line characters instead of global direction"

### What I did
- Added `pointChar(pts []image.Point, i int) rune` helper that looks at next/prev point to determine line character
- Rewrote `DrawLine`, `DrawArrowLine`, `DrawDashedLine` to call `pointChar` per point instead of one `LineChar` for the whole line
- Updated `DrawArrowLine` arrowhead to use final segment direction (last point - second-to-last) instead of overall direction
- Fixed `.gitignore`: `/cellbuf-demo` (root binary only) instead of `cellbuf-demo` (matched cmd/ directory too)
- Rewrote demo to build flowchart in a `graphmodel.Graph`, render edges via `drawutil.EdgeExit` + `DrawArrowLine`, and show `HitTest` results

### Why
- Python `grail.py:410-420` computes `_lch(pts[i+1][0] - px, pts[i+1][1] - py)` per point — this produces correct `│`→`\`→`─` transitions at direction changes in the Bresenham path
- Our single-character approach produced `\\\\\\\\` runs because the overall direction was diagonal even though individual segments alternate between horizontal and vertical steps

### What worked
- Per-point character selection matches Python output quality
- All 51 tests still pass (the tests check character types for horizontal/vertical lines, which are unaffected)
- Node-on-top drawing (edges first, nodes second) correctly covers edge endpoints on borders — same as Python
- END box was actually closed correctly all along; the user's concern was resolved once edge rendering cleaned up

### What didn't work
- Initial assumption that EdgeExit needed to return 1 cell outside the border was wrong — the Python reference returns ON the border and relies on node-on-top overwrite, which works fine

### What I learned
- **Per-point vs per-line character selection is critical for Bresenham.** A Bresenham line from (0,0) to (20,5) takes many horizontal steps with occasional vertical steps. Using the overall direction gives `\` for every cell, but per-point gives `─` for horizontal steps and `\` only at the diagonal transitions.
- **Always check the reference implementation before "fixing" things.** The Python code had already solved this problem — reading 10 lines of `grail.py` would have prevented the bug in the first place.

### What was tricky to build
- Nothing — once the Python reference was consulted, the fix was obvious and mechanical.

### What warrants a second pair of eyes
- The `pointChar` helper falls back to prev-point direction for the last point. For a single-point line (zero length), it returns `│` by default (dx=0, dy=0). This edge case might not be ideal.

### What should be done in the future
- Consider orthogonal-only routing (horizontal segments + vertical segments) for cleaner flowchart aesthetics — diagonal Bresenham lines are technically correct but visually noisy for a flowchart editor

### Code review instructions
- **Files**: `pkg/drawutil/draw.go`, `cmd/cellbuf-demo/main.go`
- **Run**: `make test` (51 tests), `GOWORK=off go run ./cmd/cellbuf-demo/` (visual)
- **Compare**: Python `grail.py:410-420` vs Go `drawutil.pointChar` — should produce equivalent characters

---

## Step 11: Implement GRAIL-005 — Scaffold (Checkpoint A) ⭐

**This was the most important step in the entire project.** The scaffold validates that the Bubbletea v2 + Lipgloss v2 stack actually works. Three major API discoveries that invalidate assumptions in the design docs:

1. **`View()` returns a `tea.View` struct, not a `string`.** AltScreen and MouseMode are fields on the View, not options passed to `NewProgram()`.
2. **`Canvas.Compose(layer)` ignores the layer's X/Y/Z.** Layer positioning only works through `Compositor`, not direct canvas composition.
3. **`Compositor` is the key API** — it flattens layer hierarchies, handles Z-sorting, and provides `Hit()` for mouse interaction. This replaces the `Canvas.Hit()` API from the design docs.

### Prompt Context

Continuing from "close and move on" prompt. This was Checkpoint A from the build plan.

**Commit:** cf7d75c — "GRAIL-005: Scaffold — Bubbletea v2 + Lipgloss v2 Compositor app"

### What I did
1. Added `charm.land/bubbletea/v2@v2-exp` dependency (resolved to v2.0.0-rc.2)
2. Explored the v2 API surface with `go doc` — discovered the View struct, Compositor, and Canvas APIs
3. Built a minimal app: dark green background, title at top, mouse coords at bottom, orange crosshair at mouse position
4. Tested initial build with `Canvas.Compose(layer)` — layers didn't position correctly
5. Read `layer.go` source — discovered `Layer.Draw()` ignores X/Y, just draws at the passed `area`
6. Found `Compositor` — creates from layers, flattens hierarchy, sorts by Z, handles positioning
7. Rewrote to use `NewCompositor(layers...) → canvas.Compose(comp)`
8. Verified in tmux: title at top, footer at bottom, background fills screen, `q` exits cleanly

### Why
- Checkpoint A: validate the entire v2 stack before investing in 8 more tickets
- Every subsequent step depends on this: layout regions, node rendering, mouse interaction

### What worked
- `charm.land/bubbletea/v2@v2-exp` resolved to RC2 and compiled cleanly
- `tea.View` struct with `AltScreen = true` and `MouseMode = tea.MouseModeAllMotion` — clean declarative config
- `Compositor` positioning works correctly — title at Y=0, footer at Y=height-1, crosshair at mouse position
- `q` key exits cleanly to normal terminal
- No panics on startup, rendering, or exit

### What didn't work
- **First attempt used `Canvas.Compose(layer)` directly** — all layers rendered at (0,0) because `Layer.Draw()` ignores X/Y positioning. The X/Y fields are metadata consumed only by `Compositor.flatten()`.
- **Design docs assumed `lipgloss.NewCanvas(layers...).Render()`** — this API doesn't exist. Canvas takes (width, height), not layers.
- **Design docs assumed `Canvas.Hit(x, y)`** — this API is on `Compositor`, not Canvas. `Compositor.Hit(x, y)` returns a `LayerHit` with ID/bounds.
- **Mouse tracking couldn't be verified via tmux scripting** — tmux doesn't relay synthetic mouse events to the child process's mouse reporting mode. Need interactive testing.

### What I learned
- **The Lipgloss v2 architecture is `Layer → Compositor → Canvas → Render()`.** Not `Layer → Canvas.Compose → Render()` as the design docs assumed. This is a cleaner separation: Layer is a pure data structure, Compositor handles flattening/sorting/hit-testing, Canvas handles cell-level rendering.
- **`tea.View` struct is the v2 way to configure terminal behavior.** No more `tea.WithAltScreen()` or `tea.WithMouseAllMotion()` program options — they're per-view fields. This means you can switch between alt-screen and inline, or toggle mouse mode, per render cycle.
- **`Compositor.Hit(x, y)` replaces the hypothetical `Canvas.Hit()`** from design doc 02. Same concept (topmost layer ID at coordinates), different location. The hit test works on flattened/z-sorted layers with absolute positions.
- **Bubbletea v2 RC2 is stable enough for development.** No crashes, clean lifecycle, familiar Update/View pattern.

### What was tricky to build
- **Finding the right API path.** The `go doc` output for `Canvas.Compose` says it takes `uv.Drawable`, and `Layer` implements `uv.Drawable`. So `canvas.Compose(layer)` compiles fine — but it draws the layer content at the canvas bounds, ignoring X/Y. You have to read the `Layer.Draw()` source to realize it doesn't use its own position. The `Compositor` is what does the coordinate math. This took 3 test scripts to figure out.
- **Background filling.** `lipgloss.NewStyle().Width(w).Height(h).Background(...).Render("")` produces a styled block but Canvas trailing-space stripping can eat it. Explicit `strings.Repeat(" ", w)` × h lines works reliably.

### What warrants a second pair of eyes
- Whether `Compositor.Render()` (which creates a temporary canvas internally) is adequate, or whether we should always use `canvas.Compose(comp)` for fixed-size output. The scaffold uses the latter for predictable sizing.
- The `OnMouse` field on `tea.View` — this allows view-level mouse dispatch without going through `Update()`. Might be useful for per-layer mouse handling later, but needs investigation.

### What should be done in the future
- **Update design docs 02 and 03** to reflect the correct API: `Compositor` not `Canvas` for positioning and hit testing, `tea.View` struct not program options for AltScreen/MouseMode
- **Checkpoint B (GRAIL-010)**: verify `Compositor.Hit(x, y)` coordinates match `MouseMsg.Mouse().X/Y`
- ~~**Interactive mouse test**: run `GOWORK=off go run ./cmd/grail/` in a real terminal~~ **DONE** — user confirmed mouse tracking works
- **Performance**: measure Compositor.Render() time with 50+ layers (node count for a typical GRaIL flowchart)

### Code review instructions
- **File**: `cmd/grail/main.go`
- **Run**: `GOWORK=off go run ./cmd/grail/` — should show dark green screen, title, footer, orange `+` crosshair. Press `q` to exit.
- **Key discovery**: `Canvas.Compose(layer)` ≠ positioned rendering. Must use `Compositor` for X/Y/Z.

---

## Step 12: Implement GRAIL-006 — tealayout Package

Built `pkg/tealayout`: declarative layout builder + chrome layer helpers. Fast and clean — completed all 4 reusable `pkg/` packages. One bug found: `image.Rect` auto-canonicalizes negative rects, so zero-size terminals produced non-empty regions. Fixed with explicit bounds check before calling `image.Rect`.

### Prompt Context

Continuing sequential implementation. GRAIL-006 is the last `pkg/` package.

**Commits:** 8eac84f (LayoutBuilder), 76afe19 (chrome helpers), f6923ae (9 tests + fix)

### What I did
- `pkg/tealayout/regions.go`: `LayoutBuilder` with `TopFixed`, `BottomFixed`, `RightFixed`, `Remaining`, `Build`
- `pkg/tealayout/chrome.go`: `ToolbarLayer`, `FooterLayer`, `VerticalSeparator`, `ModalLayer`, `FillLayer`
- `pkg/tealayout/regions_test.go`: 9 tests including overlap detection, zero-size terminal, centering

### What worked
- Layout builder correctly computes non-overlapping regions (verified by pair-wise overlap test)
- 80×24 with toolbar(3) + footer(1) + panel(34) → canvas = 46×20 at (0,3) ✅
- Modal centering works with box styles including border and padding
- FillLayer creates positioned background layers from regions

### What didn't work
- **Zero-size terminal test failed initially.** `image.Rect(0, 3, 0, 0)` auto-canonicalizes to `(0,0)-(0,3)` which has Dy()=3, not 0. Fixed by checking bounds explicitly before calling `image.Rect`.

### What I learned
- `image.Rect` always canonicalizes (swaps min/max so min <= max). You cannot use Dx()<0 to detect degenerate rects after construction. Must validate inputs before constructing.

### Code review instructions
- **Files**: `pkg/tealayout/regions.go`, `pkg/tealayout/chrome.go`, `pkg/tealayout/regions_test.go`
- **Run**: `make test` (60 tests across 4 packages)
- **Key**: `Remaining` degenerate bounds check

---

## Step 13: Implement GRAIL-007 — Nodes on Canvas

First visual milestone: the app shows 7 styled flowchart boxes on a dark background with camera panning. Created `internal/grailui/` package with Model/Update/View + data types + styles. The Compositor approach from Step 11 works perfectly for node positioning.

### Prompt Context

Continuing sequential build. User confirmed scaffold mouse tracking works, asked "how can I test [tealayout]?" — moved on since tealayout is pure math.

**Commit:** c2af1b2

### What I did
- `internal/grailui/data.go`: FlowNodeData (implements Spatial), FlowEdgeData, NodeTypeInfo registry, MakeInitialGraph (7-node sum 1..5 demo)
- `internal/grailui/styles.go`: Color palette (CRT green theme), `borderForType` (rounded/double/normal), `c()` helper for `lipgloss.Color`
- `internal/grailui/layers.go`: `buildNodeLayers` — builds positioned Layers from graph nodes with border styles, type tags, centered labels, selection/execution color overrides, visibility culling
- `internal/grailui/model.go`: Model struct with Graph, camera, tool, selection state
- `internal/grailui/update.go`: Key handling (q quit, arrows pan, s/a/c tools), mouse tracking
- `internal/grailui/view.go`: Layout (toolbar+canvas+footer), composes background + chrome + node layers via Compositor
- Simplified `cmd/grail/main.go` to just create Model and run

### What worked
- All 7 nodes render with correct border types: `╭╮╰╯` terminal, `╔╗╚╝` decision, `┌┐└┘` process/io
- Type tags render in top border: `[T]`, `[P]`, `[?]`, `[IO]`
- Text labels centered within nodes
- Camera panning works smoothly (3 cells per arrow key press)
- Footer shows live mouse coords, camera position, selection, node count
- Visibility culling skips off-screen nodes

### What didn't work
- **`lipgloss.Color` is a function in v2, not a type.** `lipgloss.Color("#00d4a0")` returns `color.Color`. Fixed by using `color.Color` as the variable type and a `c()` shorthand helper.
- **Edge color variables declared but unused** — had to use `_ = c(...)` blanks to avoid compiler errors. Will be used in GRAIL-008.

### What I learned
- **Lipgloss v2 color API change**: `lipgloss.Color` went from being a type (v1: `type Color string`) to being a constructor function (v2: `func Color(s string) color.Color`). All color variables must be typed as `color.Color`.
- **Compositor handles 10+ layers efficiently** — no visible lag with bg + toolbar + footer + 7 nodes + 7 tags = 17 layers.
- **The `internal/` convention works well** — `grailui` is app-specific, `pkg/` is reusable.

### What was tricky to build
- Nothing — with the Compositor discovery from Step 11, everything wired up cleanly.

### Code review instructions
- **Files**: `internal/grailui/*.go`, `cmd/grail/main.go`
- **Run**: `GOWORK=off go run ./cmd/grail/` — 7 nodes visible, arrow keys pan, q quits
- **Key**: `buildNodeLayers` in `layers.go` — the camera→screen coordinate transform and visibility culling

---

## Step 14: Implement GRAIL-008 — Edge Rendering

Complete flowchart now visible. The cellbuf MiniBuffer approach works exactly as designed: grid dots + Bresenham edge lines rendered into a `cellbuf.Buffer`, converted to a string, wrapped as a Z=0 Layer. Nodes at Z=2 cleanly occlude the edge lines underneath. Edge labels ("Y", "N") as Z=3 layers at midpoints.

### Prompt Context

Continuing sequential build. No issues.

**Commit:** 20c345d

### What I did
- Added `buildEdgeCanvasLayer` to `layers.go`: creates cellbuf, draws grid + all edge lines, renders as Z=0 Layer
- Added `buildEdgeLabelLayers`: edge labels at midpoints, horizontal/vertical offset based on edge direction
- Added cellbuf style keys and style map (`bufStyles`) to `layers.go`
- Wired both into `view.go` between background fills and node layers

### What worked
- All 7 edges render with correct Bresenham lines and per-point characters
- Arrowheads point in correct directions
- "Y" and "N" labels positioned at edge midpoints
- Z-ordering correct: edges(Z=0) → nodes(Z=2) → tags/labels(Z=3)
- Grid dots visible in background between nodes
- Camera panning moves edges and grid together with nodes

### What didn't work
- Nothing — clean integration of cellbuf + drawutil with the Compositor

### What I learned
- The MiniBuffer rendering cost is amortized across all edges and the grid — one `buf.Render()` call produces one Layer. This is much more efficient than individual layers per edge segment.
- Z-ordering between cellbuf (rasterized into a single string) and Lipgloss layers works correctly — the Compositor draws the cellbuf string first, then node layers on top.

### Code review instructions
- **Files**: `internal/grailui/layers.go` (new functions), `internal/grailui/view.go` (wiring)
- **Run**: `GOWORK=off go run ./cmd/grail/` — complete flowchart with edges, labels, grid
- **Key**: `buildEdgeCanvasLayer` — the cellbuf→Layer pipeline
