package evaluator

import (
	"strconv"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

func evalArithmeticExpression(e pdsl.Evaluator, expr *parser.ArithmeticExpression) px.Value {
	return calculate(expr, e.Eval(expr.Lhs()), e.Eval(expr.Rhs()))
}

func calculate(expr *parser.ArithmeticExpression, a px.Value, b px.Value) px.Value {
	op := expr.Operator()
	switch a := a.(type) {
	case *types.Hash, *types.Array, *types.UriValue:
		switch op {
		case `+`:
			return concatenate(expr, a, b)
		case `-`:
			return collectionDelete(expr, a, b)
		case `<<`:
			if av, ok := a.(*types.Array); ok {
				return av.Add(b)
			}
		}
	case px.Float:
		return lhsFloatArithmetic(expr, a.Float(), b)
	case px.Integer:
		return lhsIntArithmetic(expr, a.Int(), b)
	case px.StringValue:
		s := a.String()
		if iv, err := strconv.ParseInt(s, 0, 64); err == nil {
			return lhsIntArithmetic(expr, iv, b)
		}
		if fv, err := strconv.ParseFloat(s, 64); err == nil {
			return lhsFloatArithmetic(expr, fv, b)
		}
		panic(evalError(pdsl.NotNumeric, expr.Lhs(), issue.H{`value`: s}))
	}
	panic(evalError(pdsl.OperatorNotApplicable, expr, issue.H{`operator`: op, `left`: a.PType()}))
}

func lhsIntArithmetic(expr *parser.ArithmeticExpression, ai int64, b px.Value) px.Value {
	op := expr.Operator()
	switch b := b.(type) {
	case px.Integer:
		return types.WrapInteger(intArithmetic(expr, ai, b.Int()))
	case px.Float:
		return types.WrapFloat(floatArithmetic(expr, float64(ai), b.Float()))
	case px.StringValue:
		s := b.String()
		if iv, err := strconv.ParseInt(s, 0, 64); err == nil {
			return types.WrapInteger(intArithmetic(expr, ai, iv))
		}
		if fv, err := strconv.ParseFloat(s, 64); err == nil {
			return types.WrapFloat(floatArithmetic(expr, float64(ai), fv))
		}
		panic(evalError(pdsl.NotNumeric, expr.Rhs(), issue.H{`value`: s}))
	default:
		panic(evalError(pdsl.OperatorNotApplicableWhen, expr, issue.H{`operator`: op, `left`: `Integer`, `right`: b.PType()}))
	}
}

func lhsFloatArithmetic(expr *parser.ArithmeticExpression, af float64, b px.Value) px.Value {
	op := expr.Operator()
	switch b := b.(type) {
	case px.Float:
		return types.WrapFloat(floatArithmetic(expr, af, b.Float()))
	case px.Integer:
		return types.WrapFloat(floatArithmetic(expr, af, float64(b.Int())))
	case px.StringValue:
		s := b.String()
		if iv, err := strconv.ParseInt(s, 0, 64); err == nil {
			return types.WrapFloat(floatArithmetic(expr, af, float64(iv)))
		}
		if fv, err := strconv.ParseFloat(s, 64); err == nil {
			return types.WrapFloat(floatArithmetic(expr, af, fv))
		}
		panic(evalError(pdsl.NotNumeric, expr.Rhs(), issue.H{`value`: s}))
	default:
		panic(evalError(pdsl.OperatorNotApplicableWhen, expr, issue.H{`operator`: op, `left`: `Float`, `right`: b.PType()}))
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
		panic(evalError(pdsl.OperatorNotApplicable, expr, issue.H{`operator`: expr.Operator(), `left`: `Float`}))
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
		panic(evalError(pdsl.OperatorNotApplicable, expr, issue.H{`operator`: expr.Operator(), `left`: `Integer`}))
	}
}

func concatenate(expr *parser.ArithmeticExpression, a px.Value, b px.Value) px.Value {
	switch a := a.(type) {
	case *types.Array:
		switch b := b.(type) {
		case *types.Array:
			return a.AddAll(b)

		case *types.Hash:
			return a.AddAll(b)

		default:
			return a.Add(b)
		}
	case *types.Hash:
		switch b := b.(type) {
		case *types.Array:
			return a.Merge(types.WrapHashFromArray(b))
		case *types.Hash:
			return a.Merge(b)
		}
	case *types.UriValue:
		switch b := b.(type) {
		case px.StringValue:
			return types.WrapURI(a.URL().ResolveReference(types.ParseURI(b.String())))
		case *types.UriValue:
			return types.WrapURI(a.URL().ResolveReference(b.URL()))
		}
	}
	panic(evalError(pdsl.OperatorNotApplicableWhen, expr, issue.H{`operator`: expr.Operator(), `left`: a.PType(), `right`: b.PType()}))
}

func collectionDelete(expr *parser.ArithmeticExpression, a px.Value, b px.Value) px.Value {
	switch a := a.(type) {
	case *types.Array:
		switch b := b.(type) {
		case *types.Array:
			return a.DeleteAll(b)
		case *types.Hash:
			return a.DeleteAll(b)
		default:
			return a.Delete(b)
		}
	case *types.Hash:
		switch b := b.(type) {
		case *types.Array:
			return a.DeleteAll(b)
		case *types.Hash:
			return a.DeleteAll(b.Keys())
		default:
			return a.Delete(b)
		}
	default:
		panic(evalError(pdsl.OperatorNotApplicable, expr, issue.H{`operator`: expr.Operator(), `left`: a.PType}))
	}
}
