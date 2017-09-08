package eval

import (
	"fmt"
	"regexp"
	"strings"

	. "github.com/puppetlabs/go-evaluator/eval/evaluator"
	. "github.com/puppetlabs/go-evaluator/eval/values"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
	"github.com/puppetlabs/go-evaluator/semver"
	. "github.com/puppetlabs/go-parser/parser"
)

func (e *evaluator) eval_ComparisonExpression(expr *ComparisonExpression, c EvalContext) PValue {
	return e.compare(expr, e.eval(expr.Lhs(), c), e.eval(expr.Rhs(), c))
}

func (e *evaluator) eval_MatchExpression(expr *MatchExpression, c EvalContext) PValue {
	return WrapBoolean(e.match(expr, c.Scope(), e.eval(expr.Lhs(), c), e.eval(expr.Rhs(), c)))
}

func (e *evaluator) compare(expr *ComparisonExpression, a PValue, b PValue) PValue {
	var result bool
	switch expr.Operator() {
	case `==`:
		result = PuppetEquals(a, b)
	case `!=`:
		result = !PuppetEquals(a, b)
	default:
		result = e.compareMagnitude(expr, a, b)
	}
	return WrapBoolean(result)
}

func (e *evaluator) compareMagnitude(expr *ComparisonExpression, a PValue, b PValue) bool {
	op := expr.Operator()
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
				panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, op, A_an(a.Type())))
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
				panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, op, A_an(a.Type())))
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
				panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, op, A_an(a.Type())))
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
				panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, op, A_an(a.Type())))
			}
		}

	default:
		panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE, expr, op, A_an(a.Type())))
	}
	panic(e.evalError(EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, op, A_an(a.Type()), A_an(b.Type())))
}

func (e *evaluator) match(expr *MatchExpression, scope Scope, a PValue, b PValue) bool {
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
				panic(e.evalError(EVAL_MATCH_NOT_REGEXP, expr.Rhs(), err.Error()))
			}
		} else {
			rx = b.(*RegexpValue).Regexp()
		}

		sv, ok := a.(*StringValue)
		if !ok {
			panic(e.evalError(EVAL_MATCH_NOT_STRING, expr.Lhs(), A_an(a.Type())))
		}
		if group := rx.FindStringSubmatch(sv.String()); group != nil {
			scope.RxSet(group)
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
				panic(e.evalError(EVAL_NOT_SEMVER, expr.Lhs(), err.Error()))
			}
		} else {
			panic(e.evalError(EVAL_NOT_SEMVER, expr.Lhs(), fmt.Sprint(`A value of type %s cannot be converted to a SemVer`, a.Type().String())))
		}
		if lv, ok := b.(*SemVerValue); ok {
			result = lv.Version().Equals(version)
		} else {
			result = b.(*SemVerRangeValue).VersionRange().Includes(version)
		}

	default:
		panic(e.evalError(EVAL_MATCH_NOT_REGEXP, expr.Rhs(), fmt.Sprintf(`no conversion from %s to Regexp`, A_anUc(b.Type()))))
	}

	if expr.Operator() == `!~` {
		result = !result
	}
	return result

}
