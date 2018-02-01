package impl

import (
	"strconv"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/eval"
	. "github.com/puppetlabs/go-evaluator/types"
	. "github.com/puppetlabs/go-parser/issue"
	. "github.com/puppetlabs/go-parser/parser"
)

func (e *evaluator) eval_ArithmeticExpression(expr *ArithmeticExpression, c EvalContext) PValue {
	return e.calculate(expr, e.eval(expr.Lhs(), c), e.eval(expr.Rhs(), c))
}

func (e *evaluator) calculate(expr *ArithmeticExpression, a PValue, b PValue) PValue {
	op := expr.Operator()
	switch a.(type) {
	case *HashValue, *ArrayValue:
		switch op {
		case `+`:
			return e.concatenate(expr, a, b)
		case `-`:
			return e.collectionDelete(expr, a, b)
		case `<<`:
			if av, ok := a.(*ArrayValue); ok {
				return av.Add(b)
			}
		}
	case *FloatValue:
		return e.lhsFloatArithmetic(expr, a.(*FloatValue).Float(), b)
	case *IntegerValue:
		return e.lhsIntArithmetic(expr, a.(*IntegerValue).Int(), b)
	case *StringValue:
		sv := a.(*StringValue)
		if iv, err := strconv.ParseInt(sv.String(), 0, 64); err == nil {
			return e.lhsIntArithmetic(expr, iv, b)
		}
		if fv, err := strconv.ParseFloat(sv.String(), 64); err == nil {
			return e.lhsFloatArithmetic(expr, fv, b)
		}
		panic(e.evalError(EVAL_NOT_NUMERIC, expr.Lhs(), H{`value`: sv.String()}))
	}
	panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, H{`operator`: op, `left`: a.Type()}))
}

func (e *evaluator) lhsIntArithmetic(expr *ArithmeticExpression, ai int64, b PValue) PValue {
	op := expr.Operator()
	switch b.(type) {
	case *IntegerValue:
		return WrapInteger(e.intArithmetic(expr, ai, b.(*IntegerValue).Int()))
	case *FloatValue:
		return WrapFloat(e.floatArithmetic(expr, float64(ai), b.(*FloatValue).Float()))
	case *StringValue:
		bv := b.(*StringValue)
		if iv, err := strconv.ParseInt(bv.String(), 0, 64); err == nil {
			return WrapInteger(e.intArithmetic(expr, ai, iv))
		}
		if fv, err := strconv.ParseFloat(bv.String(), 64); err == nil {
			return WrapFloat(e.floatArithmetic(expr, float64(ai), fv))
		}
		panic(e.evalError(EVAL_NOT_NUMERIC, expr.Rhs(), H{`value`: bv.String()}))
	default:
		panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, H{`operator`: op, `left`: `Integer`, `right`: b.Type()}))
	}
}

func (e *evaluator) lhsFloatArithmetic(expr *ArithmeticExpression, af float64, b PValue) PValue {
	op := expr.Operator()
	switch b.(type) {
	case *FloatValue:
		return WrapFloat(e.floatArithmetic(expr, af, b.(*FloatValue).Float()))
	case *IntegerValue:
		return WrapFloat(e.floatArithmetic(expr, af, float64(b.(*IntegerValue).Int())))
	case *StringValue:
		bv := b.(*StringValue)
		if iv, err := strconv.ParseInt(bv.String(), 0, 64); err == nil {
			return WrapFloat(e.floatArithmetic(expr, af, float64(iv)))
		}
		if fv, err := strconv.ParseFloat(bv.String(), 64); err == nil {
			return WrapFloat(e.floatArithmetic(expr, af, fv))
		}
		panic(e.evalError(EVAL_NOT_NUMERIC, expr.Rhs(), H{`value`: bv.String()}))
	default:
		panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, H{`operator`: op, `left`: `Float`, `right`: b.Type()}))
	}
}

func (e *evaluator) floatArithmetic(expr *ArithmeticExpression, a float64, b float64) float64 {
	switch expr.Operator() {
	case `+`:
		return a + b
	case `-`:
		return a - b
	case `*`:
		return a * b
	case `/`:
		return a / b
	default:
		panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, H{`operator`: expr.Operator(), `left`: `Float`}))
	}
}

func (e *evaluator) intArithmetic(expr *ArithmeticExpression, a int64, b int64) int64 {
	switch expr.Operator() {
	case `+`:
		return a + b
	case `-`:
		return a - b
	case `*`:
		return a * b
	case `/`:
		return a / b
	case `%`:
		return a % b
	case `<<`:
		return a << uint(b)
	case `>>`:
		return a >> uint(b)
	default:
		panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, H{`operator`: expr.Operator(), `left`: `Integer`}))
	}
}

func (e *evaluator) concatenate(expr *ArithmeticExpression, a PValue, b PValue) PValue {
	switch a.(type) {
	case *ArrayValue:
		av := a.(*ArrayValue)
		switch b.(type) {
		case *ArrayValue:
			return av.AddAll(b.(*ArrayValue))

		case *HashValue:
			return av.AddAll(b.(*HashValue))

		default:
			return av.Add(b)
		}
	case *HashValue:
		switch b.(type) {
		case *ArrayValue:
			defer func() {
				err := recover()
				switch err.(type) {
				case nil:
				case *ArgumentsError:
					panic(e.evalError(EVAL_ARGUMENTS_ERROR, expr, H{`expression`: expr, `message`: err.(*ArgumentsError).Error()}))
				default:
					panic(err)
				}
			}()
			return a.(*HashValue).Merge(WrapHashFromArray(b.(*ArrayValue)))
		case *HashValue:
			return a.(*HashValue).Merge(b.(*HashValue))
		}
	}
	panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, H{`operator`: expr.Operator(), `left`: a.Type(), `right`: b.Type()}))
}

func (e *evaluator) collectionDelete(expr *ArithmeticExpression, a PValue, b PValue) PValue {
	switch a.(type) {
	case *ArrayValue:
		av := a.(*ArrayValue)
		switch b.(type) {
		case *ArrayValue:
			return av.DeleteAll(b.(*ArrayValue))
		case *HashValue:
			return av.DeleteAll(b.(*HashValue))
		default:
			return av.Delete(b)
		}
	case *HashValue:
		hv := a.(*HashValue)
		switch b.(type) {
		case *ArrayValue:
			return hv.DeleteAll(b.(*ArrayValue))
		case *HashValue:
			return hv.DeleteAll(b.(*HashValue).Keys())
		default:
			return hv.Delete(b)
		}
	default:
		panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, H{`operator`: expr.Operator(), `left`: a.Type}))
	}
}
