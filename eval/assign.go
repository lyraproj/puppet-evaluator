package eval

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
	. "github.com/puppetlabs/go-parser/parser"
)

func (e *evaluator) eval_AssignmentExpression(expr *AssignmentExpression, c EvalContext) PValue {
	return e.assign(expr, c.Scope(), e.lvalue(expr.Lhs()), e.Eval(expr.Rhs(), c))
}

func (e *evaluator) assign(expr *AssignmentExpression, scope Scope, lv PValue, rv PValue) PValue {
	if sv, ok := lv.(*StringValue); ok {
		if !scope.Set(sv.String(), rv) {
			panic(e.evalError(EVAL_ILLEGAL_REASSIGNMENT, expr, sv.String()))
		}
		return rv
	}

	names := lv.(*ArrayValue).Elements()
	switch rv.(type) {
	case *HashValue:
		h := rv.(*HashValue)
		r := make([]PValue, len(names))
		for idx, name := range names {
			v, ok := h.Get(name)
			if !ok {
				panic(e.evalError(EVAL_MISSING_MULTI_ASSIGNMENT_KEY, expr, name.String()))
			}
			r[idx] = e.assign(expr, scope, name, v)
		}
		return WrapArray(r)
	case *ArrayValue:
		values := rv.(*ArrayValue).Elements()
		if len(names) != len(values) {
			panic(e.evalError(EVAL_ILLEGAL_MULTI_ASSIGNMENT_SIZE, expr, len(names), len(values)))
		}
		for idx, name := range names {
			e.assign(expr, scope, name, values[idx])
		}
		return rv
	default:
		panic(e.evalError(EVAL_ILLEGAL_ASSIGNMENT, expr.Lhs(), A_an(expr.Lhs())))
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
	panic(e.evalError(EVAL_ILLEGAL_ASSIGNMENT, expr, A_an(expr)))
}
