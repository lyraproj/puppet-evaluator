package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"sync"
	"gonum.org/v1/gonum/graph/iterator"
)

type (
	Graph interface {
		graph.Directed

		// Nodes returns all resource nodes, sorted by file, line, pos
		AllNodes() eval.List

		// RootNodes returns the root nodes, sorted by file, line, pos
		RootNodes() eval.List

		// Edges returns all edges extending from the given node, sorted by file, line, pos of the
		// node appointed by the edge
		Edges(from Node) eval.List

		// FromNode returns all nodes that the given node has edges to, sorted by file, line, pos
		FromNode(Node) eval.List

		// ToNode returns all nodes that has edges to the given node, sorted by file, line, pos
		ToNode(Node) eval.List
	}

	concurrentGraph struct {
		lock sync.RWMutex
		g    *simple.DirectedGraph
	}
)

func NewConcurrentGraph() graph.DirectedBuilder {
	return &concurrentGraph{g: simple.NewDirectedGraph()}
}

func (cg *concurrentGraph) Node(id int64) graph.Node {
	cg.lock.RLock()
	node := cg.g.Node(id)
	cg.lock.RUnlock()
	return node
}

func (cg *concurrentGraph) AllNodes() eval.List {
	return nodeList(cg.Nodes())
}

func (cg *concurrentGraph) Edges(from Node) eval.List {
	return cg.FromNode(from).Map(func(to eval.Value) eval.Value {
		return cg.Edge(from.ID(), to.(Node).ID()).(Edge)
	})
}

func (cg *concurrentGraph) RootNodes() eval.List {
	roots := make([]eval.Value, 0)
	cg.lock.RLock()
	ni := cg.g.Nodes()
	for ni.Next() {
		n := ni.Node()
		if cg.g.To(n.ID()).Len() == 0 {
			roots = append(roots, n.(eval.Value))
		}
	}
	cg.lock.RUnlock()
	return types.WrapValues(roots)
}

func (cg *concurrentGraph) FromNode(n Node) eval.List {
	return nodeList(cg.From(n.ID()))
}

func (cg *concurrentGraph) Nodes() graph.Nodes {
	cg.lock.RLock()
	nodes := copyIterator(cg.g.Nodes())
	cg.lock.RUnlock()
	return nodes
}

func copyIterator(nodes graph.Nodes) graph.Nodes {
	result := make([]graph.Node, nodes.Len())
	i := 0
	for nodes.Next() {
		result[i] = nodes.Node()
		i++
	}
	return iterator.NewOrderedNodes(result)
}

func (cg *concurrentGraph) From(id int64) graph.Nodes {
	cg.lock.RLock()
	nodes := copyIterator(cg.g.From(id))
	cg.lock.RUnlock()
	return nodes
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

func (cg *concurrentGraph) To(id int64) graph.Nodes {
	cg.lock.RLock()
	nodes := copyIterator(cg.g.To(id))
	cg.lock.RUnlock()
	return nodes
}

func (cg *concurrentGraph) ToNode(n Node) eval.List {
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
	return &edge{from, to, false}
}

func (cg *concurrentGraph) RemoveEdge(fid, tid int64) {
	cg.lock.Lock()
	defer cg.lock.Unlock()
	cg.g.RemoveEdge(fid, tid)
}

func (cg *concurrentGraph) RemoveNode(id int64) {
	cg.lock.Lock()
	defer cg.lock.Unlock()
	cg.g.RemoveNode(id)
}

func (cg *concurrentGraph) SetEdge(e graph.Edge) {
	cg.lock.Lock()
	defer cg.lock.Unlock()
	cg.g.SetEdge(e)
}

func nodeList(nodes graph.Nodes) eval.List {
	rs := make([]eval.Value, nodes.Len())
	i := 0
	for nodes.Next() {
		rs[i] = nodes.Node().(eval.Value)
		i++
	}
	return sortByLocation(rs)
}

func sortByLocation(nodes []eval.Value) eval.List {
	return types.WrapValues(nodes).Sort(func(a, b eval.Value) bool {
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
