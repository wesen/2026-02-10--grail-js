package graphmodel

import "image"

// Node wraps a user-supplied data value with an integer ID.
type Node[N Spatial] struct {
	ID   int
	Data N
}

// Edge connects two nodes with a user-supplied label/data.
type Edge[E any] struct {
	FromID int
	ToID   int
	Data   E
}

// Graph is a generic spatial graph with stable insertion-order iteration.
type Graph[N Spatial, E any] struct {
	nodes    map[int]*Node[N]
	edges    []Edge[E]
	nextID   int
	orderIDs []int // insertion order for deterministic iteration
}

// New creates an empty graph.
func New[N Spatial, E any]() *Graph[N, E] {
	return &Graph[N, E]{
		nodes: make(map[int]*Node[N]),
	}
}

// ── Node operations ──

// AddNode inserts a node and returns its assigned ID.
func (g *Graph[N, E]) AddNode(data N) int {
	id := g.nextID
	g.nextID++
	g.nodes[id] = &Node[N]{ID: id, Data: data}
	g.orderIDs = append(g.orderIDs, id)
	return id
}

// Node returns a pointer to the node with the given ID, or nil.
func (g *Graph[N, E]) Node(id int) *Node[N] {
	return g.nodes[id]
}

// Nodes returns all nodes in insertion order.
func (g *Graph[N, E]) Nodes() []*Node[N] {
	result := make([]*Node[N], 0, len(g.orderIDs))
	for _, id := range g.orderIDs {
		if n, ok := g.nodes[id]; ok {
			result = append(result, n)
		}
	}
	return result
}

// RemoveNode deletes the node and all connected edges.
func (g *Graph[N, E]) RemoveNode(id int) {
	if _, ok := g.nodes[id]; !ok {
		return
	}
	delete(g.nodes, id)

	// Remove from orderIDs
	for i, oid := range g.orderIDs {
		if oid == id {
			g.orderIDs = append(g.orderIDs[:i], g.orderIDs[i+1:]...)
			break
		}
	}

	// Remove all connected edges
	filtered := g.edges[:0]
	for _, e := range g.edges {
		if e.FromID != id && e.ToID != id {
			filtered = append(filtered, e)
		}
	}
	g.edges = filtered
}

// MoveNode updates the position of a node. The caller provides a setter
// function since Go generics don't support interface setters cleanly.
func (g *Graph[N, E]) MoveNode(id int, pos image.Point, setPos func(*N, image.Point)) {
	if n, ok := g.nodes[id]; ok {
		setPos(&n.Data, pos)
	}
}

// ── Edge operations ──

// AddEdge adds an edge between two nodes. Duplicate (fromID, toID) pairs
// are silently ignored.
func (g *Graph[N, E]) AddEdge(fromID, toID int, data E) {
	for _, e := range g.edges {
		if e.FromID == fromID && e.ToID == toID {
			return
		}
	}
	g.edges = append(g.edges, Edge[E]{FromID: fromID, ToID: toID, Data: data})
}

// RemoveEdge removes the first edge matching (fromID, toID).
func (g *Graph[N, E]) RemoveEdge(fromID, toID int) {
	for i, e := range g.edges {
		if e.FromID == fromID && e.ToID == toID {
			g.edges = append(g.edges[:i], g.edges[i+1:]...)
			return
		}
	}
}

// Edges returns all edges.
func (g *Graph[N, E]) Edges() []Edge[E] {
	return g.edges
}

// OutEdges returns edges originating from the given node.
func (g *Graph[N, E]) OutEdges(fromID int) []Edge[E] {
	var result []Edge[E]
	for _, e := range g.edges {
		if e.FromID == fromID {
			result = append(result, e)
		}
	}
	return result
}

// InEdges returns edges terminating at the given node.
func (g *Graph[N, E]) InEdges(toID int) []Edge[E] {
	var result []Edge[E]
	for _, e := range g.edges {
		if e.ToID == toID {
			result = append(result, e)
		}
	}
	return result
}

// ── Spatial queries ──

// HitTest returns the topmost (last-inserted) node containing the point,
// or nil if no node contains it.
func (g *Graph[N, E]) HitTest(pt image.Point) *Node[N] {
	for i := len(g.orderIDs) - 1; i >= 0; i-- {
		n := g.nodes[g.orderIDs[i]]
		if n != nil && pt.In(BoundsOf(n.Data)) {
			return n
		}
	}
	return nil
}

// NodesInRect returns all nodes whose bounds intersect the given rectangle,
// in insertion order.
func (g *Graph[N, E]) NodesInRect(r image.Rectangle) []*Node[N] {
	var result []*Node[N]
	for _, id := range g.orderIDs {
		n := g.nodes[id]
		if n != nil && BoundsOf(n.Data).Overlaps(r) {
			result = append(result, n)
		}
	}
	return result
}
