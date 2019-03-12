package evaluator

import (
	"regexp"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

func evalComparisonExpression(e pdsl.Evaluator, expr *parser.ComparisonExpression) px.Value {
	return types.WrapBoolean(doCompare(expr, expr.Operator(), e.Eval(expr.Lhs()), e.Eval(expr.Rhs())))
}

func doCompare(expr parser.Expression, op string, a, b px.Value) bool {
	return compare(expr, op, a, b)
}

func evalMatchExpression(e pdsl.Evaluator, expr *parser.MatchExpression) px.Value {
	return types.WrapBoolean(match(e, expr.Lhs(), expr.Rhs(), expr.Operator(), e.Eval(expr.Lhs()), e.Eval(expr.Rhs())))
}

func compare(expr parser.Expression, op string, a px.Value, b px.Value) bool {
	var result bool
	switch op {
	case `==`:
		result = px.PuppetEquals(a, b)
	case `!=`:
		result = !px.PuppetEquals(a, b)
	default:
		result = compareMagnitude(expr, op, a, b, false)
	}
	return result
}

func compareMagnitude(expr parser.Expression, op string, a px.Value, b px.Value, caseSensitive bool) bool {
	switch a.(type) {
	case px.Type:
		left := a.(px.Type)
		switch b := b.(type) {
		case px.Type:
			switch op {
			case `<`:
				return px.IsAssignable(b, left) && !left.Equals(b, nil)
			case `<=`:
				return px.IsAssignable(b, left)
			case `>`:
				return px.IsAssignable(left, b) && !left.Equals(b, nil)
			case `>=`:
				return px.IsAssignable(left, b)
			default:
				panic(evalError(pdsl.OperatorNotApplicable, expr, issue.H{`operator`: op, `left`: a.PType()}))
			}
		}

	case px.StringValue:
		if _, ok := b.(px.StringValue); ok {
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

	case *types.SemVer:
		if rhv, ok := b.(*types.SemVer); ok {
			cmp := a.(*types.SemVer).Version().CompareTo(rhv.Version())
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

	case px.Number:
		if rhv, ok := b.(px.Number); ok {
			cmp := a.(px.Number).Float() - rhv.Float()
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

func match(c px.Context, lhs parser.Expression, rhs parser.Expression, operator string, a px.Value, b px.Value) bool {
	result := false
	switch b := b.(type) {
	case px.StringValue, *types.Regexp:
		var rx *regexp.Regexp
		if s, ok := b.(px.StringValue); ok {
			var err error
			rx, err = regexp.Compile(s.String())
			if err != nil {
				panic(px.Error2(rhs, px.MatchNotRegexp, issue.H{`detail`: err.Error()}))
			}
		} else {
			rx = b.(*types.Regexp).Regexp()
		}

		sv, ok := a.(px.StringValue)
		if !ok {
			panic(px.Error2(lhs, px.MatchNotString, issue.H{`left`: a.PType()}))
		}
		if group := rx.FindStringSubmatch(sv.String()); group != nil {
			c.Scope().(pdsl.Scope).RxSet(group)
			result = true
		}
	default:
		result = px.PuppetMatch(a, b)
	}
	if operator == `!~` {
		result = !result
	}
	return result
}
