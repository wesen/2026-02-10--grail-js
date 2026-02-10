import { useState, useRef, useCallback, useEffect } from "react";

const NODE_TYPES = {
  process: { label: "Process", shape: "rect", w: 120, h: 50 },
  decision: { label: "Decision", shape: "diamond", w: 90, h: 65 },
  terminal: { label: "Terminal", shape: "rounded", w: 100, h: 44 },
  io: { label: "I/O", shape: "parallelogram", w: 120, h: 48 },
  connector: { label: "Connector", shape: "circle", w: 36, h: 36 },
};

const INITIAL_NODES = [
  { id: 1, type: "terminal", x: 300, y: 30, text: "START", code: "" },
  { id: 2, type: "process", x: 280, y: 120, text: "INIT", code: "i = 1; sum = 0" },
  { id: 3, type: "decision", x: 310, y: 220, text: "i <= 5?", code: "i <= 5" },
  { id: 4, type: "process", x: 270, y: 330, text: "ACCUMULATE", code: "sum = sum + i; i = i + 1" },
  { id: 5, type: "connector", x: 500, y: 245, text: "", code: "" },
  { id: 6, type: "io", x: 265, y: 440, text: "PRINT SUM", code: 'print("Sum 1..5 = " + sum)' },
  { id: 7, type: "terminal", x: 300, y: 540, text: "END", code: "" },
];

const INITIAL_EDGES = [
  { from: 1, to: 2 },
  { from: 2, to: 3 },
  { from: 3, to: 4, label: "Y" },
  { from: 4, to: 5 },
  { from: 5, to: 3 },
  { from: 3, to: 6, label: "N" },
  { from: 6, to: 7 },
];

function getNodeCenter(node) {
  const info = NODE_TYPES[node.type];
  return { x: node.x + info.w / 2, y: node.y + info.h / 2 };
}

function getEdgePoint(node, targetCenter) {
  const info = NODE_TYPES[node.type];
  const cx = node.x + info.w / 2;
  const cy = node.y + info.h / 2;
  const dx = targetCenter.x - cx;
  const dy = targetCenter.y - cy;
  const angle = Math.atan2(dy, dx);
  if (info.shape === "circle") {
    const r = info.w / 2;
    return { x: cx + r * Math.cos(angle), y: cy + r * Math.sin(angle) };
  }
  if (info.shape === "diamond") {
    const hw = info.w / 2, hh = info.h / 2;
    const absCos = Math.abs(Math.cos(angle));
    const absSin = Math.abs(Math.sin(angle));
    const t = Math.min(hw / (absCos || 0.001), hh / (absSin || 0.001));
    return { x: cx + t * Math.cos(angle), y: cy + t * Math.sin(angle) };
  }
  const hw = info.w / 2, hh = info.h / 2;
  const absTan = Math.abs(Math.tan(angle));
  if (absTan < hh / hw) {
    const sign = Math.cos(angle) > 0 ? 1 : -1;
    return { x: cx + sign * hw, y: cy + sign * hw * Math.tan(angle) };
  } else {
    const sign = Math.sin(angle) > 0 ? 1 : -1;
    return { x: cx + sign * hh / Math.tan(angle), y: cy + sign * hh };
  }
}

