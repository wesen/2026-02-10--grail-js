package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

// ---------------------------------------------------------------------------
// Focus targets
// ---------------------------------------------------------------------------

type focus int

const (
	focusPTY1 focus = iota
	focusPTY2
	focusWidget
	focusCount // sentinel for cycling
)

func (f focus) String() string {
	switch f {
	case focusPTY1:
		return "PTY-1"
	case focusPTY2:
		return "PTY-2"
	case focusWidget:
		return "Todo"
	}
	return "?"
}

// ---------------------------------------------------------------------------
// PTY pane
// ---------------------------------------------------------------------------

type ptyPane struct {
	id     int
	ptmx   *os.File
	vt     vt10x.Terminal
	cmd    *exec.Cmd
	width  int
	height int
	done   bool
	mu     sync.Mutex
}

// ptyOutputMsg carries the pane id so we know which reader to re-arm.
type ptyOutputMsg struct{ id int }

// ptyExitMsg signals a pane's subprocess exited.
type ptyExitMsg struct {
	id  int
	err error
}

func newPTYPane(id int, argv []string, cols, rows int) (*ptyPane, error) {
	vt := vt10x.New(vt10x.WithSize(cols, rows))

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		fmt.Sprintf("COLUMNS=%d", cols),
		fmt.Sprintf("LINES=%d", rows),
	)

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
	if err != nil {
		return nil, fmt.Errorf("pty pane %d: %w", id, err)
	}

	return &ptyPane{
		id:     id,
		ptmx:   ptmx,
		vt:     vt,
		cmd:    cmd,
		width:  cols,
		height: rows,
	}, nil
}

func (p *ptyPane) readCmd() tea.Cmd {
	id := p.id
	ptmx := p.ptmx
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := ptmx.Read(buf)
		if n > 0 {
			p.mu.Lock()
			p.vt.Write(buf[:n])
			p.mu.Unlock()
		}
		_ = err
		return ptyOutputMsg{id: id}
	}
}

func (p *ptyPane) waitCmd() tea.Cmd {
	id := p.id
	cmd := p.cmd
	return func() tea.Msg {
		err := cmd.Wait()
		return ptyExitMsg{id: id, err: err}
	}
}

func (p *ptyPane) resize(w, h int) {
	p.width = w
	p.height = h
	_ = pty.Setsize(p.ptmx, &pty.Winsize{
		Rows: uint16(h),
		Cols: uint16(w),
	})
	p.mu.Lock()
	p.vt.Resize(w, h)
	p.mu.Unlock()
}

func (p *ptyPane) render(focused bool) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	var sb strings.Builder
	for row := 0; row < p.height; row++ {
		for col := 0; col < p.width; col++ {
			g := p.vt.Cell(col, row)
			ch := g.Char
			if ch == 0 {
				ch = ' '
			}
			style := lipgloss.NewStyle()
			style = applyColor(style, g.FG, true)
			style = applyColor(style, g.BG, false)
			sb.WriteString(style.Render(string(ch)))
		}
		if row < p.height-1 {
			sb.WriteByte('\n')
		}
	}

	borderColor := "240"
	if focused {
		borderColor = "39" // bright cyan
	}
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Width(p.width).
		Height(p.height)

	label := fmt.Sprintf(" PTY-%d: %s ", p.id+1, p.cmd.Path)
	if p.done {
		label += "(exited) "
	}
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor)).Bold(true)

	return titleStyle.Render(label) + "\n" + border.Render(sb.String())
}

func (p *ptyPane) writeBytes(b []byte) {
	if !p.done {
		p.ptmx.Write(b)
	}
}

func (p *ptyPane) close() {
	p.done = true
	p.ptmx.Close()
}

// ---------------------------------------------------------------------------
// Todo widget
// ---------------------------------------------------------------------------

type todoItem struct {
	text string
	done bool
}

type todoWidget struct {
	items    []todoItem
	cursor   int
	input    string
	editing  bool // true when typing a new item
	width    int
	height   int
}

func newTodoWidget() *todoWidget {
	return &todoWidget{
		items: []todoItem{
			{text: "Try typing in the editors", done: false},
			{text: "Switch panes with Ctrl+A, <1/2/3>", done: false},
			{text: "Add a todo item here (press 'a')", done: false},
		},
	}
}

