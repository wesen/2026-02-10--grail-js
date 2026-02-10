// Package flowinterp implements the GRaIL flow interpreter using Goja (JS runtime).
package flowinterp

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

// FlowNode is a simplified node representation for the interpreter.
type FlowNode struct {
	ID   int
	Type string // "process", "decision", "terminal", "io", "connector"
	Text string
	Code string
}

// FlowEdge is a simplified edge representation for the interpreter.
type FlowEdge struct {
	FromID int
	ToID   int
	Label  string
}

// Interpreter executes a flowchart step by step.
type Interpreter struct {
	nodes   []FlowNode
	edges   []FlowEdge
	Vars    map[string]interface{}
	Output  []string
	Current *int
	Done    bool
	Err     string

	WaitInput   bool
	InputPrompt string
	inputVar    string

	StepCount int
	MaxSteps  int
	runtime   *goja.Runtime
}

// New creates an interpreter for the given flowchart.
func New(nodes []FlowNode, edges []FlowEdge) *Interpreter {
	interp := &Interpreter{
		nodes:    nodes,
		edges:    edges,
		Vars:     make(map[string]interface{}),
		MaxSteps: 500,
		runtime:  goja.New(),
	}

	// Register print function
	interp.runtime.Set("print", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, arg := range call.Arguments {
			parts[i] = arg.String()
		}
		interp.Output = append(interp.Output, strings.Join(parts, " "))
		return goja.Undefined()
	})

	// Register str function
	interp.runtime.Set("str", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) == 0 {
			return interp.runtime.ToValue("")
		}
		return interp.runtime.ToValue(call.Arguments[0].String())
	})

	return interp
}

// Reset clears the interpreter state for re-running.
func (interp *Interpreter) Reset() {
	interp.Vars = make(map[string]interface{})
	interp.Output = nil
	interp.Current = nil
	interp.Done = false
	interp.Err = ""
	interp.WaitInput = false
	interp.InputPrompt = ""
	interp.inputVar = ""
	interp.StepCount = 0
}

// Step executes one step. Pass inputValue when WaitInput is true.
func (interp *Interpreter) Step(inputValue *string) {
	if interp.Done || interp.Err != "" {
		return
	}
	interp.StepCount++
	if interp.StepCount > interp.MaxSteps {
		interp.Err = "MAX STEPS EXCEEDED"
		interp.Done = true
		return
	}

	// Handle pending input
	if interp.WaitInput {
		if inputValue == nil {
			return
		}
		interp.Vars[interp.inputVar] = parseInputValue(*inputValue)
		interp.Output = append(interp.Output, fmt.Sprintf("> %s", *inputValue))
		interp.WaitInput = false
		interp.advance(*interp.Current)
		return
	}

	// First step: find START
	if interp.Current == nil {
		start := interp.findStart()
		if start == nil {
			interp.Err = "NO START NODE"
			interp.Done = true
			return
		}
		interp.Current = &start.ID
		interp.Output = append(interp.Output, "── PROGRAM START ──")
		interp.advance(start.ID)
		return
	}

	node := interp.findNode(*interp.Current)
	if node == nil {
		interp.Err = "BROKEN LINK"
		interp.Done = true
		return
	}

	defer func() {
		if r := recover(); r != nil {
			interp.Err = fmt.Sprintf("ERROR at %q: %v", node.Text, r)
			interp.Done = true
		}
	}()

	switch node.Type {
	case "terminal":
		interp.Output = append(interp.Output, "── PROGRAM END ──")
		interp.Done = true

	case "connector":
		interp.advance(node.ID)

	case "process":
		code := strings.TrimSpace(node.Code)
		if code != "" {
			interp.execStatements(code)
		}
		interp.advance(node.ID)

	case "decision":
		code := strings.TrimSpace(node.Code)
		result := false
		if code != "" {
			result = interp.evalBool(code)
		}
		outs := interp.outEdges(node.ID)
		var ye, ne *FlowEdge
		for i := range outs {
			switch strings.ToUpper(outs[i].Label) {
			case "Y":
				ye = &outs[i]
			case "N":
				ne = &outs[i]
			}
		}
		var next *FlowEdge
		if result {
			if ye != nil {
				next = ye
			} else if len(outs) > 0 {
				next = &outs[0]
			}
		} else {
			if ne != nil {
				next = ne
			} else if len(outs) > 0 {
				next = &outs[0]
			}
		}
		if next != nil {
			interp.Current = &next.ToID
		} else {
			interp.Done = true
		}

	case "io":
		code := strings.TrimSpace(node.Code)
		if interp.matchInput(code) {
			// waitInput is now set
		} else {
			if code != "" {
				interp.execStatements(code)
			}
			interp.advance(node.ID)
		}
	}
}

