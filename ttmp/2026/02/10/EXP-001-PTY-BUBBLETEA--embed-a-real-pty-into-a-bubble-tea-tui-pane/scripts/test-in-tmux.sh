#!/usr/bin/env bash
# test-in-tmux.sh — build the binary, launch it in a tmux session,
# verify vi renders, send :q to quit, and check clean exit.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="$SCRIPT_DIR/pty-bubbletea/pty-bubbletea"
TEST_FILE="/tmp/pty-test.txt"
SESSION="ptytest"
COLS=100
ROWS=30

# --- helpers ---------------------------------------------------------------
cleanup() {
  tmux kill-session -t "$SESSION" 2>/dev/null || true
}
trap cleanup EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }
pass() { echo "PASS: $*"; }

# --- build -----------------------------------------------------------------
echo ">>> Building binary..."
(cd "$SCRIPT_DIR/pty-bubbletea" && GOWORK=off go build -o pty-bubbletea .)
[[ -x "$BINARY" ]] || fail "binary not found at $BINARY"
pass "binary built"

# --- create test file ------------------------------------------------------
cat > "$TEST_FILE" <<'EOF'
Hello from pty-bubbletea!
This is a test file opened in vi.
Line 3
Line 4
Line 5
EOF

# --- launch in tmux --------------------------------------------------------
cleanup  # kill any leftover session
echo ">>> Launching in tmux ($COLS x $ROWS)..."
tmux new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
  "$BINARY vi $TEST_FILE; echo \"=== EXITED rc=\$? ===\"; sleep 5"
sleep 2

# --- capture and verify vi rendered ----------------------------------------
echo ">>> Capturing pane after launch..."
CAPTURE="$(tmux capture-pane -t "$SESSION" -p)"
echo "$CAPTURE"
echo "---"

echo "$CAPTURE" | grep -q "Hello from pty-bubbletea" \
  || fail "file contents not visible in vi"
pass "vi rendered file contents"

echo "$CAPTURE" | grep -q "PTY:" \
  || fail "status bar not visible"
pass "status bar visible"

echo "$CAPTURE" | grep -q '~' \
  || fail "vi tilde lines not visible"
pass "vi tilde lines visible"

# --- test keyboard: send :q to quit vi ------------------------------------
echo ">>> Sending Escape :q Enter to quit vi..."
tmux send-keys -t "$SESSION" Escape
sleep 0.3
tmux send-keys -t "$SESSION" ':q' Enter
sleep 1

CAPTURE2="$(tmux capture-pane -t "$SESSION" -p)"
echo "$CAPTURE2"
echo "---"

echo "$CAPTURE2" | grep -q "EXITED rc=0" \
  || fail "program did not exit cleanly"
pass "vi quit and program exited rc=0"

echo ""
echo "=== ALL TESTS PASSED ==="