func (t *todoWidget) handleKey(msg tea.KeyMsg) {
	if t.editing {
		switch msg.Type {
		case tea.KeyEnter:
			if strings.TrimSpace(t.input) != "" {
				t.items = append(t.items, todoItem{text: t.input})
			}
			t.input = ""
			t.editing = false
		case tea.KeyEscape:
			t.input = ""
			t.editing = false
		case tea.KeyBackspace:
			if len(t.input) > 0 {
				t.input = t.input[:len(t.input)-1]
			}
		case tea.KeySpace:
			t.input += " "
		case tea.KeyRunes:
			t.input += msg.String()
		}
		return
	}

	switch msg.String() {
	case "j", "down":
		if t.cursor < len(t.items)-1 {
			t.cursor++
		}
	case "k", "up":
		if t.cursor > 0 {
			t.cursor--
		}
	case "x", " ":
		if len(t.items) > 0 {
			t.items[t.cursor].done = !t.items[t.cursor].done
		}
	case "d":
		if len(t.items) > 0 {
			t.items = append(t.items[:t.cursor], t.items[t.cursor+1:]...)
			if t.cursor >= len(t.items) && t.cursor > 0 {
				t.cursor--
			}
		}
	case "a":
		t.editing = true
		t.input = ""
	}
}

func (t *todoWidget) render(focused bool) string {
	borderColor := "240"
	if focused {
		borderColor = "39"
	}

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(borderColor)).Bold(true)
	title := titleStyle.Render(" Todo List ")

	var sb strings.Builder

	// Items
	for i, item := range t.items {
		prefix := "  "
		if focused && i == t.cursor {
			prefix = "> "
		}
		check := "[ ]"
		if item.done {
			check = "[x]"
		}
		itemStyle := lipgloss.NewStyle()
		if item.done {
			itemStyle = itemStyle.Foreground(lipgloss.Color("242")).Strikethrough(true)
		}
		if focused && i == t.cursor {
			itemStyle = itemStyle.Bold(true).Foreground(lipgloss.Color("39"))
		}
		line := fmt.Sprintf("%s%s %s", prefix, check, itemStyle.Render(item.text))
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	// Input line
	if t.editing {
		cursor := "█"
		inputLine := fmt.Sprintf("  New: %s%s", t.input, cursor)
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(inputLine))
		sb.WriteByte('\n')
	}

	// Help
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	if focused {
		if t.editing {
			sb.WriteString(helpStyle.Render("  Enter=add  Esc=cancel"))
		} else {
			sb.WriteString(helpStyle.Render("  j/k=move  x=toggle  d=delete  a=add"))
		}
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Width(t.width).
		Height(t.height)

	return title + "\n" + border.Render(sb.String())
}

// ---------------------------------------------------------------------------
// Root model
// ---------------------------------------------------------------------------

type rootModel struct {
	panes       [2]*ptyPane
	todo        *todoWidget
	focus       focus
	prefixMode  bool // true after Ctrl+A pressed, waiting for next key
	totalWidth  int
	totalHeight int
}

func (m *rootModel) Init() tea.Cmd {
	return tea.Batch(
		m.panes[0].readCmd(),
		m.panes[0].waitCmd(),
		m.panes[1].readCmd(),
		m.panes[1].waitCmd(),
	)
}

func (m *rootModel) layout() {
	if m.totalWidth < 4 || m.totalHeight < 4 {
		return
	}

	// Layout:
	//   Row 1: two PTY panes side by side  (takes ~70% of height)
	//   Row 2: todo widget                  (takes the rest)
	//   Row 3: status bar (1 line)

	statusH := 1
	todoH := 8 // fixed height for todo
	ptyH := m.totalHeight - todoH - statusH - 6 // borders eat ~6 lines (title+border top/bottom for each row)
	if ptyH < 4 {
		ptyH = 4
	}

	// Each PTY pane gets half the width minus the border chrome (3 per pane for border L/R + gap)
	paneW := (m.totalWidth - 3) / 2 // 1 col gap between panes
	if paneW < 10 {
		paneW = 10
	}

	m.panes[0].resize(paneW, ptyH)
	m.panes[1].resize(paneW, ptyH)

	m.todo.width = m.totalWidth - 2 // border L/R
	m.todo.height = todoH
}

