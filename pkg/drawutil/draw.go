package drawutil

import (
	"image"

	"github.com/wesen/grail/pkg/cellbuf"
)

// pointChar returns the line character for a point based on its local
// direction (looking at the next or previous point). This matches the
// Python reference which computes the character per-segment.
func pointChar(pts []image.Point, i int) rune {
	var dx, dy int
	if i < len(pts)-1 {
		dx = pts[i+1].X - pts[i].X
		dy = pts[i+1].Y - pts[i].Y
	} else if i > 0 {
		dx = pts[i].X - pts[i-1].X
		dy = pts[i].Y - pts[i-1].Y
	}
	return LineChar(dx, dy)
}

// DrawLine draws a Bresenham line into buf with per-point line characters.
// Coordinates are buffer-local (not world).
func DrawLine(buf *cellbuf.Buffer, x0, y0, x1, y1 int, style cellbuf.StyleKey) {
	pts := Bresenham(x0, y0, x1, y1)
	for i, p := range pts {
		buf.Set(p.X, p.Y, pointChar(pts, i), style)
	}
}

// DrawArrowLine draws a line with an arrowhead at the endpoint.
// The line uses lineStyle and the arrowhead uses arrowStyle.
func DrawArrowLine(buf *cellbuf.Buffer, x0, y0, x1, y1 int, lineStyle, arrowStyle cellbuf.StyleKey) {
	pts := Bresenham(x0, y0, x1, y1)
	if len(pts) == 0 {
		return
	}

	// Draw all points except the last as line with per-point characters
	for i, p := range pts[:len(pts)-1] {
		buf.Set(p.X, p.Y, pointChar(pts, i), lineStyle)
	}
	// Last point is arrowhead based on final segment direction
	last := pts[len(pts)-1]
	var dx, dy int
	if len(pts) >= 2 {
		dx = last.X - pts[len(pts)-2].X
		dy = last.Y - pts[len(pts)-2].Y
	}
	buf.Set(last.X, last.Y, ArrowChar(dx, dy), arrowStyle)
}

// DrawDashedLine draws a dashed Bresenham line (every 3rd point is
// skipped). Used for connect-mode preview.
func DrawDashedLine(buf *cellbuf.Buffer, x0, y0, x1, y1 int, style cellbuf.StyleKey) {
	pts := Bresenham(x0, y0, x1, y1)
	for i, p := range pts {
		if i%3 != 2 { // skip every 3rd point
			buf.Set(p.X, p.Y, pointChar(pts, i), style)
		}
	}
}
