package impl

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

// PuppetEquals is like Equals but:
//   int and float values with same value are considered equal
//   string comparisons are case insensitive
//
func init() {
	eval.PuppetEquals = func(a eval.Value, b eval.Value) bool {
		switch a := a.(type) {
		case eval.StringValue:
			return a.EqualsIgnoreCase(b)
		case eval.IntegerValue:
			lhs := a.Int()
			switch b := b.(type) {
			case eval.IntegerValue:
				return lhs == b.Int()
			case eval.NumericValue:
				return float64(lhs) == b.Float()
			}
			return false
		case eval.FloatValue:
			lhs := a.Float()
			if rhv, ok := b.(eval.NumericValue); ok {
				return lhs == rhv.Float()
			}
			return false
		case *types.ArrayValue:
			if rhs, ok := b.(*types.ArrayValue); ok {
				if a.Len() == rhs.Len() {
					idx := 0
					return a.All(func(el eval.Value) bool {
						eq := eval.PuppetEquals(el, rhs.At(idx))
						idx++
						return eq
					})
				}
			}
			return false
		case *types.HashValue:
			if rhs, ok := b.(*types.HashValue); ok {
				if a.Len() == rhs.Len() {
					return a.AllPairs(func(key, value eval.Value) bool {
						rhsValue, ok := rhs.Get(key)
						return ok && eval.PuppetEquals(value, rhsValue)
					})
				}
			}
			return false
		default:
			return eval.Equals(a, b)
		}
	}
}
