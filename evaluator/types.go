package evaluator

import (
	"github.com/puppetlabs/go-parser/parser"
)

type (
	PType interface {
		PValue

		IsInstance(o PValue, g Guard) bool

		IsAssignable(t PType, g Guard) bool

		Name() string
	}

	SizedType interface {
		PType

		Size() PType
	}

	TypeResolver interface {
		ParseResolve(typeString string) PType

		Resolve(expr parser.Expression) PType
	}

	ResolvableType interface {
		PType

		Resolve(resolver TypeResolver)
	}

	ParameterizedType interface {
		PType

		Default() PType

		// Parameters returns the parameters that is needed in order to recreate
		// an instance of the parameterized type.
		Parameters() []PValue
	}

	// Implemented by all parameterized types that have type parameters
	Generalizable interface {
		ParameterizedType
		Generic() PType
	}
)

var IsInstance func(puppetType PType, value PValue) bool

// isAssignable answers if t is assignable to this type
var IsAssignable func(puppetType PType, other PType) bool

var Generalize func(a PType) PType

var DefaultFor func(a PType) PType

var ToArray func(elements []PValue) IndexedValue

func All(elements []PValue, predicate Predicate) bool {
	for _, elem := range elements {
		if !predicate(elem) {
			return false
		}
	}
	return true
}

func All2(array IndexedValue, predicate Predicate) bool {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		if !predicate(array.At(idx)) {
			return false
		}
	}
	return true
}

func Any(elements []PValue, predicate Predicate) bool {
	for _, elem := range elements {
		if predicate(elem) {
			return true
		}
	}
	return false
}

func Any2(array IndexedValue, predicate Predicate) bool {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		if predicate(array.At(idx)) {
			return true
		}
	}
	return false
}

func Each(elements []PValue, consumer Consumer) {
	for _, elem := range elements {
		consumer(elem)
	}
}

func Each2(array IndexedValue, consumer Consumer) {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		consumer(array.At(idx))
	}
}

func Find(elements []PValue, dflt PValue, predicate Predicate) PValue {
	for _, elem := range elements {
		if predicate(elem) {
			return elem
		}
	}
	return dflt
}

func Find2(array IndexedValue, dflt PValue, predicate Predicate) PValue {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		v := array.At(idx)
		if predicate(v) {
			return v
		}
	}
	return dflt
}

func Map(elements []PValue, mapper Mapper) []PValue {
	result := make([]PValue, len(elements))
	for idx, elem := range elements {
		result[idx] = mapper(elem)
	}
	return result
}

func Map2(array IndexedValue, mapper Mapper) IndexedValue {
	top := array.Len()
	result := make([]PValue, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = mapper(array.At(idx))
	}
	return ToArray(result)
}

func MapTypes(types []PType, mapper TypeMapper) []PValue {
	result := make([]PValue, len(types))
	for idx, elem := range types {
		result[idx] = mapper(elem)
	}
	return result
}

func Reduce(elements []PValue, memo PValue, reductor BiMapper) PValue {
	for _, elem := range elements {
		memo = reductor(memo, elem)
	}
	return memo
}

func Reduce2(array IndexedValue, memo PValue, reductor BiMapper) PValue {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		memo = reductor(memo, array.At(idx))
	}
	return memo
}

func Select(elements []PValue, predicate Predicate) []PValue {
	result := make([]PValue, 0, 8)
	for _, elem := range elements {
		if predicate(elem) {
			result = append(result, elem)
		}
	}
	return result
}

func Select2(array IndexedValue, predicate Predicate) IndexedValue {
	result := make([]PValue, 0, 8)
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		v := array.At(idx)
		if predicate(v) {
			result = append(result, v)
		}
	}
	return ToArray(result)
}

func Reject(elements []PValue, predicate Predicate) []PValue {
	result := make([]PValue, 0, 8)
	for _, elem := range elements {
		if !predicate(elem) {
			result = append(result, elem)
		}
	}
	return result
}

func Reject2(array IndexedValue, predicate Predicate) IndexedValue {
	result := make([]PValue, 0, 8)
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		v := array.At(idx)
		if !predicate(v) {
			result = append(result, v)
		}
	}
	return ToArray(result)
}
