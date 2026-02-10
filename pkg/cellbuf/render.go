package cellbuf

import (
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
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

	for y := 0; y < b.H; y++ {
		var sb strings.Builder
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
				// Flush the accumulated run
				chunk := make([]rune, x-runStart)
				for i := runStart; i < x; i++ {
					chunk[i-runStart] = row[i].Ch
				}
				if s, ok := styles[runStyle]; ok {
					sb.WriteString(s.Render(string(chunk)))
				} else {
					sb.WriteString(string(chunk))
				}
				runStart = x
				runStyle = curStyle
			}
		}

		lines[y] = sb.String()
	}

	return strings.Join(lines, "\n")
}
