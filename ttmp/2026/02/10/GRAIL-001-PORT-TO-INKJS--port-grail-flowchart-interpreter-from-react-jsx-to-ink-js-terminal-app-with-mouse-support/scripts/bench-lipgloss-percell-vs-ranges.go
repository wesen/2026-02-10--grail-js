package main

import (
    "fmt"
    "strings"
    "time"

    "github.com/charmbracelet/lipgloss/v2"
)

func main() {
    red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
    green := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))

    width := 200
    rows := 50
    iterations := 100

    // Method 1: Style.Render per cell
    start := time.Now()
    for iter := 0; iter < iterations; iter++ {
        var lines []string
        for y := 0; y < rows; y++ {
            var sb strings.Builder
            for x := 0; x < width; x++ {
                if (x+y)%2 == 0 {
                    sb.WriteString(red.Render("X"))
                } else {
                    sb.WriteString(green.Render("O"))
                }
            }
            lines = append(lines, sb.String())
        }
        _ = strings.Join(lines, "\n")
    }
    perCell := time.Since(start)

    // Method 2: StyleRanges (group runs of same style)
    start = time.Now()
    for iter := 0; iter < iterations; iter++ {
        var lines []string
        for y := 0; y < rows; y++ {
            plain := strings.Repeat("XO", width/2)
            ranges := make([]lipgloss.Range, 0, width)
            for x := 0; x < width; x++ {
                s := red
                if (x+y)%2 != 0 {
                    s = green
                }
                ranges = append(ranges, lipgloss.NewRange(x, x+1, s))
            }
            lines = append(lines, lipgloss.StyleRanges(plain, ranges...))
        }
        _ = strings.Join(lines, "\n")
    }
    perCellRanges := time.Since(start)

    // Method 3: StyleRanges with merged runs (realistic)
    start = time.Now()
    for iter := 0; iter < iterations; iter++ {
        var lines []string
        for y := 0; y < rows; y++ {
            // Simulate a row where most cells share the same style (realistic: ~5 runs per row)
            plain := strings.Repeat(".", width)
            ranges := []lipgloss.Range{
                lipgloss.NewRange(0, 40, green),
                lipgloss.NewRange(40, 60, red),
                lipgloss.NewRange(60, 140, green),
                lipgloss.NewRange(140, 160, red),
                lipgloss.NewRange(160, width, green),
            }
            lines = append(lines, lipgloss.StyleRanges(plain, ranges...))
        }
        _ = strings.Join(lines, "\n")
    }
    mergedRanges := time.Since(start)

    // Method 4: Raw ANSI per run (manual)
    start = time.Now()
    for iter := 0; iter < iterations; iter++ {
        var lines []string
        for y := 0; y < rows; y++ {
            var sb strings.Builder
            // Same 5 runs
            sb.WriteString("\033[38;2;0;255;0m")
            sb.WriteString(strings.Repeat(".", 40))
            sb.WriteString("\033[38;2;255;0;0m")
            sb.WriteString(strings.Repeat(".", 20))
            sb.WriteString("\033[38;2;0;255;0m")
            sb.WriteString(strings.Repeat(".", 80))
            sb.WriteString("\033[38;2;255;0;0m")
            sb.WriteString(strings.Repeat(".", 20))
            sb.WriteString("\033[38;2;0;255;0m")
            sb.WriteString(strings.Repeat(".", 40))
            sb.WriteString("\033[0m")
            lines = append(lines, sb.String())
        }
        _ = strings.Join(lines, "\n")
    }
    rawAnsi := time.Since(start)

    fmt.Printf("Grid: %dx%d, %d iterations\n", width, rows, iterations)
    fmt.Printf("  Per-cell Render():       %v  (%.1f µs/frame)\n", perCell, float64(perCell.Microseconds())/float64(iterations))
    fmt.Printf("  Per-cell StyleRanges():   %v  (%.1f µs/frame)\n", perCellRanges, float64(perCellRanges.Microseconds())/float64(iterations))
    fmt.Printf("  Merged StyleRanges(5/row):%v  (%.1f µs/frame)\n", mergedRanges, float64(mergedRanges.Microseconds())/float64(iterations))
    fmt.Printf("  Raw ANSI (5 runs/row):   %v  (%.1f µs/frame)\n", rawAnsi, float64(rawAnsi.Microseconds())/float64(iterations))

    // Also measure output size
    var sb1 strings.Builder
    for x := 0; x < width; x++ {
        sb1.WriteString(red.Render("X"))
    }
    perCellLine := sb1.String()

    mergedLine := lipgloss.StyleRanges(strings.Repeat(".", width),
        lipgloss.NewRange(0, 40, green),
        lipgloss.NewRange(40, 60, red),
        lipgloss.NewRange(60, 140, green),
        lipgloss.NewRange(140, 160, red),
        lipgloss.NewRange(160, width, green),
    )

    var sb2 strings.Builder
    sb2.WriteString("\033[38;2;0;255;0m" + strings.Repeat(".", 40))
    sb2.WriteString("\033[38;2;255;0;0m" + strings.Repeat(".", 20))
    sb2.WriteString("\033[38;2;0;255;0m" + strings.Repeat(".", 80))
    sb2.WriteString("\033[38;2;255;0;0m" + strings.Repeat(".", 20))
    sb2.WriteString("\033[38;2;0;255;0m" + strings.Repeat(".", 40))
    sb2.WriteString("\033[0m")
    rawLine := sb2.String()

    fmt.Printf("\nOutput size for one %d-col row:\n", width)
    fmt.Printf("  Per-cell Render():       %d bytes\n", len(perCellLine))
    fmt.Printf("  Merged StyleRanges(5):   %d bytes\n", len(mergedLine))
    fmt.Printf("  Raw ANSI (5 runs):       %d bytes\n", len(rawLine))
}
