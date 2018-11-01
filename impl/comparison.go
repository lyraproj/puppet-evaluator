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
	"strconv"
)

func init() {
	eval.PuppetMatch = func(c eval.Context, a, b eval.Value) bool {
		return match(c, nil, nil, `=~`, false, a, b)
	}
}

func (e *evaluator) eval_ComparisonExpression(expr *parser.ComparisonExpression, c eval.Context) eval.Value {
	return types.WrapBoolean(e.doCompare(expr, expr.Operator(), e.eval(expr.Lhs(), c), e.eval(expr.Rhs(), c), c))
}

func (e *evaluator) doCompare(expr parser.Expression, op string, a, b eval.Value, c eval.Context) bool {
	if c.Language() == eval.LangJavaScript {
		return e.jsCompare(expr, op, a, b)
	}
	return e.compare(expr, op, a, b)
}

func (e *evaluator) eval_MatchExpression(expr *parser.MatchExpression, c eval.Context) eval.Value {
	return types.WrapBoolean(match(c, expr.Lhs(), expr.Rhs(), expr.Operator(), true, e.eval(expr.Lhs(), c), e.eval(expr.Rhs(), c)))
}

func (e *evaluator) jsCompare(expr parser.Expression, op string, a, b eval.Value) bool {
	var result bool
	switch op {
	case `==`:
		result = JsEquals(a, b, false)
	case `!=`:
		result = !JsEquals(a, b, false)
	case `===`:
		result = JsEquals(a, b, true)
	case `!==`:
		result = !JsEquals(a, b, true)
	default:
		result = e.compareMagnitude(expr, op, a, b, true)
	}
	return result
}

// JsEquals performs JavaScript style equals of x and y and returns the result.
func JsEquals(x, y eval.Value, strict bool) bool {
	if x == y {
		return true
	}

	switch x.(type) {
	case *types.IntegerValue:
		xi := x.(*types.IntegerValue).Int()
		switch y.(type) {
		case *types.IntegerValue:
			return xi == y.(*types.IntegerValue).Int()
		case *types.FloatValue:
			return float64(xi) == y.(*types.FloatValue).Float()
		case *types.BooleanValue:
			return !strict && (xi != 0) == y.(*types.BooleanValue).Bool()
		case *types.StringValue:
			if strict {
				return false
			}
			bv := y.(*types.StringValue)
			if yi, err := strconv.ParseInt(bv.String(), 0, 64); err == nil {
				return xi == yi
			}
			if yf, err := strconv.ParseFloat(bv.String(), 64); err == nil {
				return float64(xi) == yf
			}
		}
	case *types.FloatValue:
		xf := x.(*types.FloatValue).Float()
		switch y.(type) {
		case *types.IntegerValue:
			return JsEquals(y, x, strict)
		case *types.FloatValue:
			return xf == y.(*types.FloatValue).Float()
		case *types.BooleanValue:
			return !strict && (xf != 0.0) == y.(*types.BooleanValue).Bool()
		case *types.StringValue:
			if strict {
				return false
			}
			bv := y.(*types.StringValue)
			if yf, err := strconv.ParseFloat(bv.String(), 64); err == nil {
				return xf == yf
			}
		}
	case *types.BooleanValue:
		xb := x.(*types.BooleanValue).Bool()
		switch y.(type) {
		case *types.BooleanValue:
			return xb == y.(*types.BooleanValue).Bool()
		case *types.IntegerValue, *types.FloatValue:
			return JsEquals(y, x, strict)
		case *types.StringValue:
			if strict {
				return false
			}
			bv := y.(*types.StringValue)
			if yf, err := strconv.ParseFloat(bv.String(), 64); err == nil {
				return xb == (0.0 != yf)
			}
		}
	case *types.StringValue:
		xs := x.String()
		switch y.(type) {
		case *types.StringValue:
			return xs == y.(*types.StringValue).String()
		case *types.IntegerValue, *types.FloatValue, *types.BooleanValue:
			return JsEquals(y, x, strict)
		}
	case *types.UndefValue:
		_, eq := y.(*types.UndefValue)
		return eq
	}
	return false
}

func (e *evaluator) compare(expr parser.Expression, op string, a eval.Value, b eval.Value) bool {
	var result bool
	switch op {
	case `==`:
		result = eval.PuppetEquals(a, b)
	case `!=`:
		result = !eval.PuppetEquals(a, b)
	default:
		result = e.compareMagnitude(expr, op, a, b, false)
	}
	return result
}

func (e *evaluator) compareMagnitude(expr parser.Expression, op string, a eval.Value, b eval.Value, caseSensitive bool) bool {
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
				panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
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
				panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
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
				panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
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
				panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
			}
		}

	default:
		panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE, expr, issue.H{`operator`: op, `left`: a.PType()}))
	}
	panic(e.evalError(eval.EVAL_OPERATOR_NOT_APPLICABLE_WHEN, expr, issue.H{`operator`: op, `left`: a.PType(), `right`: b.PType()}))
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
