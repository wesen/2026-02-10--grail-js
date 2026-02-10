package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

// model holds the Bubble Tea application state.
type model struct {
	ptmx   *os.File       // PTY file descriptor
	vt     vt10x.Terminal  // Virtual terminal that interprets ANSI sequences
	width  int
	height int
	cmd    *exec.Cmd
	done   bool
	mu     sync.Mutex
}

// ptyOutputMsg is sent when new output is available from the PTY.
type ptyOutputMsg struct{}

// ptyExitMsg is sent when the subprocess exits.
type ptyExitMsg struct{ err error }

func initialModel(argv []string) (*model, error) {
	cols, rows := 80, 24

	vt := vt10x.New(vt10x.WithSize(cols, rows))

	name := argv[0]
	args := argv[1:]
	cmd := exec.Command(name, args...)
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
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}

	m := &model{
		ptmx:   ptmx,
		vt:     vt,
		width:  cols,
		height: rows,
		cmd:    cmd,
	}

	return m, nil
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.readPTY(),
		m.waitForExit(),
	)
}

// readPTY reads a chunk from the PTY and feeds it into the virtual terminal.
func (m *model) readPTY() tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 4096)
		n, err := m.ptmx.Read(buf)
		if n > 0 {
			m.mu.Lock()
			m.vt.Write(buf[:n])
			m.mu.Unlock()
		}
		if err != nil {
			return ptyOutputMsg{} // will stop on ptyExitMsg
		}
		return ptyOutputMsg{}
	}
}

// waitForExit waits for the subprocess to finish.
func (m *model) waitForExit() tea.Cmd {
	return func() tea.Msg {
		err := m.cmd.Wait()
		return ptyExitMsg{err: err}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		input := keyToBytes(msg)
		if input != nil {
			m.ptmx.Write(input)
		}
		return m, nil

	case ptyOutputMsg:
		if m.done {
			return m, nil
		}
		return m, m.readPTY()

	case ptyExitMsg:
		m.done = true
		m.ptmx.Close()
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height - 1 // reserve 1 line for status bar
		if m.height < 1 {
			m.height = 1
		}
		pty.Setsize(m.ptmx, &pty.Winsize{
			Rows: uint16(m.height),
			Cols: uint16(m.width),
		})
		m.mu.Lock()
		m.vt.Resize(m.width, m.height)
		m.mu.Unlock()
		return m, nil
	}

	return m, nil
}

func (m *model) View() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var screen string

	for row := 0; row < m.height; row++ {
		line := ""
		for col := 0; col < m.width; col++ {
			g := m.vt.Cell(col, row)
			ch := g.Char
			if ch == 0 {
				ch = ' '
			}
			cell := string(ch)
			style := lipgloss.NewStyle()
			style = applyColor(style, g.FG, true)
			style = applyColor(style, g.BG, false)
			line += style.Render(cell)
		}
		if row < m.height-1 {
			line += "\n"
		}
		screen += line
	}

	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("255")).
		Width(m.width)
	status := statusStyle.Render(fmt.Sprintf(" PTY: %s | %dx%d", m.cmd.Path, m.width, m.height))

	return screen + "\n" + status
}

// applyColor converts vt10x color to lipgloss styling.
func applyColor(style lipgloss.Style, c vt10x.Color, isFg bool) lipgloss.Style {
	if c == vt10x.DefaultFG || c == vt10x.DefaultBG {
		return style
	}
	colorStr := fmt.Sprintf("%d", c)
	lc := lipgloss.Color(colorStr)
	if isFg {
		return style.Foreground(lc)
	}
	return style.Background(lc)
}

// keyToBytes translates a Bubble Tea KeyMsg into bytes to send to the PTY.
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

	// Fallback
	s := msg.String()
	if len(s) > 0 && msg.Type == tea.KeyRunes {
		return []byte(s)
	}
	return nil
}

func main() {
	argv := []string{"vi"}
	if len(os.Args) > 1 {
		argv = os.Args[1:]
	}

	m, err := initialModel(argv)
	if err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithInputTTY(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
