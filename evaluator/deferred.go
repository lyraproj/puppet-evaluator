package evaluator

import (
	"io"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

var deferredExprType px.ObjectType

func init() {
	deferredExprType = px.NewObjectType(`DeferredExpression`, `{}`)

	types.DeferredResolve = func(e types.Deferred, c px.Context) px.Value {
		fn := e.Name()
		da := e.Arguments()

		var args []px.Value
		if fn[0] == '$' {
			vn := fn[1:]
			vv, ok := c.(pdsl.EvaluationContext).Scope().Get(vn)
			if !ok {
				panic(px.Error(pdsl.UnknownVariable, issue.H{`name`: vn}))
			}
			if da.Len() == 0 {
				// No point digging with zero arguments
				return vv
			}
			fn = `dig`
			args = append(make([]px.Value, 0, 1+da.Len()), vv)
		} else {
			args = make([]px.Value, 0, da.Len())
		}
		args = da.AppendTo(args)
		for i, a := range args {
			args[i] = types.ResolveDeferred(c, a)
		}
		return px.Call(c, fn, args, nil)
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
	return px.ToString(d)
}

func (d *deferredExpr) Equals(other interface{}, guard px.Guard) bool {
	return d == other
}

func (d *deferredExpr) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	utils.WriteString(b, `DeferredExpression(`)
	utils.PuppetQuote(b, d.expression.String())
	utils.WriteString(b, `)`)
}

func (d *deferredExpr) PType() px.Type {
	return deferredExprType
}

func (d *deferredExpr) Resolve(c px.Context) px.Value {
	return pdsl.Evaluate(c.(pdsl.EvaluationContext), d.expression)
}

func objectTypeFromAST(e pdsl.Evaluator, name string, parent px.Type, initHashExpression *parser.LiteralHash) px.ObjectType {
	return types.NewParentedObjectType(name, parent, deferLiteralHash(e, initHashExpression))
}

func typeSetTypeFromAST(e pdsl.Evaluator, na px.URI, name string, initHashExpression *parser.LiteralHash) px.TypeSet {
	return types.NewTypeSet(na, name, deferLiteralHash(e, initHashExpression))
}

func deferToTypeName(expr parser.Expression) string {
	switch expr := expr.(type) {
	case *parser.QualifiedReference:
		return expr.Name()
	default:
		panic(px.Error2(expr, pdsl.IllegalWhenStaticExpression, issue.H{`expression`: expr}))
	}
}

func deferToArray(e pdsl.Evaluator, array []parser.Expression) []px.Value {
	result := make([]px.Value, 0, len(array))
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

func deferLiteralHash(e pdsl.Evaluator, expr *parser.LiteralHash) px.OrderedMap {
	entries := expr.Entries()
	top := len(entries)
	if top == 0 {
		return px.EmptyMap
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
				panic(px.Error2(op, px.DuplicateKey, issue.H{`key`: `parent`}))
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

func deferAny(e pdsl.Evaluator, expr parser.Expression) px.Value {
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
		return px.Undef
	case *parser.ResourceDefaultsExpression:
		return deferParentedObject(e, expr)
	case *parser.TextExpression:
		return types.WrapString(deferAny(e, expr.Expr()).String())
	default:
		panic(px.Error2(expr, pdsl.IllegalWhenStaticExpression, issue.H{`expression`: expr}))
	}
}
