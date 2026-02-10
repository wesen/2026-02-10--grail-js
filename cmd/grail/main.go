// GRaIL — Graphical Representation and Interpretation Language
// Terminal flowchart editor + interpreter.
//
// Scaffold: minimal Bubbletea v2 + Lipgloss v2 app with Compositor/Layer
// compositing and mouse support. This is Checkpoint A.
//
// Run: GOWORK=off go run ./cmd/grail/
package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type model struct {
	width, height  int
	mouseX, mouseY int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.MouseMsg:
		mouse := msg.Mouse()
		m.mouseX = mouse.X
		m.mouseY = mouse.Y
	}

	return m, nil
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ffc8"))

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	crossStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6600")).
			Bold(true)
)

func (m model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("")
	}

	// Background: fill entire screen with dark green
	bgLine := strings.Repeat(" ", m.width)
	bgLines := make([]string, m.height)
	for i := range bgLines {
		bgLines[i] = bgLine
	}
	bg := lipgloss.NewStyle().
		Background(lipgloss.Color("#080e0b")).
		Render(strings.Join(bgLines, "\n"))
	bgLayer := lipgloss.NewLayer(bg).X(0).Y(0).Z(0).ID("bg")

	// Title
	title := titleStyle.Render("  GRaIL — Scaffold (Checkpoint A)  ")
	titleLayer := lipgloss.NewLayer(title).X(0).Y(0).Z(1)

	// Footer with mouse coordinates
	info := footerStyle.Render(fmt.Sprintf(
		"  Mouse: (%d, %d)  Term: %dx%d  [q]uit",
		m.mouseX, m.mouseY, m.width, m.height,
	))
	infoLayer := lipgloss.NewLayer(info).X(0).Y(m.height-1).Z(1)

	// Crosshair at mouse position
	cross := crossStyle.Render("+")
	crossLayer := lipgloss.NewLayer(cross).X(m.mouseX).Y(m.mouseY).Z(2).ID("cross")

	// Compose with Compositor (handles X/Y/Z positioning and z-sort)
	comp := lipgloss.NewCompositor(bgLayer, titleLayer, infoLayer, crossLayer)

	// Render onto a fixed-size canvas
	canvas := lipgloss.NewCanvas(m.width, m.height)
	canvas.Compose(comp)

	v := tea.NewView(canvas.Render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

func main() {
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
