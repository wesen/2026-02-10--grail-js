package graphmodel

import (
	"image"
	"testing"
)

// testNode implements Spatial for testing.
type testNode struct {
	X, Y int
	W, H int
}

func (n testNode) Pos() image.Point  { return image.Pt(n.X, n.Y) }
func (n testNode) Size() image.Point { return image.Pt(n.W, n.H) }

func setPos(n *testNode, p image.Point) {
	n.X = p.X
	n.Y = p.Y
}

// ── Spatial helpers ──

func TestCenterOf(t *testing.T) {
	n := testNode{X: 10, Y: 20, W: 8, H: 4}
	c := CenterOf(n)
	if c != image.Pt(14, 22) {
		t.Errorf("CenterOf: expected (14,22), got %v", c)
	}
}

func TestBoundsOf(t *testing.T) {
	n := testNode{X: 10, Y: 20, W: 8, H: 4}
	b := BoundsOf(n)
	want := image.Rect(10, 20, 18, 24)
	if b != want {
		t.Errorf("BoundsOf: expected %v, got %v", want, b)
	}
}

// ── AddNode ──

func TestAddNodeIDsIncrement(t *testing.T) {
	g := New[testNode, string]()
	id0 := g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})
	id1 := g.AddNode(testNode{X: 10, Y: 0, W: 5, H: 3})
	id2 := g.AddNode(testNode{X: 20, Y: 0, W: 5, H: 3})
	if id0 != 0 || id1 != 1 || id2 != 2 {
		t.Errorf("expected IDs 0,1,2, got %d,%d,%d", id0, id1, id2)
	}
}

func TestNodesInsertionOrder(t *testing.T) {
	g := New[testNode, string]()
	g.AddNode(testNode{X: 30, Y: 0, W: 5, H: 3})
	g.AddNode(testNode{X: 10, Y: 0, W: 5, H: 3})
	g.AddNode(testNode{X: 20, Y: 0, W: 5, H: 3})

	nodes := g.Nodes()
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	// Insertion order, not position order
	if nodes[0].Data.X != 30 || nodes[1].Data.X != 10 || nodes[2].Data.X != 20 {
		t.Error("Nodes() not in insertion order")
	}
}

func TestNodeByID(t *testing.T) {
	g := New[testNode, string]()
	id := g.AddNode(testNode{X: 5, Y: 5, W: 10, H: 3})
	n := g.Node(id)
	if n == nil {
		t.Fatal("Node() returned nil")
	}
	if n.Data.X != 5 {
		t.Errorf("expected X=5, got %d", n.Data.X)
	}
}

func TestNodeNonExistent(t *testing.T) {
	g := New[testNode, string]()
	if g.Node(999) != nil {
		t.Error("expected nil for non-existent ID")
	}
}

// ── RemoveNode ──

func TestRemoveNode(t *testing.T) {
	g := New[testNode, string]()
	id := g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})
	g.RemoveNode(id)
	if g.Node(id) != nil {
		t.Error("node should be gone after RemoveNode")
	}
	if len(g.Nodes()) != 0 {
		t.Error("Nodes() should be empty")
	}
}

func TestRemoveNodeCleansEdges(t *testing.T) {
	g := New[testNode, string]()
	a := g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})
	b := g.AddNode(testNode{X: 10, Y: 0, W: 5, H: 3})
	c := g.AddNode(testNode{X: 20, Y: 0, W: 5, H: 3})
	g.AddEdge(a, b, "a→b")
	g.AddEdge(b, c, "b→c")
	g.AddEdge(a, c, "a→c")

	g.RemoveNode(b)
	edges := g.Edges()
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge remaining, got %d", len(edges))
	}
	if edges[0].FromID != a || edges[0].ToID != c {
		t.Errorf("expected a→c edge, got %d→%d", edges[0].FromID, edges[0].ToID)
	}
}

func TestRemoveNonExistentNode(t *testing.T) {
	g := New[testNode, string]()
	g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})
	g.RemoveNode(999) // should not panic
	if len(g.Nodes()) != 1 {
		t.Error("RemoveNode(999) should be a no-op")
	}
}

// ── MoveNode ──

func TestMoveNode(t *testing.T) {
	g := New[testNode, string]()
	id := g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})
	g.MoveNode(id, image.Pt(50, 50), setPos)
	n := g.Node(id)
	if n.Data.X != 50 || n.Data.Y != 50 {
		t.Errorf("after MoveNode: expected (50,50), got (%d,%d)", n.Data.X, n.Data.Y)
	}
}

