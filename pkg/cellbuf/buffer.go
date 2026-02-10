// Package cellbuf provides a 2D character buffer with per-cell styling
// and efficient Lipgloss-based rendering.
//
// Each cell holds a rune and a StyleKey (an int enum). At render time,
// the caller provides a map[StyleKey]lipgloss.Style so the buffer is
// decoupled from specific color schemes.
//
// Limitation: all runes are assumed to be single-width. CJK or other
// double-width characters are not handled correctly.
package cellbuf

// StyleKey identifies a visual style. The caller defines the mapping
// from StyleKey to lipgloss.Style at render time.
type StyleKey int

// Cell is a single character in the buffer with an associated style.
type Cell struct {
	Ch    rune
	Style StyleKey
}

// Buffer is a 2D grid of styled cells.
type Buffer struct {
	W, H  int
	Cells [][]Cell // [row][col]
}

// New creates a Buffer of the given size, filled with spaces in the
// given default style.
func New(w, h int, defaultStyle StyleKey) *Buffer {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	b := &Buffer{W: w, H: h, Cells: make([][]Cell, h)}
	for y := range b.Cells {
		row := make([]Cell, w)
		for x := range row {
			row[x] = Cell{Ch: ' ', Style: defaultStyle}
		}
		b.Cells[y] = row
	}
	return b
}

// InBounds reports whether (x, y) is inside the buffer.
func (b *Buffer) InBounds(x, y int) bool {
	return x >= 0 && x < b.W && y >= 0 && y < b.H
}

// Set writes a single character at (x, y). Out-of-bounds writes are
// silently ignored.
func (b *Buffer) Set(x, y int, ch rune, style StyleKey) {
	if b.InBounds(x, y) {
		b.Cells[y][x] = Cell{Ch: ch, Style: style}
	}
}

// SetString writes a string starting at (x, y), advancing x for each
// rune. Characters that fall outside the buffer are silently skipped.
func (b *Buffer) SetString(x, y int, s string, style StyleKey) {
	for i, ch := range s {
		b.Set(x+i, y, ch, style)
	}
}

// Fill resets every cell to a space with the given style.
func (b *Buffer) Fill(style StyleKey) {
	for y := range b.Cells {
		for x := range b.Cells[y] {
			b.Cells[y][x] = Cell{Ch: ' ', Style: style}
		}
	}
}
