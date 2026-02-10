package cellbuf

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss/v2"
)

// Test style keys
const (
	testBG   StyleKey = 0
	testRed  StyleKey = 1
	testBlue StyleKey = 2
)

func testStyles() map[StyleKey]lipgloss.Style {
	return map[StyleKey]lipgloss.Style{
		testBG:   lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")),
		testRed:  lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
		testBlue: lipgloss.NewStyle().Foreground(lipgloss.Color("#0000ff")),
	}
}

func TestNew(t *testing.T) {
	b := New(10, 5, testBG)
	if b.W != 10 || b.H != 5 {
		t.Fatalf("expected 10x5, got %dx%d", b.W, b.H)
	}
	if len(b.Cells) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(b.Cells))
	}
	for y := 0; y < 5; y++ {
		if len(b.Cells[y]) != 10 {
			t.Fatalf("row %d: expected 10 cols, got %d", y, len(b.Cells[y]))
		}
		for x := 0; x < 10; x++ {
			c := b.Cells[y][x]
			if c.Ch != ' ' || c.Style != testBG {
				t.Fatalf("cell (%d,%d): expected space/testBG, got %q/%d", x, y, c.Ch, c.Style)
			}
		}
	}
}

func TestNewZeroSize(t *testing.T) {
	b := New(0, 0, testBG)
	if b.W != 0 || b.H != 0 {
		t.Fatalf("expected 0x0, got %dx%d", b.W, b.H)
	}
	styles := testStyles()
	result := b.Render(styles)
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestNewNegativeSize(t *testing.T) {
	b := New(-5, -3, testBG)
	if b.W != 0 || b.H != 0 {
		t.Fatalf("expected 0x0 for negative sizes, got %dx%d", b.W, b.H)
	}
}

func TestInBounds(t *testing.T) {
	b := New(10, 5, testBG)
	tests := []struct {
		x, y int
		want bool
	}{
		{0, 0, true},
		{9, 4, true},
		{5, 2, true},
		{-1, 0, false},
		{0, -1, false},
		{10, 0, false},
		{0, 5, false},
		{10, 5, false},
	}
	for _, tc := range tests {
		got := b.InBounds(tc.x, tc.y)
		if got != tc.want {
			t.Errorf("InBounds(%d, %d) = %v, want %v", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestSet(t *testing.T) {
	b := New(10, 5, testBG)
	b.Set(3, 2, 'X', testRed)
	c := b.Cells[2][3]
	if c.Ch != 'X' || c.Style != testRed {
		t.Fatalf("expected X/testRed, got %q/%d", c.Ch, c.Style)
	}
}

func TestSetOutOfBounds(t *testing.T) {
	b := New(10, 5, testBG)
	// These should not panic
	b.Set(-1, 0, 'X', testRed)
	b.Set(0, -1, 'X', testRed)
	b.Set(10, 0, 'X', testRed)
	b.Set(0, 5, 'X', testRed)
	b.Set(100, 100, 'X', testRed)

	// Verify nothing changed
	for y := 0; y < 5; y++ {
		for x := 0; x < 10; x++ {
			if b.Cells[y][x].Ch != ' ' {
				t.Fatalf("out-of-bounds Set modified cell (%d,%d)", x, y)
			}
		}
	}
}

func TestSetString(t *testing.T) {
	b := New(10, 5, testBG)
	b.SetString(2, 1, "Hello", testBlue)

	expected := "Hello"
	for i, ch := range expected {
		c := b.Cells[1][2+i]
		if c.Ch != ch || c.Style != testBlue {
			t.Errorf("pos %d: expected %q/testBlue, got %q/%d", i, ch, c.Ch, c.Style)
		}
	}
	// Character before and after should be unchanged
	if b.Cells[1][1].Ch != ' ' {
		t.Error("cell before string was modified")
	}
	if b.Cells[1][7].Ch != ' ' {
		t.Error("cell after string was modified")
	}
}

func TestSetStringClipsAtBounds(t *testing.T) {
	b := New(5, 1, testBG)
	b.SetString(3, 0, "Hello", testRed) // only "He" fits
	if b.Cells[0][3].Ch != 'H' || b.Cells[0][4].Ch != 'e' {
		t.Error("expected H and e at positions 3,4")
	}
	// Should not panic or corrupt
}

func TestFill(t *testing.T) {
	b := New(5, 3, testBG)
	b.Set(2, 1, 'X', testRed)
	b.Fill(testBlue)
	for y := 0; y < 3; y++ {
		for x := 0; x < 5; x++ {
			c := b.Cells[y][x]
			if c.Ch != ' ' || c.Style != testBlue {
				t.Fatalf("Fill: cell (%d,%d) = %q/%d, want space/testBlue", x, y, c.Ch, c.Style)
			}
		}
	}
}

func TestRenderLineCount(t *testing.T) {
	styles := testStyles()
	b := New(20, 5, testBG)
	result := b.Render(styles)
	lines := strings.Split(result, "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
}

func TestRenderContent(t *testing.T) {
	styles := testStyles()
	b := New(10, 1, testBG)
	b.SetString(2, 0, "Hi", testRed)
	result := b.Render(styles)

	// The result should contain "Hi" somewhere (surrounded by ANSI escapes)
	if !strings.Contains(result, "Hi") {
		t.Fatalf("rendered output doesn't contain 'Hi': %q", result)
	}
}

func TestRenderMergesRuns(t *testing.T) {
	styles := testStyles()

	// All same style — should produce fewer ANSI escapes than per-cell
	b := New(50, 1, testBG)
	uniform := b.Render(styles)

	// Alternating styles — should produce more ANSI escapes
	b2 := New(50, 1, testBG)
	for x := 0; x < 50; x++ {
		if x%2 == 0 {
			b2.Set(x, 0, '.', testRed)
		} else {
			b2.Set(x, 0, '.', testBlue)
		}
	}
	alternating := b2.Render(styles)

	// Uniform should be shorter (fewer escape sequences)
	if len(uniform) >= len(alternating) {
		t.Errorf("uniform render (%d bytes) should be shorter than alternating (%d bytes)",
			len(uniform), len(alternating))
	}
}

func TestRenderMissingStyle(t *testing.T) {
	// Style key 99 not in the map — should render without ANSI (plain text)
	styles := testStyles()
	b := New(5, 1, StyleKey(99))
	b.SetString(0, 0, "plain", StyleKey(99))
	result := b.Render(styles)
	if !strings.Contains(result, "plain") {
		t.Fatalf("missing style should still render text: %q", result)
	}
}

func BenchmarkRender200x50(b *testing.B) {
	styles := testStyles()
	buf := New(200, 50, testBG)
	// Add some variety
	for y := 0; y < 50; y++ {
		for x := 0; x < 200; x++ {
			if x%5 == 0 && y%3 == 0 {
				buf.Set(x, y, '·', testRed)
			}
		}
		if y < 200 {
			buf.Set(y, y%50, '/', testBlue)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.Render(styles)
	}
}

// BenchmarkRenderRealistic simulates a GRaIL edge buffer: ~90% background,
// ~5% grid dots, ~5% edge chars. Average ~5-10 runs per row.
func BenchmarkRenderRealistic(b *testing.B) {
	styles := testStyles()
	buf := New(150, 40, testBG)
	// Grid dots
	for y := 0; y < 40; y++ {
		for x := 0; x < 150; x++ {
			if x%5 == 0 && y%3 == 0 {
				buf.Set(x, y, '·', testRed)
			}
		}
		// One diagonal edge
		if y < 150 {
			buf.Set(y, y%40, '/', testBlue)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.Render(styles)
	}
}
