package containers

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrNodeNotFound = errors.New("node not found")
)

type Node[S, T comparable] struct {
	Value  T
	In     []*Edge[S, T]
	Out    []*Edge[S, T]
	though *Node[S, T]
}

func (n *Node[S, T]) String() string {
	return fmt.Sprintf("[%v]", n.Value)
}

type Edge[S, T comparable] struct {
	From  *Node[S, T]
	To    *Node[S, T]
	Value S
}

func (e Edge[S, T]) String() string {
	return fmt.Sprintf("%v --[%v]--> %v", e.From.Value, e.Value, e.To.Value)
}

/*
This is a weak implementation of a directed graph. It *does not* detect backedges so
you must be very careful when you define your state graph to not add backedges
*/
type Graph[S, T comparable] struct {
	Nodes []*Node[S, T]
	lock  sync.Mutex
}

// New returns a new Graph with the specified root node.
func NewGraph[S, T comparable]() *Graph[S, T] {
	g := &Graph[S, T]{Nodes: make([]*Node[S, T], 0)}
	return g
}

// CreateNode returns the Node for value, creating it if not present.
func (g *Graph[S, T]) CreateNode(value T) *Node[S, T] {
	for _, n := range g.Nodes {
		if n.Value == value {
			return n
		}
	}
	n := &Node[S, T]{Value: value}
	g.Nodes = append(g.Nodes, n)
	return n
}

func (g *Graph[S, T]) GetNode(value T) *Node[S, T] {
	// from the starting point
	for _, n := range g.Nodes {
		if n.Value == value {
			return n
		}
	}
	return nil
}

// AddEdge adds the edge to graph.
// Elimination of duplicate edges is the source node's responsibility.
func (g *Graph[S, T]) AddEdge(from *Node[S, T], to *Node[S, T], value S) {
	e := &Edge[S, T]{From: from, To: to, Value: value}
	to.In = append(to.In, e)
	from.Out = append(from.Out, e)
}

func (g *Graph[S, T]) FindPath(from T, to T) ([][]map[T]S, error) {
	start := g.GetNode(from)
	if start == nil {
		return nil, fmt.Errorf("%w start node value: %v", ErrNodeNotFound, from)
	}

	end := g.GetNode(to)
	if end == nil {
		return nil, fmt.Errorf("%w end node value: %v", ErrNodeNotFound, to)
	}

	if start.Value == end.Value {
		return nil, nil
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	g.reset()
	found := [][]*Node[S, T]{}
	g.findPath(start, start, end, nil, nil, &found)

	ret := [][]map[T]S{}
	for _, f := range found {
		path := make([]map[T]S, 0)
		i := 0
		for {
			start := f[i]
			if i+1 >= len(f) {
				var none S
				path = append(path, map[T]S{start.Value: none})
				break
			}
			end := f[i+1]
			for _, edge := range start.Out {
				if edge.To == end {
					path = append(path, map[T]S{start.Value: edge.Value})
					break
				}
			}
			i++
		}
		ret = append(ret, path)
	}

	return ret, nil
}

func (g *Graph[S, T]) reset() {
	// from the starting point
	for _, n := range g.Nodes {
		n.though = nil
	}
}

func (g *Graph[S, T]) findPath(point, start, end *Node[S, T], path map[*Node[S, T]][]*Node[S, T], visit map[*Node[S, T]]bool, found *[][]*Node[S, T]) {
	if visit == nil {
		visit = make(map[*Node[S, T]]bool)
	}
	if visit[start] {
		return
	}

	if path == nil {
		path = make(map[*Node[S, T]][]*Node[S, T])
	}

	if start == end {
		// we found the path. start walking backwards to the beginning
		p := []*Node[S, T]{end}
		n := end.though
		for n != nil {
			p = append(p, n)
			n = n.though
		}
		if len(p) == 1 {
			*found = append(*found, p)
			return
		}
		// reverse p in place;
		i := len(p) - 2
		for i >= 0 {
			p = append(p, p[i])
			i--
		}
		*found = append(*found, p[len(p)/2:])
		return
	}

	visit[start] = true
	for _, edge := range start.Out {
		if p, ok := path[start.though]; ok {
			p = append(p, start)
			path[start] = p
		} else {
			path[start] = []*Node[S, T]{start}
		}

		if edge.To != point { // avoid cycles
			edge.To.though = start
			g.findPath(point, edge.To, end, path, visit, found)
		}
	}
	visit[start] = false
}
