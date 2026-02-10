package cellbuf

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Render converts the buffer into a styled string. The caller provides
// a mapping from StyleKey to lipgloss.Style.
//
// Consecutive cells with the same StyleKey are merged into runs and
// rendered with a single Style.Render() call per run. This is
// significantly faster than per-cell rendering (see GRAIL-001 perf doc).
//
// Rows are joined with "\n". An empty buffer (W==0 or H==0) returns "".
func (b *Buffer) Render(styles map[StyleKey]lipgloss.Style) string {
	if b.W == 0 || b.H == 0 {
		return ""
	}

	lines := make([]string, b.H)
	// Reusable rune buffer — avoids allocation per run
	chunk := make([]rune, b.W)

	for y := 0; y < b.H; y++ {
		var sb strings.Builder
		// Pre-size: each cell is ~1 byte content + ~10 bytes ANSI overhead
		// amortized across runs. 2× width is a reasonable estimate.
		sb.Grow(b.W * 2)
		row := b.Cells[y]

		runStart := 0
		runStyle := row[0].Style

		for x := 1; x <= b.W; x++ {
			// Use sentinel style at end to flush last run
			var curStyle StyleKey
			if x < b.W {
				curStyle = row[x].Style
			} else {
				curStyle = StyleKey(-1)
			}

			if curStyle != runStyle {
				// Flush the accumulated run into the reusable chunk
				n := x - runStart
				for i := 0; i < n; i++ {
					chunk[i] = row[runStart+i].Ch
				}
				s := string(chunk[:n])
				if style, ok := styles[runStyle]; ok {
					sb.WriteString(style.Render(s))
				} else {
					sb.WriteString(s)
				}
				runStart = x
				runStyle = curStyle
			}
		}

		lines[y] = sb.String()
	}

	return strings.Join(lines, "\n")
}
