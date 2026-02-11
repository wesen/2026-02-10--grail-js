---
title: "Integration Plan — Grail TUI Packages into Bobatea"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-014-BOBATEA
topics:
  - bobatea
  - packaging
  - go
  - lipgloss
  - bubbletea
---

# Integration Plan — Grail TUI Packages into Bobatea

## Summary

Integrate four reusable packages from the grail project into
`github.com/go-go-golems/bobatea`:

| Package | Lines | Deps | What it does |
|---|---|---|---|
| `cellbuf` | 402 | lipgloss | 2D character buffer with per-cell styling + batch render |
| `drawutil` | 489 | cellbuf | Bresenham lines, arrows, grid dots, edge exit geometry |
| `graphmodel` | 458 | (none) | Generic spatial graph with CRUD, hit testing, rect query |
| `tealayout` | 326 | lipgloss | Region layout builder + compositor chrome helpers |

**Total:** ~1675 lines, 68 tests.

## Current bobatea structure

```
github.com/go-go-golems/bobatea/
├── pkg/
│   ├── autocomplete/    ← input components
│   ├── buttons/
│   ├── chat/
│   ├── commandpalette/
│   ├── diff/
│   ├── eventbus/
│   ├── filepicker/
│   ├── listbox/
│   ├── logutil/
│   ├── mode-keymap/
│   ├── overlay/         ← closest to tealayout (manual layer compositing)
│   ├── repl/
│   ├── sparkline/
│   ├── textarea/
│   ├── thirdparty/
│   └── timeline/
├── cmd/                 ← demo binaries per component
├── docs/                ← glazed-frontmatter .md per component
├── examples/            ← runnable example apps
├── go.mod               ← BT v1 + lipgloss v1
└── Makefile
```

**Key observations:**
- All packages are under `pkg/` — flat namespace, one package = one directory
- BT v1 / lipgloss v1 throughout (no v2 yet)
- Each component has: `pkg/<name>/`, optional `docs/<name>.md`, optional `examples/<name>/`, optional `cmd/<name>/`
- Doc frontmatter uses glazed format: `Title`, `Slug`, `Topics`, `SectionType`
- AGENT.md specifies: cobra for CLIs, `pkg/errors` for wrapping, `testify/assert` for tests
- `overlay/` does manual string-level compositing (pre-lipgloss v2 Compositor)

## Dependency challenge: BT v1 → v2

Bobatea currently depends on **Bubbletea v1** + **lipgloss v1**. The grail
packages depend on **lipgloss v2** (for `tealayout` and `cellbuf`) or have
**no Bubbletea dependency at all** (`graphmodel`, `drawutil`).

### Dependency matrix

| Package | lipgloss v1 | lipgloss v2 | bubbletea | Notes |
|---|---|---|---|---|
| `graphmodel` | ✗ | ✗ | ✗ | Zero external deps — port as-is |
| `drawutil` | ✗ | ✗ | ✗ | Only depends on cellbuf — port as-is |
| `cellbuf` | ✗ | **v2** | ✗ | `Render()` uses `lipgloss.Style` for cell styling |
| `tealayout` | ✗ | **v2** | ✗ | Uses `lipgloss.NewLayer`, `lipgloss.Style` |

### Options

**Option A (recommended): Add as lipgloss v2 packages alongside v1 packages.**

Go supports importing both `lipgloss` v1 and v2 in the same module (different
module paths: `github.com/charmbracelet/lipgloss` vs `charm.land/lipgloss/v2`).
The new packages would import v2; existing packages keep v1. No migration needed.

When bobatea eventually migrates fully to v2, these packages are already there.

**Option B: Abstract the lipgloss dependency.**

Make cellbuf's `Render()` accept an interface instead of `lipgloss.Style`.
This removes the lipgloss dep but adds complexity and loses type safety.
Not recommended — lipgloss v2 is the future.

**Option C: Wait for bobatea v2 migration.**

Defer integration until all of bobatea moves to BT v2 / lipgloss v2. Risk:
indefinite delay. The packages are stable and useful now.

**Recommendation: Option A.** The packages are the vanguard of bobatea's
v2 migration. Document them as "v2-ready" in their docs.

## Target layout in bobatea

