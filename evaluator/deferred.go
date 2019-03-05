package evaluator

import (
	"io"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

var deferredExprType eval.ObjectType

func init() {
	deferredExprType = eval.NewObjectType(`DeferredExpression`, `{}`)

	types.DeferredResolve = func(e types.Deferred, c eval.Context) eval.Value {
		fn := e.Name()
		da := e.Arguments()

		var args []eval.Value
		if fn[0] == '$' {
			vn := fn[1:]
			vv, ok := c.(pdsl.EvaluationContext).Scope().Get(vn)
			if !ok {
				panic(eval.Error(pdsl.UnknownVariable, issue.H{`name`: vn}))
			}
			if da.Len() == 0 {
				// No point digging with zero arguments
				return vv
			}
			fn = `dig`
			args = append(make([]eval.Value, 0, 1+da.Len()), vv)
		} else {
			args = make([]eval.Value, 0, da.Len())
		}
		args = da.AppendTo(args)
		for i, a := range args {
			args[i] = types.ResolveDeferred(c, a)
		}
		return eval.Call(c, fn, args, nil)
	}
}

func newDeferredExpression(expression parser.Expression) types.Deferred {
	return &deferredExpr{expression}
}

type deferredExpr struct {
	expression parser.Expression
}

func (d *deferredExpr) Name() string {
	return ``
}

func (d *deferredExpr) Arguments() *types.ArrayValue {
	return nil
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
	return pdsl.Evaluate(c.(pdsl.EvaluationContext), d.expression)
}

func objectTypeFromAST(e pdsl.Evaluator, name string, parent eval.Type, initHashExpression *parser.LiteralHash) eval.ObjectType {
	return types.NewParentedObjectType(name, parent, deferLiteralHash(e, initHashExpression))
}

func typeSetTypeFromAST(e pdsl.Evaluator, na eval.URI, name string, initHashExpression *parser.LiteralHash) eval.TypeSet {
	return types.NewTypeSet(na, name, deferLiteralHash(e, initHashExpression))
}

func deferToTypeName(expr parser.Expression) string {
	switch expr := expr.(type) {
	case *parser.QualifiedReference:
		return expr.Name()
	default:
		panic(eval.Error2(expr, pdsl.IllegalWhenStaticExpression, issue.H{`expression`: expr}))
	}
}

func deferToArray(e pdsl.Evaluator, array []parser.Expression) []eval.Value {
	result := make([]eval.Value, 0, len(array))
	for _, ex := range array {
		ex = unwindParenthesis(ex)
		if u, ok := ex.(*parser.UnfoldExpression); ok {
			ev := deferAny(e, u.Expr())
			switch ev := ev.(type) {
			case *types.ArrayValue:
				result = ev.AppendTo(result)
			default:
				result = append(result, ev)
			}
		} else {
			result = append(result, deferAny(e, ex))
		}
	}
	return result
}

func deferLiteralHash(e pdsl.Evaluator, expr *parser.LiteralHash) eval.OrderedMap {
	entries := expr.Entries()
	top := len(entries)
	if top == 0 {
		return eval.EmptyMap
	}
	result := make([]*types.HashEntry, top)
	for i, en := range entries {
		result[i] = deferKeyedEntry(e, en.(*parser.KeyedEntry))
	}
	return types.WrapHash(result)
}

func deferKeyedEntry(e pdsl.Evaluator, expr *parser.KeyedEntry) *types.HashEntry {
	return types.WrapHashEntry(deferAny(e, expr.Key()), deferAny(e, expr.Value()))
}

func deferParentedObject(e pdsl.Evaluator, expr *parser.ResourceDefaultsExpression) *types.DeferredType {
	// This is actually a <Parent> { <key-value entries> } notation. Convert to a literal
	// hash that contains the parent
	entries := make([]*types.HashEntry, 0)
	name := deferToTypeName(expr.TypeRef())
	if name != `Object` && name != `TypeSet` {
		// the name `parent` is not allowed here
		for _, op := range expr.Operations() {
			if op.(issue.Named).Name() == `parent` {
				panic(eval.Error2(op, eval.DuplicateKey, issue.H{`key`: `parent`}))
			}
		}
		entries = append(entries, types.WrapHashEntry2(`parent`, types.WrapString(name)))
		name = `Object`
	}
	for _, op := range expr.Operations() {
		if ao, ok := op.(*parser.AttributeOperation); ok {
			entries = append(entries, types.WrapHashEntry2(ao.Name(), deferAny(e, ao.Value())))
		}
	}
	return types.NewDeferredType(name, types.WrapHash(entries))
}

func deferAny(e pdsl.Evaluator, expr parser.Expression) eval.Value {
	switch expr := expr.(type) {
	case *parser.AccessExpression:
		return types.NewDeferredType(deferToTypeName(expr.Operand()), deferToArray(e, expr.Keys())...)
	case *parser.CallNamedFunctionExpression:
		return types.NewDeferred(deferToTypeName(expr.Functor()), deferToArray(e, expr.Arguments())...)
	case *parser.HeredocExpression:
		return evalHeredocExpression(e, expr)
	case *parser.KeyedEntry:
		return deferKeyedEntry(e, expr)
	case *parser.LiteralHash:
		return deferLiteralHash(e, expr)
	case *parser.LiteralList:
		return types.WrapValues(deferToArray(e, expr.Elements()))
	case *parser.QualifiedName:
		return evalQualifiedName(expr)
	case *parser.QualifiedReference:
		return types.NewDeferredType(expr.Name())
	case *parser.RegexpExpression:
		return evalRegexpExpression(expr)
	case *parser.LiteralBoolean:
		return evalLiteralBoolean(expr)
	case *parser.LiteralDefault:
		return evalLiteralDefault()
	case *parser.LiteralFloat:
		return evalLiteralFloat(expr)
	case *parser.LiteralInteger:
		return evalLiteralInteger(expr)
	case *parser.LiteralString:
		return evalLiteralString(expr)
	case *parser.LiteralUndef, *parser.Nop:
		return eval.Undef
	case *parser.ResourceDefaultsExpression:
		return deferParentedObject(e, expr)
	case *parser.TextExpression:
		return types.WrapString(deferAny(e, expr.Expr()).String())
	default:
		panic(eval.Error2(expr, pdsl.IllegalWhenStaticExpression, issue.H{`expression`: expr}))
	}
}
