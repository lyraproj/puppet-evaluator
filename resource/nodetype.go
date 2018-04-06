package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-parser/issue"
	"io"
	"github.com/puppetlabs/go-evaluator/types"
	"gonum.org/v1/gonum/graph"
	"github.com/puppetlabs/go-parser/parser"
	"sync"
)

var Node_Type eval.ObjectType

func init() {
	Node_Type = eval.NewObjectType(`ResourceNode`, `{
	attributes => {
		id => Integer,
		ref => String,
	  value => Any, 
	}
}`)
}

type(
	// A Node represents a Future value. It can be evaluated once all edges
	// pointing to it has been evaluated
	Node interface {
		graph.Node
		eval.PValue

		Location() issue.Location

		// Resolved returns true if all promises has been fulfilled for this
		// value. In essence, that means that this node is not appointed by
		// edges from any unresolved nodes.
		Resolved(c eval.Context) bool

		Value(c eval.Context) eval.PValue
	}

	// node represents a PuppetObject in the graph with a unique ID
	node struct {
		id int64
		lock sync.Mutex
		ref string
		value eval.PValue
		resources map[string]*handle
		expression parser.Expression
	}
)

func (rn *node) ID() int64 {
	return rn.id
}

func (rn *node) Resolved(c eval.Context) bool {
	return rn.value != nil
}

func (rn *node) Value(c eval.Context) eval.PValue {
	rn.lock.Lock()
	defer rn.lock.Unlock()
	if rn.value != nil {
		return rn.value
	}

	if rn.expression == nil {
		name, title, _ := SplitRef(rn.ref)
		panic(eval.Error(c, EVAL_UNKNOWN_RESOURCE, issue.H {`type_name`: name, `title`: title}))
	}

	before := c.Evaluator().(Evaluator).Graph().To(rn.ID())
	count := len(before)
	if count > 0 {
		done := make(chan bool, count)
		for _, bn := range before {
			go func() {
				bn.(*node).Value(c.Fork())
				done <- true
			}()
		}

		// Wait for count done's to arrive
		for i := 0; i < count; i++ {
			<-done
		}
	}
	rn.value = c.Evaluate(rn.expression)
	return rn.value
}

func (rn *node) Equals(other interface{}, guard eval.Guard) bool {
	on, ok := other.(*node)
	return ok && rn.id == on.id
}

func (rn *node) Get(key string) (value eval.PValue, ok bool) {
	switch key {
	case `id`:
		return types.WrapInteger(rn.id), true
	case `ref`:
		return types.WrapString(rn.ref), true
	case `value`:
		if rn.value == nil {
			return eval.UNDEF, false
		}
		return rn.value, true
	}
	return eval.UNDEF, false
}

func (rn *node) InitHash() eval.KeyedValue {
	hash := map[string]eval.PValue {
		`id`: types.WrapInteger(rn.id),
		`ref`: types.WrapString(rn.ref),
	}
	if rn.value != nil {
		hash[`value`] = rn.value
	}
	return types.WrapHash3(hash)
}

func (rn *node) Location() issue.Location {
	return rn.expression
}

func (rn *node) String() string {
	return eval.ToString2(rn, types.NONE)
}

func (rn *node) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(rn, s, b, g)
}

func (rn *node) Type() eval.PType {
	return Node_Type
}