```
github.com/go-go-golems/bobatea/
├── pkg/
│   ├── cellbuf/              ← NEW: 2D styled character buffer
│   │   ├── buffer.go
│   │   ├── render.go
│   │   └── buffer_test.go
│   ├── drawutil/             ← NEW: terminal drawing primitives
│   │   ├── draw.go
│   │   ├── edge.go
│   │   ├── grid.go
│   │   ├── line.go
│   │   └── line_test.go
│   ├── graphmodel/           ← NEW: generic spatial graph
│   │   ├── graph.go
│   │   ├── spatial.go
│   │   └── graph_test.go
│   ├── tealayout/            ← NEW: layout builder + chrome layers
│   │   ├── chrome.go
│   │   ├── regions.go
│   │   └── regions_test.go
│   ├── overlay/              ← EXISTING (v1 compositing — keep as-is)
│   │   ...
│   ...existing packages...
├── cmd/
│   ├── cellbuf-demo/         ← NEW: visual demo of cellbuf + drawutil + graphmodel
│   │   └── main.go
│   ...existing demos...
├── docs/
│   ├── cellbuf.md            ← NEW: glazed-frontmatter doc
│   ├── drawutil.md           ← NEW
│   ├── graphmodel.md         ← NEW
│   ├── tealayout.md          ← NEW
│   ...existing docs...
├── examples/
│   ├── cellbuf-canvas/       ← NEW: simple buffer → render example
│   │   ├── main.go
│   │   └── README.md
│   ├── graph-editor/         ← NEW: graphmodel + drawutil interactive demo
│   │   ├── main.go
│   │   └── README.md
│   ...existing examples...
```

### Package naming

Keep the grail package names as-is — they don't conflict with anything in bobatea:

