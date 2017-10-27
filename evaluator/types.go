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

	Generalizable interface {
		Generic() PType
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

		// Parameters returns the parameters that is needed in order to recreate
		// an instance of the parameterized type.
		Parameters() []PValue
	}
)

var IsInstance func(puppetType PType, value PValue) bool

// isAssignable answers if t is assignable to this type
var IsAssignable func(puppetType PType, other PType) bool

var Generalize func(a PType) PType

var ToArray func(elements []PValue) IndexedValue

func All(array IndexedValue, predicate Predicate) bool {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		if !predicate(array.At(idx)) {
			return false
		}
	}
	return true
}

func Any(array IndexedValue, predicate Predicate) bool {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		if predicate(array.At(idx)) {
			return true
		}
	}
	return false
}

func Each(array IndexedValue, consumer Consumer) {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		consumer(array.At(idx))
	}
}

func Find(array IndexedValue, dflt PValue, predicate Predicate) PValue {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		v := array.At(idx)
		if predicate(v) {
			return v
		}
	}
	return dflt
}

func Map(array IndexedValue, mapper Mapper) IndexedValue {
	top := array.Len()
	result := make([]PValue, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = mapper(array.At(idx))
	}
	return ToArray(result)
}

func Reduce(array IndexedValue, memo PValue, reductor BiMapper) PValue {
	top := array.Len()
	for idx := 0; idx < top; idx++ {
		memo = reductor(memo, array.At(idx))
	}
	return memo
}

func Select(array IndexedValue, predicate Predicate) IndexedValue {
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

func Reject(array IndexedValue, predicate Predicate) IndexedValue {
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
