package evaluator

import (
	"regexp"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

func evalComparisonExpression(e pdsl.Evaluator, expr *parser.ComparisonExpression) eval.Value {
	return types.WrapBoolean(doCompare(expr, expr.Operator(), e.Eval(expr.Lhs()), e.Eval(expr.Rhs())))
}

func doCompare(expr parser.Expression, op string, a, b eval.Value) bool {
	return compare(expr, op, a, b)
}

func evalMatchExpression(e pdsl.Evaluator, expr *parser.MatchExpression) eval.Value {
	return types.WrapBoolean(match(e, expr.Lhs(), expr.Rhs(), expr.Operator(), e.Eval(expr.Lhs()), e.Eval(expr.Rhs())))
}

func compare(expr parser.Expression, op string, a eval.Value, b eval.Value) bool {
	var result bool
	switch op {
	case `==`:
		result = eval.PuppetEquals(a, b)
	case `!=`:
		result = !eval.PuppetEquals(a, b)
	default:
		result = compareMagnitude(expr, op, a, b, false)
	}
	return result
}

func compareMagnitude(expr parser.Expression, op string, a eval.Value, b eval.Value, caseSensitive bool) bool {
	switch a.(type) {
	case eval.Type:
		left := a.(eval.Type)
		switch b := b.(type) {
		case eval.Type:
			switch op {
			case `<`:
				return eval.IsAssignable(b, left) && !eval.Equals(left, b)
			case `<=`:
				return eval.IsAssignable(b, left)
			case `>`:
				return eval.IsAssignable(left, b) && !eval.Equals(left, b)
			case `>=`:
				return eval.IsAssignable(left, b)
			default:
				panic(evalError(pdsl.OperatorNotApplicable, expr, issue.H{`operator`: op, `left`: a.PType()}))
			}
		}

	case eval.StringValue:
		if _, ok := b.(eval.StringValue); ok {
			sa := a.String()
			sb := b.String()
			if !caseSensitive {
				sa = strings.ToLower(sa)
				sb = strings.ToLower(sb)
			}
			// Case insensitive compare
			cmp := strings.Compare(sa, sb)
			switch op {
			case `<`:
				return cmp < 0
			case `<=`:
				return cmp <= 0
			case `>`:
				return cmp > 0
			case `>=`:
				return cmp >= 0
			default:
				panic(evalError(pdsl.OperatorNotApplicable, expr, issue.H{`operator`: op, `left`: a.PType()}))
			}
		}

	case *types.SemVerValue:
		if rhv, ok := b.(*types.SemVerValue); ok {
			cmp := a.(*types.SemVerValue).Version().CompareTo(rhv.Version())
			switch op {
			case `<`:
				return cmp < 0.0
			case `<=`:
				return cmp <= 0.0
			case `>`:
				return cmp > 0.0
			case `>=`:
				return cmp >= 0.0
			default:
				panic(evalError(pdsl.OperatorNotApplicable, expr, issue.H{`operator`: op, `left`: a.PType()}))
			}
		}

	case eval.NumericValue:
		if rhv, ok := b.(eval.NumericValue); ok {
			cmp := a.(eval.NumericValue).Float() - rhv.Float()
			switch op {
			case `<`:
				return cmp < 0.0
			case `<=`:
				return cmp <= 0.0
			case `>`:
				return cmp > 0.0
			case `>=`:
				return cmp >= 0.0
			default:
				panic(evalError(pdsl.OperatorNotApplicable, expr, issue.H{`operator`: op, `left`: a.PType()}))
			}
		}

	default:
		panic(evalError(pdsl.OperatorNotApplicable, expr, issue.H{`operator`: op, `left`: a.PType()}))
	}
	panic(evalError(pdsl.OperatorNotApplicableWhen, expr, issue.H{`operator`: op, `left`: a.PType(), `right`: b.PType()}))
}

func match(c eval.Context, lhs parser.Expression, rhs parser.Expression, operator string, a eval.Value, b eval.Value) bool {
	result := false
	switch b := b.(type) {
	case eval.StringValue, *types.RegexpValue:
		var rx *regexp.Regexp
		if s, ok := b.(eval.StringValue); ok {
			var err error
			rx, err = regexp.Compile(s.String())
			if err != nil {
				panic(eval.Error2(rhs, eval.MatchNotRegexp, issue.H{`detail`: err.Error()}))
			}
		} else {
			rx = b.(*types.RegexpValue).Regexp()
		}

		sv, ok := a.(eval.StringValue)
		if !ok {
			panic(eval.Error2(lhs, eval.MatchNotString, issue.H{`left`: a.PType()}))
		}
		if group := rx.FindStringSubmatch(sv.String()); group != nil {
			c.(pdsl.EvaluationContext).Scope().RxSet(group)
			result = true
		}
	default:
		result = eval.PuppetMatch(a, b)
	}
	if operator == `!~` {
		result = !result
	}
	return result
}
