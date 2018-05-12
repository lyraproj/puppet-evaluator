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
	eval.PuppetMatch = func(c eval.Context, a, b eval.PValue) bool {
		return match(c, nil, nil, `=~`, false, a, b)
	}
}
func (e *evaluator) eval_ComparisonExpression(expr *parser.ComparisonExpression, c eval.Context) eval.PValue {
	return e.compare(expr, expr.Operator(), e.eval(expr.Lhs(), c), e.eval(expr.Rhs(), c))
}

func (e *evaluator) eval_MatchExpression(expr *parser.MatchExpression, c eval.Context) eval.PValue {
	return types.WrapBoolean(match(c, expr.Lhs(), expr.Rhs(), expr.Operator(), true, e.eval(expr.Lhs(), c), e.eval(expr.Rhs(), c)))
}

func (e *evaluator) compare(expr parser.Expression, op string, a eval.PValue, b eval.PValue) eval.PValue {
	var result bool
	switch op {
	case `==`:
		result = eval.PuppetEquals(a, b)
	case `!=`:
		result = !eval.PuppetEquals(a, b)
	default:
		result = e.compareMagnitude(expr, op, a, b)
	}
	return types.WrapBoolean(result)
}

func (e *evaluator) compareMagnitude(expr parser.Expression, op string, a eval.PValue, b eval.PValue) bool {
	switch a.(type) {
	case eval.PType:
		left := a.(eval.PType)
		switch b.(type) {
		case eval.PType:
			right := b.(eval.PType)
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
				panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.Type()}))
			}
		}

	case *types.StringValue:
		if right, ok := b.(*types.StringValue); ok {
			// Case insensitive compare
			cmp := strings.Compare(strings.ToLower(a.(*types.StringValue).String()), strings.ToLower(right.String()))
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
				panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.Type()}))
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
				panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.Type()}))
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
				panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.Type()}))
			}
		}

	default:
		panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.Type()}))
	}
	panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, issue.H{`operator`: op, `left`: a.Type(), `right`: b.Type()}))
}

func match(c eval.Context, lhs parser.Expression, rhs parser.Expression, operator string, updateScope bool, a eval.PValue, b eval.PValue) bool {
	result := false
	switch b.(type) {
	case eval.PType:
		result = eval.IsInstance(c, b.(eval.PType), a)

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
			panic(eval.Error2(lhs, eval.EVAL_MATCH_NOT_STRING, issue.H{`left`: a.Type()}))
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
				issue.H{`detail`: fmt.Sprintf(`A value of type %s cannot be converted to a SemVer`, a.Type().String())}))
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
