#!/usr/bin/env bash
# test-insert-mode.sh — launch vi, enter insert mode, type text,
# save and quit, then verify the file was modified.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="$SCRIPT_DIR/pty-bubbletea/pty-bubbletea"
TEST_FILE="/tmp/pty-insert-test.txt"
SESSION="ptytest-insert"
COLS=100
ROWS=30

cleanup() {
  tmux kill-session -t "$SESSION" 2>/dev/null || true
}
trap cleanup EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }
pass() { echo "PASS: $*"; }

# --- build -----------------------------------------------------------------
echo ">>> Building binary..."
(cd "$SCRIPT_DIR/pty-bubbletea" && GOWORK=off go build -o pty-bubbletea .)
[[ -x "$BINARY" ]] || fail "binary not found"
pass "binary built"

# --- create test file ------------------------------------------------------
echo "original content" > "$TEST_FILE"

# --- launch in tmux --------------------------------------------------------
cleanup
echo ">>> Launching vi in tmux..."
tmux new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
  "$BINARY vi $TEST_FILE; echo \"=== EXITED rc=\$? ===\"; sleep 5"
sleep 2

# --- enter insert mode and type text --------------------------------------
echo ">>> Entering insert mode (o) and typing text..."
tmux send-keys -t "$SESSION" 'o'      # open new line below in vi
sleep 0.3
tmux send-keys -t "$SESSION" 'inserted by pty-bubbletea'
sleep 0.3

# Capture mid-edit
echo ">>> Mid-edit capture:"
tmux capture-pane -t "$SESSION" -p
echo "---"

# --- save and quit ---------------------------------------------------------
echo ">>> Escape then :wq to save and quit..."
tmux send-keys -t "$SESSION" Escape
sleep 0.3
tmux send-keys -t "$SESSION" ':wq' Enter
sleep 1

CAPTURE="$(tmux capture-pane -t "$SESSION" -p)"
echo "$CAPTURE"
echo "---"

echo "$CAPTURE" | grep -q "EXITED rc=0" \
  || fail "program did not exit cleanly"
pass "vi saved and exited rc=0"

# --- verify file content ---------------------------------------------------
echo ">>> File contents:"
cat "$TEST_FILE"
echo "---"

grep -q "inserted by pty-bubbletea" "$TEST_FILE" \
  || fail "inserted text not found in file"
pass "file contains inserted text"

echo ""
echo "=== ALL TESTS PASSED ==="
