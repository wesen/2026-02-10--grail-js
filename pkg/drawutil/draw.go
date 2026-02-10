package drawutil

import "github.com/wesen/grail/pkg/cellbuf"

// DrawLine draws a Bresenham line into buf with appropriate line characters.
// Coordinates are buffer-local (not world).
func DrawLine(buf *cellbuf.Buffer, x0, y0, x1, y1 int, style cellbuf.StyleKey) {
	pts := Bresenham(x0, y0, x1, y1)
	dx := x1 - x0
	dy := y1 - y0
	ch := LineChar(dx, dy)
	for _, p := range pts {
		buf.Set(p.X, p.Y, ch, style)
	}
}

// DrawArrowLine draws a line with an arrowhead at the endpoint.
// The line uses lineStyle and the arrowhead uses arrowStyle.
func DrawArrowLine(buf *cellbuf.Buffer, x0, y0, x1, y1 int, lineStyle, arrowStyle cellbuf.StyleKey) {
	pts := Bresenham(x0, y0, x1, y1)
	if len(pts) == 0 {
		return
	}

	dx := x1 - x0
	dy := y1 - y0
	ch := LineChar(dx, dy)

	// Draw all points except the last as line
	for _, p := range pts[:len(pts)-1] {
		buf.Set(p.X, p.Y, ch, lineStyle)
	}
	// Last point is arrowhead
	last := pts[len(pts)-1]
	buf.Set(last.X, last.Y, ArrowChar(dx, dy), arrowStyle)
}

// DrawDashedLine draws a dashed Bresenham line (every 3rd point is
// skipped). Used for connect-mode preview.
func DrawDashedLine(buf *cellbuf.Buffer, x0, y0, x1, y1 int, style cellbuf.StyleKey) {
	pts := Bresenham(x0, y0, x1, y1)
	dx := x1 - x0
	dy := y1 - y0
	ch := LineChar(dx, dy)
	for i, p := range pts {
		if i%3 != 2 { // skip every 3rd point
			buf.Set(p.X, p.Y, ch, style)
		}
	}
}
