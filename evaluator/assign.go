package evaluator

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

func evalAssignmentExpression(e pdsl.Evaluator, expr *parser.AssignmentExpression) px.Value {
	return assign(expr, e.Scope(), lvalue(expr.Lhs()), e.Eval(expr.Rhs()))
}

func assign(expr *parser.AssignmentExpression, scope pdsl.Scope, lv px.Value, rv px.Value) px.Value {
	if sv, ok := lv.(px.StringValue); ok {
		if !scope.Set(sv.String(), rv) {
			panic(evalError(pdsl.IllegalReassignment, expr, issue.H{`var`: sv.String()}))
		}
		return rv
	}

	names := lv.(*types.ArrayValue)
	switch rv := rv.(type) {
	case *types.HashValue:
		r := make([]px.Value, names.Len())
		names.EachWithIndex(func(name px.Value, idx int) {
			v, ok := rv.Get(name)
			if !ok {
				panic(evalError(pdsl.MissingMultiAssignmentKey, expr, issue.H{`name`: name.String()}))
			}
			r[idx] = assign(expr, scope, name, v)
		})
		return types.WrapValues(r)
	case *types.ArrayValue:
		if names.Len() != rv.Len() {
			panic(evalError(pdsl.IllegalMultiAssignmentSize, expr, issue.H{`expected`: names.Len(), `actual`: rv.Len()}))
		}
		names.EachWithIndex(func(name px.Value, idx int) {
			assign(expr, scope, name, rv.At(idx))
		})
		return rv
	default:
		panic(evalError(pdsl.IllegalAssignment, expr.Lhs(), issue.H{`value`: expr.Lhs()}))
	}
}

func lvalue(expr parser.Expression) px.Value {
	switch expr := expr.(type) {
	case *parser.VariableExpression:
		if name, ok := expr.Name(); ok {
			return types.WrapString(name)
		}
	case *parser.LiteralList:
		le := expr.Elements()
		ev := make([]px.Value, len(le))
		for idx, ex := range le {
			ev[idx] = lvalue(ex)
		}
		return types.WrapValues(ev)
	}
	panic(evalError(pdsl.IllegalAssignment, expr, issue.H{`value`: expr}))
}
