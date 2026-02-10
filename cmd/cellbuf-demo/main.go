// cellbuf-demo renders a sample buffer to the terminal to visually verify
// that cellbuf + drawutil + lipgloss styling works correctly.
//
// Run: GOWORK=off go run ./cmd/cellbuf-demo/
package main

import (
	"fmt"
	"image"

	"charm.land/lipgloss/v2"
	"github.com/wesen/grail/pkg/cellbuf"
	"github.com/wesen/grail/pkg/drawutil"
)

// Style keys
const (
	BG       cellbuf.StyleKey = 0
	Grid     cellbuf.StyleKey = 1
	Edge     cellbuf.StyleKey = 2
	EdgeHot  cellbuf.StyleKey = 3
	NodeBox  cellbuf.StyleKey = 4
	NodeText cellbuf.StyleKey = 5
	Label    cellbuf.StyleKey = 6
)

func main() {
	styles := map[cellbuf.StyleKey]lipgloss.Style{
		BG:       lipgloss.NewStyle().Foreground(lipgloss.Color("#333333")).Background(lipgloss.Color("#0a0a0a")),
		Grid:     lipgloss.NewStyle().Foreground(lipgloss.Color("#1a3a1a")).Background(lipgloss.Color("#0a0a0a")),
		Edge:     lipgloss.NewStyle().Foreground(lipgloss.Color("#00d4a0")).Background(lipgloss.Color("#0a0a0a")),
		EdgeHot:  lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6600")).Background(lipgloss.Color("#0a0a0a")).Bold(true),
		NodeBox:  lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff88")).Background(lipgloss.Color("#0a1510")),
		NodeText: lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffcc")).Background(lipgloss.Color("#0a1510")).Bold(true),
		Label:    lipgloss.NewStyle().Foreground(lipgloss.Color("#ffcc00")).Background(lipgloss.Color("#0a0a0a")),
	}

	buf := cellbuf.New(60, 25, BG)

	// Grid dots via drawutil
	drawutil.DrawGrid(buf, 0, 0, 4, 2, Grid)

	// Draw node boxes
	drawBox(buf, 5, 2, 12, 3, NodeBox, '╭', '╮', '╰', '╯', '─', '│')
	buf.SetString(7, 3, " START  ", NodeText)

	drawBox(buf, 3, 10, 16, 3, NodeBox, '┌', '┐', '└', '┘', '─', '│')
	buf.SetString(5, 11, " x = x + 1  ", NodeText)

	drawBox(buf, 33, 6, 14, 3, NodeBox, '◇', '◇', '◇', '◇', '─', '│')
	buf.SetString(35, 7, " x > 10 ?  ", NodeText)

	drawBox(buf, 35, 18, 10, 3, NodeBox, '╭', '╮', '╰', '╯', '─', '│')
	buf.SetString(37, 19, "  END   ", NodeText)

	// Edges using drawutil — EdgeExit computes exit points from node rects
	startRect := image.Rect(5, 2, 17, 5)
	decisionRect := image.Rect(33, 6, 47, 9)
	processRect := image.Rect(3, 10, 19, 13)
	endRect := image.Rect(35, 18, 45, 21)

	// START → decision (arrow line)
	e1a := drawutil.EdgeExit(startRect, decisionRect.Min)
	e1b := drawutil.EdgeExit(decisionRect, startRect.Min)
	drawutil.DrawArrowLine(buf, e1a.X, e1a.Y, e1b.X, e1b.Y, Edge, Edge)

	// decision → END (NO branch, down)
	e2a := drawutil.EdgeExit(decisionRect, endRect.Min)
	e2b := drawutil.EdgeExit(endRect, decisionRect.Min)
	drawutil.DrawArrowLine(buf, e2a.X, e2a.Y, e2b.X, e2b.Y, Edge, Edge)
	buf.SetString(42, 12, "NO", Label)

	// decision → process (YES branch, dashed preview style)
	e3a := drawutil.EdgeExit(decisionRect, processRect.Min)
	e3b := drawutil.EdgeExit(processRect, decisionRect.Min)
	drawutil.DrawDashedLine(buf, e3a.X, e3a.Y, e3b.X, e3b.Y, EdgeHot)
	buf.SetString(24, 8, "YES", Label)

	// process → START (loop back)
	e4a := drawutil.EdgeExit(processRect, startRect.Min)
	e4b := drawutil.EdgeExit(startRect, processRect.Min)
	drawutil.DrawArrowLine(buf, e4a.X, e4a.Y, e4b.X, e4b.Y, EdgeHot, EdgeHot)

	// Title
	fmt.Println()
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffcc")).
		Bold(true).
		Underline(true)
	fmt.Println(title.Render("  cellbuf visual demo — GRaIL-style flowchart"))
	fmt.Println()

	// Render and print
	fmt.Println(buf.Render(styles))

	fmt.Println()
	legend := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	fmt.Println(legend.Render("  Grid=dim dots  Edge=green lines  EdgeHot=orange active  Nodes=boxed"))
	fmt.Println()
}

func drawBox(buf *cellbuf.Buffer, x, y, w, h int, style cellbuf.StyleKey,
	tl, tr, bl, br, horiz, vert rune) {
	// Top edge
	buf.Set(x, y, tl, style)
	for i := 1; i < w-1; i++ {
		buf.Set(x+i, y, horiz, style)
	}
	buf.Set(x+w-1, y, tr, style)
	// Bottom edge
	buf.Set(x, y+h-1, bl, style)
	for i := 1; i < w-1; i++ {
		buf.Set(x+i, y+h-1, horiz, style)
	}
	buf.Set(x+w-1, y+h-1, br, style)
	// Sides
	for j := 1; j < h-1; j++ {
		buf.Set(x, y+j, vert, style)
		buf.Set(x+w-1, y+j, vert, style)
		// Fill interior with space in node style
		for i := 1; i < w-1; i++ {
			buf.Set(x+i, y+j, ' ', style)
		}
	}
}
