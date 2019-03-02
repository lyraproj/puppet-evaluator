package types

import (
	"io"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/utils"
	"github.com/lyraproj/puppet-parser/parser"
)

var DeferredMetaType eval.ObjectType
var deferredExprType eval.ObjectType

func init() {
	DeferredMetaType = newObjectType(`Deferred`, `{
    attributes => {
      # Fully qualified name of the function
      name  => { type => Pattern[/\A[$]?[a-z][0-9A-Za-z_]*(?:::[a-z][0-9A-Za-z_]*)*\z/] },
      arguments => { type => Optional[Array[Any]], value => undef},
    }}`,
		func(ctx eval.Context, args []eval.Value) eval.Value {
			return newDeferred2(args...)
		},
		func(ctx eval.Context, args []eval.Value) eval.Value {
			return newDeferredFromHash(args[0].(*HashValue))
		})

	// For internal use only
	deferredExprType = newObjectType(`DeferredExpression`, `{}`)
}

type Deferred interface {
	eval.Value

	Resolve(c eval.Context) eval.Value
}

type deferred struct {
	name      string
	arguments *ArrayValue
}

func NewDeferred(name string, arguments ...eval.Value) *deferred {
	return &deferred{name, WrapValues(arguments)}
}

func newDeferred2(args ...eval.Value) *deferred {
	argc := len(args)
	if argc < 1 || argc > 2 {
		panic(errors.NewIllegalArgumentCount(`deferred[]`, `1 - 2`, argc))
	}
	if name, ok := args[0].(stringValue); ok {
		if argc == 1 {
			return &deferred{string(name), emptyArray}
		}
		if as, ok := args[1].(*ArrayValue); ok {
			return &deferred{string(name), as}
		}
		panic(NewIllegalArgumentType(`deferred[]`, 1, `Array`, args[1]))
	}
	panic(NewIllegalArgumentType(`deferred[]`, 0, `String`, args[0]))
}

func newDeferredFromHash(hash *HashValue) *deferred {
	name := hash.Get5(`name`, eval.EmptyString).String()
	arguments := hash.Get5(`arguments`, eval.EmptyArray).(*ArrayValue)
	return &deferred{name, arguments}
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

func (e *deferred) PType() eval.Type {
	return DeferredMetaType
}

func (e *deferred) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `name`:
		return stringValue(e.name), true
	case `arguments`:
		return e.arguments, true
	}
	return nil, false
}

func (e *deferred) InitHash() eval.OrderedMap {
	return WrapHash([]*HashEntry{WrapHashEntry2(`name`, stringValue(e.name)), WrapHashEntry2(`arguments`, e.arguments)})
}

func (e *deferred) Resolve(c eval.Context) eval.Value {
	fn := e.name

	var args []eval.Value
	if fn[0] == '$' {
		vn := fn[1:]
		vv, ok := c.(eval.EvaluationContext).Scope().Get(vn)
		if !ok {
			panic(eval.Error(eval.UnknownVariable, issue.H{`name`: vn}))
		}
		if e.arguments.Len() == 0 {
			// No point digging with zero arguments
			return vv
		}
		fn = `dig`
		args = append(make([]eval.Value, 0, 1+e.arguments.Len()), vv)
	} else {
		args = make([]eval.Value, 0, e.arguments.Len())
	}
	args = e.arguments.AppendTo(args)
	for i, a := range args {
		args[i] = ResolveDeferred(c, a)
	}
	return eval.Call(c, fn, args, nil)
}

// ResolveDeferred will resolve all occurrences of a DeferredValue in its
// given argument. Array and Hash arguments will be resolved recursively.
func ResolveDeferred(c eval.Context, a eval.Value) eval.Value {
	switch a := a.(type) {
	case Deferred:
		return a.Resolve(c)
	case *ArrayValue:
		return a.Map(func(v eval.Value) eval.Value {
			return ResolveDeferred(c, v)
		})
	case *HashValue:
		return a.MapEntries(func(v eval.MapEntry) eval.MapEntry {
			return WrapHashEntry(ResolveDeferred(c, v.Key()), ResolveDeferred(c, v.Value()))
		})
	default:
		return a
	}
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
	utils.WriteString(b, `DeferredExpression(`)
	utils.PuppetQuote(b, d.expression.String())
	utils.WriteString(b, `)`)
}

func (d *deferredExpr) PType() eval.Type {
	return deferredExprType
}

func (d *deferredExpr) Resolve(c eval.Context) eval.Value {
	return eval.Evaluate(c.(eval.EvaluationContext), d.expression)
}
