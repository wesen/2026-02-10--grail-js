package flowinterp

import (
	"strings"
	"testing"
)

// makeSum15 creates the standard "sum 1..5" flowchart.
func makeSum15() ([]FlowNode, []FlowEdge) {
	nodes := []FlowNode{
		{ID: 0, Type: "terminal", Text: "START"},
		{ID: 1, Type: "process", Text: "INIT", Code: "i = 1; sum = 0"},
		{ID: 2, Type: "decision", Text: "i <= 5?", Code: "i <= 5"},
		{ID: 3, Type: "process", Text: "ACCUMULATE", Code: "sum = sum + i; i = i + 1"},
		{ID: 4, Type: "connector", Text: ""},
		{ID: 5, Type: "io", Text: "PRINT SUM", Code: `print("Sum 1..5 = " + str(sum))`},
		{ID: 6, Type: "terminal", Text: "END"},
	}
	edges := []FlowEdge{
		{FromID: 0, ToID: 1},
		{FromID: 1, ToID: 2},
		{FromID: 2, ToID: 3, Label: "Y"},
		{FromID: 3, ToID: 4},
		{FromID: 4, ToID: 2},
		{FromID: 2, ToID: 5, Label: "N"},
		{FromID: 5, ToID: 6},
	}
	return nodes, edges
}

func TestSum15Integration(t *testing.T) {
	nodes, edges := makeSum15()
	interp := New(nodes, edges)

	for !interp.Done && interp.Err == "" && interp.StepCount < 200 {
		interp.Step(nil)
	}

	if interp.Err != "" {
		t.Fatalf("unexpected error: %s", interp.Err)
	}
	if !interp.Done {
		t.Fatal("interpreter didn't finish")
	}

	// Check variables
	sum, ok := interp.Vars["sum"]
	if !ok {
		t.Fatal("missing var 'sum'")
	}
	if sumVal, ok := sum.(int64); !ok || sumVal != 15 {
		t.Fatalf("sum = %v (%T), want 15", sum, sum)
	}

	i, ok := interp.Vars["i"]
	if !ok {
		t.Fatal("missing var 'i'")
	}
	if iVal, ok := i.(int64); !ok || iVal != 6 {
		t.Fatalf("i = %v (%T), want 6", i, i)
	}

	// Check output
	output := strings.Join(interp.Output, "\n")
	if !strings.Contains(output, "PROGRAM START") {
		t.Error("output missing PROGRAM START")
	}
	if !strings.Contains(output, "Sum 1..5 = 15") {
		t.Errorf("output missing 'Sum 1..5 = 15', got:\n%s", output)
	}
	if !strings.Contains(output, "PROGRAM END") {
		t.Error("output missing PROGRAM END")
	}

	if interp.StepCount > 100 {
		t.Errorf("too many steps: %d", interp.StepCount)
	}

	t.Logf("Sum15 completed in %d steps", interp.StepCount)
	t.Logf("Output:\n%s", output)
}

func TestNoStartNode(t *testing.T) {
	nodes := []FlowNode{
		{ID: 0, Type: "process", Text: "NOPE"},
	}
	interp := New(nodes, nil)
	interp.Step(nil)

	if interp.Err != "NO START NODE" {
		t.Errorf("err = %q, want 'NO START NODE'", interp.Err)
	}
}

func TestBrokenLink(t *testing.T) {
	nodes := []FlowNode{
		{ID: 0, Type: "terminal", Text: "START"},
	}
	edges := []FlowEdge{
		{FromID: 0, ToID: 99}, // points to nonexistent
	}
	interp := New(nodes, edges)
	interp.Step(nil) // finds START, advances to 99
	interp.Step(nil) // tries to execute 99 → broken link

	if interp.Err != "BROKEN LINK" {
		t.Errorf("err = %q, want 'BROKEN LINK'", interp.Err)
	}
}

