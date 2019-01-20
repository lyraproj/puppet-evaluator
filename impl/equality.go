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
		switch a.(type) {
		case eval.StringValue:
			return a.(eval.StringValue).EqualsIgnoreCase(b)
		case *types.IntegerValue:
			lhs := a.(*types.IntegerValue).Int()
			switch b.(type) {
			case *types.IntegerValue:
				return lhs == b.(*types.IntegerValue).Int()
			case eval.NumericValue:
				return float64(lhs) == b.(eval.NumericValue).Float()
			}
			return false
		case *types.FloatValue:
			lhs := a.(*types.FloatValue).Float()
			if rhv, ok := b.(eval.NumericValue); ok {
				return lhs == rhv.Float()
			}
			return false
		case *types.ArrayValue:
			if rhs, ok := b.(*types.ArrayValue); ok {
				lhs := a.(*types.ArrayValue)
				if lhs.Len() == rhs.Len() {
					idx := 0
					return lhs.All(func(el eval.Value) bool {
						eq := eval.PuppetEquals(el, rhs.At(idx))
						idx++
						return eq
					})
				}
			}
			return false
		case *types.HashValue:
			if rhs, ok := b.(*types.HashValue); ok {
				lhs := a.(*types.HashValue)
				if lhs.Len() == rhs.Len() {
					return lhs.AllPairs(func(key, value eval.Value) bool {
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
