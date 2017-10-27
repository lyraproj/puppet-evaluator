package types

import (
	. "io"

	"math"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type CollectionType struct {
	size *IntegerType
}

func DefaultCollectionType() *CollectionType {
	return collectionType_DEFAULT
}

func NewCollectionType(size *IntegerType) *CollectionType {
	if size == nil || *size == *integerType_POSITIVE {
		return DefaultCollectionType()
	}
	return &CollectionType{size}
}

func NewCollectionType2(args ...PValue) *CollectionType {
	switch len(args) {
	case 0:
		return DefaultCollectionType()
	case 1:
		arg := args[0]
		size, ok := arg.(*IntegerType)
		if !ok {
			sz, ok := toInt(arg)
			if !ok {
				if _, ok := arg.(*DefaultValue); !ok {
					panic(NewIllegalArgumentType2(`Collection[]`, 0, `Variant[Integer, Default, Type[Integer]]`, arg))
				}
				sz = 0
			}
			size = NewIntegerType(sz, math.MaxInt64)
		}
		return NewCollectionType(size)
	case 2:
		arg := args[0]
		min, ok := toInt(arg)
		if !ok {
			if _, ok := arg.(*DefaultValue); !ok {
				panic(NewIllegalArgumentType2(`Collection[]`, 0, `Variant[Integer, Default]`, arg))
			}
			min = 0
		}
		arg = args[1]
		max, ok := toInt(arg)
		if !ok {
			if _, ok := arg.(*DefaultValue); !ok {
				panic(NewIllegalArgumentType2(`Collection[]`, 1, `Variant[Integer, Default]`, arg))
			}
			max = math.MaxInt64
		}
		return NewCollectionType(NewIntegerType(min, max))
	default:
		panic(NewIllegalArgumentCount(`Collection[]`, `0 - 2`, len(args)))
	}
}

func (t *CollectionType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*CollectionType); ok {
		return t.size.Equals(ot.size, g)
	}
	return false
}

func (t *CollectionType) Generic() PType {
	return collectionType_DEFAULT
}

func (t *CollectionType) IsAssignable(o PType, g Guard) bool {
	var osz *IntegerType
	switch o.(type) {
	case *CollectionType:
		osz = o.(*CollectionType).size
	case *ArrayType:
		osz = o.(*ArrayType).size
	case *HashType:
		osz = o.(*HashType).size
	case *TupleType:
		osz = o.(*TupleType).givenOrActualSize
	case *StructType:
		n := int64(len(o.(*StructType).elements))
		osz = NewIntegerType(n, n)
	default:
		return false
	}
	return t.size.IsAssignable(osz, g)
}

func (t *CollectionType) IsInstance(o PValue, g Guard) bool {
	return t.IsAssignable(o.Type(), g)
}

func (t *CollectionType) Name() string {
	return `Collection`
}

func (t *CollectionType) Parameters() []PValue {
	if *t.size == *integerType_POSITIVE {
		return EMPTY_VALUES
	}
	return t.size.SizeParameters()
}

func (t *CollectionType) Size() *IntegerType {
	return t.size
}

func (t *CollectionType) String() string {
	return ToString2(t, NONE)
}

func (t *CollectionType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *CollectionType) Type() PType {
	return &TypeType{t}
}

var collectionType_DEFAULT = &CollectionType{integerType_POSITIVE}
