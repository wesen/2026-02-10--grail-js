package main

import (
    "fmt"
    "strings"
    "time"

    "github.com/charmbracelet/lipgloss/v2"
)

func main() {
    bg := lipgloss.NewStyle().Foreground(lipgloss.Color("#1a3a2a")).Background(lipgloss.Color("#080e0b"))
    grid := lipgloss.NewStyle().Foreground(lipgloss.Color("#0e2e20")).Background(lipgloss.Color("#080e0b"))
    edge := lipgloss.NewStyle().Foreground(lipgloss.Color("#00d4a0")).Background(lipgloss.Color("#080e0b"))
    edgeActive := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffcc00")).Background(lipgloss.Color("#080e0b")).Bold(true)
    _ = edgeActive

    width := 150
    rows := 40
    iterations := 200

    // Realistic: ~90% bg, ~5% grid dots, ~5% edge chars
    // Typical run count per row: 5-15 runs

    type cell struct {
        ch    rune
        style int // 0=bg, 1=grid, 2=edge
    }
    styles := []lipgloss.Style{bg, grid, edge}

    // Build a fake buffer
    buffer := make([][]cell, rows)
    for y := 0; y < rows; y++ {
        buffer[y] = make([]cell, width)
        for x := 0; x < width; x++ {
            buffer[y][x] = cell{' ', 0}
            if x%5 == 0 && y%3 == 0 {
                buffer[y][x] = cell{'·', 1}
            }
        }
        // Add a diagonal edge
        if y < width {
            buffer[y][y] = cell{'/', 2}
            if y+1 < width { buffer[y][y+1] = cell{'/', 2} }
        }
    }

    // Count average runs per row
    totalRuns := 0
    for y := 0; y < rows; y++ {
        runs := 1
        for x := 1; x < width; x++ {
            if buffer[y][x].style != buffer[y][x-1].style {
                runs++
            }
        }
        totalRuns += runs
    }
    fmt.Printf("Average runs/row: %.1f\n", float64(totalRuns)/float64(rows))

    // Method A: StyleRanges with merged runs
    start := time.Now()
    for iter := 0; iter < iterations; iter++ {
        var lines []string
        for y := 0; y < rows; y++ {
            row := make([]rune, width)
            for x := 0; x < width; x++ {
                row[x] = buffer[y][x].ch
            }
            plain := string(row)

            var ranges []lipgloss.Range
            runStart := 0
            runStyle := buffer[y][0].style
            for x := 1; x <= width; x++ {
                cs := -1
                if x < width { cs = buffer[y][x].style }
                if cs != runStyle {
                    ranges = append(ranges, lipgloss.NewRange(runStart, x, styles[runStyle]))
                    runStart = x
                    runStyle = cs
                }
            }
            lines = append(lines, lipgloss.StyleRanges(plain, ranges...))
        }
        _ = strings.Join(lines, "\n")
    }
    mergedTime := time.Since(start)

    // Method B: Manual render per run (Style.Render on each run)
    start = time.Now()
    for iter := 0; iter < iterations; iter++ {
        var lines []string
        for y := 0; y < rows; y++ {
            var sb strings.Builder
            runStart := 0
            runStyle := buffer[y][0].style
            for x := 1; x <= width; x++ {
                cs := -1
                if x < width { cs = buffer[y][x].style }
                if cs != runStyle {
                    chunk := make([]rune, x-runStart)
                    for i := runStart; i < x; i++ {
                        chunk[i-runStart] = buffer[y][i].ch
                    }
                    sb.WriteString(styles[runStyle].Render(string(chunk)))
                    runStart = x
                    runStyle = cs
                }
            }
            lines = append(lines, sb.String())
        }
        _ = strings.Join(lines, "\n")
    }
    renderTime := time.Since(start)

    fmt.Printf("Grid: %dx%d, %d iterations\n", width, rows, iterations)
    fmt.Printf("  Merged StyleRanges: %v  (%.1f µs/frame)\n", mergedTime, float64(mergedTime.Microseconds())/float64(iterations))
    fmt.Printf("  Render-per-run:     %v  (%.1f µs/frame)\n", renderTime, float64(renderTime.Microseconds())/float64(iterations))
}
