// Package graphmodel provides a generic spatial graph with positioned nodes,
// labeled edges, stable iteration order, and hit testing.
package graphmodel

import "image"

// Spatial is the minimal interface for a positioned, sized element.
type Spatial interface {
	Pos() image.Point
	Size() image.Point
}

// CenterOf returns the center point of a Spatial element.
func CenterOf(s Spatial) image.Point {
	p := s.Pos()
	sz := s.Size()
	return image.Pt(p.X+sz.X/2, p.Y+sz.Y/2)
}

// BoundsOf returns the bounding rectangle of a Spatial element.
func BoundsOf(s Spatial) image.Rectangle {
	p := s.Pos()
	sz := s.Size()
	return image.Rect(p.X, p.Y, p.X+sz.X, p.Y+sz.Y)
}
