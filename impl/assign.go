package impl

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-parser/parser"
)

func evalAssignmentExpression(e eval.Evaluator, expr *parser.AssignmentExpression) eval.Value {
	return assign(expr, e.Scope(), lvalue(expr.Lhs()), e.Eval(expr.Rhs()))
}

func assign(expr *parser.AssignmentExpression, scope eval.Scope, lv eval.Value, rv eval.Value) eval.Value {
	if sv, ok := lv.(*types.StringValue); ok {
		if !scope.Set(sv.String(), rv) {
			panic(evalError(eval.EVAL_ILLEGAL_REASSIGNMENT, expr, issue.H{`var`: sv.String()}))
		}
		return rv
	}

	names := lv.(*types.ArrayValue)
	switch rv.(type) {
	case *types.HashValue:
		h := rv.(*types.HashValue)
		r := make([]eval.Value, names.Len())
		names.EachWithIndex(func(name eval.Value, idx int) {
			v, ok := h.Get(name)
			if !ok {
				panic(evalError(eval.EVAL_MISSING_MULTI_ASSIGNMENT_KEY, expr, issue.H{`name`: name.String()}))
			}
			r[idx] = assign(expr, scope, name, v)
		})
		return types.WrapValues(r)
	case *types.ArrayValue:
		values := rv.(*types.ArrayValue)
		if names.Len() != values.Len() {
			panic(evalError(eval.EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE, expr, issue.H{`expected`: names.Len(), `actual`: values.Len()}))
		}
		names.EachWithIndex(func(name eval.Value, idx int) {
			assign(expr, scope, name, values.At(idx))
		})
		return rv
	default:
		panic(evalError(eval.EVAL_ILLEGAL_ASSIGNMENT, expr.Lhs(), issue.H{`value`: expr.Lhs()}))
	}
}

func lvalue(expr parser.Expression) eval.Value {
	switch expr.(type) {
	case *parser.VariableExpression:
		ve := expr.(*parser.VariableExpression)
		if name, ok := ve.Name(); ok {
			return types.WrapString(name)
		}
	case *parser.LiteralList:
		le := expr.(*parser.LiteralList).Elements()
		ev := make([]eval.Value, len(le))
		for idx, ex := range le {
			ev[idx] = lvalue(ex)
		}
		return types.WrapValues(ev)
	}
	panic(evalError(eval.EVAL_ILLEGAL_ASSIGNMENT, expr, issue.H{`value`: expr}))
}
