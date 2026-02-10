#!/usr/bin/env python3
"""GRaIL — Graphical Representation and Interpretation Language

Terminal flowchart editor + interpreter built with Textual.
Ported from reference.jsx (React/SVG).

Usage:
    python3 grail.py

Keys:
    s/a/c       Select / Add / Connect mode
    1-5         Choose node type (in Add mode)
    e           Edit selected node
    Delete/d    Delete selected node
    r           Run program
    n           Step
    g           Auto-run
    p           Pause
    x           Stop
    Arrow keys  Pan canvas
    q           Quit

Mouse:
    Click       Select node
    Drag        Move node
    Click+Click Connect nodes (in Connect mode)
    Click empty Add node (in Add mode)
"""

from __future__ import annotations

import math
import re
from dataclasses import dataclass
from typing import Optional, Any

from textual.app import App, ComposeResult
from textual.screen import ModalScreen
from textual.widget import Widget
from textual.widgets import Static, Input, Button, Label, Footer
from textual.containers import Horizontal, Vertical, VerticalScroll
from textual.strip import Strip
from textual.binding import Binding
from textual import events, on

from rich.segment import Segment
from rich.style import Style
from rich.text import Text


# ═══════════════════════════ STYLES ═══════════════════════════

S = Style

BG = S(color="#1a3a2a", bgcolor="#080e0b")
GRID = S(color="#0e2e20", bgcolor="#080e0b")

NODE_COLORS = {
    "process":   {"border": S(color="#00d4a0", bgcolor="#080e0b"),
                  "text":   S(color="#00ffc8", bgcolor="#080e0b", bold=True),
                  "tag": "P"},
    "decision":  {"border": S(color="#00ccee", bgcolor="#080e0b"),
                  "text":   S(color="#66ffee", bgcolor="#080e0b", bold=True),
                  "tag": "?"},
    "terminal":  {"border": S(color="#44ff88", bgcolor="#080e0b"),
                  "text":   S(color="#88ffbb", bgcolor="#080e0b", bold=True),
                  "tag": "T"},
    "io":        {"border": S(color="#ddaa44", bgcolor="#080e0b"),
                  "text":   S(color="#ffcc66", bgcolor="#080e0b", bold=True),
                  "tag": "IO"},
    "connector": {"border": S(color="#1a6a4a", bgcolor="#080e0b"),
                  "text":   S(color="#00d4a0", bgcolor="#080e0b", bold=True),
                  "tag": ""},
}

SEL_BORDER = S(color="#00ffee", bgcolor="#0a1a15", bold=True)
SEL_TEXT = S(color="#00ffee", bgcolor="#0a1a15", bold=True)
EXEC_BORDER = S(color="#ffcc00", bgcolor="#12120a", bold=True)
EXEC_TEXT = S(color="#ffee66", bgcolor="#12120a", bold=True)

EDGE_S = S(color="#00d4a0", bgcolor="#080e0b")
EDGE_ACT = S(color="#ffcc00", bgcolor="#080e0b", bold=True)
EDGE_LBL = S(color="#00ffc8", bgcolor="#080e0b", bold=True)
CONN_S = S(color="#00ffc8", bgcolor="#080e0b")


# ═══════════════════════════ DATA MODEL ═══════════════════════════

@dataclass
class NodeType:
    label: str
    w: int
    h: int


NODE_TYPES: dict[str, NodeType] = {
    "process":   NodeType("Process",   22, 3),
    "decision":  NodeType("Decision",  22, 3),
    "terminal":  NodeType("Terminal",  22, 3),
    "io":        NodeType("I/O",       22, 3),
    "connector": NodeType("Connector",  7, 3),
}


@dataclass
class FlowNode:
    id: int
    type: str
    x: int
    y: int
    text: str
    code: str = ""

    @property
    def info(self) -> NodeType:
        return NODE_TYPES[self.type]

    @property
    def cx(self) -> float:
        return self.x + self.info.w / 2

    @property
    def cy(self) -> float:
        return self.y + self.info.h / 2


@dataclass
class FlowEdge:
    from_id: int
    to_id: int
    label: str = ""


def make_initial_nodes() -> list[FlowNode]:
    return [
        FlowNode(1, "terminal",  5, 1,  "START"),
        FlowNode(2, "process",   4, 5,  "INIT",       "i = 1; sum = 0"),
        FlowNode(3, "decision",  4, 9,  "i <= 5?",    "i <= 5"),
        FlowNode(4, "process",   4, 17, "ACCUMULATE",  "sum = sum + i; i = i + 1"),
        FlowNode(5, "connector", 32, 13, "",           ""),
        FlowNode(6, "io",        44, 9, "PRINT SUM",  'print("Sum 1..5 = " + str(sum))'),
        FlowNode(7, "terminal",  46, 14, "END"),
    ]


