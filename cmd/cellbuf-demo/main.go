// cellbuf-demo renders a sample buffer to the terminal to visually verify
// that cellbuf + lipgloss styling works correctly.
//
// Run: GOWORK=off go run ./cmd/cellbuf-demo/
package main

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/wesen/grail/pkg/cellbuf"
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

	// Grid dots
	for y := 0; y < 25; y++ {
		for x := 0; x < 60; x++ {
			if x%4 == 0 && y%2 == 0 {
				buf.Set(x, y, '·', Grid)
			}
		}
	}

	// Draw a node box: "START" at (5, 2)
	drawBox(buf, 5, 2, 12, 3, NodeBox, '╭', '╮', '╰', '╯', '─', '│')
	buf.SetString(7, 3, " START  ", NodeText)

	// Draw a node box: "x = x + 1" at (5, 10)
	drawBox(buf, 3, 10, 16, 3, NodeBox, '┌', '┐', '└', '┘', '─', '│')
	buf.SetString(5, 11, " x = x + 1  ", NodeText)

	// Draw a node box: "x > 10?" at (35, 6)
	drawBox(buf, 33, 6, 14, 3, NodeBox, '◇', '◇', '◇', '◇', '─', '│')
	buf.SetString(35, 7, " x > 10 ?  ", NodeText)

	// Draw a node box: "END" at (35, 18)
	drawBox(buf, 35, 18, 10, 3, NodeBox, '╭', '╮', '╰', '╯', '─', '│')
	buf.SetString(37, 19, "  END   ", NodeText)

	// Vertical edge: START → decision
	for y := 5; y <= 7; y++ {
		buf.Set(10, y, '│', Edge)
	}
	buf.Set(10, 5, '▼', Edge)
	// Horizontal edge to decision
	for x := 11; x <= 33; x++ {
		buf.Set(x, 7, '─', Edge)
	}

	// Vertical edge: decision → process (YES branch, going left then down)
	buf.Set(33, 9, '│', EdgeHot)
	for y := 9; y <= 10; y++ {
		buf.Set(20, y, '│', EdgeHot)
	}
	for x := 20; x <= 33; x++ {
		buf.Set(x, 9, '─', EdgeHot)
	}
	buf.SetString(24, 8, "YES", Label)

	// Vertical edge: decision → END (NO branch, going down)
	for y := 9; y <= 18; y++ {
		buf.Set(40, y, '│', Edge)
	}
	buf.Set(40, 18, '▼', Edge)
	buf.SetString(42, 12, "NO", Label)

	// Loop back: process → decision
	for y := 6; y <= 10; y++ {
		buf.Set(2, y, '│', EdgeHot)
	}
	for x := 2; x <= 3; x++ {
		buf.Set(x, 10, '─', EdgeHot)
	}
	for x := 2; x <= 10; x++ {
		buf.Set(x, 6, '─', EdgeHot)
	}
	buf.Set(10, 6, '▼', EdgeHot)

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
