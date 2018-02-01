package eval

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
	. "github.com/puppetlabs/go-parser/issue"
	. "github.com/puppetlabs/go-parser/parser"
)

func (e *evaluator) eval_AssignmentExpression(expr *AssignmentExpression, c EvalContext) PValue {
	return e.assign(expr, c.Scope(), e.lvalue(expr.Lhs()), e.Eval(expr.Rhs(), c))
}

func (e *evaluator) assign(expr *AssignmentExpression, scope Scope, lv PValue, rv PValue) PValue {
	if sv, ok := lv.(*StringValue); ok {
		if !scope.Set(sv.String(), rv) {
			panic(e.evalError(EVAL_ILLEGAL_REASSIGNMENT, expr, H{`var`: sv.String()}))
		}
		return rv
	}

	names := lv.(*ArrayValue)
	switch rv.(type) {
	case *HashValue:
		h := rv.(*HashValue)
		r := make([]PValue, names.Len())
		names.EachWithIndex(func(name PValue, idx int) {
			v, ok := h.Get(name)
			if !ok {
				panic(e.evalError(EVAL_MISSING_MULTI_ASSIGNMENT_KEY, expr, H{`name`: name.String()}))
			}
			r[idx] = e.assign(expr, scope, name, v)
		})
		return WrapArray(r)
	case *ArrayValue:
		values := rv.(*ArrayValue)
		if names.Len() != values.Len() {
			panic(e.evalError(EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE, expr, H{`expected`: names.Len(), `actual`: values.Len()}))
		}
		names.EachWithIndex(func(name PValue, idx int) {
			e.assign(expr, scope, name, values.At(idx))
		})
		return rv
	default:
		panic(e.evalError(EVAL_ILLEGAL_ASSIGNMENT, expr.Lhs(), H{`value`: expr.Lhs()}))
	}
}

func (e *evaluator) lvalue(expr Expression) PValue {
	switch expr.(type) {
	case *VariableExpression:
		ve := expr.(*VariableExpression)
		if name, ok := ve.Name(); ok {
			return WrapString(name)
		}
	case *LiteralList:
		le := expr.(*LiteralList).Elements()
		ev := make([]PValue, len(le))
		for idx, ex := range le {
			ev[idx] = e.lvalue(ex)
		}
		return WrapArray(ev)
	}
	panic(e.evalError(EVAL_ILLEGAL_ASSIGNMENT, expr, H{`value`: expr}))
}