// ── Edges ──

func TestAddEdge(t *testing.T) {
	g := New[testNode, string]()
	a := g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})
	b := g.AddNode(testNode{X: 10, Y: 0, W: 5, H: 3})
	g.AddEdge(a, b, "hello")

	edges := g.Edges()
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Data != "hello" {
		t.Errorf("edge data: expected 'hello', got %q", edges[0].Data)
	}
}

func TestAddEdgeDuplicate(t *testing.T) {
	g := New[testNode, string]()
	a := g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})
	b := g.AddNode(testNode{X: 10, Y: 0, W: 5, H: 3})
	g.AddEdge(a, b, "first")
	g.AddEdge(a, b, "second") // duplicate, should be ignored

	if len(g.Edges()) != 1 {
		t.Error("duplicate edge should be ignored")
	}
}

func TestRemoveEdge(t *testing.T) {
	g := New[testNode, string]()
	a := g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})
	b := g.AddNode(testNode{X: 10, Y: 0, W: 5, H: 3})
	g.AddEdge(a, b, "test")
	g.RemoveEdge(a, b)
	if len(g.Edges()) != 0 {
		t.Error("edge should be removed")
	}
}

func TestOutEdges(t *testing.T) {
	g := New[testNode, string]()
	a := g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})
	b := g.AddNode(testNode{X: 10, Y: 0, W: 5, H: 3})
	c := g.AddNode(testNode{X: 20, Y: 0, W: 5, H: 3})
	g.AddEdge(a, b, "a→b")
	g.AddEdge(a, c, "a→c")
	g.AddEdge(b, c, "b→c")

	out := g.OutEdges(a)
	if len(out) != 2 {
		t.Fatalf("expected 2 out-edges from a, got %d", len(out))
	}
}

func TestInEdges(t *testing.T) {
	g := New[testNode, string]()
	a := g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})
	b := g.AddNode(testNode{X: 10, Y: 0, W: 5, H: 3})
	c := g.AddNode(testNode{X: 20, Y: 0, W: 5, H: 3})
	g.AddEdge(a, c, "a→c")
	g.AddEdge(b, c, "b→c")

	in := g.InEdges(c)
	if len(in) != 2 {
		t.Fatalf("expected 2 in-edges to c, got %d", len(in))
	}
}

// ── HitTest ──

func TestHitTestInside(t *testing.T) {
	g := New[testNode, string]()
	g.AddNode(testNode{X: 10, Y: 10, W: 8, H: 4})
	hit := g.HitTest(image.Pt(14, 12))
	if hit == nil {
		t.Fatal("expected hit, got nil")
	}
}

func TestHitTestOutside(t *testing.T) {
	g := New[testNode, string]()
	g.AddNode(testNode{X: 10, Y: 10, W: 8, H: 4})
	hit := g.HitTest(image.Pt(0, 0))
	if hit != nil {
		t.Error("expected nil for miss")
	}
}

func TestHitTestTopmost(t *testing.T) {
	g := New[testNode, string]()
	g.AddNode(testNode{X: 10, Y: 10, W: 10, H: 10}) // id 0, bottom
	g.AddNode(testNode{X: 12, Y: 12, W: 10, H: 10}) // id 1, top (overlapping)
	// Point in overlap region
	hit := g.HitTest(image.Pt(15, 15))
	if hit == nil {
		t.Fatal("expected hit")
	}
	if hit.ID != 1 {
		t.Errorf("expected topmost (ID=1), got ID=%d", hit.ID)
	}
}

func TestHitTestEmptyGraph(t *testing.T) {
	g := New[testNode, string]()
	if g.HitTest(image.Pt(0, 0)) != nil {
		t.Error("empty graph should return nil")
	}
}

// ── NodesInRect ──

func TestNodesInRect(t *testing.T) {
	g := New[testNode, string]()
	g.AddNode(testNode{X: 0, Y: 0, W: 5, H: 3})   // in
	g.AddNode(testNode{X: 50, Y: 50, W: 5, H: 3})  // out
	g.AddNode(testNode{X: 8, Y: 0, W: 5, H: 3})    // in (overlaps query rect)

	r := image.Rect(0, 0, 10, 10)
	nodes := g.NodesInRect(r)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes in rect, got %d", len(nodes))
	}
}