// --- helpers ---

var inputRe = regexp.MustCompile(`(?i)^(?:input|read)\s*\(\s*["']?([^"']*)["']?\s*,?\s*["']?([a-zA-Z_]\w*)["']?\s*\)$`)

func (interp *Interpreter) matchInput(code string) bool {
	m := inputRe.FindStringSubmatch(code)
	if m == nil {
		return false
	}
	interp.InputPrompt = m[1]
	if interp.InputPrompt == "" {
		interp.InputPrompt = "INPUT:"
	}
	interp.inputVar = m[2]
	if interp.inputVar == "" {
		interp.inputVar = "x"
	}
	interp.WaitInput = true
	interp.Output = append(interp.Output, interp.InputPrompt)
	return true
}

func (interp *Interpreter) findStart() *FlowNode {
	for i := range interp.nodes {
		n := &interp.nodes[i]
		if n.Type == "terminal" && strings.Contains(strings.ToUpper(n.Text), "START") {
			return n
		}
	}
	return nil
}

func (interp *Interpreter) findNode(id int) *FlowNode {
	for i := range interp.nodes {
		if interp.nodes[i].ID == id {
			return &interp.nodes[i]
		}
	}
	return nil
}

func (interp *Interpreter) outEdges(id int) []FlowEdge {
	var out []FlowEdge
	for _, e := range interp.edges {
		if e.FromID == id {
			out = append(out, e)
		}
	}
	return out
}

func (interp *Interpreter) advance(id int) {
	outs := interp.outEdges(id)
	if len(outs) > 0 {
		interp.Current = &outs[0].ToID
	} else {
		interp.Current = nil
		interp.Done = true
	}
}

func (interp *Interpreter) syncVarsToRuntime() {
	for k, v := range interp.Vars {
		interp.runtime.Set(k, v)
	}
}

func (interp *Interpreter) syncVarsFromRuntime() {
	// After execution, read back all exported vars from the runtime's global scope
	// This is necessary because goja may update values in-place
}

var assignRe = regexp.MustCompile(`^([a-zA-Z_]\w*)\s*=\s*(.+)$`)

func (interp *Interpreter) execStatements(code string) {
	for _, stmt := range strings.Split(code, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		m := assignRe.FindStringSubmatch(stmt)
		if m != nil {
			varName := m[1]
			expr := m[2]
			interp.syncVarsToRuntime()
			val, err := interp.runtime.RunString(expr)
			if err != nil {
				panic(fmt.Sprintf("eval %q: %v", expr, err))
			}
			interp.Vars[varName] = val.Export()
		} else {
			interp.syncVarsToRuntime()
			_, err := interp.runtime.RunString(stmt)
			if err != nil {
				panic(fmt.Sprintf("exec %q: %v", stmt, err))
			}
		}
	}
}

func (interp *Interpreter) evalBool(code string) bool {
	interp.syncVarsToRuntime()
	val, err := interp.runtime.RunString(code)
	if err != nil {
		panic(fmt.Sprintf("eval %q: %v", code, err))
	}
	return val.ToBoolean()
}

func parseInputValue(s string) interface{} {
	// Try parsing as integer
	s = strings.TrimSpace(s)
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err == nil {
		// Verify the whole string was consumed
		if fmt.Sprintf("%d", n) == s {
			return n
		}
	}
	return s
}