def make_initial_edges() -> list[FlowEdge]:
    return [
        FlowEdge(1, 2),
        FlowEdge(2, 3),
        FlowEdge(3, 4, "Y"),
        FlowEdge(4, 5),
        FlowEdge(5, 3),
        FlowEdge(3, 6, "N"),
        FlowEdge(6, 7),
    ]


# ═══════════════════════════ INTERPRETER ═══════════════════════════

class FlowInterpreter:
    """Step-through flowchart interpreter. Ported from reference.jsx."""

    def __init__(self, nodes: list[FlowNode], edges: list[FlowEdge]):
        self.nodes = list(nodes)
        self.edges = list(edges)
        self.vars: dict[str, Any] = {}
        self.output: list[str] = []
        self.current_id: int | None = None
        self.done = False
        self.error: str | None = None
        self.waiting_input = False
        self.input_prompt = ""
        self.input_var = ""
        self.step_count = 0
        self.max_steps = 500

    # ── helpers ──

    def _find_start(self) -> FlowNode | None:
        return next(
            (n for n in self.nodes
             if n.type == "terminal" and "START" in n.text.upper()),
            None,
        )

    def _out_edges(self, nid: int) -> list[FlowEdge]:
        return [e for e in self.edges if e.from_id == nid]

    def _env(self) -> dict:
        env: dict[str, Any] = dict(self.vars)
        out = self.output
        env["print"] = lambda *a: out.append(" ".join(str(x) for x in a))
        for fn in (str, int, float, abs, min, max, len, range):
            env[fn.__name__] = fn
        return env

    def _eval(self, code: str) -> Any:
        return eval(code, {"__builtins__": {}}, self._env())

    def _exec(self, code: str) -> None:
        for stmt in (s.strip() for s in code.split(";") if s.strip()):
            m = re.match(r"^([a-zA-Z_]\w*)\s*=\s*(.+)$", stmt)
            if m:
                self.vars[m[1]] = eval(
                    m[2], {"__builtins__": {}}, self._env()
                )
            else:
                exec(stmt, {"__builtins__": {}}, self._env())

    def _advance(self, nid: int) -> None:
        outs = self._out_edges(nid)
        self.current_id = outs[0].to_id if outs else None
        if self.current_id is None:
            self.done = True

    # ── main step ──

    def step(self, input_value: str | None = None) -> None:
        if self.done or self.error:
            return
        self.step_count += 1
        if self.step_count > self.max_steps:
            self.error = "MAX STEPS EXCEEDED"
            self.done = True
            return

        # Handle pending input
        if self.waiting_input:
            if input_value is None:
                return
            try:
                parsed: Any = int(input_value) if input_value.lstrip("-").isdigit() else input_value
            except ValueError:
                parsed = input_value
            self.vars[self.input_var] = parsed
            self.output.append(f"> {input_value}")
            self.waiting_input = False
            self._advance(self.current_id)  # type: ignore
            return

        # First step — find START
        if self.current_id is None:
            start = self._find_start()
            if not start:
                self.error = "NO START NODE"
                self.done = True
                return
            self.current_id = start.id
            self.output.append("── PROGRAM START ──")
            self._advance(start.id)
            return

        node = next((n for n in self.nodes if n.id == self.current_id), None)
        if not node:
            self.error = "BROKEN LINK"
            self.done = True
            return

        try:
            if node.type == "terminal":
                self.output.append("── PROGRAM END ──")
                self.done = True

            elif node.type == "connector":
                self._advance(node.id)

            elif node.type == "process":
                if node.code and node.code.strip():
                    self._exec(node.code)
                self._advance(node.id)

            elif node.type == "decision":
                result = bool(self._eval(node.code)) if node.code and node.code.strip() else False
                outs = self._out_edges(node.id)
                ye = next((e for e in outs if (e.label or "").upper() == "Y"), None)
                ne = next((e for e in outs if (e.label or "").upper() == "N"), None)
                nxt = (ye or outs[0]) if result else (ne or outs[0])
                self.current_id = nxt.to_id if nxt else None
                if self.current_id is None:
                    self.done = True

            elif node.type == "io":
                code = (node.code or "").strip()
                im = re.match(
                    r'^(?:input|read)\s*\(\s*["\']?([^"\']*)["\']?\s*'
                    r',?\s*["\']?([a-zA-Z_]\w*)?["\']?\s*\)$',
                    code, re.I,
                )
                if im:
                    self.input_prompt = im[1] or "INPUT:"
                    self.input_var = im[2] or "x"
                    self.waiting_input = True
                    self.output.append(self.input_prompt)
                else:
                    if code:
                        self._exec(code)
                    self._advance(node.id)

        except Exception as exc:
            self.error = f'ERROR at "{node.text}": {exc}'
            self.done = True


