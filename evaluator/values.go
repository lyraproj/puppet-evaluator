package evaluator

import (
	"bytes"
	"io"
)

type (
	RDetect map[interface{}]bool

	PValue interface {
		Equality
		String() string
		ToString(bld io.Writer, format FormatContext, g RDetect)
		Type() PType
	}

	DetailedTypeValue interface {
		PValue
		DetailedType() PType
	}

	SizedValue interface {
		PValue
		Len() int
	}

	InterfaceValue interface {
		PValue
		Interface() interface{}
	}

	IterableValue interface {
		Iterator() Iterator
		ElementType() PType
	}

	IteratorValue interface {
		PValue
		DynamicValue() Iterator
	}

	IndexedValue interface {
		SizedValue
		IterableValue
		Add(PValue) IndexedValue
		AddAll(IndexedValue) IndexedValue
		At(index int) PValue
		Delete(PValue) IndexedValue
		DeleteAll(IndexedValue) IndexedValue
		Elements() []PValue
	}

	HashKey string

	HashKeyValue interface {
		ToKey() HashKey
	}

	StreamHashKeyValue interface {
		ToKey(b *bytes.Buffer)
	}

	EntryValue interface {
		Key() PValue
		Value() PValue
	}

	KeyedValue interface {
		SizedValue
		IterableValue
		Entries() IndexedValue
		Keys() IndexedValue
		Values() IndexedValue
		Get(key PValue) (PValue, bool)
		Get2(key string) PValue
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
var EMPTY_VALUES = []PValue{}
var UNDEF PValue

var CommonType func(a PType, b PType) PType
var DetailedType func(value PValue) PType
var GenericType func(value PValue) PType
var ToKey func(value PValue) HashKey
var IsTruthy func(tv PValue) bool

var ToInt func(v PValue) (int64, bool)
var ToFloat func(v PValue) (float64, bool)

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