func (m *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.totalWidth = msg.Width
		m.totalHeight = msg.Height
		m.layout()
		return m, nil

	case ptyOutputMsg:
		p := m.panes[msg.id]
		if p.done {
			return m, nil
		}
		return m, p.readCmd()

	case ptyExitMsg:
		m.panes[msg.id].close()
		// Quit only if both panes are done
		if m.panes[0].done && m.panes[1].done {
			return m, tea.Quit
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *rootModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// --- Prefix mode: Ctrl+A was pressed, interpret next key as command ---
	if m.prefixMode {
		m.prefixMode = false
		switch msg.String() {
		case "1":
			m.focus = focusPTY1
			return m, nil
		case "2":
			m.focus = focusPTY2
			return m, nil
		case "3":
			m.focus = focusWidget
			return m, nil
		case "tab":
			m.focus = (m.focus + 1) % focusCount
			return m, nil
		case "ctrl+a":
			// Ctrl+A, Ctrl+A → send literal Ctrl+A to focused PTY
			if m.focus == focusPTY1 || m.focus == focusPTY2 {
				idx := 0
				if m.focus == focusPTY2 {
					idx = 1
				}
				m.panes[idx].writeBytes([]byte{0x01})
			}
			return m, nil
		case "q":
			// Ctrl+A, q → quit the whole app
			for _, p := range m.panes {
				if !p.done {
					p.close()
				}
			}
			return m, tea.Quit
		}
		// Unknown prefix command — ignore
		return m, nil
	}

	// --- Ctrl+A enters prefix mode ---
	if msg.Type == tea.KeyCtrlA {
		m.prefixMode = true
		return m, nil
	}

	// --- Dispatch to focused target ---
	switch m.focus {
	case focusPTY1:
		m.panes[0].writeBytes(keyToBytes(msg))
	case focusPTY2:
		m.panes[1].writeBytes(keyToBytes(msg))
	case focusWidget:
		m.todo.handleKey(msg)
	}

	return m, nil
}

func (m *rootModel) View() string {
	// Render two PTY panes side by side
	left := m.panes[0].render(m.focus == focusPTY1)
	right := m.panes[1].render(m.focus == focusPTY2)

	paneRow := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)

	// Render todo widget
	todoRow := m.todo.render(m.focus == focusWidget)

	// Status bar
	prefixIndicator := ""
	if m.prefixMode {
		prefixIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true).
			Render(" [Ctrl+A] ")
	}

	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("255")).
		Width(m.totalWidth)

	status := statusStyle.Render(fmt.Sprintf(
		" Focus: %s%s │ Ctrl+A,1/2/3=switch │ Ctrl+A,Tab=cycle │ Ctrl+A,q=quit",
		m.focus, prefixIndicator,
	))

	return paneRow + "\n" + todoRow + "\n" + status
}

// ---------------------------------------------------------------------------
// Color helper
// ---------------------------------------------------------------------------

func applyColor(style lipgloss.Style, c vt10x.Color, isFg bool) lipgloss.Style {
	if c == vt10x.DefaultFG || c == vt10x.DefaultBG {
		return style
	}
	lc := lipgloss.Color(fmt.Sprintf("%d", c))
	if isFg {
		return style.Foreground(lc)
	}
	return style.Background(lc)
}

// ---------------------------------------------------------------------------
// Key translation (same as before)
// ---------------------------------------------------------------------------

