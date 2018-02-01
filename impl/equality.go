package impl

import (
	"strings"

	. "github.com/puppetlabs/go-evaluator/eval"
	. "github.com/puppetlabs/go-evaluator/types"
)

// PuppetEquals is like Equals but:
//   int and float values with same value are considered equal
//   string comparisons are case insensitive
//
func init() {
	PuppetEquals = func(a PValue, b PValue) bool {
		switch a.(type) {
		case *StringValue:
			bs, ok := b.(*StringValue)
			return ok && strings.ToLower(a.(*StringValue).String()) == strings.ToLower(bs.String())
		case *IntegerValue:
			lhs := a.(*IntegerValue).Int()
			switch b.(type) {
			case *IntegerValue:
				return lhs == b.(*IntegerValue).Int()
			case NumericValue:
				return float64(lhs) == b.(NumericValue).Float()
			}
			return false
		case *FloatValue:
			lhs := a.(*FloatValue).Float()
			if rhv, ok := b.(NumericValue); ok {
				return lhs == rhv.Float()
			}
			return false
		case *ArrayValue:
			if rhs, ok := b.(*ArrayValue); ok {
				lhs := a.(*ArrayValue)
				if lhs.Len() == rhs.Len() {
					idx := 0
					return lhs.All(func(el PValue) bool {
						eq := PuppetEquals(el, rhs.At(idx))
						idx++
						return eq
					})
				}
			}
			return false
		case *HashValue:
			if rhs, ok := b.(*HashValue); ok {
				lhs := a.(*HashValue)
				if lhs.Len() == rhs.Len() {
					return lhs.AllPairs(func(key, value PValue) bool {
						rhsValue, ok := rhs.Get(key)
						return ok && PuppetEquals(value, rhsValue)
					})
				}
			}
			return false
		default:
			return Equals(a, b)
		}
	}
}
