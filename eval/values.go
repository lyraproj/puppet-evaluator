package eval

import (
	"bytes"
	"io"
	"fmt"
)

type (
	RDetect map[interface{}]bool

	PValue interface {
		fmt.Stringer
		Equality
		ToString(bld io.Writer, format FormatContext, g RDetect)
		Type() PType
	}

	// Comparator returns true when a is less than b.
	Comparator func(a, b PValue) bool

	ObjectValue interface {
		PValue
		Initialize(c Context, arguments []PValue)
		InitFromHash(c Context, hash KeyedValue)
	}

	DetailedTypeValue interface {
		PValue
		DetailedType() PType
	}

	SizedValue interface {
		PValue
		Len() int
		IsEmpty() bool
	}

	InterfaceValue interface {
		PValue
		Interface() interface{}
	}

	IterableValue interface {
		Iterator() Iterator
		ElementType() PType
		IsHashStyle() bool
	}

	IteratorValue interface {
		PValue
		DynamicValue() Iterator
	}

	// IndexedValue represents an Array. The iterative methods will not catch break exceptions. If
	//	// that is desired, then use an Iterator instead.
	IndexedValue interface {
		SizedValue
		IterableValue
		Add(PValue) IndexedValue
		AddAll(IndexedValue) IndexedValue
		All(predicate Predicate) bool
		Any(predicate Predicate) bool
		AppendTo(slice []PValue) []PValue
		At(index int) PValue
		Delete(PValue) IndexedValue
		DeleteAll(IndexedValue) IndexedValue
		Each(Consumer)
		EachSlice(int, SliceConsumer)
		EachWithIndex(consumer IndexedConsumer)
		Find(predicate Predicate) (PValue, bool)
		Flatten() IndexedValue
		Map(mapper Mapper) IndexedValue
		Select(predicate Predicate) IndexedValue
		Slice(i int, j int) IndexedValue
		Reduce(redactor BiMapper) PValue
		Reduce2(initialValue PValue, redactor BiMapper) PValue
		Reject(predicate Predicate) IndexedValue
		Unique() IndexedValue
	}

	SortableValue interface {
		IndexedValue
		Sort(comparator Comparator) IndexedValue
	}

	HashKey string

	HashKeyValue interface {
		ToKey() HashKey
	}

	StreamHashKeyValue interface {
		ToKey(b *bytes.Buffer)
	}

	EntryValue interface {
		PValue
		Key() PValue
		Value() PValue
	}

	// KeyedValue represents a Hash. The iterative methods will not catch break exceptions. If
	// that is desired, then use an Iterator instead.
	KeyedValue interface {
		IndexedValue
		AllPairs(BiPredicate) bool
		AnyPair(BiPredicate) bool
		Entries() IndexedValue
		EachKey(Consumer)
		EachPair(BiConsumer)
		EachValue(Consumer)

		Get(key PValue) (PValue, bool)
		Get2(key PValue, dflt PValue) PValue
		Get3(key PValue, dflt Producer) PValue
		Get4(key string) (PValue, bool)
		Get5(key string, dflt PValue) PValue
		Get6(key string, dflt Producer) PValue

		Keys() IndexedValue

		// MapValues returns a new KeyedValue with the exact same keys as
		// before but where each value has been converted using the given
		// mapper function
		MapValues(mapper Mapper) KeyedValue

		Merge(KeyedValue) KeyedValue

		Values() IndexedValue
		SelectPairs(BiPredicate) KeyedValue
		RejectPairs(BiPredicate) KeyedValue
	}

	NumericValue interface {
		PValue
		Int() int64
		Float() float64
		Abs() NumericValue
	}
)

var EMPTY_ARRAY IndexedValue
var EMPTY_MAP KeyedValue
var EMPTY_STRING PValue
var EMPTY_VALUES []PValue
var UNDEF PValue

var DetailedValueType func(value PValue) PType
var GenericValueType func(value PValue) PType
var ToKey func(value PValue) HashKey
var IsTruthy func(tv PValue) bool

var ToInt func(v PValue) (int64, bool)
var ToFloat func(v PValue) (float64, bool)
var WrapUnknown func(v interface{}) PValue

func ToString(t PValue) string {
	return ToString2(t, DEFAULT_FORMAT_CONTEXT)
}

func ToString2(t PValue, format FormatContext) string {
	bld := bytes.NewBufferString(``)
	t.ToString(bld, format, nil)
	return bld.String()
}

func ToString3(t PValue, writer io.Writer) {
	ToString4(t, DEFAULT_FORMAT_CONTEXT, writer)
}

func ToString4(t PValue, format FormatContext, writer io.Writer) {
	t.ToString(writer, format, nil)
}

func CopyValues(src []PValue) []PValue {
	dst := make([]PValue, len(src))
	for i, v := range src {
		dst[i] = v
	}
	return dst
}
