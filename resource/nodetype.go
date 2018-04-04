package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-parser/issue"
	"io"
	"github.com/puppetlabs/go-evaluator/types"
	"gonum.org/v1/gonum/graph"
)

var Node_Type eval.ObjectType

func init() {
	Node_Type = eval.NewObjectType(`ResourceNode`, `{
	attributes => {
		id => Integer,
		ref => String,
	  value => { type => Optional[Resource], value => undef }, 
	  resolved => { type => Boolean, kind => derived },
	}
}`)
}

type(
	Node interface {
		graph.Node
		eval.PValue

		Location() issue.Location

		Resolved() bool

		Value(c eval.EvalContext) eval.PuppetObject
	}

	// node represents a PuppetObject in the graph with a unique ID
	node struct {
		id int64
		ref string
		value eval.PuppetObject
		location issue.Location
	}
)

func (rn *node) ID() int64 {
	return rn.id
}

func (rn *node) Resolved() bool {
	return rn.value != nil
}

func (rn *node) Value(c eval.EvalContext) eval.PuppetObject {
	if rn.value == nil {
		name, title, _ := SplitRef(rn.ref)
		panic(eval.Error(c, EVAL_UNKNOWN_RESOURCE, issue.H {`type_name`: name, `title`: title}))
	}
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
	case `resolved`:
		return types.WrapBoolean(rn.Resolved()), true
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
	return rn.location
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
