package types

import (
	"strings"

	. "github.com/puppetlabs/go-evaluator/evaluator"
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
				if len(lhs.elements) == len(rhs.elements) {
					for idx, el := range lhs.elements {
						if !PuppetEquals(el, rhs.elements[idx]) {
							return false
						}
					}
					return true
				}
			}
			return false
		case *HashValue:
			if rhs, ok := b.(*HashValue); ok {
				lhs := a.(*HashValue)
				if len(lhs.entries) == len(rhs.entries) {
					for _, entry := range lhs.entries {
						var rhsValue PValue
						rhsValue, ok = rhs.Get(entry.key)
						if !(ok && PuppetEquals(entry.value, rhsValue)) {
							return false
						}
					}
					return true
				}
			}
			return false
		default:
			return Equals(a, b)
		}
	}
}