# ═══════════════════════════ DRAWING ═══════════════════════════

def bresenham(x0: int, y0: int, x1: int, y1: int) -> list[tuple[int, int]]:
    """Integer Bresenham line. Returns list of (x, y) points."""
    pts: list[tuple[int, int]] = []
    dx, dy = abs(x1 - x0), abs(y1 - y0)
    sx = 1 if x0 < x1 else -1
    sy = 1 if y0 < y1 else -1
    err = dx - dy
    x, y = x0, y0
    for _ in range(dx + dy + 2):
        pts.append((x, y))
        if x == x1 and y == y1:
            break
        e2 = 2 * err
        if e2 > -dy:
            err -= dy
            x += sx
        if e2 < dx:
            err += dx
            y += sy
    return pts


def _lch(dx: int, dy: int) -> str:
    """Line character for a direction vector."""
    if dx == 0:
        return "│"
    if dy == 0:
        return "─"
    return "\\" if (dx > 0) == (dy > 0) else "/"


def _ach(dx: int, dy: int) -> str:
    """Arrow-head character."""
    if abs(dy) > abs(dx):
        return "▼" if dy > 0 else "▲"
    return "►" if dx > 0 else "◄"


def get_edge_exit(node: FlowNode, target: FlowNode) -> tuple[int, int]:
    """Border point of *node* facing *target* (on the border itself)."""
    info = node.info
    dx = target.cx - node.cx
    dy = target.cy - node.cy
    hw, hh = info.w / 2, info.h / 2
    if abs(dx) < 0.01 and abs(dy) < 0.01:
        return (int(node.cx), int(node.cy))
    ndx = dx / hw if hw else 0.0
    ndy = dy / hh if hh else 0.0
    if abs(ndx) > abs(ndy):
        if dx > 0:
            return (node.x + info.w - 1, int(round(node.cy)))
        return (node.x, int(round(node.cy)))
    if dy > 0:
        return (int(round(node.cx)), node.y + info.h - 1)
    return (int(round(node.cx)), node.y)


# ── buffer builder ──

