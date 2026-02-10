package tealayout

import (
	"image"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestLayoutBasic(t *testing.T) {
	l := NewLayoutBuilder(80, 24).
		TopFixed("toolbar", 3).
		BottomFixed("footer", 1).
		RightFixed("panel", 34).
		Remaining("canvas").
		Build()

	if l.TermW != 80 || l.TermH != 24 {
		t.Fatalf("term size: expected 80x24, got %dx%d", l.TermW, l.TermH)
	}

	tb := l.Get("toolbar")
	if tb.Rect != image.Rect(0, 0, 80, 3) {
		t.Errorf("toolbar: expected (0,0)-(80,3), got %v", tb.Rect)
	}

	ft := l.Get("footer")
	if ft.Rect != image.Rect(0, 23, 80, 24) {
		t.Errorf("footer: expected (0,23)-(80,24), got %v", ft.Rect)
	}

	pn := l.Get("panel")
	if pn.Rect != image.Rect(46, 3, 80, 23) {
		t.Errorf("panel: expected (46,3)-(80,23), got %v", pn.Rect)
	}

	cv := l.Get("canvas")
	if cv.Rect != image.Rect(0, 3, 46, 23) {
		t.Errorf("canvas: expected (0,3)-(46,23), got %v", cv.Rect)
	}
}

func TestLayoutRemainingOnly(t *testing.T) {
	l := NewLayoutBuilder(80, 24).
		Remaining("full").
		Build()

	r := l.Get("full")
	if r.Rect != image.Rect(0, 0, 80, 24) {
		t.Errorf("full: expected (0,0)-(80,24), got %v", r.Rect)
	}
}

func TestLayoutZeroSize(t *testing.T) {
	l := NewLayoutBuilder(0, 0).
		TopFixed("toolbar", 3).
		Remaining("canvas").
		Build()

	cv := l.Get("canvas")
	// With 0-height terminal and 3 rows consumed from top, remaining is negative → clamped to zero
	if cv.Rect.Dx() != 0 || cv.Rect.Dy() != 0 {
		t.Errorf("zero term canvas: expected empty rect, got %v", cv.Rect)
	}
}

func TestLayoutNoOverlap(t *testing.T) {
	l := NewLayoutBuilder(80, 24).
		TopFixed("toolbar", 3).
		BottomFixed("footer", 1).
		RightFixed("panel", 34).
		Remaining("canvas").
		Build()

	regions := []Region{
		l.Get("toolbar"),
		l.Get("footer"),
		l.Get("panel"),
		l.Get("canvas"),
	}

	for i := 0; i < len(regions); i++ {
		for j := i + 1; j < len(regions); j++ {
			ri, rj := regions[i], regions[j]
			if ri.Rect.Overlaps(rj.Rect) {
				t.Errorf("overlap: %s %v and %s %v",
					ri.Name, ri.Rect, rj.Name, rj.Rect)
			}
		}
	}
}

func TestLayoutCanvasDimensions(t *testing.T) {
	l := NewLayoutBuilder(80, 24).
		TopFixed("toolbar", 3).
		BottomFixed("footer", 1).
		RightFixed("panel", 34).
		Remaining("canvas").
		Build()

	cv := l.Get("canvas")
	// 80 - 34 = 46 wide, 24 - 3 - 1 = 20 tall
	if cv.Rect.Dx() != 46 || cv.Rect.Dy() != 20 {
		t.Errorf("canvas dims: expected 46x20, got %dx%d", cv.Rect.Dx(), cv.Rect.Dy())
	}
}

func TestGetNonExistent(t *testing.T) {
	l := NewLayoutBuilder(80, 24).Build()
	r := l.Get("missing")
	if r.Name != "" {
		t.Errorf("non-existent: expected empty, got %v", r)
	}
}

func TestModalLayer(t *testing.T) {
	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Width(20).
		Padding(1, 2)

	layer := ModalLayer("test content", 80, 24, style)
	if layer.GetID() != "modal" {
		t.Errorf("modal ID: expected 'modal', got %q", layer.GetID())
	}
	if layer.GetZ() != 100 {
		t.Errorf("modal Z: expected 100, got %d", layer.GetZ())
	}
	// Should be roughly centered
	x, y := layer.GetX(), layer.GetY()
	if x < 20 || x > 40 {
		t.Errorf("modal X not centered: %d", x)
	}
	if y < 5 || y > 15 {
		t.Errorf("modal Y not centered: %d", y)
	}
}

func TestFillLayer(t *testing.T) {
	r := Region{Name: "test", Rect: image.Rect(10, 5, 30, 15)}
	style := lipgloss.NewStyle().Background(lipgloss.Color("#080e0b"))
	layer := FillLayer(r, style, "bg", 0)

	if layer.GetID() != "bg" {
		t.Errorf("fill ID: expected 'bg', got %q", layer.GetID())
	}
	if layer.GetX() != 10 || layer.GetY() != 5 {
		t.Errorf("fill pos: expected (10,5), got (%d,%d)", layer.GetX(), layer.GetY())
	}
}

func TestFillLayerEmpty(t *testing.T) {
	r := Region{Name: "empty", Rect: image.Rectangle{}}
	style := lipgloss.NewStyle()
	layer := FillLayer(r, style, "bg", 0)
	// Should not panic, returns empty layer
	if layer.GetContent() != "" {
		t.Error("empty fill should have no content")
	}
}
