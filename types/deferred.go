package types

import (
	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"io"
)

var deferredType eval.ObjectType
var deferredExprType eval.ObjectType

func init() {
	deferredType = newObjectType(`Deferred`, `{
    attributes => {
      # Fully qualified name of the function
      name  => { type => Pattern[/\A[$]?[a-z][a-z0-9_]*(?:::[a-z][a-z0-9_]*)*\z/] },
      arguments => { type => Optional[Array[Any]], value => undef},
    }}`,
		func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return NewDeferred2(ctx, args...)
		},
		func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return newDeferredFromHash(ctx, args[0].(*HashValue))
		})

	// For internal use only
	deferredExprType = newObjectType(`DeferredExpression`, `{}`)
}

type Deferred interface {
	eval.PValue

	Resolve(c eval.Context) eval.PValue
}

type deferred struct {
	name      string
  arguments *ArrayValue
}

func NewDeferred(name string, arguments ...eval.PValue) *deferred {
	return &deferred{name, WrapArray(arguments)}
}

func NewDeferred2(c eval.Context, args ...eval.PValue) *deferred {
	argc := len(args)
	if argc < 1 || argc > 2 {
		panic(errors.NewIllegalArgumentCount(`deferred[]`, `1 - 2`, argc))
	}
	if name, ok := args[0].(*StringValue); ok {
    if argc == 1 {
			return &deferred{name.String(), _EMPTY_ARRAY}
		}
		if as, ok := args[1].(*ArrayValue); ok {
			return &deferred{name.String(), as}
		}
		panic(NewIllegalArgumentType2(`deferred[]`, 1, `Array`, args[1]))
	}
	panic(NewIllegalArgumentType2(`deferred[]`, 0, `String`, args[0]))
}

func newDeferredFromHash(c eval.Context, hash *HashValue) *deferred {
	ev := &deferred{}
	ev.name = hash.Get5(`name`, eval.EMPTY_STRING).String()
	ev.arguments = hash.Get5(`arguments`, eval.EMPTY_ARRAY).(*ArrayValue)
	return ev
}

func (e *deferred) Name() string {
	return e.name
}

func (e *deferred) Arguments() *ArrayValue {
	return e.arguments
}

func (e *deferred) String() string {
	return eval.ToString(e)
}

func (e *deferred) Equals(other interface{}, guard eval.Guard) bool {
	if o, ok := other.(*deferred); ok {
		return e.name == o.name &&
			eval.GuardedEquals(e.arguments, o.arguments, guard)
	}
	return false
}

func (e *deferred) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	ObjectToString(e, s, b, g)
}

func (e *deferred) Type() eval.PType {
	return deferredType
}

func (e *deferred) Get(key string) (value eval.PValue, ok bool) {
	switch key {
	case `name`:
		return WrapString(e.name), true
	case `arguments`:
		return e.arguments, true
	}
	return nil, false
}

func (e *deferred) InitHash() eval.KeyedValue {
	return WrapHash([]*HashEntry{WrapHashEntry2(`name`, WrapString(e.name)), WrapHashEntry2(`arguments`, e.arguments)})
}

func (e *deferred) Resolve(c eval.Context) eval.PValue {
	fn := e.name

	var args []eval.PValue
	if fn[0] == '$' {
		vn := fn[1:]
		vv, ok := c.Scope().Get(vn)
		if !ok {
			panic(eval.Error(eval.EVAL_UNKNOWN_VARIABLE, issue.H{`name`: fn}))
		}
		if e.arguments.Len() == 0 {
			// No point digging with zero arguments
			return vv
		}
		fn = `dig`
		args = append(make([]eval.PValue, 0, 1 + e.arguments.Len()), vv)
	} else {
		args = make([]eval.PValue, 0, e.arguments.Len())
	}
	args = e.arguments.AppendTo(args)
	for i, a := range args {
		args[i] = ResolveDeferred(c, a)
	}
	return eval.Call(c, fn, args, nil)
}

// ResolveDeferred will resolve all occurences of a DeferredValue in its
// given argument. Array and Hash arguments will be resolved recursively.
func ResolveDeferred(c eval.Context, a eval.PValue) eval.PValue {
	switch a.(type) {
	case Deferred:
		a = a.(Deferred).Resolve(c)
	case *ArrayValue:
		a = a.(*ArrayValue).Map(func(v eval.PValue) eval.PValue {
			return ResolveDeferred(c, v)
		})
	case *HashValue:
		a = a.(*HashValue).MapEntries(func(v eval.EntryValue) eval.EntryValue {
			return WrapHashEntry(ResolveDeferred(c, v.Key()), ResolveDeferred(c, v.Value()))
		})
	}
	return a
}

func NewDeferredExpression(expression parser.Expression) Deferred {
	return &deferredExpr{expression}
}

type deferredExpr struct {
	expression parser.Expression
}

func (d *deferredExpr) String() string {
	return eval.ToString(d)
}

func (d *deferredExpr) Equals(other interface{}, guard eval.Guard) bool {
	return d == other
}

func (d *deferredExpr) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	io.WriteString(b, `DeferredExpression(`)
	utils.PuppetQuote(b, d.expression.String())
	io.WriteString(b, `)`)
}

func (d *deferredExpr) Type() eval.PType {
	return deferredExprType
}

func (d *deferredExpr) Resolve(c eval.Context) eval.PValue {
	return eval.Evaluate(c, d.expression)
}