def build_buffer(
    nodes: list[FlowNode],
    edges: list[FlowEdge],
    sel_id: int | None,
    exec_id: int | None,
    conn_id: int | None,
    mouse_cx: int,
    mouse_cy: int,
    cam_x: int,
    cam_y: int,
    w: int,
    h: int,
) -> list[list[tuple[str, Style]]]:
    """Build the visible canvas buffer (w × h)."""
    buf = [[(" ", BG) for _ in range(w)] for _ in range(h)]

    def put(cx: int, cy: int, ch: str, st: Style) -> None:
        bx, by = cx - cam_x, cy - cam_y
        if 0 <= bx < w and 0 <= by < h:
            buf[by][bx] = (ch, st)

    def puts(cx: int, cy: int, txt: str, st: Style) -> None:
        for i, ch in enumerate(txt):
            put(cx + i, cy, ch, st)

    # Grid dots
    for r in range(h):
        for c in range(w):
            gc, gr = c + cam_x, r + cam_y
            if gc % 5 == 0 and gr % 3 == 0:
                buf[r][c] = ("·", GRID)

    nmap = {n.id: n for n in nodes}

    # ── edges ──
    for edge in edges:
        fn, tn = nmap.get(edge.from_id), nmap.get(edge.to_id)
        if not fn or not tn:
            continue
        active = exec_id == edge.to_id
        es = EDGE_ACT if active else EDGE_S
        p1 = get_edge_exit(fn, tn)
        p2 = get_edge_exit(tn, fn)
        pts = bresenham(p1[0], p1[1], p2[0], p2[1])
        for i, (px, py) in enumerate(pts):
            if i < len(pts) - 1:
                ddx, ddy = pts[i + 1][0] - px, pts[i + 1][1] - py
            elif i > 0:
                ddx, ddy = px - pts[i - 1][0], py - pts[i - 1][1]
            else:
                ddx, ddy = 0, 0
            put(px, py, _lch(ddx, ddy), es)
        # arrowhead
        if len(pts) >= 2:
            ax = pts[-1][0] - pts[-2][0]
            ay = pts[-1][1] - pts[-2][1]
            put(pts[-1][0], pts[-1][1], _ach(ax, ay), es)
        # label
        if edge.label:
            mx = (p1[0] + p2[0]) // 2
            my = (p1[1] + p2[1]) // 2
            horiz = abs(p2[0] - p1[0]) >= abs(p2[1] - p1[1])
            if horiz:
                puts(mx - len(edge.label) // 2, my - 1, edge.label, EDGE_LBL)
            else:
                puts(mx + 1, my, edge.label, EDGE_LBL)

    # ── connect preview ──
    if conn_id and conn_id in nmap:
        cn = nmap[conn_id]
        for i, (px, py) in enumerate(
            bresenham(int(cn.cx), int(cn.cy), mouse_cx, mouse_cy)
        ):
            if i % 3 < 2:
                put(px, py, "·", CONN_S)

    # ── nodes (drawn last, on top) ──
    for node in nodes:
        info = node.info
        nc = NODE_COLORS[node.type]
        if node.id == exec_id:
            bs, ts = EXEC_BORDER, EXEC_TEXT
        elif node.id == sel_id:
            bs, ts = SEL_BORDER, SEL_TEXT
        else:
            bs, ts = nc["border"], nc["text"]

        x, y2 = node.x, node.y
        nw, nh = info.w, info.h
        tag = nc["tag"]

        # corners
        tl, tr, bl, br = "┌", "┐", "└", "┘"
        if node.type == "terminal":
            tl, tr, bl, br = "╭", "╮", "╰", "╯"
        elif node.type == "decision":
            tl, tr, bl, br = "╔", "╗", "╚", "╝"

        put(x, y2, tl, bs)
        put(x + nw - 1, y2, tr, bs)
        put(x, y2 + nh - 1, bl, bs)
        put(x + nw - 1, y2 + nh - 1, br, bs)

        hch = "═" if node.type == "decision" else "─"
        for c in range(x + 1, x + nw - 1):
            put(c, y2, hch, bs)
            put(c, y2 + nh - 1, hch, bs)
        vch = "║" if node.type == "decision" else "│"
        for r in range(y2 + 1, y2 + nh - 1):
            put(x, r, vch, bs)
            put(x + nw - 1, r, vch, bs)

        # tag in top border
        if tag:
            puts(x + 2, y2, f"[{tag}]", bs)

        # clear interior
        for r in range(y2 + 1, y2 + nh - 1):
            for c in range(x + 1, x + nw - 1):
                put(c, r, " ", ts)

        # text
        mid = y2 + nh // 2
        label = node.text[: nw - 4]
        tx = x + (nw - len(label)) // 2
        puts(tx, mid, label, ts)

        # connector ○
        if node.type == "connector" and not node.text:
            put(x + nw // 2, y2 + nh // 2, "○", ts)

    return buf


# ═══════════════════════════ EDIT SCREEN ═══════════════════════════

class EditScreen(ModalScreen):
    """Modal for editing node label + code."""

    CSS = """
    EditScreen { align: center middle; }
    #edit-box {
        width: 56;
        height: auto;
        border: solid #00d4a0;
        background: #0a1510;
        padding: 1 2;
    }
    #edit-box Label { color: #00d4a0; margin: 0 0 0 0; }
    #edit-box Input { margin: 0 0 1 0; }
    #edit-box Button { margin: 1 1 0 0; }
    """

    BINDINGS = [Binding("escape", "cancel", "Cancel")]

    def __init__(self, node: FlowNode):
        super().__init__()
        self.node = node

    def compose(self) -> ComposeResult:
        info = NODE_TYPES[self.node.type]
        hints = {
            "process": "statements separated by ;",
            "decision": "boolean expression",
            "io": 'print("...") or input("prompt", var)',
        }
        hint = hints.get(self.node.type, "")
        with Vertical(id="edit-box"):
            yield Label(f"✏️  EDIT — {info.label.upper()}")
            yield Label("Label:")
            yield Input(value=self.node.text, id="inp-text")
            yield Label(f"Code ({hint}):" if hint else "Code:")
            yield Input(value=self.node.code, id="inp-code")
            with Horizontal():
                yield Button("Cancel", id="btn-cancel")
                yield Button("💾 Save", id="btn-save", variant="primary")

    @on(Button.Pressed, "#btn-save")
    def do_save(self) -> None:
        t = self.query_one("#inp-text", Input).value
        c = self.query_one("#inp-code", Input).value
        self.dismiss((t.upper(), c))

    @on(Button.Pressed, "#btn-cancel")
    def do_cancel(self) -> None:
        self.dismiss(None)

    def action_cancel(self) -> None:
        self.dismiss(None)


# ═══════════════════════════ CANVAS WIDGET ═══════════════════════════

class FlowCanvas(Widget, can_focus=True):
    """Custom widget: renders the flowchart canvas with mouse interaction."""

    def __init__(self, **kw: Any):
        super().__init__(**kw)
        self.cam_x = 0
        self.cam_y = 0
        self._mx = 0  # mouse in canvas coords
        self._my = 0
        self._drag_id: int | None = None
        self._drag_off: tuple[int, int] | None = None
        self._buf: list[list[tuple[str, Style]]] = []

    @property
    def _app(self) -> "GRaILApp":
        return self.app  # type: ignore

    # ── rendering ──

    def _rebuild(self) -> None:
        w, h = self.size.width, self.size.height
        if w <= 0 or h <= 0:
            self._buf = []
            return
        a = self._app
        self._buf = build_buffer(
            a.nodes, a.edges, a.selected_id, a.exec_node_id,
            a.connecting_id, self._mx, self._my,
            self.cam_x, self.cam_y, w, h,
        )

    def render_line(self, y: int) -> Strip:
        if not self._buf:
            self._rebuild()
        if 0 <= y < len(self._buf):
            row = self._buf[y]
            return Strip([Segment(ch, st) for ch, st in row], self.size.width)
        return Strip([Segment(" " * max(1, self.size.width), BG)], max(1, self.size.width))

    def refresh_canvas(self) -> None:
        self._buf = []
        self.refresh()

    def on_resize(self, event: events.Resize) -> None:
        self.refresh_canvas()

    # ── coordinate helpers ──

    def _to_canvas(self, sx: int, sy: int) -> tuple[int, int]:
        return sx + self.cam_x, sy + self.cam_y

    def _hit(self, sx: int, sy: int) -> FlowNode | None:
        cx, cy = self._to_canvas(sx, sy)
        for node in reversed(self._app.nodes):
            info = node.info
            if node.x <= cx < node.x + info.w and node.y <= cy < node.y + info.h:
                return node
        return None

    # ── mouse ──

    def on_mouse_down(self, event: events.MouseDown) -> None:
        self.focus()
        a = self._app
        cx, cy = self._to_canvas(event.x, event.y)
        hit = self._hit(event.x, event.y)

        if a.tool == "add":
            info = NODE_TYPES[a.new_node_type]
            a.add_node(a.new_node_type, cx - info.w // 2, cy - info.h // 2)
            a.tool = "select"
            a.refresh_all()
            return

        if a.tool == "connect":
            if hit:
                if a.connecting_id is None:
                    a.connecting_id = hit.id
                else:
                    if hit.id != a.connecting_id:
                        a.add_edge(a.connecting_id, hit.id)
                    a.connecting_id = None
                    a.tool = "select"
                a.refresh_all()
            return

        # select mode
        if hit:
            a.selected_id = hit.id
            self._drag_id = hit.id
            self._drag_off = (cx - hit.x, cy - hit.y)
        else:
            a.selected_id = None
            self._drag_id = None
        a.refresh_all()

    def on_mouse_move(self, event: events.MouseMove) -> None:
        cx, cy = self._to_canvas(event.x, event.y)
        self._mx, self._my = cx, cy
        a = self._app
        if self._drag_id and self._drag_off:
            node = next((n for n in a.nodes if n.id == self._drag_id), None)
            if node:
                node.x = cx - self._drag_off[0]
                node.y = cy - self._drag_off[1]
                self.refresh_canvas()
        elif a.connecting_id:
            self.refresh_canvas()

    def on_mouse_up(self, event: events.MouseUp) -> None:
        self._drag_id = None
        self._drag_off = None

    # ── keyboard panning ──

    def on_key(self, event: events.Key) -> None:
        pan = {"up": (0, -2), "down": (0, 2), "left": (-3, 0), "right": (3, 0)}
        if event.key in pan:
            dx, dy = pan[event.key]
            self.cam_x = max(0, self.cam_x + dx)
            self.cam_y = max(0, self.cam_y + dy)
            self.refresh_canvas()
            event.prevent_default()
            event.stop()


# ═══════════════════════════ MAIN APP ═══════════════════════════

class GRaILApp(App):
    """GRaIL — terminal flowchart editor + interpreter."""

    CSS = """
    Screen { background: #080e0b; }

    #toolbar {
        height: 3;
        background: #0a1510;
        border-bottom: solid #1a4a3a;
        content-align: left middle;
        padding: 0 1;
    }

    #main-area { height: 1fr; }

    #canvas {
        width: 1fr;
    }

    #panel {
        width: 34;
        border-left: solid #1a4a3a;
        background: #050c0a;
    }

    #vars-sec {
        height: auto;
        max-height: 10;
        border-bottom: solid #1a4a3a;
        padding: 0 1;
    }

    #console-scroll {
        height: 1fr;
        scrollbar-size: 1 1;
    }

    #console-text {
        padding: 0 1;
    }

    #help-sec {
        height: auto;
        max-height: 9;
        border-top: solid #0e2e20;
        padding: 0 1;
    }

    #input-sec {
        height: 3;
        border-top: solid #1a4a3a;
        padding: 0 1;
        display: none;
    }

    #input-sec.visible { display: block; }

    #prog-input {
        width: 1fr;
    }
    """

    BINDINGS = [
        Binding("s", "tool_select", "Select"),
        Binding("a", "tool_add", "Add"),
        Binding("c", "tool_connect", "Connect"),
        Binding("d", "delete_node", "Del"),
        Binding("delete", "delete_node", "Delete", show=False),
        Binding("e", "edit_node", "Edit"),
        Binding("r", "run_program", "▶Run"),
        Binding("n", "step_program", "Step"),
        Binding("g", "auto_run", "Auto"),
        Binding("p", "pause_program", "Pause"),
        Binding("x", "stop_program", "Stop"),
        Binding("escape", "cancel_action", "Esc"),
        Binding("1", "ntype_1", show=False),
        Binding("2", "ntype_2", show=False),
        Binding("3", "ntype_3", show=False),
        Binding("4", "ntype_4", show=False),
        Binding("5", "ntype_5", show=False),
        Binding("q", "quit", "Quit"),
    ]

    # ── state ──
    nodes: list[FlowNode]
    edges: list[FlowEdge]
    selected_id: int | None = None
    tool: str = "select"
    new_node_type: str = "process"
    connecting_id: int | None = None
    _next_id: int = 20

    interp: FlowInterpreter | None = None
    exec_node_id: int | None = None
    console_output: list[str]
    variables: dict[str, Any]
    running: bool = False
    auto_running: bool = False
    waiting_input: bool = False
    speed: float = 0.4
    _auto_timer: Any = None

    def __init__(self) -> None:
        super().__init__()
        self.nodes = make_initial_nodes()
        self.edges = make_initial_edges()
        self.console_output = []
        self.variables = {}

    def compose(self) -> ComposeResult:
        yield Static(id="toolbar")
        with Horizontal(id="main-area"):
            yield FlowCanvas(id="canvas")
            with Vertical(id="panel"):
                yield Static(id="vars-sec")
                with VerticalScroll(id="console-scroll"):
                    yield Static(id="console-text")
                with Horizontal(id="input-sec"):
                    yield Input(placeholder="type input…", id="prog-input")
                yield Static(id="help-sec")
        yield Footer()

    def on_mount(self) -> None:
        self.query_one("#canvas", FlowCanvas).focus()
        self.refresh_all()

    # ═══ UI refresh ═══

    def refresh_all(self) -> None:
        self._draw_toolbar()
        self._draw_vars()
        self._draw_console()
        self._draw_help()
        self.query_one("#canvas", FlowCanvas).refresh_canvas()

    def _draw_toolbar(self) -> None:
        t = Text()
        t.append("  GRaIL ", S(color="#00ffc8", bold=True))
        t.append("FLOWCHART INTERPRETER  ", S(color="#1a4a3a"))

        tools = [("s", "SEL", self.tool == "select"),
                 ("a", "ADD", self.tool == "add"),
                 ("c", "LINK", self.tool == "connect")]
        for key, lbl, active in tools:
            if active:
                t.append(f" [{key}]{lbl} ",
                         S(color="#080e0b", bgcolor="#00d4a0", bold=True))
            else:
                t.append(f" [{key}]{lbl} ", S(color="#00d4a0"))
            t.append(" ")

        if self.tool == "add":
            types = list(NODE_TYPES.keys())
            labels = ["1:Proc", "2:Dec", "3:Term", "4:IO", "5:Conn"]
            cur = types.index(self.new_node_type) if self.new_node_type in types else 0
            for i, lb in enumerate(labels):
                if i == cur:
                    t.append(f" {lb} ", S(color="#080e0b", bgcolor="#ffcc66", bold=True))
                else:
                    t.append(f" {lb} ", S(color="#ffcc66"))
            t.append(" ")

        t.append(" │ ", S(color="#1a4a3a"))

        if not self.running:
            t.append(" [r]▶ RUN ", S(color="#44ff88", bold=True))
        else:
            tag = "▶ AUTO" if self.auto_running else "⏸ READY"
            t.append(f" {tag} ", S(color="#ffcc00", bold=True))
            t.append(" [n]STEP [g]GO [p]PAUSE [x]STOP ", S(color="#44ddff"))

        if self.selected_id is not None:
            t.append(" │ ", S(color="#1a4a3a"))
            node = next((n for n in self.nodes if n.id == self.selected_id), None)
            if node:
                t.append(f" [{node.type[0].upper()}] ", S(color="#ffcc66"))
            t.append("[e]EDIT [d]DEL ", S(color="#ff8866"))

        if self.connecting_id is not None:
            t.append(" │ ", S(color="#1a4a3a"))
            t.append(" LINKING… click target ", S(color="#00ffc8", bold=True))

        self.query_one("#toolbar", Static).update(t)

    def _draw_vars(self) -> None:
        t = Text()
        t.append("📦 VARIABLES\n", S(color="#1a6a4a", bold=True))
        if not self.variables:
            t.append("  (none)\n", S(color="#1a4a3a", italic=True))
        else:
            for k, v in self.variables.items():
                t.append(f"  {k}", S(color="#ffcc66"))
                t.append("=", S(color="#1a4a3a"))
                vc = "#88ddff" if isinstance(v, str) else "#44ff88"
                vs = f'"{v}"' if isinstance(v, str) else str(v)
                t.append(f"{vs} ", S(color=vc))
            t.append("\n")
        self.query_one("#vars-sec", Static).update(t)

    def _draw_console(self) -> None:
        t = Text()
        t.append("🖥️  CONSOLE\n", S(color="#1a6a4a", bold=True))
        for line in self.console_output[-50:]:
            if line.startswith("⚠"):
                t.append(f"  {line}\n", S(color="#ff6666"))
            elif line.startswith("──"):
                t.append(f"  {line}\n", S(color="#1a6a4a"))
            elif line.startswith(">"):
                t.append(f"  {line}\n", S(color="#ffcc66"))
            else:
                t.append(f"  {line}\n", S(color="#00ffc8"))
        self.query_one("#console-text", Static).update(t)

    def _draw_help(self) -> None:
        t = Text()
        t.append("HELP\n", S(color="#1a6a4a", bold=True))
        lines = [
            "Mouse: click=select, drag=move",
            "[s]Select [a]Add [c]Connect",
            "[e]Edit  [d]Delete selected",
            "[r]Run [n]Step [g]Auto [x]Stop",
            "Add mode: [1-5] node type",
            "Arrows: pan canvas",
        ]
        for ln in lines:
            t.append(f"  {ln}\n", S(color="#0e4e30"))
        self.query_one("#help-sec", Static).update(t)

    # ═══ node / edge ops ═══

    def add_node(self, ntype: str, x: int, y: int) -> int:
        nid = self._next_id
        self._next_id += 1
        info = NODE_TYPES[ntype]
        self.nodes.append(FlowNode(nid, ntype, x, y, info.label.upper()))
        self.selected_id = nid
        return nid

    def add_edge(self, from_id: int, to_id: int) -> None:
        if any(e.from_id == from_id and e.to_id == to_id for e in self.edges):
            return
        label = ""
        fn = next((n for n in self.nodes if n.id == from_id), None)
        if fn and fn.type == "decision":
            existing = [e for e in self.edges if e.from_id == from_id]
            if not any((e.label or "").upper() == "Y" for e in existing):
                label = "Y"
            elif not any((e.label or "").upper() == "N" for e in existing):
                label = "N"
        self.edges.append(FlowEdge(from_id, to_id, label))

    # ═══ tool actions ═══

    def action_tool_select(self) -> None:
        self.tool = "select"
        self.connecting_id = None
        self.refresh_all()

    def action_tool_add(self) -> None:
        self.tool = "add"
        self.connecting_id = None
        self.refresh_all()

    def action_tool_connect(self) -> None:
        self.tool = "connect"
        self.connecting_id = None
        self.refresh_all()

    def action_ntype_1(self) -> None:
        self.new_node_type = "process"; self.refresh_all()

    def action_ntype_2(self) -> None:
        self.new_node_type = "decision"; self.refresh_all()

    def action_ntype_3(self) -> None:
        self.new_node_type = "terminal"; self.refresh_all()

    def action_ntype_4(self) -> None:
        self.new_node_type = "io"; self.refresh_all()

    def action_ntype_5(self) -> None:
        self.new_node_type = "connector"; self.refresh_all()

    def action_delete_node(self) -> None:
        if self.selected_id is not None:
            sid = self.selected_id
            self.nodes = [n for n in self.nodes if n.id != sid]
            self.edges = [e for e in self.edges
                          if e.from_id != sid and e.to_id != sid]
            self.selected_id = None
            self.refresh_all()

    def action_edit_node(self) -> None:
        if self.selected_id is None:
            return
        node = next((n for n in self.nodes if n.id == self.selected_id), None)
        if node:
            self.push_screen(EditScreen(node), self._on_edit)

    def _on_edit(self, result: tuple[str, str] | None) -> None:
        if result is not None and self.selected_id is not None:
            node = next((n for n in self.nodes if n.id == self.selected_id), None)
            if node:
                node.text, node.code = result
        self.refresh_all()

    def action_cancel_action(self) -> None:
        self.tool = "select"
        self.connecting_id = None
        self.refresh_all()

    # ═══ interpreter actions ═══

    def action_run_program(self) -> None:
        self.interp = FlowInterpreter(self.nodes, self.edges)
        self.console_output = []
        self.variables = {}
        self.exec_node_id = None
        self.running = True
        self.auto_running = False
        self.waiting_input = False
        self._show_input(False)
        self.interp.step()
        self._sync()
        self.refresh_all()

    def action_step_program(self) -> None:
        if not self.running or not self.interp:
            return
        if self.interp.done or self.interp.error or self.interp.waiting_input:
            return
        self.interp.step()
        self._sync()
        self.refresh_all()

    def action_auto_run(self) -> None:
        if not self.running or not self.interp:
            return
        self.auto_running = True
        self._start_auto()
        self.refresh_all()

    def action_pause_program(self) -> None:
        self.auto_running = False
        self._stop_auto()
        self.refresh_all()

    def action_stop_program(self) -> None:
        self.auto_running = False
        self._stop_auto()
        self.running = False
        self.exec_node_id = None
        self.interp = None
        self.waiting_input = False
        self._show_input(False)
        self.refresh_all()

    def _sync(self) -> None:
        i = self.interp
        if not i:
            return
        self.exec_node_id = i.current_id
        self.console_output = list(i.output)
        self.variables = dict(i.vars)
        self.waiting_input = i.waiting_input
        if i.waiting_input:
            self._show_input(True)
            self._stop_auto()
            self.auto_running = False
        if i.done or i.error:
            self.auto_running = False
            self._stop_auto()
            if i.error:
                self.console_output.append(f"⚠ {i.error}")

    def _start_auto(self) -> None:
        self._stop_auto()
        self._auto_timer = self.set_interval(self.speed, self._auto_step)

    def _stop_auto(self) -> None:
        if self._auto_timer:
            self._auto_timer.stop()
            self._auto_timer = None

    def _auto_step(self) -> None:
        if not self.auto_running or not self.interp:
            self._stop_auto()
            return
        if self.interp.done or self.interp.error or self.interp.waiting_input:
            self.auto_running = False
            self._stop_auto()
            self.refresh_all()
            return
        self.interp.step()
        self._sync()
        self.refresh_all()

    def _show_input(self, show: bool) -> None:
        sec = self.query_one("#input-sec")
        if show:
            sec.add_class("visible")
            try:
                self.query_one("#prog-input", Input).focus()
            except Exception:
                pass
        else:
            sec.remove_class("visible")

    @on(Input.Submitted, "#prog-input")
    def _on_submit_input(self, event: Input.Submitted) -> None:
        if not self.interp or not self.interp.waiting_input:
            return
        val = event.value
        event.input.value = ""
        self.interp.step(val)
        self._sync()
        if not self.waiting_input:
            self._show_input(False)
            self.query_one("#canvas", FlowCanvas).focus()
        self.refresh_all()


# ═══════════════════════════ ENTRY ═══════════════════════════

if __name__ == "__main__":
    GRaILApp().run()
