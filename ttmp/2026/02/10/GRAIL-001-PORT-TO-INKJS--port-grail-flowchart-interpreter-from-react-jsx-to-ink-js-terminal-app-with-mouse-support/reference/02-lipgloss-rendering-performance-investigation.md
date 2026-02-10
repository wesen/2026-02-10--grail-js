---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: ttmp/2026/02/10/GRAIL-001-PORT-TO-INKJS--port-grail-flowchart-interpreter-from-react-jsx-to-ink-js-terminal-app-with-mouse-support/scripts/bench-lipgloss-percell-vs-ranges.go
      Note: 'Benchmark: per-cell Render vs StyleRanges vs raw ANSI'
    - Path: ttmp/2026/02/10/GRAIL-001-PORT-TO-INKJS--port-grail-flowchart-interpreter-from-react-jsx-to-ink-js-terminal-app-with-mouse-support/scripts/bench-lipgloss-realistic-buffer.go
      Note: 'Benchmark: realistic edge buffer'
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Lipgloss Rendering Performance Investigation

## Goal

Determine the fastest way to render a 2D character buffer (CellBuffer /
MiniBuffer) into a styled string using Lipgloss, for use in Bubbletea
terminal applications that need per-character control (graph editors,
games, diagram renderers).

## Context

When building a terminal-based flowchart editor, edges and grid dots are
drawn into a 2D character buffer where each cell has a character and a
style key. The buffer must be converted to a styled string for Bubbletea's
`View() string`. There are several ways to do this with Lipgloss:

1. **Per-cell `Style.Render()`** — call `Render()` on every single character
2. **Per-cell `StyleRanges()`** — one `Range` per character
3. **Merged-run `StyleRanges()`** — run-length encode same-styled cells, then apply
4. **Merged-run `Style.Render()`** — run-length encode, then `Render()` per run
5. **Raw ANSI** — manual `\033[...m` escape sequences per run

The question: which is fastest, and which produces the smallest output?

## Quick Reference: Results

### Synthetic benchmark — 200×50 grid, worst-case alternating styles

100 iterations, 200 columns × 50 rows, every cell alternates between
red and green (maximally fragmented — unrealistic but stress-tests overhead).

| Method | µs/frame | Relative | Output bytes/row |
|---|---|---|---|
| Per-cell `Style.Render()` | 18,866 | 1.0× | 3,800 |
| Per-cell `StyleRanges()` | 65,101 | 3.5× slower | — |
| Merged `StyleRanges()` (5 runs/row) | 1,002 | 19× faster | 290 |
| Raw ANSI (5 runs/row) | 24 | 786× faster | 279 |

### Realistic benchmark — 150×40 edge buffer, ~23 style runs/row

200 iterations, 150 columns × 40 rows, ~90% background cells, ~5% grid
dots, ~5% edge characters. Average 23.2 style transitions per row.

| Method | µs/frame | Relative |
|---|---|---|
| Merged `StyleRanges()` | 7,374 | 1.0× |
| `Render()`-per-run | 1,885 | **3.9× faster** |

### Output size comparison (single 200-column row)

| Method | Bytes | Ratio |
|---|---|---|
| Per-cell `Render()` | 3,800 | 13.6× |
| Merged `StyleRanges()` (5 runs) | 290 | 1.04× |
| Raw ANSI (5 runs) | 279 | 1.0× |

## Key Findings

### 1. Run-length encoding is what matters

The single most important optimization is **merging consecutive same-styled
cells into runs** before rendering. Per-cell anything is slow:

- Per-cell `Render()`: 18.9ms/frame → merged `Render()`: 1.9ms/frame = **10× speedup**
- Per-cell `StyleRanges()`: 65.1ms/frame → merged `StyleRanges()`: 7.4ms/frame = **8.8× speedup**
- Output shrinks from 3,800 bytes/row to ~290 bytes/row = **13× smaller**

### 2. `Render()`-per-run beats `StyleRanges()` on merged runs