func keyToBytes(msg tea.KeyMsg) []byte {
	switch msg.Type {
	case tea.KeyCtrlA:
		return []byte{0x01}
	case tea.KeyCtrlB:
		return []byte{0x02}
	case tea.KeyCtrlC:
		return []byte{0x03}
	case tea.KeyCtrlD:
		return []byte{0x04}
	case tea.KeyCtrlE:
		return []byte{0x05}
	case tea.KeyCtrlF:
		return []byte{0x06}
	case tea.KeyCtrlG:
		return []byte{0x07}
	case tea.KeyCtrlH:
		return []byte{0x08}
	case tea.KeyTab:
		return []byte{0x09}
	case tea.KeyCtrlJ:
		return []byte{0x0a}
	case tea.KeyCtrlK:
		return []byte{0x0b}
	case tea.KeyCtrlL:
		return []byte{0x0c}
	case tea.KeyEnter:
		return []byte{0x0d}
	case tea.KeyCtrlN:
		return []byte{0x0e}
	case tea.KeyCtrlO:
		return []byte{0x0f}
	case tea.KeyCtrlP:
		return []byte{0x10}
	case tea.KeyCtrlQ:
		return []byte{0x11}
	case tea.KeyCtrlR:
		return []byte{0x12}
	case tea.KeyCtrlS:
		return []byte{0x13}
	case tea.KeyCtrlT:
		return []byte{0x14}
	case tea.KeyCtrlU:
		return []byte{0x15}
	case tea.KeyCtrlV:
		return []byte{0x16}
	case tea.KeyCtrlW:
		return []byte{0x17}
	case tea.KeyCtrlX:
		return []byte{0x18}
	case tea.KeyCtrlY:
		return []byte{0x19}
	case tea.KeyCtrlZ:
		return []byte{0x1a}
	case tea.KeyEscape:
		return []byte{0x1b}
	case tea.KeyBackspace:
		return []byte{0x7f}
	case tea.KeyUp:
		return []byte("\x1b[A")
	case tea.KeyDown:
		return []byte("\x1b[B")
	case tea.KeyRight:
		return []byte("\x1b[C")
	case tea.KeyLeft:
		return []byte("\x1b[D")
	case tea.KeyHome:
		return []byte("\x1b[H")
	case tea.KeyEnd:
		return []byte("\x1b[F")
	case tea.KeyPgUp:
		return []byte("\x1b[5~")
	case tea.KeyPgDown:
		return []byte("\x1b[6~")
	case tea.KeyDelete:
		return []byte("\x1b[3~")
	case tea.KeyF1:
		return []byte("\x1bOP")
	case tea.KeyF2:
		return []byte("\x1bOQ")
	case tea.KeyF3:
		return []byte("\x1bOR")
	case tea.KeyF4:
		return []byte("\x1bOS")
	case tea.KeyF5:
		return []byte("\x1b[15~")
	case tea.KeyF6:
		return []byte("\x1b[17~")
	case tea.KeyF7:
		return []byte("\x1b[18~")
	case tea.KeyF8:
		return []byte("\x1b[19~")
	case tea.KeyF9:
		return []byte("\x1b[20~")
	case tea.KeyF10:
		return []byte("\x1b[21~")
	case tea.KeyF11:
		return []byte("\x1b[23~")
	case tea.KeyF12:
		return []byte("\x1b[24~")
	case tea.KeySpace:
		return []byte{' '}
	case tea.KeyRunes:
		s := msg.String()
		if utf8.ValidString(s) {
			return []byte(s)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	// Create two test files
	for i, content := range []string{
		"=== File 1 ===\nEdit me in the LEFT pane.\nLine 3\nLine 4\nLine 5\n",
		"=== File 2 ===\nEdit me in the RIGHT pane.\nLine 3\nLine 4\nLine 5\n",
	} {
		f := fmt.Sprintf("/tmp/pty-test-%d.txt", i+1)
		os.WriteFile(f, []byte(content), 0644)
	}

	argv1 := []string{"vi", "/tmp/pty-test-1.txt"}
	argv2 := []string{"vi", "/tmp/pty-test-2.txt"}

	// Allow overriding via args: program [-- cmd1 args -- cmd2 args]
	if len(os.Args) > 1 {
		parts := splitArgs(os.Args[1:])
		if len(parts) >= 1 {
			argv1 = parts[0]
		}
		if len(parts) >= 2 {
			argv2 = parts[1]
		}
	}

	cols, rows := 40, 15
	pane1, err := newPTYPane(0, argv1, cols, rows)
	if err != nil {
		log.Fatal(err)
	}
	pane2, err := newPTYPane(1, argv2, cols, rows)
	if err != nil {
		log.Fatal(err)
	}

	root := &rootModel{
		panes: [2]*ptyPane{pane1, pane2},
		todo:  newTodoWidget(),
		focus: focusPTY1,
	}

	p := tea.NewProgram(
		root,
		tea.WithAltScreen(),
		tea.WithInputTTY(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// splitArgs splits os.Args on "--" into multiple command arg slices.
func splitArgs(args []string) [][]string {
	var result [][]string
	var current []string
	for _, a := range args {
		if a == "--" {
			if len(current) > 0 {
				result = append(result, current)
			}
			current = nil
			continue
		}
		current = append(current, a)
	}
	if len(current) > 0 {
		result = append(result, current)
	}
	return result
}
