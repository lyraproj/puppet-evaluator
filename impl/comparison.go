package impl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-semver/semver"
)

func init() {
	eval.PuppetMatch = func(c eval.Context, a, b eval.Value) bool {
		return match(c, nil, nil, `=~`, false, a, b)
	}
}

func eval_ComparisonExpression(e eval.Evaluator, expr *parser.ComparisonExpression, c eval.Context) eval.Value {
	return types.WrapBoolean(doCompare(expr, expr.Operator(), e.Eval(expr.Lhs(), c), e.Eval(expr.Rhs(), c), c))
}

func doCompare(expr parser.Expression, op string, a, b eval.Value, c eval.Context) bool {
	return compare(expr, op, a, b)
}

func eval_MatchExpression(e eval.Evaluator, expr *parser.MatchExpression, c eval.Context) eval.Value {
	return types.WrapBoolean(match(c, expr.Lhs(), expr.Rhs(), expr.Operator(), true, e.Eval(expr.Lhs(), c), e.Eval(expr.Rhs(), c)))
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
		switch b.(type) {
		case eval.Type:
			right := b.(eval.Type)
			switch op {
			case `<`:
				return eval.IsAssignable(right, left) && !eval.Equals(left, right)
			case `<=`:
				return eval.IsAssignable(right, left)
			case `>`:
				return eval.IsAssignable(left, right) && !eval.Equals(left, right)
			case `>=`:
				return eval.IsAssignable(left, right)
			default:
				panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
			}
		}

	case *types.StringValue:
		if _, ok := b.(*types.StringValue); ok {
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
				panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
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
				panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
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
				panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
			}
		}

	default:
		panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
	}
	panic(evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, issue.H{`operator`: op, `left`: a.PType(), `right`: b.PType()}))
}

func match(c eval.Context, lhs parser.Expression, rhs parser.Expression, operator string, updateScope bool, a eval.Value, b eval.Value) bool {
	result := false
	switch b.(type) {
	case eval.Type:
		result = eval.IsInstance(b.(eval.Type), a)

	case *types.StringValue, *types.RegexpValue:
		var rx *regexp.Regexp
		if s, ok := b.(*types.StringValue); ok {
			var err error
			rx, err = regexp.Compile(s.String())
			if err != nil {
				panic(eval.Error2(rhs, eval.EVAL_MATCH_NOT_REGEXP, issue.H{`detail`: err.Error()}))
			}
		} else {
			rx = b.(*types.RegexpValue).Regexp()
		}

		sv, ok := a.(*types.StringValue)
		if !ok {
			panic(eval.Error2(lhs, eval.EVAL_MATCH_NOT_STRING, issue.H{`left`: a.PType()}))
		}
		if group := rx.FindStringSubmatch(sv.String()); group != nil {
			if updateScope {
				c.Scope().RxSet(group)
			}
			result = true
		}

	case *types.SemVerValue, *types.SemVerRangeValue:
		var version semver.Version

		if v, ok := a.(*types.SemVerValue); ok {
			version = v.Version()
		} else if s, ok := a.(*types.StringValue); ok {
			var err error
			version, err = semver.ParseVersion(s.String())
			if err != nil {
				panic(eval.Error2(lhs, eval.EVAL_NOT_SEMVER, issue.H{`detail`: err.Error()}))
			}
		} else {
			panic(eval.Error2(lhs, eval.EVAL_NOT_SEMVER,
				issue.H{`detail`: fmt.Sprintf(`A value of type %s cannot be converted to a SemVer`, a.PType().String())}))
		}
		if lv, ok := b.(*types.SemVerValue); ok {
			result = lv.Version().Equals(version)
		} else {
			result = b.(*types.SemVerRangeValue).VersionRange().Includes(version)
		}

	default:
		result = eval.PuppetEquals(b, a)
	}

	if operator == `!~` {
		result = !result
	}
	return result
}
