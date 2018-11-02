package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"gonum.org/v1/gonum/graph"
	"io"
)

var Edge_Type eval.ObjectType

func init() {
	Edge_Type = eval.NewObjectType(`ResourceEdge`, `{
	attributes => {
		from => ResourceNode,
		to => ResourceNode,
		subscribe => Boolean,
	}
}`)
}

type (
	Edge interface {
		graph.Edge
		eval.Value
		Subscribe() bool
		Value() eval.Value
	}

	// edge denotes a relationship between two ResourceNodes
	edge struct {
		from      graph.Node
		to        graph.Node
		subscribe bool
	}
)

func newEdge(from, to Node, subscribe bool) Edge {
	return &edge{from, to, subscribe}
}

func (re *edge) Equals(other interface{}, guard eval.Guard) bool {
	if oe, ok := other.(*edge); ok {
		return re.from.ID() == oe.from.ID() && re.to.ID() == oe.to.ID()
	}
	return false
}

func (re *edge) From() graph.Node {
	return re.from
}

func (re *edge) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `from`:
		return re.from.(eval.Value), true
	case `to`:
		return re.to.(eval.Value), true
	case `subscribe`:
		return types.WrapBoolean(re.subscribe), true
	}
	return eval.UNDEF, false
}

func (re *edge) InitHash() eval.OrderedMap {
	return types.WrapHashSorted(map[string]eval.Value{
		`from`:      re.from.(eval.Value),
		`to`:        re.to.(eval.Value),
		`subscribe`: types.WrapBoolean(re.subscribe),
	})
}

func (re *edge) String() string {
	return eval.ToString2(re, types.NONE)
}

func (re *edge) Subscribe() bool {
	return re.subscribe
}

func (re *edge) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(re, s, b, g)
}

func (re *edge) To() graph.Node {
	return re.to
}

func (re *edge) PType() eval.Type {
	return Edge_Type
}

func (re *edge) Value() eval.Value {
	return re.to.(Node).Value()
}
