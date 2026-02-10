---
title: "Implementation Plan — Interpreter UI"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-012-INTERP-UI
topics:
  - interpreter
  - bubbletea
  - go
---

# Implementation Plan — Interpreter UI

## Overview

Wire the flow interpreter to the UI: toolbar run state, console output,
live variables, execution highlighting, auto-step timer, and program
input field.

## Dependencies

- `internal/flowinterp` (GRAIL-011) — the interpreter engine
- Side panel (GRAIL-009) — variables/console layers to populate
- Mouse interaction (GRAIL-010) — keyboard routing context

## Blocked by

- GRAIL-009-PANEL, GRAIL-010-MOUSE, GRAIL-011-INTERPRETER

## Blocks

- GRAIL-013-EDIT-MODAL (modal needs running state awareness)

## What this changes

```
internal/grailui/
├── update.go    # ADD: startProgram, stepProgram, autoRun, pause, stop
├── model.go     # ADD: interp, execNodeID, running, autoRunning fields
├── view.go      # CHANGE: wire live data to panel layers
├── layers.go    # ADD: buildInputOverlayLayer; CHANGE: buildVars/Console with live data
└── keys.go      # ADD: r/n/g/p/x key handlers, FocusInput routing
```

## Implementation details

### State fields on Model

```go
interp      *flowinterp.Interpreter
execNodeID  *int          // current execution node (yellow highlight)
running     bool          // interpreter is active
autoRunning bool          // auto-stepping with timer
waitInput   bool          // waiting for user input
speed       time.Duration // auto-step interval (400ms)
```

### syncInterpreter

After each `Step()`, copy interpreter state to model:
- `m.execNodeID = interp.CurrentID()`
- `m.consoleLines = interp.Output()`
- `m.variables = interp.Vars()`
- `m.waitInput = interp.WaitingInput()`
- If `waitInput` → switch focus to input field, stop auto
- If `done` or `err` → stop auto, append error to console if present

### Keyboard actions

| Key | Action | Guard |
|---|---|---|
| `r` | Create interpreter, step once | not running |
| `n` | Step once | running, not done, not waiting |
| `g` | Start auto-run (set autoRunning, return tickCmd) | running |
| `p` | Pause (set autoRunning=false) | auto-running |
| `x` | Stop (clear interpreter, exec highlight, running state) | running |

### TickMsg auto-step

```go
case TickMsg:
    if m.autoRunning && m.interp != nil {
        if !m.interp.Done() && m.interp.Err() == "" && !m.interp.WaitingInput() {
            m.interp.Step(nil)
            syncInterpreter(&m)
            return m, tickCmd(m.speed)
        }
        m.autoRunning = false
    }
```

### Input overlay layer

When `waitInput == true`, show a text input field over the console area:
- `bubbles/v2/textinput` sub-model
- Layer at Z=10, positioned over the console panel area
- Focus routes to textinput (FocusInput)
- Enter submits value → `interp.Step(&value)` → sync → hide input
- After input, focus returns to canvas

### Toolbar updates

When running, toolbar shows:
- "▶ AUTO" or "⏸ READY" tag in yellow
- `[n]STEP [g]GO [p]PAUSE [x]STOP` controls in cyan

### Execution highlighting

In `buildNodeLayer`, check `isExec = (m.execNodeID != nil && *m.execNodeID == node.ID)`.
If true, use yellow border + text styles instead of normal node colors.

### Variables panel (live)

Wire `buildVarsPanelLayer(m.variables)` — now displays actual interpreter
variables with proper formatting (gold names, green numbers, blue strings).

### Console panel (live)

Wire `buildConsolePanelLayer(m.consoleLines)` — shows last N lines of
output, styled by prefix (errors red, separators dim, input yellow, output green).

## Visual validation

- `r` → "PROGRAM START" in console, START node highlights yellow
- `n` → steps through, INIT highlights, variables show i=1 sum=0
- `g` → auto-runs, nodes highlight in sequence, variables update
- `p` → pauses mid-execution
- `x` → clears everything
- Full run → console shows "Sum 1..5 = 15"

## Estimated effort

~100 lines of code, ~45 minutes.