Once you've merged runs, `Style.Render()` called once per run is **~4×
faster** than `StyleRanges()` with one Range per run.

**Why:** `StyleRanges` must parse existing ANSI escapes in the base string
to correctly splice in new styles (it "takes into account existing styles"
per the docs). `Render()` on a plain substring has no existing escapes to
parse — it just wraps with open/close sequences. Less work per call.

### 3. Raw ANSI is 80× faster but loses Lipgloss features

Manual `\033[38;2;R;G;Bm` sequences avoid all Lipgloss overhead. At
24µs/frame vs 1,885µs, the difference is dramatic. But you lose:

- Automatic color profile downsampling (truecolor → 256 → 16 → none)
- Consistent style API across your codebase
- Future-proofing for Lipgloss improvements

For a 150×40 buffer, 1.9ms is well within the 16ms frame budget (60fps).
Raw ANSI is only worth it if you're rendering very large buffers.

### 4. StyleRanges is designed for a different use case

`StyleRanges` is designed for **syntax highlighting**: you have a
pre-rendered string (possibly with existing ANSI escapes) and want to
colorize portions of it. It's not designed for building styled output
from scratch — `Render()` is.

## API Reference

### `lipgloss.StyleRanges` (v1.1.0+ and v2.0.0-beta.2+)

```go
func StyleRanges(s string, ranges ...Range) string
```

> StyleRanges allows to, given a string, style ranges of it differently.
> The function will take into account existing styles. Ranges should not
> overlap.

```go
type Range struct {
    Start, End int
    Style      Style
}

func NewRange(start, end int, style Style) Range
```

**When to use:** You have a string (possibly already styled) and want to
apply different styles to subranges. Syntax highlighting, search result
highlighting, diff coloring.

**When NOT to use:** Building styled output from a character buffer. Use
`Style.Render()` per run instead.

### `lipgloss.Style.Render` (all versions)

```go
func (s Style) Render(strs ...string) string
```

> Render applies the defined style formatting to a given string.

**When to use:** You have unstyled text and want to apply a style. This
is the primary styling primitive for building output from scratch.

## Recommended MiniBuffer Renderer

```go
// Render converts the buffer to a styled string using Render-per-run.
// Pre-condition: bufStyles maps StyleKey → lipgloss.Style (pre-built).
func (buf *MiniBuffer) Render() string {
    lines := make([]string, buf.H)

    for y := 0; y < buf.H; y++ {
        var sb strings.Builder
        runStart := 0
        runStyle := buf.Cells[y][0].Style

        for x := 1; x <= buf.W; x++ {
            var cur StyleKey
            if x < buf.W {
                cur = buf.Cells[y][x].Style
            } else {
                cur = StyleKey(-1) // sentinel: force flush
            }
            if cur != runStyle {
                // Flush run
                chunk := make([]rune, x-runStart)
                for i := runStart; i < x; i++ {
                    chunk[i-runStart] = buf.Cells[y][i].Ch
                }
                sb.WriteString(bufStyles[runStyle].Render(string(chunk)))
                runStart = x
                runStyle = cur
            }
        }
        lines[y] = sb.String()
    }
    return strings.Join(lines, "\n")
}
```

**Performance for a 150×40 buffer with ~23 runs/row: ~1.9ms/frame.**

## Running the Benchmarks

```bash
cd ttmp/.../GRAIL-001-PORT-TO-INKJS.../scripts/

# Synthetic worst-case (alternating styles every cell)
go run bench-lipgloss-percell-vs-ranges.go

# Realistic edge buffer (~23 runs/row)
go run bench-lipgloss-realistic-buffer.go
```

## Corrections to Design Docs

The following sections in docs 01 and 02 recommended `StyleRanges` for
buffer rendering. Based on this investigation, they should use
`Render()`-per-run instead:

- **Doc 01, §2.3** ("Alternative: Use Lipgloss for Cell Styling"): Replace
  `StyleRanges` recommendation with `Render()`-per-run
- **Doc 02, §6.2** ("Rendering MiniBuffer to String"): Same correction
