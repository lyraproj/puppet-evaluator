package impl

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"io"
)

type parameter struct {
	name  string
	typ   eval.Type
	value eval.Value
	captures bool
}

func NewParameter(name string, typ eval.Type, value eval.Value, capturesRest bool) eval.Parameter {
	return &parameter{name, typ, value, capturesRest}
}

func (p *parameter) Name() string {
	return p.name
}

func (p *parameter) Value() eval.Value {
	return p.value
}

func (p *parameter) ValueType() eval.Type {
	return p.typ
}

func (p *parameter) CapturesRest() bool {
	return p.captures
}

func (p *parameter) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `name`:
		return types.WrapString(p.name), true
	case `type`:
		return p.typ, true
	case `value`:
		return p.value, true
	}
	return nil, false
}

func (p *parameter) InitHash() eval.OrderedMap {
	es := make([]*types.HashEntry, 0, 3)
	es = append(es, types.WrapHashEntry2(`name`, types.WrapString(p.name)))
	es = append(es, types.WrapHashEntry2(`type`, p.typ))
	if p.value != nil {
		es = append(es, types.WrapHashEntry2(`value`, p.value))
	}
	return types.WrapHash(es)
}

var Parameter_Type eval.Type

func (p *parameter) Equals(other interface{}, guard eval.Guard) bool {
	return p == other
}

func (p *parameter) String() string {
	return eval.ToString(p)
}

func (p *parameter) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(p, format, bld, g)
}

func (p *parameter) Type() eval.Type {
	return Parameter_Type
}

func init() {
	Parameter_Type = eval.NewObjectType(`Parameter`, `{
    attributes => {
      'name' => String,
      'type' => Type,
      'value' => { type => Variant[Deferred,Data], value => undef },
      'captures_rest' => { type => Boolean, value => false },
    }
  }`, func(ctx eval.Context, args []eval.Value) eval.Value {
		n := args[0].(*types.StringValue).String()
		t := args[1].(eval.Type)
		l := eval.UNDEF
		if len(args) > 2 {
			l = args[2]
		}
		c := false
		if len(args) > 3 {
			c = args[3].(*types.BooleanValue).Bool()
		}
		return NewParameter(n, t, l, c)
	}, func(ctx eval.Context, args []eval.Value) eval.Value {
		h := args[0].(*types.HashValue)
		n := h.Get5(`name`, eval.EMPTY_STRING).(*types.StringValue).String()
		t := h.Get5(`type`, types.DefaultDataType()).(eval.Type)
		l := eval.UNDEF
		if x, ok := h.Get4(`value`); ok {
			l = x
		}
		c := false
		if x, ok := h.Get4(`captures_rest`); ok {
			c = x.(*types.BooleanValue).Bool()
		}
		return NewParameter(n, t, l, c)
	})
}
