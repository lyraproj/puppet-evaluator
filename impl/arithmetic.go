package impl

import (
	"strconv"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-parser/parser"
)

func (e *evaluator) eval_ArithmeticExpression(expr *parser.ArithmeticExpression, c eval.EvalContext) eval.PValue {
	return e.calculate(expr, e.eval(expr.Lhs(), c), e.eval(expr.Rhs(), c))
}

func (e *evaluator) calculate(expr *parser.ArithmeticExpression, a eval.PValue, b eval.PValue) eval.PValue {
	op := expr.Operator()
	switch a.(type) {
	case *types.HashValue, *types.ArrayValue, *types.UriValue:
		switch op {
		case `+`:
			return e.concatenate(expr, a, b)
		case `-`:
			return e.collectionDelete(expr, a, b)
		case `<<`:
			if av, ok := a.(*types.ArrayValue); ok {
				return av.Add(b)
			}
		}
	case *types.FloatValue:
		return e.lhsFloatArithmetic(expr, a.(*types.FloatValue).Float(), b)
	case *types.IntegerValue:
		return e.lhsIntArithmetic(expr, a.(*types.IntegerValue).Int(), b)
	case *types.StringValue:
		sv := a.(*types.StringValue)
		if iv, err := strconv.ParseInt(sv.String(), 0, 64); err == nil {
			return e.lhsIntArithmetic(expr, iv, b)
		}
		if fv, err := strconv.ParseFloat(sv.String(), 64); err == nil {
			return e.lhsFloatArithmetic(expr, fv, b)
		}
		panic(e.evalError(eval.EVAL_NOT_NUMERIC, expr.Lhs(), issue.H{`value`: sv.String()}))
	}
	panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.Type()}))
}

func (e *evaluator) lhsIntArithmetic(expr *parser.ArithmeticExpression, ai int64, b eval.PValue) eval.PValue {
	op := expr.Operator()
	switch b.(type) {
	case *types.IntegerValue:
		return types.WrapInteger(e.intArithmetic(expr, ai, b.(*types.IntegerValue).Int()))
	case *types.FloatValue:
		return types.WrapFloat(e.floatArithmetic(expr, float64(ai), b.(*types.FloatValue).Float()))
	case *types.StringValue:
		bv := b.(*types.StringValue)
		if iv, err := strconv.ParseInt(bv.String(), 0, 64); err == nil {
			return types.WrapInteger(e.intArithmetic(expr, ai, iv))
		}
		if fv, err := strconv.ParseFloat(bv.String(), 64); err == nil {
			return types.WrapFloat(e.floatArithmetic(expr, float64(ai), fv))
		}
		panic(e.evalError(eval.EVAL_NOT_NUMERIC, expr.Rhs(), issue.H{`value`: bv.String()}))
	default:
		panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, issue.H{`operator`: op, `left`: `Integer`, `right`: b.Type()}))
	}
}

func (e *evaluator) lhsFloatArithmetic(expr *parser.ArithmeticExpression, af float64, b eval.PValue) eval.PValue {
	op := expr.Operator()
	switch b.(type) {
	case *types.FloatValue:
		return types.WrapFloat(e.floatArithmetic(expr, af, b.(*types.FloatValue).Float()))
	case *types.IntegerValue:
		return types.WrapFloat(e.floatArithmetic(expr, af, float64(b.(*types.IntegerValue).Int())))
	case *types.StringValue:
		bv := b.(*types.StringValue)
		if iv, err := strconv.ParseInt(bv.String(), 0, 64); err == nil {
			return types.WrapFloat(e.floatArithmetic(expr, af, float64(iv)))
		}
		if fv, err := strconv.ParseFloat(bv.String(), 64); err == nil {
			return types.WrapFloat(e.floatArithmetic(expr, af, fv))
		}
		panic(e.evalError(eval.EVAL_NOT_NUMERIC, expr.Rhs(), issue.H{`value`: bv.String()}))
	default:
		panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, issue.H{`operator`: op, `left`: `Float`, `right`: b.Type()}))
	}
}

func (e *evaluator) floatArithmetic(expr *parser.ArithmeticExpression, a float64, b float64) float64 {
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
		panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: expr.Operator(), `left`: `Float`}))
	}
}

func (e *evaluator) intArithmetic(expr *parser.ArithmeticExpression, a int64, b int64) int64 {
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
		panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: expr.Operator(), `left`: `Integer`}))
	}
}

func (e *evaluator) concatenate(expr *parser.ArithmeticExpression, a eval.PValue, b eval.PValue) eval.PValue {
	switch a.(type) {
	case *types.ArrayValue:
		av := a.(*types.ArrayValue)
		switch b.(type) {
		case *types.ArrayValue:
			return av.AddAll(b.(*types.ArrayValue))

		case *types.HashValue:
			return av.AddAll(b.(*types.HashValue))

		default:
			return av.Add(b)
		}
	case *types.HashValue:
		switch b.(type) {
		case *types.ArrayValue:
			defer func() {
				err := recover()
				switch err.(type) {
				case nil:
				case *errors.ArgumentsError:
					panic(e.evalError(eval.EVAL_ARGUMENTS_ERROR, expr, issue.H{`expression`: expr, `message`: err.(*errors.ArgumentsError).Error()}))
				default:
					panic(err)
				}
			}()
			return a.(*types.HashValue).Merge(types.WrapHashFromArray(b.(*types.ArrayValue)))
		case *types.HashValue:
			return a.(*types.HashValue).Merge(b.(*types.HashValue))
		}
	case *types.UriValue:
		switch b.(type) {
		case *types.StringValue:
			return types.WrapURI(a.(*types.UriValue).URL().ResolveReference(types.ParseURI(b.String())))
		case *types.UriValue:
			return types.WrapURI(a.(*types.UriValue).URL().ResolveReference(b.(*types.UriValue).URL()))
		}
	}
	panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, issue.H{`operator`: expr.Operator(), `left`: a.Type(), `right`: b.Type()}))
}

func (e *evaluator) collectionDelete(expr *parser.ArithmeticExpression, a eval.PValue, b eval.PValue) eval.PValue {
	switch a.(type) {
	case *types.ArrayValue:
		av := a.(*types.ArrayValue)
		switch b.(type) {
		case *types.ArrayValue:
			return av.DeleteAll(b.(*types.ArrayValue))
		case *types.HashValue:
			return av.DeleteAll(b.(*types.HashValue))
		default:
			return av.Delete(b)
		}
	case *types.HashValue:
		hv := a.(*types.HashValue)
		switch b.(type) {
		case *types.ArrayValue:
			return hv.DeleteAll(b.(*types.ArrayValue))
		case *types.HashValue:
			return hv.DeleteAll(b.(*types.HashValue).Keys())
		default:
			return hv.Delete(b)
		}
	default:
		panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: expr.Operator(), `left`: a.Type}))
	}
}
