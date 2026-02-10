// Package tealayout provides declarative layout computation and common
// chrome layer builders for Bubbletea v2 + Lipgloss v2 apps.
package tealayout

import "image"

// Region is a named rectangular area of the terminal.
type Region struct {
	Name string
	Rect image.Rectangle
}

// Layout holds the computed regions for a given terminal size.
type Layout struct {
	TermW, TermH int
	Regions      map[string]Region
}

// Get returns the region with the given name, or a zero Region.
func (l Layout) Get(name string) Region {
	return l.Regions[name]
}

// LayoutBuilder accumulates fixed regions and computes the remainder.
type LayoutBuilder struct {
	termW, termH int
	top, bottom  int // rows consumed from top/bottom
	right        int // columns consumed from right
	regions      []Region
}

// NewLayoutBuilder creates a builder for the given terminal size.
func NewLayoutBuilder(termW, termH int) *LayoutBuilder {
	return &LayoutBuilder{termW: termW, termH: termH}
}

// TopFixed reserves rows from the top. Returns the builder for chaining.
func (b *LayoutBuilder) TopFixed(name string, height int) *LayoutBuilder {
	y := b.top
	b.regions = append(b.regions, Region{
		Name: name,
		Rect: image.Rect(0, y, b.termW, y+height),
	})
	b.top += height
	return b
}

// BottomFixed reserves rows from the bottom. Returns the builder for chaining.
func (b *LayoutBuilder) BottomFixed(name string, height int) *LayoutBuilder {
	y := b.termH - b.bottom - height
	b.regions = append(b.regions, Region{
		Name: name,
		Rect: image.Rect(0, y, b.termW, y+height),
	})
	b.bottom += height
	return b
}

// RightFixed reserves columns from the right, spanning the area between
// top and bottom fixed regions. Returns the builder for chaining.
func (b *LayoutBuilder) RightFixed(name string, width int) *LayoutBuilder {
	x := b.termW - b.right - width
	b.regions = append(b.regions, Region{
		Name: name,
		Rect: image.Rect(x, b.top, x+width, b.termH-b.bottom),
	})
	b.right += width
	return b
}

// Remaining assigns whatever rectangle is left after fixed allocations.
func (b *LayoutBuilder) Remaining(name string) *LayoutBuilder {
	b.regions = append(b.regions, Region{
		Name: name,
		Rect: image.Rect(0, b.top, b.termW-b.right, b.termH-b.bottom),
	})
	return b
}

// Build computes and returns the final Layout.
func (b *LayoutBuilder) Build() Layout {
	l := Layout{
		TermW:   b.termW,
		TermH:   b.termH,
		Regions: make(map[string]Region, len(b.regions)),
	}
	for _, r := range b.regions {
		// Clamp negative dimensions to zero
		if r.Rect.Dx() < 0 || r.Rect.Dy() < 0 {
			r.Rect = image.Rectangle{}
		}
		l.Regions[r.Name] = r
	}
	return l
}
