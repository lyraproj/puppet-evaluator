package impl

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-parser/parser"
)

func evalAssignmentExpression(e eval.Evaluator, expr *parser.AssignmentExpression) eval.Value {
	return assign(expr, e.Scope(), lvalue(expr.Lhs()), e.Eval(expr.Rhs()))
}

func assign(expr *parser.AssignmentExpression, scope eval.Scope, lv eval.Value, rv eval.Value) eval.Value {
	if sv, ok := lv.(eval.StringValue); ok {
		if !scope.Set(sv.String(), rv) {
			panic(evalError(eval.IllegalReassignment, expr, issue.H{`var`: sv.String()}))
		}
		return rv
	}

	names := lv.(*types.ArrayValue)
	switch rv := rv.(type) {
	case *types.HashValue:
		r := make([]eval.Value, names.Len())
		names.EachWithIndex(func(name eval.Value, idx int) {
			v, ok := rv.Get(name)
			if !ok {
				panic(evalError(eval.MissingMultiAssignmentKey, expr, issue.H{`name`: name.String()}))
			}
			r[idx] = assign(expr, scope, name, v)
		})
		return types.WrapValues(r)
	case *types.ArrayValue:
		if names.Len() != rv.Len() {
			panic(evalError(eval.IllegalMultiAssignmentSize, expr, issue.H{`expected`: names.Len(), `actual`: rv.Len()}))
		}
		names.EachWithIndex(func(name eval.Value, idx int) {
			assign(expr, scope, name, rv.At(idx))
		})
		return rv
	default:
		panic(evalError(eval.IllegalAssignment, expr.Lhs(), issue.H{`value`: expr.Lhs()}))
	}
}

func lvalue(expr parser.Expression) eval.Value {
	switch expr := expr.(type) {
	case *parser.VariableExpression:
		if name, ok := expr.Name(); ok {
			return types.WrapString(name)
		}
	case *parser.LiteralList:
		le := expr.Elements()
		ev := make([]eval.Value, len(le))
		for idx, ex := range le {
			ev[idx] = lvalue(ex)
		}
		return types.WrapValues(ev)
	}
	panic(evalError(eval.IllegalAssignment, expr, issue.H{`value`: expr}))
}
