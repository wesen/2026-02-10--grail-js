package drawutil

import "image"

// EdgeExit returns the point on the border of rect that faces toward
// the target point. It picks the side (left/right/top/bottom) by
// comparing normalized dx/dy against the rect's half-dimensions.
//
// If rect has zero size or target equals the rect center, the rect
// center is returned.
func EdgeExit(rect image.Rectangle, target image.Point) image.Point {
	cx := (rect.Min.X + rect.Max.X) / 2
	cy := (rect.Min.Y + rect.Max.Y) / 2
	hw := (rect.Max.X - rect.Min.X) / 2
	hh := (rect.Max.Y - rect.Min.Y) / 2

	dx := target.X - cx
	dy := target.Y - cy

	if dx == 0 && dy == 0 {
		return image.Pt(cx, cy)
	}
	if hw == 0 && hh == 0 {
		return image.Pt(cx, cy)
	}

	// Normalize dx/dy by half-dimensions to decide exit side
	var ndx, ndy float64
	if hw > 0 {
		ndx = float64(dx) / float64(hw)
	}
	if hh > 0 {
		ndy = float64(dy) / float64(hh)
	}

	if abs(int(ndx*1000)) > abs(int(ndy*1000)) {
		// Exit horizontally
		if dx > 0 {
			return image.Pt(rect.Max.X-1, cy)
		}
		return image.Pt(rect.Min.X, cy)
	}
	// Exit vertically
	if dy > 0 {
		return image.Pt(cx, rect.Max.Y-1)
	}
	return image.Pt(cx, rect.Min.Y)
}
