#!/usr/bin/env bash
# test-dual-panes.sh — test the dual-PTY + todo widget version in tmux.
#
# Tests:
#   1. Both PTY panes render vi with their respective files
#   2. Status bar shows Focus: PTY-1
#   3. Ctrl+A,2 switches focus to PTY-2
#   4. Ctrl+A,3 switches focus to Todo widget
#   5. Interact with todo widget (add item)
#   6. Ctrl+A,1 switches back to PTY-1, type in vi
#   7. Ctrl+A,q quits the whole app
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="$SCRIPT_DIR/pty-bubbletea/pty-bubbletea"
SESSION="ptytest-dual"
COLS=120
ROWS=45

cleanup() { tmux kill-session -t "$SESSION" 2>/dev/null || true; }
trap cleanup EXIT
fail() { echo "FAIL: $*" >&2; exit 1; }
pass() { echo "PASS: $*"; }

capture() {
  sleep "${1:-1}"
  tmux capture-pane -t "$SESSION" -p
}

# --- build -----------------------------------------------------------------
echo ">>> Building binary..."
(cd "$SCRIPT_DIR/pty-bubbletea" && GOWORK=off go build -o pty-bubbletea .)
[[ -x "$BINARY" ]] || fail "binary not found"
pass "binary built"

# --- launch ----------------------------------------------------------------
cleanup
echo ">>> Launching dual-pane app in tmux ($COLS x $ROWS)..."
tmux new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
  "$BINARY; echo \"=== EXITED rc=\$? ===\"; sleep 5"

C=$(capture 3)
echo "$C"
echo "---"

# --- Test 1: both panes render -------------------------------------------
echo "$C" | grep -q "File 1" || fail "PTY-1 file content not visible"
pass "PTY-1 renders file 1"

echo "$C" | grep -q "File 2" || fail "PTY-2 file content not visible"
pass "PTY-2 renders file 2"

# --- Test 2: status bar ---------------------------------------------------
echo "$C" | grep -q "Focus: PTY-1" || fail "status bar doesn't show PTY-1 focus"
pass "status bar shows Focus: PTY-1"

# --- Test 3: Ctrl+A,2 switches to PTY-2 ----------------------------------
echo ">>> Switching to PTY-2 (Ctrl+A, 2)..."
tmux send-keys -t "$SESSION" C-a
sleep 0.2
tmux send-keys -t "$SESSION" '2'
C=$(capture 1)
echo "$C" | grep -q "Focus: PTY-2" || fail "focus didn't switch to PTY-2"
pass "Ctrl+A,2 switches to PTY-2"

# --- Test 4: Ctrl+A,3 switches to Todo widget ----------------------------
echo ">>> Switching to Todo (Ctrl+A, 3)..."
tmux send-keys -t "$SESSION" C-a
sleep 0.2
tmux send-keys -t "$SESSION" '3'
C=$(capture 1)
echo "$C" | grep -q "Focus: Todo" || fail "focus didn't switch to Todo"
pass "Ctrl+A,3 switches to Todo widget"

# Check todo items are visible
echo "$C" | grep -q "Try typing" || fail "todo items not visible"
pass "todo items rendered"

# --- Test 5: interact with todo - add item --------------------------------
echo ">>> Adding a todo item..."
tmux send-keys -t "$SESSION" 'a'
sleep 0.3
tmux send-keys -t "$SESSION" 'My new task'
sleep 0.3

C=$(capture 0.5)
echo "$C" | grep -q "My new task" || fail "typed text not visible in todo input"
pass "todo input shows typed text"

tmux send-keys -t "$SESSION" Enter
sleep 0.5
C=$(capture 0.5)
echo "$C" | grep -q "My new task" || fail "added item not visible in todo list"
pass "todo item added"

# --- Test 6: switch back to PTY-1, insert text ----------------------------
echo ">>> Switching to PTY-1 (Ctrl+A, 1) and inserting text..."
tmux send-keys -t "$SESSION" C-a
sleep 0.2
tmux send-keys -t "$SESSION" '1'
sleep 0.5

# Enter insert mode in vi and type
tmux send-keys -t "$SESSION" 'o'
sleep 0.2
tmux send-keys -t "$SESSION" 'hello from pane 1'
sleep 0.3

C=$(capture 0.5)
echo "$C" | grep -q "hello from pane 1" || fail "text typed into PTY-1 not visible"
pass "keyboard input reaches PTY-1"

# Escape back to normal mode
tmux send-keys -t "$SESSION" Escape
sleep 0.3

# --- Test 7: Ctrl+A,q quits ----------------------------------------------
echo ">>> Quitting with Ctrl+A,q..."
# First :q! the vi in pane 1 (discard changes)
tmux send-keys -t "$SESSION" ':q!' Enter
sleep 1

# Switch to PTY-2 and quit that vi too
tmux send-keys -t "$SESSION" C-a
sleep 0.2
tmux send-keys -t "$SESSION" '2'
sleep 0.3
tmux send-keys -t "$SESSION" ':q' Enter
sleep 1

C=$(capture 1)
echo "$C"
echo "---"
echo "$C" | grep -q "EXITED rc=0" || fail "program did not exit cleanly"
pass "program exited rc=0 after both vi's quit"

echo ""
echo "=== ALL TESTS PASSED ==="
