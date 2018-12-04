package impl

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
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

func (p *parameter) HasValue() bool {
	return p.value != nil
}

func (p *parameter) Name() string {
	return p.name
}

func (p *parameter) Value() eval.Value {
	return p.value
}

func (p *parameter) Type() eval.Type {
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
		return p.Value(), true
	case `has_value`:
		return types.WrapBoolean(p.value != nil), true
	case `captures_rest`:
		return types.WrapBoolean(p.captures), true
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
	if p.value == eval.UNDEF {
		es = append(es, types.WrapHashEntry2(`has_value`, types.Boolean_TRUE))
	}
	if p.captures {
		es = append(es, types.WrapHashEntry2(`captures_rest`, types.Boolean_TRUE))
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

func (p *parameter) PType() eval.Type {
	return Parameter_Type
}

func init() {
	Parameter_Type = eval.NewObjectType(`Parameter`, `{
    attributes => {
      'name' => String,
      'type' => Type,
      'has_value' => { type => Boolean, value => false },
      'value' => { type => Variant[Deferred,Data], value => undef },
      'captures_rest' => { type => Boolean, value => false },
    }
  }`, func(ctx eval.Context, args []eval.Value) eval.Value {
		n := args[0].(*types.StringValue).String()
		t := args[1].(eval.Type)
		h := false
		if len(args) > 2 {
			h = args[2].(*types.BooleanValue).Bool()
		}
		var v eval.Value
		if len(args) > 3 {
			v = args[3]
		}
		c := false
		if len(args) > 4 {
			c = args[4].(*types.BooleanValue).Bool()
		}
		if h && v == nil {
			v = eval.UNDEF
		}
		return NewParameter(n, t, v, c)
	}, func(ctx eval.Context, args []eval.Value) eval.Value {
		h := args[0].(*types.HashValue)
		n := h.Get5(`name`, eval.EMPTY_STRING).(*types.StringValue).String()
		t := h.Get5(`type`, types.DefaultDataType()).(eval.Type)
		var v eval.Value
		if x, ok := h.Get4(`value`); ok {
			v = x
		}
		hv := false
		if x, ok := h.Get4(`has_value`); ok {
			hv = x.(*types.BooleanValue).Bool()
		}
		c := false
		if x, ok := h.Get4(`captures_rest`); ok {
			c = x.(*types.BooleanValue).Bool()
		}
		if hv && v == nil {
			v = eval.UNDEF
		}
		return NewParameter(n, t, v, c)
	})
}