func TestMaxStepsExceeded(t *testing.T) {
	// Infinite loop: process → connector → process
	nodes := []FlowNode{
		{ID: 0, Type: "terminal", Text: "START"},
		{ID: 1, Type: "process", Text: "LOOP"},
		{ID: 2, Type: "connector", Text: ""},
	}
	edges := []FlowEdge{
		{FromID: 0, ToID: 1},
		{FromID: 1, ToID: 2},
		{FromID: 2, ToID: 1},
	}
	interp := New(nodes, edges)
	interp.MaxSteps = 20

	for !interp.Done && interp.Err == "" {
		interp.Step(nil)
	}

	if interp.Err != "MAX STEPS EXCEEDED" {
		t.Errorf("err = %q, want 'MAX STEPS EXCEEDED'", interp.Err)
	}
}

func TestInvalidCode(t *testing.T) {
	nodes := []FlowNode{
		{ID: 0, Type: "terminal", Text: "START"},
		{ID: 1, Type: "process", Text: "BAD", Code: "x = ???invalid"},
	}
	edges := []FlowEdge{
		{FromID: 0, ToID: 1},
	}
	interp := New(nodes, edges)
	interp.Step(nil) // START → advance to 1
	interp.Step(nil) // execute bad code

	if !strings.Contains(interp.Err, "ERROR at") {
		t.Errorf("err = %q, want it to contain 'ERROR at'", interp.Err)
	}
}

func TestInputHandling(t *testing.T) {
	nodes := []FlowNode{
		{ID: 0, Type: "terminal", Text: "START"},
		{ID: 1, Type: "io", Text: "ASK", Code: `input("Name?", name)`},
		{ID: 2, Type: "terminal", Text: "END"},
	}
	edges := []FlowEdge{
		{FromID: 0, ToID: 1},
		{FromID: 1, ToID: 2},
	}
	interp := New(nodes, edges)

	interp.Step(nil) // START → advance to 1
	interp.Step(nil) // IO → sets waitInput

	if !interp.WaitInput {
		t.Fatal("expected waitInput=true")
	}
	if interp.InputPrompt != "Name?" {
		t.Errorf("prompt = %q, want 'Name?'", interp.InputPrompt)
	}

	val := "Alice"
	interp.Step(&val)

	if interp.WaitInput {
		t.Error("expected waitInput=false after input")
	}
	if interp.Vars["name"] != "Alice" {
		t.Errorf("vars[name] = %v, want 'Alice'", interp.Vars["name"])
	}

	interp.Step(nil) // END
	if !interp.Done {
		t.Error("expected done after END")
	}
}

func TestInputNumericParsing(t *testing.T) {
	nodes := []FlowNode{
		{ID: 0, Type: "terminal", Text: "START"},
		{ID: 1, Type: "io", Text: "ASK", Code: `input("Age?", age)`},
		{ID: 2, Type: "terminal", Text: "END"},
	}
	edges := []FlowEdge{
		{FromID: 0, ToID: 1},
		{FromID: 1, ToID: 2},
	}
	interp := New(nodes, edges)
	interp.Step(nil)
	interp.Step(nil)

	val := "42"
	interp.Step(&val)

	if age, ok := interp.Vars["age"].(int); !ok || age != 42 {
		t.Errorf("vars[age] = %v (%T), want 42 (int)", interp.Vars["age"], interp.Vars["age"])
	}
}

func TestReset(t *testing.T) {
	nodes, edges := makeSum15()
	interp := New(nodes, edges)

	// Run to completion
	for !interp.Done && interp.Err == "" && interp.StepCount < 200 {
		interp.Step(nil)
	}
	if !interp.Done {
		t.Fatal("should be done")
	}

	// Reset and verify clean state
	interp.Reset()
	if interp.Done || interp.Err != "" || interp.Current != nil || len(interp.Output) != 0 || len(interp.Vars) != 0 {
		t.Error("reset didn't clear state")
	}

	// Run again
	for !interp.Done && interp.Err == "" && interp.StepCount < 200 {
		interp.Step(nil)
	}
	sum := interp.Vars["sum"]
	if sumVal, ok := sum.(int64); !ok || sumVal != 15 {
		t.Fatalf("after reset: sum = %v, want 15", sum)
	}
}
