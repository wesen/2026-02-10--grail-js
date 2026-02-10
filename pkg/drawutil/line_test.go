package drawutil

import (
	"image"
	"testing"

	"github.com/wesen/grail/pkg/cellbuf"
)

// ── Bresenham ──

func TestBresenhamHorizontal(t *testing.T) {
	pts := Bresenham(0, 0, 5, 0)
	if len(pts) != 6 {
		t.Fatalf("expected 6 points, got %d: %v", len(pts), pts)
	}
	for i, p := range pts {
		if p.X != i || p.Y != 0 {
			t.Errorf("point %d: expected (%d,0), got %v", i, i, p)
		}
	}
}

func TestBresenhamVertical(t *testing.T) {
	pts := Bresenham(0, 0, 0, 5)
	if len(pts) != 6 {
		t.Fatalf("expected 6 points, got %d: %v", len(pts), pts)
	}
	for i, p := range pts {
		if p.X != 0 || p.Y != i {
			t.Errorf("point %d: expected (0,%d), got %v", i, i, p)
		}
	}
}

func TestBresenhamDiagonal(t *testing.T) {
	pts := Bresenham(0, 0, 5, 5)
	if len(pts) != 6 {
		t.Fatalf("expected 6 points, got %d: %v", len(pts), pts)
	}
	for i, p := range pts {
		if p.X != i || p.Y != i {
			t.Errorf("point %d: expected (%d,%d), got %v", i, i, i, p)
		}
	}
}

func TestBresenhamSteep(t *testing.T) {
	pts := Bresenham(0, 0, 2, 8)
	// Should have more y-steps than x-steps
	if len(pts) < 9 {
		t.Fatalf("steep line should have at least 9 points, got %d", len(pts))
	}
	// First and last points must match
	if pts[0] != image.Pt(0, 0) {
		t.Errorf("first point: expected (0,0), got %v", pts[0])
	}
	if pts[len(pts)-1] != image.Pt(2, 8) {
		t.Errorf("last point: expected (2,8), got %v", pts[len(pts)-1])
	}
}

func TestBresenhamReverse(t *testing.T) {
	pts := Bresenham(5, 3, 0, 0)
	if pts[0] != image.Pt(5, 3) {
		t.Errorf("first point: expected (5,3), got %v", pts[0])
	}
	if pts[len(pts)-1] != image.Pt(0, 0) {
		t.Errorf("last point: expected (0,0), got %v", pts[len(pts)-1])
	}
}

func TestBresenhamZeroLength(t *testing.T) {
	pts := Bresenham(3, 3, 3, 3)
	if len(pts) != 1 {
		t.Fatalf("zero-length line: expected 1 point, got %d", len(pts))
	}
	if pts[0] != image.Pt(3, 3) {
		t.Errorf("expected (3,3), got %v", pts[0])
	}
}

// ── LineChar ──

func TestLineChar(t *testing.T) {
	tests := []struct {
		dx, dy int
		want   rune
	}{
		{0, 1, '│'},
		{0, -1, '│'},
		{1, 0, '─'},
		{-1, 0, '─'},
		{1, 1, '\\'},
		{-1, -1, '\\'},
		{-1, 1, '/'},
		{1, -1, '/'},
	}
	for _, tc := range tests {
		got := LineChar(tc.dx, tc.dy)
		if got != tc.want {
			t.Errorf("LineChar(%d,%d) = %c, want %c", tc.dx, tc.dy, got, tc.want)
		}
	}
}

// ── ArrowChar ──

func TestArrowChar(t *testing.T) {
	tests := []struct {
		dx, dy int
		want   rune
	}{
		{0, 1, '▼'},
		{0, -1, '▲'},
		{1, 0, '►'},
		{-1, 0, '◄'},
		{1, 5, '▼'},  // steep → vertical arrow
		{5, 1, '►'},  // shallow → horizontal arrow
		{-3, 1, '◄'}, // dx dominant
	}
	for _, tc := range tests {
		got := ArrowChar(tc.dx, tc.dy)
		if got != tc.want {
			t.Errorf("ArrowChar(%d,%d) = %c, want %c", tc.dx, tc.dy, got, tc.want)
		}
	}
}

// ── EdgeExit ──

func TestEdgeExitRight(t *testing.T) {
	rect := image.Rect(10, 10, 20, 14) // 10 wide, 4 tall
	target := image.Pt(50, 12)          // far to the right
	exit := EdgeExit(rect, target)
	if exit.X != 19 { // Max.X - 1
		t.Errorf("expected right exit X=19, got %d", exit.X)
	}
}