function NodeShape({ node, selected, executing, onMouseDown, onDoubleClick }) {
  const info = NODE_TYPES[node.type];
  const isExec = executing;
  const glow = isExec
    ? "drop-shadow(0 0 14px #ffcc00) drop-shadow(0 0 6px #ffaa00)"
    : selected
      ? "drop-shadow(0 0 8px #00ffcc)"
      : "drop-shadow(0 0 3px rgba(0,255,180,0.4))";
  const strokeColor = isExec ? "#ffcc00" : selected ? "#00ffee" : "#00d4a0";
  const textColor = isExec ? "#ffee66" : "#00ffc8";

  const style = {
    position: "absolute",
    left: node.x,
    top: node.y,
    width: info.w,
    height: info.h,
    cursor: "grab",
    filter: glow,
    transition: "filter 0.15s",
  };

  const textEl = (
    <text
      x={info.w / 2} y={info.h / 2}
      textAnchor="middle" dominantBaseline="central"
      fill={textColor} fontSize="11"
      fontFamily="'Courier New', monospace" fontWeight="bold"
      style={{ textShadow: `0 0 6px ${textColor}` }}
    >
      {node.text}
    </text>
  );

  const strokeProps = {
    stroke: strokeColor,
    strokeWidth: isExec ? 2.5 : selected ? 2.2 : 1.5,
    fill: isExec ? "rgba(255,204,0,0.06)" : "none",
  };

  let shape;
  if (info.shape === "rect") shape = <rect x={1} y={1} width={info.w - 2} height={info.h - 2} {...strokeProps} />;
  else if (info.shape === "rounded") shape = <rect x={1} y={1} width={info.w - 2} height={info.h - 2} rx={20} ry={20} {...strokeProps} />;
  else if (info.shape === "diamond") {
    const cx = info.w / 2, cy = info.h / 2;
    shape = <polygon points={`${cx},2 ${info.w - 2},${cy} ${cx},${info.h - 2} 2,${cy}`} {...strokeProps} />;
  } else if (info.shape === "parallelogram") {
    const off = 14;
    shape = <polygon points={`${off},1 ${info.w - 1},1 ${info.w - off},${info.h - 1} 1,${info.h - 1}`} {...strokeProps} />;
  } else if (info.shape === "circle") shape = <circle cx={info.w / 2} cy={info.h / 2} r={info.w / 2 - 2} {...strokeProps} />;

  return (
    <div style={style} onMouseDown={onMouseDown} onDoubleClick={onDoubleClick}>
      <svg width={info.w} height={info.h}>{shape}{textEl}</svg>
    </div>
  );
}

function Scanlines() {
  return <div style={{ position: "absolute", inset: 0, pointerEvents: "none", zIndex: 100, background: "repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0,0,0,0.15) 2px, rgba(0,0,0,0.15) 4px)", mixBlendMode: "multiply" }} />;
}

function CRTOverlay() {
  return <div style={{ position: "absolute", inset: 0, pointerEvents: "none", zIndex: 101, borderRadius: 12, boxShadow: "inset 0 0 80px rgba(0,0,0,0.6), inset 0 0 200px rgba(0,0,0,0.3)" }} />;
}

/* ---------- Interpreter Engine ---------- */
class FlowInterpreter {
  constructor(nodes, edges) {
    this.nodes = nodes;
    this.edges = edges;
    this.vars = {};
    this.output = [];
    this.currentId = null;
    this.done = false;
    this.error = null;
    this.waitingInput = false;
    this.inputPrompt = "";
    this.inputVar = "";
    this.stepCount = 0;
    this.maxSteps = 500;
  }

  findStart() {
    return this.nodes.find((n) => n.type === "terminal" && n.text.toUpperCase().includes("START"));
  }

  getOutEdges(id) {
    return this.edges.filter((e) => e.from === id);
  }

  evalExpr(code) {
    const keys = Object.keys(this.vars);
    const vals = Object.values(this.vars);
    const output = this.output;
    const fn = new Function(
      ...keys, "print",
      `"use strict"; ${code.includes("return") ? code : "return (" + code + ")"}`
    );
    return fn(...vals, (msg) => output.push(String(msg)));
  }

  execStatements(code) {
    const output = this.output;
    const varsCopy = { ...this.vars };
    const assignments = code.split(";").map((s) => s.trim()).filter(Boolean);
    for (const stmt of assignments) {
      const m = stmt.match(/^\s*([a-zA-Z_$][a-zA-Z0-9_$]*)\s*=\s*(.+)$/);
      if (m) {
        const [, varName, expr] = m;
        const ekeys = Object.keys(varsCopy);
        const evals = Object.values(varsCopy);
        const fn = new Function(...ekeys, "print", `"use strict"; return (${expr})`);
        varsCopy[varName] = fn(...evals, (msg) => output.push(String(msg)));
      } else {
        const ekeys = Object.keys(varsCopy);
        const evals = Object.values(varsCopy);
        const fn = new Function(...ekeys, "print", `"use strict"; ${stmt}`);
        fn(...evals, (msg) => output.push(String(msg)));
      }
    }
    this.vars = varsCopy;
  }

