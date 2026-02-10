package grailui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// codeHints provides input hints per node type.
var codeHints = map[string]string{
	"process":  " (stmts separated by ;)",
	"decision": " (boolean expression)",
	"io":       ` (print("..") or input("prompt", var))`,
}

// openEditModal opens the edit modal for the selected node.
func (m Model) openEditModal() (tea.Model, tea.Cmd) {
	if m.SelectedID == nil {
		return m, nil
	}
	node := m.Graph.Node(*m.SelectedID)
	if node == nil {
		return m, nil
	}

	m.EditOpen = true
	m.EditNodeID = *m.SelectedID
	m.EditFocus = 0

	m.EditLabel = textinput.New()
	m.EditLabel.Prompt = ""
	m.EditLabel.CharLimit = 30
	m.EditLabel.SetValue(node.Data.Text)

	m.EditCode = textinput.New()
	m.EditCode.Prompt = ""
	m.EditCode.CharLimit = 80
	m.EditCode.SetValue(node.Data.Code)

	cmd := m.EditLabel.Focus()
	return m, cmd
}

// handleEditKeys processes keys when the edit modal is open.
func (m Model) handleEditKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc", "escape":
		m.EditOpen = false
		return m, nil

	case "enter":
		// Save and close
		node := m.Graph.Node(m.EditNodeID)
		if node != nil {
			node.Data.Text = strings.ToUpper(strings.TrimSpace(m.EditLabel.Value()))
			node.Data.Code = strings.TrimSpace(m.EditCode.Value())
		}
		m.EditOpen = false
		return m, nil

	case "tab", "shift+tab":
		// Toggle focus
		if m.EditFocus == 0 {
			m.EditFocus = 1
			m.EditLabel.Blur()
			cmd := m.EditCode.Focus()
			return m, cmd
		} else {
			m.EditFocus = 0
			m.EditCode.Blur()
			cmd := m.EditLabel.Focus()
			return m, cmd
		}

	default:
		// Forward to active textinput
		var cmd tea.Cmd
		if m.EditFocus == 0 {
			m.EditLabel, cmd = m.EditLabel.Update(msg)
		} else {
			m.EditCode, cmd = m.EditCode.Update(msg)
		}
		return m, cmd
	}
}

// buildEditModalLayer renders the edit modal as a centered Z=100 Layer.
func buildEditModalLayer(m Model, screenW, screenH int) *lipgloss.Layer {
	node := m.Graph.Node(m.EditNodeID)
	if node == nil {
		return nil
	}

	info := nodeTypeInfo[node.Data.Type]

	titleStyle := lipgloss.NewStyle().
		Foreground(c("#00ffc8")).
		Background(c("#0a1510")).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(c("#ddaa44")).
		Background(c("#0a1510"))

	hintStyle := lipgloss.NewStyle().
		Foreground(c("#336655")).
		Background(c("#0a1510")).
		Italic(true)

	// Active field indicator
	focusLabel := "  "
	focusCode := "  "
	if m.EditFocus == 0 {
		focusLabel = "▸ "
	} else {
		focusCode = "▸ "
	}

	hint := codeHints[node.Data.Type]

	lines := []string{
		titleStyle.Render(fmt.Sprintf("  ✏️  EDIT — %s", strings.ToUpper(info.Label))),
		"",
		labelStyle.Render(focusLabel + "Label:"),
		"  " + m.EditLabel.View(),
		"",
		labelStyle.Render(focusCode + "Code" + hint + ":"),
		"  " + m.EditCode.View(),
		"",
		hintStyle.Render("  [tab] switch  [enter] save  [esc] cancel"),
	}

	content := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(c("#00d4a0")).
		Background(c("#0a1510")).
		Width(52).
		Padding(1, 2)

	rendered := boxStyle.Render(content)

	// Center on screen
	renderedW := lipgloss.Width(rendered)
	renderedH := lipgloss.Height(rendered)
	cx := (screenW - renderedW) / 2
	cy := (screenH - renderedH) / 2
	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}

	return lipgloss.NewLayer(rendered).X(cx).Y(cy).Z(100).ID("edit-modal")
}