func TestEdgeExitLeft(t *testing.T) {
	rect := image.Rect(10, 10, 20, 14)
	target := image.Pt(0, 12) // far to the left
	exit := EdgeExit(rect, target)
	if exit.X != 10 {
		t.Errorf("expected left exit X=10, got %d", exit.X)
	}
}

func TestEdgeExitBottom(t *testing.T) {
	rect := image.Rect(10, 10, 20, 14)
	target := image.Pt(15, 50) // far below
	exit := EdgeExit(rect, target)
	if exit.Y != 13 { // Max.Y - 1
		t.Errorf("expected bottom exit Y=13, got %d", exit.Y)
	}
}

func TestEdgeExitTop(t *testing.T) {
	rect := image.Rect(10, 10, 20, 14)
	target := image.Pt(15, 0) // far above
	exit := EdgeExit(rect, target)
	if exit.Y != 10 {
		t.Errorf("expected top exit Y=10, got %d", exit.Y)
	}
}

func TestEdgeExitSameCenter(t *testing.T) {
	rect := image.Rect(10, 10, 20, 14)
	center := image.Pt(15, 12) // center of rect
	exit := EdgeExit(rect, center)
	if exit != center {
		t.Errorf("same-center: expected %v, got %v", center, exit)
	}
}

// ── Draw functions ──

func TestDrawLine(t *testing.T) {
	buf := cellbuf.New(10, 10, 0)
	DrawLine(buf, 0, 0, 9, 0, 1)
	// All cells on row 0 should have style 1
	for x := 0; x < 10; x++ {
		c := buf.Cells[0][x]
		if c.Style != 1 {
			t.Errorf("DrawLine: cell (%d,0) style=%d, want 1", x, c.Style)
		}
		if c.Ch != '─' {
			t.Errorf("DrawLine: cell (%d,0) char=%c, want ─", x, c.Ch)
		}
	}
}

func TestDrawArrowLine(t *testing.T) {
	buf := cellbuf.New(10, 10, 0)
	DrawArrowLine(buf, 5, 0, 5, 5, 1, 2)
	// Last point should be arrowhead with style 2
	c := buf.Cells[5][5]
	if c.Ch != '▼' {
		t.Errorf("arrowhead: expected ▼, got %c", c.Ch)
	}
	if c.Style != 2 {
		t.Errorf("arrowhead style: expected 2, got %d", c.Style)
	}
	// Middle points should be line with style 1
	c = buf.Cells[2][5]
	if c.Ch != '│' {
		t.Errorf("line body: expected │, got %c", c.Ch)
	}
	if c.Style != 1 {
		t.Errorf("line body style: expected 1, got %d", c.Style)
	}
}

func TestDrawDashedLine(t *testing.T) {
	buf := cellbuf.New(20, 1, 0)
	DrawDashedLine(buf, 0, 0, 19, 0, 1)
	// Every 3rd point (index 2, 5, 8, ...) should be skipped (style 0)
	drawn := 0
	for x := 0; x < 20; x++ {
		if buf.Cells[0][x].Style == 1 {
			drawn++
		}
	}
	// 20 points, skip indices 2,5,8,11,14,17 = 6 skipped, 14 drawn
	if drawn != 14 {
		t.Errorf("dashed line: expected 14 drawn points, got %d", drawn)
	}
}

func TestDrawGrid(t *testing.T) {
	buf := cellbuf.New(20, 10, 0)
	DrawGrid(buf, 0, 0, 5, 3, 1)
	// (0,0), (5,0), (10,0), (15,0) should have dots
	for _, x := range []int{0, 5, 10, 15} {
		if buf.Cells[0][x].Ch != '·' {
			t.Errorf("grid: expected dot at (%d,0), got %c", x, buf.Cells[0][x].Ch)
		}
	}
	// (1,0) should not have a dot
	if buf.Cells[0][1].Ch == '·' {
		t.Error("grid: unexpected dot at (1,0)")
	}
	// Row 3 should have dots (3%3==0)
	if buf.Cells[3][0].Ch != '·' {
		t.Error("grid: expected dot at (0,3)")
	}
	// Row 1 should not
	if buf.Cells[1][0].Ch == '·' {
		t.Error("grid: unexpected dot at (0,1)")
	}
}

func TestDrawGridWithCamera(t *testing.T) {
	buf := cellbuf.New(20, 10, 0)
	DrawGrid(buf, 2, 1, 5, 3, 1)
	// World (2,1) is buffer (0,0) — not a grid point (2%5!=0)
	if buf.Cells[0][0].Ch == '·' {
		t.Error("grid+cam: unexpected dot at buf(0,0) = world(2,1)")
	}
	// World (5,3) is buffer (3,2) — grid point
	if buf.Cells[2][3].Ch != '·' {
		t.Error("grid+cam: expected dot at buf(3,2) = world(5,3)")
	}
}
