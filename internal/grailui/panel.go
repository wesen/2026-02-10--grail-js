package grailui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

const panelWidth = 34

// panelBG is the panel background color, defined inline to avoid init-order issues.
var panelBG = c("#1a2a20") // slightly lighter than canvas bg for visible distinction

// Panel styles — all share the same background for consistency.
var (
	panelTitleStyle = lipgloss.NewStyle().
			Foreground(c("#00ffc8")).
			Background(panelBG).
			Bold(true)

	panelDimStyle = lipgloss.NewStyle().
			Foreground(c("#336655")).
			Background(panelBG)

	panelTextStyle = lipgloss.NewStyle().
			Foreground(c("#00d4a0")).
			Background(panelBG)

	panelVarNameStyle = lipgloss.NewStyle().
				Foreground(c("#ddaa44")).
				Background(panelBG)

	panelVarValStyle = lipgloss.NewStyle().
				Foreground(c("#00ffc8")).
				Background(panelBG)

	panelSepStyle = lipgloss.NewStyle().
			Foreground(c("#1a4a3a")).
			Background(panelBG)

	// panelLineStyle wraps padding with consistent background.
	panelLineStyle = lipgloss.NewStyle().
			Background(panelBG)
)

// padLine right-pads and renders a line with consistent background to the given width.
func padLine(s string, width int) string {
	// Measure visible width of the already-styled string
	vis := lipgloss.Width(s)
	pad := width - vis
	if pad > 0 {
		s += panelLineStyle.Render(strings.Repeat(" ", pad))
	}
	return s
}

// buildVarsPanelLayer renders the variables section.
func buildVarsPanelLayer(vars map[string]any, x, y, width, height int) *lipgloss.Layer {
	var lines []string
	lines = append(lines, panelTitleStyle.Render("📦 VARIABLES"))
	lines = append(lines, panelDimStyle.Render(strings.Repeat("─", width-2)))

	if len(vars) == 0 {
		lines = append(lines, panelDimStyle.Render("  (none)"))
	} else {
		// Sort keys alphabetically
		keys := make([]string, 0, len(vars))
		for k := range vars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := vars[k]
			line := panelVarNameStyle.Render(fmt.Sprintf("  %s", k)) +
				panelDimStyle.Render(" = ") +
				panelVarValStyle.Render(fmt.Sprintf("%v", v))
			lines = append(lines, line)
		}
	}

	// Pad to height with bg-styled empty lines
	for len(lines) < height {
		lines = append(lines, "")
	}
	lines = lines[:height]

	// Right-pad every line to full width for consistent background
	for i, l := range lines {
		lines[i] = padLine(l, width)
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewLayer(content).X(x).Y(y).Z(1).ID("panel-vars")
}

// buildConsolePanelLayer renders the console/output section.
func buildConsolePanelLayer(output []string, x, y, width, height int) *lipgloss.Layer {
	var lines []string
	lines = append(lines, panelTitleStyle.Render("🖥️  CONSOLE"))
	lines = append(lines, panelDimStyle.Render(strings.Repeat("─", width-2)))

	if len(output) == 0 {
		lines = append(lines, panelDimStyle.Render("  (empty)"))
	} else {
		maxLines := height - 2
		start := 0
		if len(output) > maxLines {
			start = len(output) - maxLines
		}
		for _, line := range output[start:] {
			lines = append(lines, panelTextStyle.Render("  "+line))
		}
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	lines = lines[:height]

	for i, l := range lines {
		lines[i] = padLine(l, width)
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewLayer(content).X(x).Y(y).Z(1).ID("panel-console")
}

// buildHelpPanelLayer renders the static help section.
func buildHelpPanelLayer(x, y, width, height int) *lipgloss.Layer {
	helpLines := []string{
		panelTitleStyle.Render("❓ HELP"),
		panelDimStyle.Render(strings.Repeat("─", width-2)),
		panelTextStyle.Render("  click=select drag=move"),
		panelTextStyle.Render("  [s]Select [a]Add [c]Connect"),
		panelTextStyle.Render("  [e]Edit  [d]Delete"),
		panelTextStyle.Render("  [r]Run [n]Step [g]Auto"),
		panelTextStyle.Render("  [p]Pause [x]Stop"),
		panelTextStyle.Render("  Arrows: pan canvas"),
	}

	for len(helpLines) < height {
		helpLines = append(helpLines, "")
	}
	helpLines = helpLines[:height]

	for i, l := range helpLines {
		helpLines[i] = padLine(l, width)
	}

	content := strings.Join(helpLines, "\n")
	return lipgloss.NewLayer(content).X(x).Y(y).Z(1).ID("panel-help")
}

// buildSeparatorLayer creates a vertical separator line.
func buildSeparatorLayer(x, y, height int) *lipgloss.Layer {
	lines := make([]string, height)
	for i := range lines {
		lines[i] = panelSepStyle.Render("│")
	}
	content := strings.Join(lines, "\n")
	return lipgloss.NewLayer(content).X(x).Y(y).Z(1).ID("separator")
}
