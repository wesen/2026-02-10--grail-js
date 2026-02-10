---
title: "Implementation Plan — Flow Interpreter"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-011-INTERPRETER
topics:
  - interpreter
  - go
---

# Implementation Plan — Flow Interpreter

## Overview

Port the Python `FlowInterpreter` to Go using Goja (embedded JS runtime)
for expression evaluation. Pure logic — no UI dependencies. This is
**Checkpoint D**: the integration test must pass with `sum == 15`.

## Dependencies

- `github.com/dop251/goja` — JS runtime for eval/exec
- No dependency on cellbuf, drawutil, tealayout, or Bubbletea

## Blocked by

Nothing — can be built in parallel with UI steps.

## Blocks

- GRAIL-012-INTERP-UI (wires interpreter to the UI)

## File plan

```
internal/flowinterp/
├── interpreter.go      # Interpreter struct, New, Step, helpers
└── interpreter_test.go  # Integration + error tests
```

## Implementation details

### Interpreter struct

```go
type FlowNode struct {
    ID   int
    Type string   // "process", "decision", "terminal", "io", "connector"
    Text string
    Code string
}

type FlowEdge struct {
    FromID, ToID int
    Label        string
}

type Interpreter struct {
    nodes       []FlowNode
    edges       []FlowEdge
    vars        map[string]interface{}
    output      []string
    currentID   *int
    done        bool
    err         string
    waitInput   bool
    inputPrompt string
    inputVar    string
    stepCount   int
    maxSteps    int
    runtime     *goja.Runtime
}
```

Note: the interpreter uses its own `FlowNode`/`FlowEdge` types (simple
value types) rather than `graphmodel.Node`. This keeps the interpreter
decoupled from the generic graph package. The UI layer converts between
the two when creating an interpreter instance.

### Goja setup

```go
func New(nodes []FlowNode, edges []FlowEdge) *Interpreter {
    i := &Interpreter{...}
    i.runtime = goja.New()
    i.runtime.Set("print", func(call goja.FunctionCall) goja.Value {
        // append to i.output
    })
    i.runtime.Set("str", func(call goja.FunctionCall) goja.Value {
        // convert to string
    })
    return i
}
```

### Step function

Direct port of Python `FlowInterpreter.step()` from `grail.py:220-300`.
Key dispatch:

| Node type | Action |
|---|---|
| terminal | Append "PROGRAM END", set done=true |
| connector | Advance to next node |
| process | execStatements(code), advance |
| decision | evalExpr(code) → bool, follow Y or N edge |
| io | If input() call → set waitInput; if print() → exec, advance |

### execStatements

Split code on `;`. For each statement:
- If matches `varName = expr`: evaluate `expr` via Goja, store in vars
- Else: execute as JS via `runtime.RunString(stmt)`

Sync Go vars → Goja runtime before each eval.

### evalExpr

Sync vars → runtime, then `runtime.RunString(code)`, convert result to bool.

### Input handling

Pattern match: `input("prompt", varName)` or `read("prompt", varName)`.
Set `waitInput=true`, store prompt and varName. Next `Step()` call with
a non-nil `inputValue` stores the value and advances.

## Test cases

### Integration test (Checkpoint D)

Run the initial 7-node flowchart to completion:
- Create interpreter with `makeInitialNodes()` + `makeInitialEdges()`
- Call `Step()` repeatedly until `Done()` or step limit
- Assert: `vars["sum"] == 15`, `vars["i"] == 6`
- Assert: output contains "PROGRAM START", "Sum 1..5 = 15", "PROGRAM END"
- Assert: step count < 100

### Error tests

- No START node → `err == "NO START NODE"`
- Broken link (edge to non-existent node) → `err == "BROKEN LINK"`
- Max steps exceeded (infinite loop) → `err == "MAX STEPS EXCEEDED"`
- Invalid code → `err` contains "ERROR at"

### Input test

- I/O node with `input("Name?", name)` → waitInput=true, prompt="Name?"
- Call Step with "Alice" → vars["name"] == "Alice", waitInput=false

## Estimated effort

~200 lines of code, ~60 minutes.
