package types

import (
	"io"
	"math"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
)

type CollectionType struct {
	size *IntegerType
}

var Collection_Type eval.ObjectType

func init() {
	Collection_Type = newObjectType(`Pcore::CollectionType`, `Pcore::AnyType {
  attributes => {
    'size_type' => { type => Type[Integer], value => Integer[0] }
  }
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return NewCollectionType2(args...)
	})
}

func DefaultCollectionType() *CollectionType {
	return collectionType_DEFAULT
}

func NewCollectionType(size *IntegerType) *CollectionType {
	if size == nil || *size == *IntegerType_POSITIVE {
		return DefaultCollectionType()
	}
	return &CollectionType{size}
}

func NewCollectionType2(args ...eval.Value) *CollectionType {
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
		panic(errors.NewIllegalArgumentCount(`Collection[]`, `0 - 2`, len(args)))
	}
}

func (t *CollectionType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.size.Accept(v, g)
}

func (t *CollectionType) Default() eval.Type {
	return collectionType_DEFAULT
}

func (t *CollectionType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*CollectionType); ok {
		return t.size.Equals(ot.size, g)
	}
	return false
}

func (t *CollectionType) Generic() eval.Type {
	return collectionType_DEFAULT
}

func (t *CollectionType) Get(key string) (eval.Value, bool) {
	switch key {
	case `size_type`:
		if t.size == nil {
			return eval.UNDEF, true
		}
		return t.size, true
	default:
		return nil, false
	}
}

func (t *CollectionType) IsAssignable(o eval.Type, g eval.Guard) bool {
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

func (t *CollectionType) IsInstance(o eval.Value, g eval.Guard) bool {
	return t.IsAssignable(o.Type(), g)
}

func (t *CollectionType) MetaType() eval.ObjectType {
	return Collection_Type
}

func (t *CollectionType) Name() string {
	return `Collection`
}

func (t *CollectionType) Parameters() []eval.Value {
	if *t.size == *IntegerType_POSITIVE {
		return eval.EMPTY_VALUES
	}
	return t.size.SizeParameters()
}

func (t *CollectionType) Size() *IntegerType {
	return t.size
}

func (t *CollectionType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *CollectionType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *CollectionType) Type() eval.Type {
	return &TypeType{t}
}

var collectionType_DEFAULT = &CollectionType{IntegerType_POSITIVE}
