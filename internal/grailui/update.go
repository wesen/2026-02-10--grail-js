package grailui

import tea "charm.land/bubbletea/v2"

const panStep = 3

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		// Camera panning
		case "up":
			m.CamY -= panStep
		case "down":
			m.CamY += panStep
		case "left":
			m.CamX -= panStep
		case "right":
			m.CamX += panStep

		// Tool selection
		case "s":
			m.CurrentTool = ToolSelect
		case "a":
			m.CurrentTool = ToolAdd
		case "c":
			m.CurrentTool = ToolConnect
		}

	case tea.MouseMsg:
		mouse := msg.Mouse()
		m.MouseX = mouse.X
		m.MouseY = mouse.Y
	}

	return m, nil
}
