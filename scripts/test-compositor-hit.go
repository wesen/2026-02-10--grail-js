package main

import (
	"fmt"
	"charm.land/lipgloss/v2"
)

func main() {
	a := lipgloss.NewLayer("AAAA").X(10).Y(5).Z(0).ID("target-a")
	b := lipgloss.NewLayer("BBBB").X(20).Y(8).Z(0).ID("target-b")

	comp := lipgloss.NewCompositor(a, b)

	tests := []struct {
		x, y   int
		wantID string
	}{
		{10, 5, "target-a"}, // exact top-left of A
		{13, 5, "target-a"}, // inside A (width=4)
		{20, 8, "target-b"}, // exact top-left of B
		{0, 0, ""},          // miss
		{14, 5, ""},         // just past A (width=4, so x=10..13)
		{9, 5, ""},          // just before A
	}

	allPass := true
	for _, tc := range tests {
		hit := comp.Hit(tc.x, tc.y)
		gotID := hit.ID()
		status := "PASS"
		if gotID != tc.wantID {
			status = "FAIL"
			allPass = false
		}
		fmt.Printf("  %s  Hit(%d,%d) = %q  want %q\n", status, tc.x, tc.y, gotID, tc.wantID)
	}

	if allPass {
		fmt.Println("\n✅ Checkpoint B: All hit tests passed!")
	} else {
		fmt.Println("\n❌ Checkpoint B: Some hit tests FAILED")
	}
}
