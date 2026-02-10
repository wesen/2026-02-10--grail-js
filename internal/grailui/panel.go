package grailui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

const panelWidth = 34

var (
	panelTitleStyle = lipgloss.NewStyle().
			Foreground(c("#00ffc8")).
			Background(c("#080e0b")).
			Bold(true)

	panelDimStyle = lipgloss.NewStyle().
			Foreground(c("#336655")).
			Background(c("#080e0b"))

	panelTextStyle = lipgloss.NewStyle().
			Foreground(c("#00d4a0")).
			Background(c("#080e0b"))

	panelVarNameStyle = lipgloss.NewStyle().
				Foreground(c("#ddaa44")).
				Background(c("#080e0b"))

	panelVarValStyle = lipgloss.NewStyle().
				Foreground(c("#00ffc8")).
				Background(c("#080e0b"))

	panelSepStyle = lipgloss.NewStyle().
			Foreground(c("#1a4a3a")).
			Background(c("#080e0b"))
)

// buildVarsPanelLayer renders the variables section.
func buildVarsPanelLayer(vars map[string]any, x, y, width, height int) *lipgloss.Layer {
	var lines []string
	lines = append(lines, panelTitleStyle.Render("📦 VARIABLES"))
	lines = append(lines, panelDimStyle.Render(strings.Repeat("─", width-2)))

	if len(vars) == 0 {
		lines = append(lines, panelDimStyle.Render("  (none)"))
	} else {
		for k, v := range vars {
			line := fmt.Sprintf("  %s = %v",
				panelVarNameStyle.Render(k),
				panelVarValStyle.Render(fmt.Sprintf("%v", v)))
			lines = append(lines, line)
		}
	}

	// Pad to height
	for len(lines) < height {
		lines = append(lines, "")
	}
	lines = lines[:height]

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
		// Show last N lines that fit
		maxLines := height - 2
		start := 0
		if len(output) > maxLines {
			start = len(output) - maxLines
		}
		for _, line := range output[start:] {
			styled := panelTextStyle.Render("  " + line)
			lines = append(lines, styled)
		}
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	lines = lines[:height]

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
