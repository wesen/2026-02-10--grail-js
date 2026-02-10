package tealayout

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// ToolbarLayer creates a Layer for a toolbar at the top of the screen.
func ToolbarLayer(content string, width int, style lipgloss.Style) *lipgloss.Layer {
	rendered := style.Width(width).Render(content)
	return lipgloss.NewLayer(rendered).X(0).Y(0).Z(0).ID("toolbar")
}

// FooterLayer creates a Layer for a footer at a given y position.
func FooterLayer(content string, width, y int, style lipgloss.Style) *lipgloss.Layer {
	rendered := style.Width(width).Render(content)
	return lipgloss.NewLayer(rendered).X(0).Y(y).Z(0).ID("footer")
}

// VerticalSeparator creates a Layer with a vertical line of │ characters.
func VerticalSeparator(x, y, height int, style lipgloss.Style) *lipgloss.Layer {
	lines := make([]string, height)
	for i := range lines {
		lines[i] = "│"
	}
	rendered := style.Render(strings.Join(lines, "\n"))
	return lipgloss.NewLayer(rendered).X(x).Y(y).Z(0).ID("separator")
}

// ModalLayer creates a centered high-Z overlay Layer.
// The content is rendered inside boxStyle, then centered on the terminal.
func ModalLayer(content string, termW, termH int, boxStyle lipgloss.Style) *lipgloss.Layer {
	rendered := boxStyle.Render(content)
	w := lipgloss.Width(rendered)
	h := lipgloss.Height(rendered)
	cx := (termW - w) / 2
	cy := (termH - h) / 2
	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}
	return lipgloss.NewLayer(rendered).X(cx).Y(cy).Z(100).ID("modal")
}

// FillLayer creates a Layer filled with the given style at a region's position.
// Useful for creating background layers that fill a layout region.
func FillLayer(r Region, style lipgloss.Style, id string, z int) *lipgloss.Layer {
	w := r.Rect.Dx()
	h := r.Rect.Dy()
	if w <= 0 || h <= 0 {
		return lipgloss.NewLayer("").X(r.Rect.Min.X).Y(r.Rect.Min.Y).Z(z).ID(id)
	}
	line := strings.Repeat(" ", w)
	lines := make([]string, h)
	for i := range lines {
		lines[i] = line
	}
	rendered := style.Render(strings.Join(lines, "\n"))
	return lipgloss.NewLayer(rendered).X(r.Rect.Min.X).Y(r.Rect.Min.Y).Z(z).ID(id)
}
