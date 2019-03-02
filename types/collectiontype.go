package types

import (
	"io"
	"math"

	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
)

type CollectionType struct {
	size *IntegerType
}

var CollectionMetaType eval.ObjectType

func init() {
	CollectionMetaType = newObjectType(`Pcore::CollectionType`, `Pcore::AnyType {
  attributes => {
    'size_type' => { type => Type[Integer], value => Integer[0] }
  }
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
		return newCollectionType2(args...)
	})
}

func DefaultCollectionType() *CollectionType {
	return collectionTypeDefault
}

func NewCollectionType(size *IntegerType) *CollectionType {
	if size == nil || *size == *IntegerTypePositive {
		return DefaultCollectionType()
	}
	return &CollectionType{size}
}

func newCollectionType2(args ...eval.Value) *CollectionType {
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
					panic(NewIllegalArgumentType(`Collection[]`, 0, `Variant[Integer, Default, Type[Integer]]`, arg))
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
				panic(NewIllegalArgumentType(`Collection[]`, 0, `Variant[Integer, Default]`, arg))
			}
			min = 0
		}
		arg = args[1]
		max, ok := toInt(arg)
		if !ok {
			if _, ok := arg.(*DefaultValue); !ok {
				panic(NewIllegalArgumentType(`Collection[]`, 1, `Variant[Integer, Default]`, arg))
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
	return collectionTypeDefault
}

func (t *CollectionType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*CollectionType); ok {
		return t.size.Equals(ot.size, g)
	}
	return false
}

func (t *CollectionType) Generic() eval.Type {
	return collectionTypeDefault
}

func (t *CollectionType) Get(key string) (eval.Value, bool) {
	switch key {
	case `size_type`:
		if t.size == nil {
			return eval.Undef, true
		}
		return t.size, true
	default:
		return nil, false
	}
}

func (t *CollectionType) IsAssignable(o eval.Type, g eval.Guard) bool {
	var osz *IntegerType
	switch o := o.(type) {
	case *CollectionType:
		osz = o.size
	case *ArrayType:
		osz = o.size
	case *HashType:
		osz = o.size
	case *TupleType:
		osz = o.givenOrActualSize
	case *StructType:
		n := int64(len(o.elements))
		osz = NewIntegerType(n, n)
	default:
		return false
	}
	return t.size.IsAssignable(osz, g)
}

func (t *CollectionType) IsInstance(o eval.Value, g eval.Guard) bool {
	return t.IsAssignable(o.PType(), g)
}

func (t *CollectionType) MetaType() eval.ObjectType {
	return CollectionMetaType
}

func (t *CollectionType) Name() string {
	return `Collection`
}

func (t *CollectionType) Parameters() []eval.Value {
	if *t.size == *IntegerTypePositive {
		return eval.EmptyValues
	}
	return t.size.SizeParameters()
}

func (t *CollectionType) CanSerializeAsString() bool {
	return true
}

func (t *CollectionType) SerializationString() string {
	return t.String()
}

func (t *CollectionType) Size() *IntegerType {
	return t.size
}

func (t *CollectionType) String() string {
	return eval.ToString2(t, None)
}

func (t *CollectionType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *CollectionType) PType() eval.Type {
	return &TypeType{t}
}

var collectionTypeDefault = &CollectionType{IntegerTypePositive}
