package impl

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
)

func (e *evaluator) eval_AssignmentExpression(expr *parser.AssignmentExpression, c eval.Context) eval.PValue {
	return e.assign(expr, c.Scope(), e.lvalue(expr.Lhs()), e.eval(expr.Rhs(), c))
}

func (e *evaluator) assign(expr *parser.AssignmentExpression, scope eval.Scope, lv eval.PValue, rv eval.PValue) eval.PValue {
	if sv, ok := lv.(*types.StringValue); ok {
		if !scope.Set(sv.String(), rv) {
			panic(e.evalError(eval.EVAL_ILLEGAL_REASSIGNMENT, expr, issue.H{`var`: sv.String()}))
		}
		return rv
	}

	names := lv.(*types.ArrayValue)
	switch rv.(type) {
	case *types.HashValue:
		h := rv.(*types.HashValue)
		r := make([]eval.PValue, names.Len())
		names.EachWithIndex(func(name eval.PValue, idx int) {
			v, ok := h.Get(name)
			if !ok {
				panic(e.evalError(eval.EVAL_MISSING_MULTI_ASSIGNMENT_KEY, expr, issue.H{`name`: name.String()}))
			}
			r[idx] = e.assign(expr, scope, name, v)
		})
		return types.WrapArray(r)
	case *types.ArrayValue:
		values := rv.(*types.ArrayValue)
		if names.Len() != values.Len() {
			panic(e.evalError(eval.EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE, expr, issue.H{`expected`: names.Len(), `actual`: values.Len()}))
		}
		names.EachWithIndex(func(name eval.PValue, idx int) {
			e.assign(expr, scope, name, values.At(idx))
		})
		return rv
	default:
		panic(e.evalError(eval.EVAL_ILLEGAL_ASSIGNMENT, expr.Lhs(), issue.H{`value`: expr.Lhs()}))
	}
}

func (e *evaluator) lvalue(expr parser.Expression) eval.PValue {
	switch expr.(type) {
	case *parser.VariableExpression:
		ve := expr.(*parser.VariableExpression)
		if name, ok := ve.Name(); ok {
			return types.WrapString(name)
		}
	case *parser.LiteralList:
		le := expr.(*parser.LiteralList).Elements()
		ev := make([]eval.PValue, len(le))
		for idx, ex := range le {
			ev[idx] = e.lvalue(ex)
		}
		return types.WrapArray(ev)
	}
	panic(e.evalError(eval.EVAL_ILLEGAL_ASSIGNMENT, expr, issue.H{`value`: expr}))
}
