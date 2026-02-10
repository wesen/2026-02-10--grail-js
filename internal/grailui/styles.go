package grailui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// c is shorthand for lipgloss.Color.
func c(hex string) color.Color { return lipgloss.Color(hex) }

// Color palette — CRT green terminal aesthetic.
var (
	colorBG = c("#080e0b")

	// Node type colors
	nodeColors = map[string]struct{ border, text color.Color }{
		"process":   {border: c("#00d4a0"), text: c("#00ffc8")},
		"decision":  {border: c("#00ccee"), text: c("#66ffee")},
		"terminal":  {border: c("#44ff88"), text: c("#88ffbb")},
		"io":        {border: c("#ddaa44"), text: c("#ffcc66")},
		"connector": {border: c("#1a6a4a"), text: c("#00d4a0")},
	}

	// Selection / execution override colors
	selBorder  = c("#00ffee")
	selText    = c("#00ffee")
	selBG      = c("#0a1a15")
	execBorder = c("#ffcc00")
	execText   = c("#ffee66")
	execBG     = c("#12120a")

	// Edge colors (used in later tickets)
	_ = c("#00d4a0") // edgeColor
	_ = c("#ffcc00") // edgeActColor
	_ = c("#00ffc8") // edgeLblColor

	// Chrome colors
	toolbarColor = c("#00ffc8")
	footerColor  = c("#666666")
)

// borderForType returns the border style for a given node type.
func borderForType(nodeType string) lipgloss.Border {
	switch nodeType {
	case "terminal":
		return lipgloss.RoundedBorder()
	case "decision":
		return lipgloss.DoubleBorder()
	default:
		return lipgloss.NormalBorder()
	}
}
