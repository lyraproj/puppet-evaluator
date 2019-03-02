package impl

import (
	"strconv"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-parser/parser"
)

func evalArithmeticExpression(e eval.Evaluator, expr *parser.ArithmeticExpression) eval.Value {
	return calculate(expr, e.Eval(expr.Lhs()), e.Eval(expr.Rhs()))
}

func calculate(expr *parser.ArithmeticExpression, a eval.Value, b eval.Value) eval.Value {
	op := expr.Operator()
	switch a := a.(type) {
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
	case eval.FloatValue:
		return lhsFloatArithmetic(expr, a.Float(), b)
	case eval.IntegerValue:
		return lhsIntArithmetic(expr, a.Int(), b)
	case eval.StringValue:
		s := a.String()
		if iv, err := strconv.ParseInt(s, 0, 64); err == nil {
			return lhsIntArithmetic(expr, iv, b)
		}
		if fv, err := strconv.ParseFloat(s, 64); err == nil {
			return lhsFloatArithmetic(expr, fv, b)
		}
		panic(evalError(eval.NotNumeric, expr.Lhs(), issue.H{`value`: s}))
	}
	panic(evalError(eval.OperatorNotApplicable, expr, issue.H{`operator`: op, `left`: a.PType()}))
}

func lhsIntArithmetic(expr *parser.ArithmeticExpression, ai int64, b eval.Value) eval.Value {
	op := expr.Operator()
	switch b := b.(type) {
	case eval.IntegerValue:
		return types.WrapInteger(intArithmetic(expr, ai, b.Int()))
	case eval.FloatValue:
		return types.WrapFloat(floatArithmetic(expr, float64(ai), b.Float()))
	case eval.StringValue:
		s := b.String()
		if iv, err := strconv.ParseInt(s, 0, 64); err == nil {
			return types.WrapInteger(intArithmetic(expr, ai, iv))
		}
		if fv, err := strconv.ParseFloat(s, 64); err == nil {
			return types.WrapFloat(floatArithmetic(expr, float64(ai), fv))
		}
		panic(evalError(eval.NotNumeric, expr.Rhs(), issue.H{`value`: s}))
	default:
		panic(evalError(eval.OperatorNotApplicableWhen, expr, issue.H{`operator`: op, `left`: `Integer`, `right`: b.PType()}))
	}
}

func lhsFloatArithmetic(expr *parser.ArithmeticExpression, af float64, b eval.Value) eval.Value {
	op := expr.Operator()
	switch b := b.(type) {
	case eval.FloatValue:
		return types.WrapFloat(floatArithmetic(expr, af, b.Float()))
	case eval.IntegerValue:
		return types.WrapFloat(floatArithmetic(expr, af, float64(b.Int())))
	case eval.StringValue:
		s := b.String()
		if iv, err := strconv.ParseInt(s, 0, 64); err == nil {
			return types.WrapFloat(floatArithmetic(expr, af, float64(iv)))
		}
		if fv, err := strconv.ParseFloat(s, 64); err == nil {
			return types.WrapFloat(floatArithmetic(expr, af, fv))
		}
		panic(evalError(eval.NotNumeric, expr.Rhs(), issue.H{`value`: s}))
	default:
		panic(evalError(eval.OperatorNotApplicableWhen, expr, issue.H{`operator`: op, `left`: `Float`, `right`: b.PType()}))
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
		panic(evalError(eval.OperatorNotApplicable, expr, issue.H{`operator`: expr.Operator(), `left`: `Float`}))
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
		panic(evalError(eval.OperatorNotApplicable, expr, issue.H{`operator`: expr.Operator(), `left`: `Integer`}))
	}
}

func concatenate(expr *parser.ArithmeticExpression, a eval.Value, b eval.Value) eval.Value {
	switch a := a.(type) {
	case *types.ArrayValue:
		switch b := b.(type) {
		case *types.ArrayValue:
			return a.AddAll(b)

		case *types.HashValue:
			return a.AddAll(b)

		default:
			return a.Add(b)
		}
	case *types.HashValue:
		switch b := b.(type) {
		case *types.ArrayValue:
			defer func() {
				err := recover()
				switch err := err.(type) {
				case nil:
				case *errors.ArgumentsError:
					panic(evalError(eval.ArgumentsError, expr, issue.H{`expression`: expr, `message`: err.Error()}))
				default:
					panic(err)
				}
			}()
			return a.Merge(types.WrapHashFromArray(b))
		case *types.HashValue:
			return a.Merge(b)
		}
	case *types.UriValue:
		switch b := b.(type) {
		case eval.StringValue:
			return types.WrapURI(a.URL().ResolveReference(types.ParseURI(b.String())))
		case *types.UriValue:
			return types.WrapURI(a.URL().ResolveReference(b.URL()))
		}
	}
	panic(evalError(eval.OperatorNotApplicableWhen, expr, issue.H{`operator`: expr.Operator(), `left`: a.PType(), `right`: b.PType()}))
}

func collectionDelete(expr *parser.ArithmeticExpression, a eval.Value, b eval.Value) eval.Value {
	switch a := a.(type) {
	case *types.ArrayValue:
		switch b := b.(type) {
		case *types.ArrayValue:
			return a.DeleteAll(b)
		case *types.HashValue:
			return a.DeleteAll(b)
		default:
			return a.Delete(b)
		}
	case *types.HashValue:
		switch b := b.(type) {
		case *types.ArrayValue:
			return a.DeleteAll(b)
		case *types.HashValue:
			return a.DeleteAll(b.Keys())
		default:
			return a.Delete(b)
		}
	default:
		panic(evalError(eval.OperatorNotApplicable, expr, issue.H{`operator`: expr.Operator(), `left`: a.PType}))
	}
}
