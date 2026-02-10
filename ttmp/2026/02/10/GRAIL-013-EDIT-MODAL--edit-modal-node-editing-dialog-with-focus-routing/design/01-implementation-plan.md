---
title: "Implementation Plan — Edit Modal"
doc_type: implementation-plan
status: active
intent: long-term
ticket: GRAIL-013-EDIT-MODAL
topics:
  - bubbletea
  - lipgloss-v2
  - go
  - layout
---

# Implementation Plan — Edit Modal

## Overview

Press `e` on a selected node → centered modal dialog at Z=100 with label
and code fields. Tab switches fields. Enter saves. Esc cancels. Uses
`tealayout.ModalLayer` for positioning and Bubbletea v2 `textinput` for
the form fields.

## Dependencies

- `pkg/tealayout` (GRAIL-006) — ModalLayer for centering
- `bubbles/v2/textinput` — input fields
- Mouse interaction (GRAIL-010) — selection state, focus routing

## Blocked by

- GRAIL-006-TEALAYOUT, GRAIL-010-MOUSE

## Blocks

Nothing — this is the final step.

## What this adds

```
internal/grailui/
├── layers.go    # ADD: buildEditModalLayer
├── keys.go      # ADD: handleEditKeys
└── update.go    # ADD: openEditModal, commitEdit, cancelEdit
```

## Implementation details

### Model fields

```go
editOpen     bool
editNodeID   int
editLabel    textinput.Model    // label field
editCode     textinput.Model    // code field
focus        FocusTarget        // FocusEditLabel or FocusEditCode when editing
```

### Opening the modal

When `e` is pressed and `selectedID != nil`:
1. Set `editOpen = true`, `editNodeID = *selectedID`
2. Initialize `editLabel.SetValue(node.Text)`, `editCode.SetValue(node.Code)`
3. Focus label field: `editLabel.Focus()`, `focus = FocusEditLabel`

### buildEditModalLayer

```go
func buildEditModalLayer(m Model) lipgloss.Layer {
    node := nodeByID(m.editNodeID)
    info := nodeTypeInfo[node.Type]
    
    content := lipgloss.JoinVertical(lipgloss.Left,
        titleStyle.Render("✏️  EDIT — " + strings.ToUpper(info.Label)),
        "",
        labelStyle.Render("Label:"),
        m.editLabel.View(),
        "",
        labelStyle.Render("Code" + hintForType(node.Type) + ":"),
        m.editCode.View(),
        "",
        hintStyle.Render("[tab] switch  [enter] save  [esc] cancel"),
    )
    
    boxStyle := lipgloss.NewStyle().
        Border(lipgloss.NormalBorder()).
        BorderForeground(color("#00d4a0")).
        Background(color("#0a1510")).
        Width(50).Padding(1, 2)
    
    return tealayout.ModalLayer(content, m.termW, m.termH, boxStyle)
}
```

### Code hints per type

| Type | Hint |
|---|---|
| process | "(statements separated by ;)" |
| decision | "(boolean expression)" |
| io | `(print("...") or input("prompt", var))` |
| terminal | (none) |
| connector | (none) |

### Focus routing: handleEditKeys

```
tab/shift+tab → toggle FocusEditLabel ↔ FocusEditCode
    Blur one field, Focus the other
enter → commitEdit:
    node.Text = strings.ToUpper(editLabel.Value())
    node.Code = editCode.Value()
    editOpen = false, focus = FocusCanvas
esc → cancelEdit:
    editOpen = false, focus = FocusCanvas
other keys → forward to active textinput
```

### Click on modal

`Canvas.Hit()` returns the modal layer (ID="modal") for clicks inside it.
Handle by doing nothing (don't deselect). Clicks outside the modal could
either close it or be ignored — match Python behavior: ignore (modal is
dismissed only by Enter or Esc).

## Visual validation

- `e` on selected node → centered box appears over canvas
- Label field shows current node text, code field shows current code
- Tab switches between fields (cursor moves)
- Typing updates the active field
- Enter saves and closes — node text updates on canvas
- Esc closes without saving

## Estimated effort

~60 lines of code, ~30 minutes.
