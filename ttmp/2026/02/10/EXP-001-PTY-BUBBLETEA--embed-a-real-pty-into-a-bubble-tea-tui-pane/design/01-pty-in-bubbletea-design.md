---
title: "PTY-in-BubbleTea Design"
ticket: EXP-001-PTY-BUBBLETEA
doc_type: design
status: active
intent: long-term
topics:
  - pty
  - bubbletea
  - tui
  - go
  - experiment
created: 2026-02-10
---

# PTY-in-BubbleTea Design

## Goal

Prove that a real program (e.g. `vi`) can be run inside a real PTY and its
output rendered into a Bubble Tea pane, with full keyboard forwarding—making it
possible to embed interactive terminal applications inside a Go TUI.

## Architecture

```
┌─────────────────────────────────────┐
│  Bubble Tea Program                 │
│  ┌───────────────────────────────┐  │
│  │  PTY pane (View)              │  │
│  │  vt10x virtual terminal       │  │
│  │  renders cell-by-cell          │  │
│  │  with lipgloss color mapping   │  │
│  └───────────────────────────────┘  │
│  ┌───────────────────────────────┐  │
│  │  Status bar                    │  │
│  └───────────────────────────────┘  │
│                                     │
│  KeyMsg → keyToBytes() → ptmx.Write│
│  ptmx.Read → vt10x.Write → View()  │
└─────────────────────────────────────┘
```

### Components

1. **PTY** (`creack/pty`): spawns subprocess in a pseudo-terminal with proper
   `TERM=xterm-256color` and window size.

2. **Virtual Terminal** (`hinshun/vt10x`): a headless VT100/xterm terminal
   emulator that parses ANSI escape sequences and maintains a cell grid
   (character + foreground + background per cell).

3. **Bubble Tea**: the TUI framework. Owns the real terminal, reads keystrokes,
   and renders views.

4. **lipgloss**: maps vt10x 256-color indices to styled terminal output.

### Data Flow

- **Input**: `tea.KeyMsg` → `keyToBytes()` translates to raw bytes → written
  to `ptmx` (PTY master).
- **Output**: goroutine reads `ptmx` → feeds bytes into `vt10x.Write()` →
  sends `ptyOutputMsg` back to Bubble Tea → `View()` renders the vt10x cell
  grid character-by-character with color.
- **Resize**: `tea.WindowSizeMsg` → `pty.Setsize()` + `vt10x.Resize()`.
- **Exit**: `cmd.Wait()` completes → `ptyExitMsg` → `tea.Quit`.

### Key Translation

Full mapping of Bubble Tea key types to raw terminal bytes including:
- All Ctrl+A through Ctrl+Z
- Arrow keys, Home, End, PgUp, PgDown, Delete
- Function keys F1-F12
- Escape, Tab, Enter, Space, Backspace
- Unicode runes (pass-through)

## Usage

```bash
# Default: runs vi
./pty-bubbletea

# Run vi on a specific file
./pty-bubbletea vi myfile.txt

# Run any interactive program
./pty-bubbletea htop
./pty-bubbletea bash
```

## Known Limitations

- **Performance**: rendering cell-by-cell with lipgloss per character is slow
  for large terminals. A production version would batch contiguous same-style
  cells into single lipgloss.Render calls.
- **vt10x fidelity**: `hinshun/vt10x` is a basic VT100 emulator. It may not
  handle all vim features (e.g., some SGR attributes, 24-bit color, mouse
  reporting).
- **Mouse**: mouse events from Bubble Tea are not forwarded to the PTY
  subprocess yet.
- **Bold/italic/underline**: vt10x `Glyph.Mode` flags for text attributes are
  not mapped to lipgloss styles yet.

## Future Directions

- Batch cell rendering for performance
- Map glyph mode flags (bold, italic, underline, reverse) to lipgloss
- Forward mouse events to the PTY
- Support splitting the terminal into multiple PTY panes
- Use a more complete terminal emulator (e.g., a Go port of libvte)
