package impl

import (
	"strconv"

	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-parser/parser"
)

func evalArithmeticExpression(e eval.Evaluator, expr *parser.ArithmeticExpression) eval.Value {
	return calculate(expr, e.Eval(expr.Lhs()), e.Eval(expr.Rhs()))
}

func calculate(expr *parser.ArithmeticExpression, a eval.Value, b eval.Value) eval.Value {
	op := expr.Operator()
	switch a.(type) {
	case *types.HashValue, *types.ArrayValue, *types.UriValue:
		switch op {
		case `+`:
			return concatenate(expr, a, b)
		case `-`:
			return collectionDelete(expr, a, b)
		case `<<`:
			if av, ok := a.(*types.ArrayValue); ok {
				return av.Add(b)
			}
		}
	case *types.FloatValue:
		return lhsFloatArithmetic(expr, a.(*types.FloatValue).Float(), b)
	case *types.IntegerValue:
		return lhsIntArithmetic(expr, a.(*types.IntegerValue).Int(), b)
	case *types.StringValue:
		sv := a.(*types.StringValue)
		if iv, err := strconv.ParseInt(sv.String(), 0, 64); err == nil {
			return lhsIntArithmetic(expr, iv, b)
		}
		if fv, err := strconv.ParseFloat(sv.String(), 64); err == nil {
			return lhsFloatArithmetic(expr, fv, b)
		}
		panic(evalError(eval.EVAL_NOT_NUMERIC, expr.Lhs(), issue.H{`value`: sv.String()}))
	}
	panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
}

func lhsIntArithmetic(expr *parser.ArithmeticExpression, ai int64, b eval.Value) eval.Value {
	op := expr.Operator()
	switch b.(type) {
	case *types.IntegerValue:
		return types.WrapInteger(intArithmetic(expr, ai, b.(*types.IntegerValue).Int()))
	case *types.FloatValue:
		return types.WrapFloat(floatArithmetic(expr, float64(ai), b.(*types.FloatValue).Float()))
	case *types.StringValue:
		bv := b.(*types.StringValue)
		if iv, err := strconv.ParseInt(bv.String(), 0, 64); err == nil {
			return types.WrapInteger(intArithmetic(expr, ai, iv))
		}
		if fv, err := strconv.ParseFloat(bv.String(), 64); err == nil {
			return types.WrapFloat(floatArithmetic(expr, float64(ai), fv))
		}
		panic(evalError(eval.EVAL_NOT_NUMERIC, expr.Rhs(), issue.H{`value`: bv.String()}))
	default:
		panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, issue.H{`operator`: op, `left`: `Integer`, `right`: b.PType()}))
	}
}

func lhsFloatArithmetic(expr *parser.ArithmeticExpression, af float64, b eval.Value) eval.Value {
	op := expr.Operator()
	switch b.(type) {
	case *types.FloatValue:
		return types.WrapFloat(floatArithmetic(expr, af, b.(*types.FloatValue).Float()))
	case *types.IntegerValue:
		return types.WrapFloat(floatArithmetic(expr, af, float64(b.(*types.IntegerValue).Int())))
	case *types.StringValue:
		bv := b.(*types.StringValue)
		if iv, err := strconv.ParseInt(bv.String(), 0, 64); err == nil {
			return types.WrapFloat(floatArithmetic(expr, af, float64(iv)))
		}
		if fv, err := strconv.ParseFloat(bv.String(), 64); err == nil {
			return types.WrapFloat(floatArithmetic(expr, af, fv))
		}
		panic(evalError(eval.EVAL_NOT_NUMERIC, expr.Rhs(), issue.H{`value`: bv.String()}))
	default:
		panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, issue.H{`operator`: op, `left`: `Float`, `right`: b.PType()}))
	}
}

func floatArithmetic(expr *parser.ArithmeticExpression, a float64, b float64) float64 {
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
		panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: expr.Operator(), `left`: `Float`}))
	}
}

func intArithmetic(expr *parser.ArithmeticExpression, a int64, b int64) int64 {
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
		panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: expr.Operator(), `left`: `Integer`}))
	}
}

func concatenate(expr *parser.ArithmeticExpression, a eval.Value, b eval.Value) eval.Value {
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
					panic(evalError(eval.EVAL_ARGUMENTS_ERROR, expr, issue.H{`expression`: expr, `message`: err.(*errors.ArgumentsError).Error()}))
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
	panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, issue.H{`operator`: expr.Operator(), `left`: a.PType(), `right`: b.PType()}))
}

func collectionDelete(expr *parser.ArithmeticExpression, a eval.Value, b eval.Value) eval.Value {
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
		panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: expr.Operator(), `left`: a.PType}))
	}
}
