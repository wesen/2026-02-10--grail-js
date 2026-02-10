// GRaIL — Graphical Representation and Interpretation Language
// Terminal flowchart editor + interpreter.
//
// Run: GOWORK=off go run ./cmd/grail/
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/wesen/grail/internal/grailui"
)

func main() {
	p := tea.NewProgram(grailui.NewModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