  step(inputValue) {
    if (this.done || this.error) return;
    if (this.stepCount++ > this.maxSteps) {
      this.error = "MAX STEPS EXCEEDED (infinite loop?)";
      this.done = true;
      return;
    }

    if (this.waitingInput) {
      if (inputValue == null) return;
      const parsed = isNaN(Number(inputValue)) ? inputValue : Number(inputValue);
      this.vars[this.inputVar] = parsed;
      this.output.push(`> ${inputValue}`);
      this.waitingInput = false;
      const outs = this.getOutEdges(this.currentId);
      this.currentId = outs.length > 0 ? outs[0].to : null;
      if (!this.currentId) this.done = true;
      return;
    }

    if (this.currentId == null) {
      const start = this.findStart();
      if (!start) { this.error = "NO START NODE FOUND"; this.done = true; return; }
      this.currentId = start.id;
      this.output.push("── PROGRAM START ──");
      const outs = this.getOutEdges(start.id);
      this.currentId = outs.length > 0 ? outs[0].to : null;
      if (!this.currentId) this.done = true;
      return;
    }

    const node = this.nodes.find((n) => n.id === this.currentId);
    if (!node) { this.error = "BROKEN LINK"; this.done = true; return; }

    try {
      if (node.type === "terminal") {
        this.output.push("── PROGRAM END ──");
        this.done = true;
        return;
      }

      if (node.type === "connector") {
        const outs = this.getOutEdges(node.id);
        this.currentId = outs.length > 0 ? outs[0].to : null;
        if (!this.currentId) this.done = true;
        return;
      }

      if (node.type === "process") {
        if (node.code && node.code.trim()) this.execStatements(node.code);
        const outs = this.getOutEdges(node.id);
        this.currentId = outs.length > 0 ? outs[0].to : null;
        if (!this.currentId) this.done = true;
        return;
      }

      if (node.type === "decision") {
        let result = false;
        if (node.code && node.code.trim()) result = !!this.evalExpr(node.code);
        const outs = this.getOutEdges(node.id);
        const yEdge = outs.find((e) => (e.label || "").toUpperCase() === "Y");
        const nEdge = outs.find((e) => (e.label || "").toUpperCase() === "N");
        const next = result ? (yEdge || outs[0]) : (nEdge || outs[0]);
        this.currentId = next ? next.to : null;
        if (!this.currentId) this.done = true;
        return;
      }

      if (node.type === "io") {
        const code = (node.code || "").trim();
        const inputMatch = code.match(/^(?:input|read)\s*\(\s*"?([^"]*)"?\s*,?\s*"?([a-zA-Z_$][a-zA-Z0-9_$]*)?"?\s*\)$/i);
        if (inputMatch) {
          this.inputPrompt = inputMatch[1] || "INPUT:";
          this.inputVar = inputMatch[2] || "x";
          this.waitingInput = true;
          this.output.push(this.inputPrompt);
          return;
        }
        if (code) this.execStatements(code);
        const outs = this.getOutEdges(node.id);
        this.currentId = outs.length > 0 ? outs[0].to : null;
        if (!this.currentId) this.done = true;
        return;
      }
    } catch (e) {
      this.error = `ERROR at "${node.text}": ${e.message}`;
      this.done = true;
    }
  }
}

