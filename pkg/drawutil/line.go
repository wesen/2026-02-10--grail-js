// Package drawutil provides terminal drawing primitives: Bresenham lines,
// directional line/arrow character lookup, edge exit-point geometry, and
// convenience functions that draw into a cellbuf.Buffer.
package drawutil

import "image"

// Bresenham returns the integer points on the line from (x0,y0) to (x1,y1)
// using Bresenham's line algorithm. The result always includes both endpoints.
// The loop is capped at dx+dy+2 iterations to prevent infinite loops.
func Bresenham(x0, y0, x1, y1 int) []image.Point {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy
	x, y := x0, y0

	pts := make([]image.Point, 0, dx+dy+1)
	for range dx + dy + 2 {
		pts = append(pts, image.Pt(x, y))
		if x == x1 && y == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
	return pts
}

// LineChar returns the box-drawing character for a line segment with the
// given direction vector (dx, dy).
func LineChar(dx, dy int) rune {
	if dx == 0 {
		return '│'
	}
	if dy == 0 {
		return '─'
	}
	if (dx > 0) == (dy > 0) {
		return '\\'
	}
	return '/'
}

// ArrowChar returns an arrow-head character pointing in the dominant
// direction of (dx, dy).
func ArrowChar(dx, dy int) rune {
	if abs(dy) > abs(dx) {
		if dy > 0 {
			return '▼'
		}
		return '▲'
	}
	if dx > 0 {
		return '►'
	}
	return '◄'
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
