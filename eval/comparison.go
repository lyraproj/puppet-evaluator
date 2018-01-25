package eval

import (
	"fmt"
	"regexp"
	"strings"

	. "github.com/puppetlabs/go-evaluator/evaluator"
	"github.com/puppetlabs/go-evaluator/semver"
	. "github.com/puppetlabs/go-evaluator/types"
	. "github.com/puppetlabs/go-parser/parser"
	. "github.com/puppetlabs/go-parser/issue"
)

func init() {
  PuppetMatch = func(a, b PValue) bool {
  	return match(nil, nil, `=~`, nil, a, b)
	}
}
func (e *evaluator) eval_ComparisonExpression(expr *ComparisonExpression, c EvalContext) PValue {
	return e.compare(expr, expr.Operator(), e.eval(expr.Lhs(), c), e.eval(expr.Rhs(), c))
}

func (e *evaluator) eval_MatchExpression(expr *MatchExpression, c EvalContext) PValue {
	return WrapBoolean(match(expr.Lhs(), expr.Rhs(), expr.Operator(), c.Scope(), e.eval(expr.Lhs(), c), e.eval(expr.Rhs(), c)))
}

func (e *evaluator) compare(expr Expression, op string, a PValue, b PValue) PValue {
	var result bool
	switch op {
	case `==`:
		result = PuppetEquals(a, b)
	case `!=`:
		result = !PuppetEquals(a, b)
	default:
		result = e.compareMagnitude(expr, op, a, b)
	}
	return WrapBoolean(result)
}

func (e *evaluator) compareMagnitude(expr Expression, op string, a PValue, b PValue) bool {
	switch a.(type) {
	case PType:
		left := a.(PType)
		switch b.(type) {
		case PType:
			right := b.(PType)
			switch op {
			case `<`:
				return IsAssignable(right, left) && !Equals(left, right)
			case `<=`:
				return IsAssignable(right, left)
			case `>`:
				return IsAssignable(left, right) && !Equals(left, right)
			case `>=`:
				return IsAssignable(left, right)
			default:
				panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, H{`operator`: op, `left`: a.Type()}))
			}
		}

	case *StringValue:
		if right, ok := b.(*StringValue); ok {
			// Case insensitive compare
			cmp := strings.Compare(strings.ToLower(a.(*StringValue).String()), strings.ToLower(right.String()))
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
				panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, H{`operator`: op, `left`: a.Type()}))
			}
		}

	case *SemVerValue:
		if rhv, ok := b.(*SemVerValue); ok {
			cmp := a.(*SemVerValue).Version().CompareTo(rhv.Version())
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
				panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, H{`operator`: op, `left`: a.Type()}))
			}
		}

	case NumericValue:
		if rhv, ok := b.(NumericValue); ok {
			cmp := a.(NumericValue).Float() - rhv.Float()
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
				panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, H{`operator`: op, `left`: a.Type()}))
			}
		}

	default:
		panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, H{`operator`: op, `left`: a.Type()}))
	}
	panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, H{`operator`: op, `left`: a.Type(), `right`: b.Type()}))
}

func match(lhs Expression, rhs Expression, operator string, scope Scope, a PValue, b PValue) bool {
	result := false
	switch b.(type) {
	case PType:
		result = IsInstance(b.(PType), a)

	case *StringValue, *RegexpValue:
		var rx *regexp.Regexp
		if s, ok := b.(*StringValue); ok {
			var err error
			rx, err = regexp.Compile(s.String())
			if err != nil {
				panic(Error2(rhs, EVAL_MATCH_NOT_REGEXP, H{`detail`: err.Error()}))
			}
		} else {
			rx = b.(*RegexpValue).Regexp()
		}

		sv, ok := a.(*StringValue)
		if !ok {
			panic(Error2(lhs, EVAL_MATCH_NOT_STRING, H{`left`: a.Type()}))
		}
		if group := rx.FindStringSubmatch(sv.String()); group != nil {
			if scope != nil {
				scope.RxSet(group)
			}
			result = true
		}

	case *SemVerValue, *SemVerRangeValue:
		var version *semver.Version

		if v, ok := a.(*SemVerValue); ok {
			version = v.Version()
		} else if s, ok := a.(*StringValue); ok {
			var err error
			version, err = semver.ParseVersion(s.String())
			if err != nil {
				panic(Error2(lhs, EVAL_NOT_SEMVER, H{`detail`: err.Error()}))
			}
		} else {
			panic(Error2(lhs, EVAL_NOT_SEMVER,
				H{`detail`: fmt.Sprintf(`A value of type %s cannot be converted to a SemVer`, a.Type().String())}))
		}
		if lv, ok := b.(*SemVerValue); ok {
			result = lv.Version().Equals(version)
		} else {
			result = b.(*SemVerRangeValue).VersionRange().Includes(version)
		}

	default:
		result = PuppetEquals(b, a)
	}

	if operator == `!~` {
		result = !result
	}
	return result

}
