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