| Grail name | Bobatea import path | Conflicts? |
|---|---|---|
| `cellbuf` | `bobatea/pkg/cellbuf` | No (charmbracelet's `x/cellbuf` is different, internal) |
| `drawutil` | `bobatea/pkg/drawutil` | No |
| `graphmodel` | `bobatea/pkg/graphmodel` | No |
| `tealayout` | `bobatea/pkg/tealayout` | No (`overlay` exists but does something different) |

### Relationship to `overlay` package

The existing `pkg/overlay` does manual string-level compositing (pre-v2 approach:
`PlaceOverlay(x, y, fg, bg)` by parsing ANSI escape codes character by character).
The new `tealayout` uses lipgloss v2's native `Compositor` + `Layer` API.

These are complementary:
- `overlay` → works with BT v1, string-level compositing
- `tealayout` → works with BT v2 / lipgloss v2, native layer compositing

Both should coexist. When bobatea fully migrates to v2, `overlay` can be
deprecated in favor of `tealayout`.

## Import path changes

All internal imports change from `github.com/wesen/grail/pkg/X` to
`github.com/go-go-golems/bobatea/pkg/X`:

```go
// Before (in grail)
import "github.com/wesen/grail/pkg/cellbuf"
import "github.com/wesen/grail/pkg/drawutil"

// After (in bobatea)
import "github.com/go-go-golems/bobatea/pkg/cellbuf"
import "github.com/go-go-golems/bobatea/pkg/drawutil"
```

The grail app's `go.mod` then imports bobatea instead of having the packages inline.

## Code changes required

### Minimal changes

The packages are already clean and self-contained. Required changes:

1. **Package declarations** — already correct (no `package grailXXX` to rename)
2. **Import paths** — `drawutil` imports `cellbuf`, update to bobatea path
3. **go.mod** — add `charm.land/lipgloss/v2` to bobatea's go.mod
4. **Test framework** — grail uses stdlib `testing`; bobatea prefers `testify/assert`.
   **Decision:** Keep stdlib `testing` for these packages (they're self-contained,
   don't use testify patterns). This matches existing bobatea packages like
   `sparkline` which also use stdlib testing.

### No changes needed

- No cobra dependency (these aren't CLI commands)
- No `pkg/errors` wrapping needed (errors are simple strings / panics)
- No goroutines or context (pure synchronous code)
- No global state

## Documentation plan

### docs/ files (glazed frontmatter)

Four new docs, following the established pattern:

**`docs/cellbuf.md`**
```yaml
Title: CellBuf — 2D Character Buffer
Slug: cellbuf
Short: Styled 2D character buffer for terminal canvas rendering
Topics: [components, rendering, canvas, lipgloss-v2]
SectionType: GeneralTopic
```

Content: Overview, API reference (New, Set, SetString, Fill, Render),
StyleKey concept, integration with drawutil, performance notes
(render-per-run strategy), usage with lipgloss v2 Compositor.

**`docs/drawutil.md`**
```yaml
Title: DrawUtil — Terminal Drawing Primitives
Slug: drawutil
Short: Bresenham lines, arrows, grids, and edge geometry for terminal canvases
Topics: [components, rendering, canvas, drawing]
SectionType: GeneralTopic
```

Content: Bresenham algorithm, line/arrow characters, DrawLine/DrawArrowLine/
DrawDashedLine, DrawGrid, EdgeExit for connecting boxes, usage with cellbuf.

**`docs/graphmodel.md`**
```yaml
Title: GraphModel — Generic Spatial Graph
Slug: graphmodel
Short: Generic graph data structure with spatial awareness and hit testing
Topics: [components, data-model, graph, spatial]
SectionType: GeneralTopic
```

Content: Generic `Graph[N Spatial, E any]`, CRUD operations, MoveNode with
custom setter, HitTest, NodesInRect, Spatial interface, usage examples for
flowcharts/diagrams/node editors.

**`docs/tealayout.md`**
```yaml
Title: TeaLayout — Bubbletea v2 Layout Builder
Slug: tealayout
Short: Region-based layout builder and compositor chrome helpers for Bubbletea v2
Topics: [components, layout, bubbletea-v2, lipgloss-v2, compositor]
SectionType: GeneralTopic
```

Content: LayoutBuilder (TopFixed, BottomFixed, RightFixed, Remaining),
Region struct, chrome helpers (ToolbarLayer, FooterLayer, VerticalSeparator,
ModalLayer, FillLayer), integration with lipgloss v2 Compositor, Z-index
conventions.

### Examples

**`examples/cellbuf-canvas/`** — Minimal: create buffer, draw grid + some lines,
render to string, print. No Bubbletea dependency. ~40 lines.

**`examples/graph-editor/`** — Interactive: Bubbletea v2 app with graphmodel
nodes, cellbuf edge rendering, tealayout chrome. Click to select, drag to move.
~150 lines. Demonstrates all four packages working together.

### cmd/ demos

**`cmd/cellbuf-demo/`** — Port the existing `cmd/cellbuf-demo/main.go` from grail.
Visual showcase of all drawing primitives.

## Migration steps (for the porting session)

### Phase 1: Copy packages (30 min)

1. Copy `pkg/cellbuf/`, `pkg/drawutil/`, `pkg/graphmodel/`, `pkg/tealayout/`
   into bobatea's `pkg/` directory
2. Update import paths in `drawutil` (`cellbuf` import)
3. Add `charm.land/lipgloss/v2` to bobatea's `go.mod`
4. Run `go mod tidy`
5. Run `go test ./pkg/cellbuf/... ./pkg/drawutil/... ./pkg/graphmodel/... ./pkg/tealayout/...`
6. Verify all 68 tests pass

### Phase 2: Documentation (30 min)

7. Create `docs/cellbuf.md`, `docs/drawutil.md`, `docs/graphmodel.md`, `docs/tealayout.md`
8. Follow glazed frontmatter format matching existing docs
9. Write API reference + usage examples in each doc

### Phase 3: Examples + demos (30 min)

10. Port `cmd/cellbuf-demo/` to bobatea (update imports)
11. Create `examples/cellbuf-canvas/` (minimal, no BT dependency)
12. Create `examples/graph-editor/` (interactive BT v2 demo)
13. Add README.md to each example directory

### Phase 4: README + integration (15 min)

14. Update bobatea's `README.md` — add four new components to the list,
    note "lipgloss v2 / Bubbletea v2" badge
15. Update bobatea's `go.sum` (go mod tidy)
16. Run full `make test` + `make build`
17. Commit with descriptive message

### Phase 5: Update grail to import from bobatea (15 min)

18. Update grail's `go.mod` — replace inline `pkg/` with bobatea dependency
19. Update all import paths in `internal/grailui/`
20. Remove `pkg/` from grail repo
21. Verify grail builds and runs

## go.mod impact on bobatea

### New direct dependencies

```
charm.land/lipgloss/v2 v2.0.0-beta.3.0.20260210014823-2f36a2f1ba17
```

### New transitive dependencies

```
charm.land/bubbletea/v2 v2.0.0-rc.2  (only if examples/graph-editor uses BT v2)
```

### Risk: pre-release deps

`charm.land/lipgloss/v2` is at `v2.0.0-beta.3` — a pre-release pseudo-version.
This is acceptable because:
- bobatea already tracks charmbracelet pre-releases (lipgloss v1.1.1-0.2025...)
- The new packages are clearly documented as "v2-ready / experimental"
- cellbuf and tealayout APIs are stable (tested, proven in grail)
- When lipgloss v2 goes stable, only go.mod version bump needed

If this is unacceptable, **Option B fallback**: only port `graphmodel` and
`drawutil` now (zero lipgloss deps), defer `cellbuf` + `tealayout` until
lipgloss v2 stabilizes.

## API surface added to bobatea

### cellbuf (5 exported)

```go
type StyleKey int
type Cell struct { Ch rune; Style StyleKey }
type Buffer struct { ... }
func New(w, h int, defaultStyle StyleKey) *Buffer
func (b *Buffer) InBounds(x, y int) bool
func (b *Buffer) Set(x, y int, ch rune, style StyleKey)
func (b *Buffer) SetString(x, y int, s string, style StyleKey)
func (b *Buffer) Fill(ch rune, style StyleKey)
func (b *Buffer) Render(styles map[StyleKey]lipgloss.Style) string
```

### drawutil (8 exported)

```go
func Bresenham(x0, y0, x1, y1 int) []image.Point
func LineChar(dx, dy int) rune
func ArrowChar(dx, dy int) rune
func DrawLine(buf *cellbuf.Buffer, x0, y0, x1, y1 int, style cellbuf.StyleKey)
func DrawArrowLine(buf, x0, y0, x1, y1, lineStyle, arrowStyle cellbuf.StyleKey)
func DrawDashedLine(buf, x0, y0, x1, y1, style cellbuf.StyleKey)
func DrawGrid(buf, camX, camY, spacingX, spacingY, style cellbuf.StyleKey)
func EdgeExit(rect image.Rectangle, target image.Point) image.Point
```

### graphmodel (10 exported)

```go
type Spatial interface { Pos() image.Point; Size() image.Point }
func CenterOf(s Spatial) image.Point
func BoundsOf(s Spatial) image.Rectangle
type Node[N Spatial] struct { ID int; Data N }
type Edge[E any] struct { FromID, ToID int; Data E }
type Graph[N Spatial, E any] struct { ... }
func New[N Spatial, E any]() *Graph[N, E]
func (g *Graph) AddNode/Node/Nodes/RemoveNode/MoveNode
func (g *Graph) AddEdge/RemoveEdge/Edges/OutEdges/InEdges
func (g *Graph) HitTest/NodesInRect
```

### tealayout (8 exported)

```go
type Region struct { Name string; Rect image.Rectangle }
type Layout struct { ... }
func (l Layout) Get(name string) Region
type LayoutBuilder struct { ... }
func NewLayoutBuilder(termW, termH int) *LayoutBuilder
func (b *LayoutBuilder) TopFixed/BottomFixed/RightFixed/Remaining/Build
func ToolbarLayer/FooterLayer/VerticalSeparator/ModalLayer/FillLayer
```

## Open questions for the porting session

1. **Package naming**: `tealayout` vs `layout` vs `compositor`? (I recommend
   `tealayout` — specific, no conflicts, signals BT v2 orientation)

2. **graphmodel generics**: Go 1.24 is bobatea's min version. Generics are
   fine (available since 1.18), but verify bobatea's CI/toolchain handles them.

3. **Should `overlay` be deprecated?** Not yet — it's used by existing
   components. Add a doc note: "For lipgloss v2 projects, see `tealayout`."

4. **Example scope**: Should `examples/graph-editor/` be a full mini-GRaIL,
   or a minimal node-drag demo? Recommend minimal (demonstrates packages
   without the interpreter complexity).

5. **Should `cellbuf.Render()` accept `lipgloss.Style` or an interface?**
   Recommend keeping `lipgloss.Style` — it's the natural API and avoids
   unnecessary abstraction.
