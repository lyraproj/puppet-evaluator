package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"sync"
)

type (
	Graph interface {
		graph.Directed

		// Nodes returns all resource nodes, sorted by file, line, pos
		AllNodes() eval.IndexedValue

		// RootNodes returns the root nodes, sorted by file, line, pos
		RootNodes() eval.IndexedValue

		// Edges returns all edges extending from the given node, sorted by file, line, pos of the
		// node appointed by the edge
		Edges(from Node) eval.IndexedValue

		// FromNode returns all nodes that the given node has edges to, sorted by file, line, pos
		FromNode(Node) eval.IndexedValue

		// ToNode returns all nodes that has edges to the given node, sorted by file, line, pos
		ToNode(Node) eval.IndexedValue
	}

	concurrentGraph struct {
		lock sync.RWMutex
		g    *simple.DirectedGraph
	}
)

func NewConcurrentGraph() graph.DirectedBuilder {
	return &concurrentGraph{g: simple.NewDirectedGraph()}
}

func (cg *concurrentGraph) Has(id int64) bool {
	cg.lock.RLock()
	result := cg.g.Has(id)
	cg.lock.RUnlock()
	return result
}

func (cg *concurrentGraph) AllNodes() eval.IndexedValue {
	return nodeList(cg.Nodes())
}

func (cg *concurrentGraph) Edges(from Node) eval.IndexedValue {
	return cg.FromNode(from).Map(func(to eval.PValue) eval.PValue {
		return cg.Edge(from.ID(), to.(Node).ID()).(Edge)
	})
}

func (cg *concurrentGraph) RootNodes() eval.IndexedValue {
	roots := make([]graph.Node, 0)
	cg.lock.RLock()
	for _, n := range cg.g.Nodes() {
		if len(cg.g.To(n.ID())) == 0 {
			roots = append(roots, n)
		}
	}
	cg.lock.RUnlock()
	return nodeList(roots)
}

func (cg *concurrentGraph) FromNode(n Node) eval.IndexedValue {
	return nodeList(cg.From(n.ID()))
}

func (cg *concurrentGraph) Nodes() []graph.Node {
	cg.lock.RLock()
	nodes := cg.g.Nodes()
	result := make([]graph.Node, len(nodes))
	copy(result, nodes)
	cg.lock.RUnlock()
	return result
}

func (cg *concurrentGraph) From(id int64) []graph.Node {
	cg.lock.RLock()
	nodes := cg.g.From(id)
	result := make([]graph.Node, len(nodes))
	copy(result, nodes)
	cg.lock.RUnlock()
	return result
}

func (cg *concurrentGraph) HasEdgeBetween(xid, yid int64) bool {
	cg.lock.RLock()
	result := cg.g.HasEdgeBetween(xid, yid)
	cg.lock.RUnlock()
	return result
}

func (cg *concurrentGraph) Edge(uid, vid int64) graph.Edge {
	cg.lock.RLock()
	result := cg.g.Edge(uid, vid)
	cg.lock.RUnlock()
	return result
}

func (cg *concurrentGraph) HasEdgeFromTo(uid, vid int64) bool {
	cg.lock.RLock()
	result := cg.g.HasEdgeFromTo(uid, vid)
	cg.lock.RUnlock()
	return result
}

func (cg *concurrentGraph) To(id int64) []graph.Node {
	cg.lock.RLock()
	nodes := cg.g.To(id)
	result := make([]graph.Node, len(nodes))
	copy(result, nodes)
	cg.lock.RUnlock()
	return result
}

func (cg *concurrentGraph) ToNode(n Node) eval.IndexedValue {
	return nodeList(cg.To(n.ID()))
}

func (cg *concurrentGraph) NewNode() graph.Node {
	cg.lock.Lock()
	n := cg.g.NewNode()
	cg.lock.Unlock()
	return &node{id: n.ID(), resolved: make(chan bool)}
}

func (cg *concurrentGraph) AddNode(n graph.Node) {
	cg.lock.Lock()
	defer cg.lock.Unlock()
	cg.g.AddNode(n)
}

func (cg *concurrentGraph) NewEdge(from, to graph.Node) graph.Edge {
	return &edge{from.(*node), to.(*node), false}
}

func (cg *concurrentGraph) RemoveEdge(fid, tid int64) {
	cg.lock.Lock()
	defer cg.lock.Unlock()
	cg.g.RemoveEdge(fid, tid)
}

func (cg *concurrentGraph) SetEdge(e graph.Edge) {
	cg.lock.Lock()
	defer cg.lock.Unlock()
	cg.g.SetEdge(e)
}

func nodeList(nodes []graph.Node) eval.IndexedValue {
	rs := make([]eval.PValue, len(nodes))
	for i, n := range nodes {
		rs[i] = n.(eval.PValue)
	}
	return sortByLocation(rs)
}

func sortByLocation(nodes []eval.PValue) eval.IndexedValue {
	return types.WrapArray(nodes).Sort(func(a, b eval.PValue) bool {
		l1 := a.(issue.Located).Location()
		l2 := b.(issue.Located).Location()
		if l1.File() == l2.File() {
			ld := l1.Line() - l2.Line()
			if ld == 0 {
				return l1.Pos() < l2.Pos()
			}
			return ld < 0
		}
		return l1.File() < l2.File()
	})
}