/* ---------- Main Component ---------- */
export default function GRaIL() {
  const [nodes, setNodes] = useState(INITIAL_NODES);
  const [edges, setEdges] = useState(INITIAL_EDGES);
  const [selected, setSelected] = useState(null);
  const [dragging, setDragging] = useState(null);
  const [connecting, setConnecting] = useState(null);
  const [mousePos, setMousePos] = useState({ x: 0, y: 0 });
  const [tool, setTool] = useState("select");
  const [newNodeType, setNewNodeType] = useState("process");

  const [editNode, setEditNode] = useState(null);
  const [editText, setEditText] = useState("");
  const [editCode, setEditCode] = useState("");

  const [interp, setInterp] = useState(null);
  const [execNodeId, setExecNodeId] = useState(null);
  const [consoleOutput, setConsoleOutput] = useState([]);
  const [vars, setVars] = useState({});
  const [running, setRunning] = useState(false);
  const [autoRun, setAutoRun] = useState(false);
  const [inputVal, setInputVal] = useState("");
  const [waitingInput, setWaitingInput] = useState(false);
  const [speed, setSpeed] = useState(400);
  const [showPanel, setShowPanel] = useState(true);

  const canvasRef = useRef(null);
  const consoleEndRef = useRef(null);
  const nextId = useRef(20);
  const interpRef = useRef(null);
  const autoRef = useRef(false);

  useEffect(() => { consoleEndRef.current?.scrollIntoView({ behavior: "smooth" }); }, [consoleOutput]);

  useEffect(() => { autoRef.current = autoRun; }, [autoRun]);

  useEffect(() => {
    if (!autoRun || !interp || interp.done || interp.error || interp.waitingInput) return;
    const t = setTimeout(() => {
      if (!autoRef.current) return;
      doStep();
    }, speed);
    return () => clearTimeout(t);
  }, [autoRun, interp, execNodeId, speed]);

  function startProgram() {
    const i = new FlowInterpreter(nodes, edges);
    interpRef.current = i;
    setInterp(i);
    setConsoleOutput([]);
    setVars({});
    setExecNodeId(null);
    setRunning(true);
    setWaitingInput(false);
    setAutoRun(false);
    setShowPanel(true);
    i.step();
    syncState(i);
  }

  function doStep(inputValue) {
    const i = interpRef.current;
    if (!i || i.done || i.error) return;
    if (i.waitingInput && inputValue == null) return;
    i.step(inputValue);
    syncState(i);
  }

  function syncState(i) {
    setExecNodeId(i.currentId);
    setConsoleOutput([...i.output]);
    setVars({ ...i.vars });
    setWaitingInput(i.waitingInput);
    if (i.done || i.error) {
      setAutoRun(false);
      if (i.error) setConsoleOutput((o) => [...o, `⚠ ${i.error}`]);
      if (i.done && !i.error) setConsoleOutput((o) => [...o, ""]);
    }
  }

  function submitInput() {
    const i = interpRef.current;
    if (!i || !i.waitingInput) return;
    doStep(inputVal);
    setInputVal("");
  }

  function stopProgram() {
    setAutoRun(false);
    setRunning(false);
    setExecNodeId(null);
    interpRef.current = null;
    setInterp(null);
    setWaitingInput(false);
  }

  const handleCanvasMouseDown = useCallback((e) => {
    if (tool === "add") {
      const rect = canvasRef.current.getBoundingClientRect();
      const info = NODE_TYPES[newNodeType];
      const x = e.clientX - rect.left - info.w / 2;
      const y = e.clientY - rect.top - info.h / 2;
      const id = nextId.current++;
      setNodes((n) => [...n, { id, type: newNodeType, x, y, text: info.label.toUpperCase(), code: "" }]);
      setSelected(id);
      setTool("select");
    } else {
      setSelected(null);
    }
  }, [tool, newNodeType]);

  const handleNodeMouseDown = useCallback((e, node) => {
    e.stopPropagation();
    if (tool === "connect") { setConnecting(node.id); return; }
    setSelected(node.id);
    const rect = canvasRef.current.getBoundingClientRect();
    setDragging({ id: node.id, offX: e.clientX - rect.left - node.x, offY: e.clientY - rect.top - node.y });
  }, [tool]);

  const handleMouseMove = useCallback((e) => {
    const rect = canvasRef.current.getBoundingClientRect();
    const mx = e.clientX - rect.left;
    const my = e.clientY - rect.top;
    setMousePos({ x: mx, y: my });
    if (dragging) {
      setNodes((ns) => ns.map((n) => n.id === dragging.id ? { ...n, x: mx - dragging.offX, y: my - dragging.offY } : n));
    }
  }, [dragging]);

  const handleMouseUp = useCallback((e) => {
    if (connecting) {
      const rect = canvasRef.current.getBoundingClientRect();
      const mx = e.clientX - rect.left;
      const my = e.clientY - rect.top;
      const target = nodes.find((n) => {
        const info = NODE_TYPES[n.type];
        return mx >= n.x && mx <= n.x + info.w && my >= n.y && my <= n.y + info.h;
      });
      if (target && target.id !== connecting) {
        const fromNode = nodes.find((n) => n.id === connecting);
        const isDecision = fromNode && fromNode.type === "decision";
        const existingEdges = edges.filter((e) => e.from === connecting);
        let label = "";
        if (isDecision) {
          const hasY = existingEdges.some((e) => (e.label || "").toUpperCase() === "Y");
          const hasN = existingEdges.some((e) => (e.label || "").toUpperCase() === "N");
          if (!hasY) label = "Y";
          else if (!hasN) label = "N";
        }
        setEdges((es) => {
          if (es.some((ed) => ed.from === connecting && ed.to === target.id)) return es;
          return [...es, { from: connecting, to: target.id, label }];
        });
      }
      setConnecting(null);
      setTool("select");
    }
    setDragging(null);
  }, [connecting, nodes, edges]);

  const handleDoubleClick = useCallback((e, node) => {
    e.stopPropagation();
    setEditNode(node.id);
    setEditText(node.text);
    setEditCode(node.code || "");
  }, []);

  const commitEdit = useCallback(() => {
    if (editNode != null) {
      setNodes((ns) => ns.map((n) => n.id === editNode ? { ...n, text: editText, code: editCode } : n));
      setEditNode(null);
    }
  }, [editNode, editText, editCode]);

  const deleteSelected = useCallback(() => {
    if (selected) {
      setNodes((ns) => ns.filter((n) => n.id !== selected));
      setEdges((es) => es.filter((e) => e.from !== selected && e.to !== selected));
      setSelected(null);
    }
  }, [selected]);

  useEffect(() => {
    const handler = (e) => {
      if (e.key === "Delete" || (e.key === "Backspace" && editNode == null && !waitingInput)) {
        if (editNode == null) deleteSelected();
      }
      if (e.key === "Escape") { setTool("select"); setConnecting(null); setEditNode(null); }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [deleteSelected, editNode, waitingInput]);

  const nodesById = Object.fromEntries(nodes.map((n) => [n.id, n]));

  const toolBtn = (active, extra) => ({
    padding: "5px 12px",
    background: active ? "rgba(0,255,200,0.15)" : "rgba(0,255,200,0.04)",
    border: active ? "1px solid #00ffc8" : "1px solid rgba(0,255,200,0.2)",
    color: "#00ffc8",
    fontFamily: "'Courier New', monospace",
    fontSize: 11,
    cursor: "pointer",
    borderRadius: 2,
    textShadow: active ? "0 0 8px #00ffc8" : "none",
    transition: "all 0.2s",
    whiteSpace: "nowrap",
    ...extra,
  });

  const runBtn = (color, glow) => ({
    ...toolBtn(false),
    borderColor: color,
    color: color,
    textShadow: `0 0 8px ${glow}`,
  });

  return (
    <div style={{ width: "100%", height: "100vh", background: "#0a0a0a", display: "flex", flexDirection: "column", fontFamily: "'Courier New', monospace", overflow: "hidden" }}>
      {/* Header */}
      <div style={{ padding: "8px 16px", display: "flex", alignItems: "center", gap: 8, borderBottom: "1px solid rgba(0,255,200,0.15)", background: "rgba(0,20,15,0.8)", flexShrink: 0, flexWrap: "wrap" }}>
        <span style={{ color: "#00ffc8", fontSize: 15, fontWeight: "bold", letterSpacing: 4, textShadow: "0 0 10px #00ffc8, 0 0 20px rgba(0,255,200,0.3)", marginRight: 12 }}>GRaIL</span>
        <span style={{ color: "rgba(0,255,200,0.25)", fontSize: 10, marginRight: 12 }}>FLOWCHART INTERPRETER</span>

        <button style={toolBtn(tool === "select")} onClick={() => { setTool("select"); setConnecting(null); }}>✋ SEL</button>
        <button style={toolBtn(tool === "add")} onClick={() => setTool("add")}>➕ ADD</button>
        <button style={toolBtn(tool === "connect")} onClick={() => setTool("connect")}>🔗 LINK</button>

        {tool === "add" && (
          <select value={newNodeType} onChange={(e) => setNewNodeType(e.target.value)} style={{ background: "rgba(0,20,15,0.9)", color: "#00ffc8", border: "1px solid rgba(0,255,200,0.3)", fontFamily: "'Courier New', monospace", fontSize: 11, padding: "4px 6px", borderRadius: 2 }}>
            {Object.entries(NODE_TYPES).map(([k, v]) => <option key={k} value={k}>{v.label}</option>)}
          </select>
        )}

        {selected && <button style={{ ...toolBtn(false), borderColor: "#ff4444", color: "#ff6666" }} onClick={deleteSelected}>🗑️ DEL</button>}

        <div style={{ width: 1, height: 20, background: "rgba(0,255,200,0.15)", margin: "0 4px" }} />

        {!running ? (
          <button style={runBtn("#44ff88", "#44ff88")} onClick={startProgram}>▶ RUN</button>
        ) : (
          <>
            <button style={runBtn("#44ff88", "#44ff88")} onClick={() => doStep()} disabled={autoRun || waitingInput || (interp && (interp.done || interp.error))}>⏭ STEP</button>
            <button style={runBtn(autoRun ? "#ffaa00" : "#44ddff", autoRun ? "#ffaa00" : "#44ddff")} onClick={() => setAutoRun(!autoRun)} disabled={waitingInput || (interp && (interp.done || interp.error))}>
              {autoRun ? "⏸ PAUSE" : "⏩ AUTO"}
            </button>
            <button style={runBtn("#ff6666", "#ff4444")} onClick={stopProgram}>⏹ STOP</button>
            <label style={{ display: "flex", alignItems: "center", gap: 4, color: "rgba(0,255,200,0.4)", fontSize: 10 }}>
              SPD
              <input type="range" min={50} max={1000} step={50} value={speed} onChange={(e) => setSpeed(Number(e.target.value))}
                style={{ width: 60, accentColor: "#00ffc8" }}
              />
              <span style={{ color: "#00ffc8", width: 30 }}>{speed}ms</span>
            </label>
          </>
        )}

        <button style={{ ...toolBtn(showPanel), marginLeft: "auto" }} onClick={() => setShowPanel(!showPanel)}>
          {showPanel ? "◀ HIDE" : "▶ PANEL"}
        </button>
      </div>

      <div style={{ flex: 1, display: "flex", overflow: "hidden" }}>
        {/* Canvas */}
        <div
          ref={canvasRef}
          style={{ flex: 1, position: "relative", overflow: "hidden", background: "radial-gradient(ellipse at center, #0d1a16 0%, #050a08 60%, #020504 100%)", cursor: tool === "add" ? "crosshair" : tool === "connect" ? "pointer" : "default" }}
          onMouseDown={handleCanvasMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
        >
          <Scanlines />
          <CRTOverlay />

          <svg style={{ position: "absolute", inset: 0, width: "100%", height: "100%", zIndex: 0 }}>
            <defs>
              <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
                <circle cx="20" cy="20" r="0.5" fill="rgba(0,255,200,0.07)" />
              </pattern>
            </defs>
            <rect width="100%" height="100%" fill="url(#grid)" />
          </svg>

          <svg style={{ position: "absolute", inset: 0, width: "100%", height: "100%", zIndex: 1, pointerEvents: "none" }}>
            <defs>
              <marker id="arrow" markerWidth="8" markerHeight="6" refX="7" refY="3" orient="auto">
                <polygon points="0 0, 8 3, 0 6" fill="#00d4a0" />
              </marker>
              <marker id="arrowExec" markerWidth="8" markerHeight="6" refX="7" refY="3" orient="auto">
                <polygon points="0 0, 8 3, 0 6" fill="#ffcc00" />
              </marker>
              <filter id="edgeGlow">
                <feGaussianBlur stdDeviation="1.5" result="blur" />
                <feMerge><feMergeNode in="blur" /><feMergeNode in="SourceGraphic" /></feMerge>
              </filter>
            </defs>
            {edges.map((edge, i) => {
              const fromNode = nodesById[edge.from];
              const toNode = nodesById[edge.to];
              if (!fromNode || !toNode) return null;
              const toC = getNodeCenter(toNode);
              const fromC = getNodeCenter(fromNode);
              const p1 = getEdgePoint(fromNode, toC);
              const p2 = getEdgePoint(toNode, fromC);
              const mx = (p1.x + p2.x) / 2, my = (p1.y + p2.y) / 2;
              const isActive = running && execNodeId === edge.to;
              return (
                <g key={i} filter="url(#edgeGlow)">
                  <line x1={p1.x} y1={p1.y} x2={p2.x} y2={p2.y}
                    stroke={isActive ? "#ffcc00" : "#00d4a0"} strokeWidth={isActive ? 2 : 1.3}
                    markerEnd={isActive ? "url(#arrowExec)" : "url(#arrow)"} opacity={0.85}
                  />
                  {edge.label && (
                    <text x={mx} y={my - 6} fill={isActive ? "#ffee66" : "#00ffc8"} fontSize="10" fontFamily="'Courier New', monospace" textAnchor="middle">
                      {edge.label}
                    </text>
                  )}
                </g>
              );
            })}
            {connecting && nodesById[connecting] && (
              <line
                x1={getNodeCenter(nodesById[connecting]).x} y1={getNodeCenter(nodesById[connecting]).y}
                x2={mousePos.x} y2={mousePos.y}
                stroke="#00ffc8" strokeWidth={1} strokeDasharray="6,4" opacity={0.6}
              />
            )}
          </svg>

          <div style={{ position: "absolute", inset: 0, zIndex: 2 }}>
            {nodes.map((node) => (
              <NodeShape key={node.id} node={node} selected={selected === node.id}
                executing={execNodeId === node.id}
                onMouseDown={(e) => handleNodeMouseDown(e, node)}
                onDoubleClick={(e) => handleDoubleClick(e, node)}
              />
            ))}
          </div>

          {/* Edit modal */}
          {editNode != null && nodesById[editNode] && (
            <div style={{ position: "absolute", inset: 0, zIndex: 200, display: "flex", alignItems: "center", justifyContent: "center", background: "rgba(0,0,0,0.6)" }}
              onClick={(e) => { if (e.target === e.currentTarget) commitEdit(); }}
            >
              <div style={{ background: "rgba(5,15,12,0.97)", border: "1px solid rgba(0,255,200,0.3)", padding: 20, borderRadius: 4, width: 360, boxShadow: "0 0 40px rgba(0,255,200,0.1)" }}>
                <div style={{ color: "#00ffc8", fontSize: 12, marginBottom: 12, letterSpacing: 2 }}>
                  ✏️ EDIT NODE — {NODE_TYPES[nodesById[editNode]?.type]?.label?.toUpperCase()}
                </div>
                <label style={{ color: "rgba(0,255,200,0.5)", fontSize: 10, display: "block", marginBottom: 4 }}>LABEL</label>
                <input value={editText} onChange={(e) => setEditText(e.target.value.toUpperCase())}
                  style={{ width: "100%", background: "rgba(0,20,15,0.9)", color: "#00ffc8", border: "1px solid rgba(0,255,200,0.3)", fontFamily: "'Courier New', monospace", fontSize: 13, padding: "6px 10px", borderRadius: 2, marginBottom: 12, boxSizing: "border-box", outline: "none" }}
                />
                <label style={{ color: "rgba(0,255,200,0.5)", fontSize: 10, display: "block", marginBottom: 4 }}>
                  JAVASCRIPT CODE {nodesById[editNode]?.type === "decision" ? "(boolean expr)" : nodesById[editNode]?.type === "io" ? '(print("...") or input("prompt", varName))' : "(statements separated by ;)"}
                </label>
                <textarea value={editCode} onChange={(e) => setEditCode(e.target.value)} rows={3}
                  style={{ width: "100%", background: "rgba(0,20,15,0.9)", color: "#ffcc66", border: "1px solid rgba(0,255,200,0.3)", fontFamily: "'Courier New', monospace", fontSize: 12, padding: "6px 10px", borderRadius: 2, boxSizing: "border-box", outline: "none", resize: "vertical" }}
                  placeholder={nodesById[editNode]?.type === "process" ? 'x = 10; y = x * 2' : nodesById[editNode]?.type === "decision" ? 'x > 0' : nodesById[editNode]?.type === "io" ? 'print("Hello " + name)' : ''}
                />
                <div style={{ display: "flex", gap: 8, marginTop: 14, justifyContent: "flex-end" }}>
                  <button style={toolBtn(false)} onClick={() => setEditNode(null)}>CANCEL</button>
                  <button style={{ ...toolBtn(true) }} onClick={commitEdit}>💾 SAVE</button>
                </div>
              </div>
            </div>
          )}

          <div style={{ position: "absolute", inset: 0, pointerEvents: "none", zIndex: 99, background: "rgba(0,255,200,0.008)", animation: "flicker 0.15s infinite alternate" }} />
        </div>

        {/* Right Panel */}
        {showPanel && (
          <div style={{ width: 300, borderLeft: "1px solid rgba(0,255,200,0.15)", background: "rgba(5,12,10,0.95)", display: "flex", flexDirection: "column", flexShrink: 0 }}>
            {/* Variables */}
            <div style={{ borderBottom: "1px solid rgba(0,255,200,0.12)", padding: "10px 14px" }}>
              <div style={{ color: "rgba(0,255,200,0.5)", fontSize: 10, letterSpacing: 2, marginBottom: 8 }}>📦 VARIABLES</div>
              {Object.keys(vars).length === 0 ? (
                <div style={{ color: "rgba(0,255,200,0.2)", fontSize: 11, fontStyle: "italic" }}>No variables yet</div>
              ) : (
                <div style={{ display: "flex", flexWrap: "wrap", gap: "4px 10px" }}>
                  {Object.entries(vars).map(([k, v]) => (
                    <div key={k} style={{ fontSize: 12, color: "#00ffc8" }}>
                      <span style={{ color: "#ffcc66" }}>{k}</span>
                      <span style={{ color: "rgba(0,255,200,0.3)" }}>=</span>
                      <span style={{ color: typeof v === "string" ? "#88ddff" : "#44ff88" }}>
                        {typeof v === "string" ? `"${v}"` : String(v)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Console */}
            <div style={{ flex: 1, display: "flex", flexDirection: "column", minHeight: 0 }}>
              <div style={{ padding: "10px 14px 4px", color: "rgba(0,255,200,0.5)", fontSize: 10, letterSpacing: 2 }}>🖥️ CONSOLE OUTPUT</div>
              <div style={{ flex: 1, overflow: "auto", padding: "4px 14px 10px", fontSize: 12 }}>
                {consoleOutput.map((line, i) => (
                  <div key={i} style={{
                    color: line.startsWith("⚠") ? "#ff6666"
                      : line.startsWith("──") ? "rgba(0,255,200,0.35)"
                        : line.startsWith(">") ? "#ffcc66"
                          : "#00ffc8",
                    marginBottom: 2,
                    fontFamily: "'Courier New', monospace",
                    textShadow: line.startsWith("⚠") ? "0 0 6px #ff4444" : "0 0 4px rgba(0,255,200,0.3)",
                  }}>{line || "\u00A0"}</div>
                ))}
                <div ref={consoleEndRef} />
              </div>

              {waitingInput && (
                <div style={{ padding: "8px 14px", borderTop: "1px solid rgba(0,255,200,0.12)", display: "flex", gap: 6 }}>
                  <input
                    autoFocus
                    value={inputVal}
                    onChange={(e) => setInputVal(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && submitInput()}
                    placeholder="type input..."
                    style={{ flex: 1, background: "rgba(0,20,15,0.9)", color: "#ffcc66", border: "1px solid rgba(255,204,0,0.4)", fontFamily: "'Courier New', monospace", fontSize: 12, padding: "5px 8px", borderRadius: 2, outline: "none" }}
                  />
                  <button style={{ ...toolBtn(false), borderColor: "#ffcc00", color: "#ffcc00" }} onClick={submitInput}>⏎</button>
                </div>
              )}

              {/* Help */}
              <div style={{ padding: "8px 14px", borderTop: "1px solid rgba(0,255,200,0.08)", color: "rgba(0,255,200,0.2)", fontSize: 9, lineHeight: 1.6 }}>
                <strong style={{ color: "rgba(0,255,200,0.35)" }}>NODE TYPES:</strong><br />
                ▸ Process: <span style={{ color: "rgba(255,204,102,0.5)" }}>x = 1; y = x + 2</span><br />
                ▸ Decision: <span style={{ color: "rgba(255,204,102,0.5)" }}>x {">"} 0</span> → Y/N edges<br />
                ▸ I/O: <span style={{ color: "rgba(255,204,102,0.5)" }}>print("hi")</span> or <span style={{ color: "rgba(255,204,102,0.5)" }}>input("prompt", varName)</span><br />
                ▸ Double-click any node to edit
              </div>
            </div>
          </div>
        )}
      </div>

      <style>{`
        @keyframes flicker { 0% { opacity: 1; } 50% { opacity: 0.97; } 100% { opacity: 1; } }
        ::selection { background: rgba(0,255,200,0.3); color: #00ffc8; }
        ::-webkit-scrollbar { width: 6px; }
        ::-webkit-scrollbar-track { background: rgba(0,255,200,0.03); }
        ::-webkit-scrollbar-thumb { background: rgba(0,255,200,0.15); border-radius: 3px; }
      `}</style>
    </div>
  );
}
