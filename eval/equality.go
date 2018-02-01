package eval

import "reflect"

type (
	visit struct {
		a1 interface{}
		a2 interface{}
	}

	// Guard helps tracking endless recursion. The comparison algorithm assumes that all checks in progress
	// are true when it reencounters them. Visited comparisons are stored in a map
	// indexed by visit.
	//
	// (algorithm copied from golang reflect/deepequal.go)
	Guard map[visit]bool

	Equality interface {
		Equals(other interface{}, guard Guard) bool
	}
)

func (g Guard) Seen(a, b interface{}) bool {
	v := visit{a, b}
	if _, ok := g[v]; ok {
		return true
	}
	g[v] = true
	return false
}

// EqSlice converts a slice of values that implement the Equality interface to []Equality. The
// method will panic if the given argument is not a slice or array, or if not all
// elements implement the Equality interface
func EqSlice(slice interface{}) []Equality {
	sv := reflect.ValueOf(slice)
	top := sv.Len()
	result := make([]Equality, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = sv.Index(idx).Interface().(Equality)
	}
	return result
}

func Equals(a interface{}, b interface{}) bool {
	switch a.(type) {
	case nil:
		return b == nil
	case Equality:
		return a.(Equality).Equals(b, nil)
	case string:
		bs, ok := b.(string)
		return ok && a.(string) == bs
	default:
		return reflect.DeepEqual(a, b)
	}
}

func GuardedEquals(a interface{}, b interface{}, g Guard) bool {
	switch a.(type) {
	case nil:
		return b == nil
	case Equality:
		return a.(Equality).Equals(b, g)
	case string:
		bs, ok := b.(string)
		return ok && a.(string) == bs
	default:
		return reflect.DeepEqual(a, b)
	}
}

func IncludesAll(a []Equality, b []Equality) bool {
	return GuardedIncludesAll(a, b, nil)
}

func IndexOf(a []Equality, b Equality) int {
	return GuardedIndexFrom(a, b, 0, nil)
}

func ReverseIndexOf(a []Equality, b Equality) int {
	return GuardedReverseIndexFrom(a, b, len(a), nil)
}

func IndexFrom(a []Equality, b Equality, startPos int) int {
	return GuardedIndexFrom(a, b, startPos, nil)
}

func ReverseIndexFrom(a []Equality, b Equality, startPos int) int {
	return GuardedReverseIndexFrom(a, b, startPos, nil)
}

func GuardedIncludesAll(a []Equality, b []Equality, g Guard) bool {
	for _, v := range a {
		found := false
		for _, ov := range b {
			if v.Equals(ov, g) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func GuardedIndexOf(a []Equality, b Equality, g Guard) int {
	return GuardedIndexFrom(a, b, 0, g)
}

func GuardedReverseIndexOf(a []Equality, b Equality, g Guard) int {
	return GuardedReverseIndexFrom(a, b, len(a), g)
}

func GuardedIndexFrom(a []Equality, b Equality, startPos int, g Guard) int {
	top := len(a)
	for idx := startPos; idx < top; idx++ {
		if a[idx].Equals(b, g) {
			return idx
		}
	}
	return -1
}

func GuardedReverseIndexFrom(a []Equality, b Equality, startPos int, g Guard) int {
	top := len(a)
	idx := startPos
	if idx >= top {
		idx = top - 1
	}
	for ; idx >= 0; idx-- {
		if a[idx].Equals(b, g) {
			return idx
		}
	}
	return -1
}

// PuppetEquals is like Equals but:
//   int and float values with same value are considered equal
//   string comparisons are case insensitive
//
var PuppetEquals func(a, b PValue) bool

var PuppetMatch func(a, b PValue) bool
