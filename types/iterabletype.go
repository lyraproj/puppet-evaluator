package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
)

type IterableType struct {
	typ eval.PType
}

var Iterable_Type eval.ObjectType

func init() {
	Iterable_Type = newObjectType(`Pcore::IterableType`,
		`Pcore::AnyType {
			attributes => {
				type => {
					type => Optional[Type],
					value => Any
				},
			}
		}`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return NewIterableType2(args...)
		})
}

func DefaultIterableType() *IterableType {
	return iterableType_DEFAULT
}

func NewIterableType(elementType eval.PType) *IterableType {
	if elementType == nil || elementType == anyType_DEFAULT {
		return DefaultIterableType()
	}
	return &IterableType{elementType}
}

func NewIterableType2(args ...eval.PValue) *IterableType {
	switch len(args) {
	case 0:
		return DefaultIterableType()
	case 1:
		containedType, ok := args[0].(eval.PType)
		if !ok {
			panic(NewIllegalArgumentType2(`Iterable[]`, 0, `Type`, args[0]))
		}
		return NewIterableType(containedType)
	default:
		panic(errors.NewIllegalArgumentCount(`Iterable[]`, `0 - 1`, len(args)))
	}
}

func (t *IterableType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *IterableType) Default() eval.PType {
	return iterableType_DEFAULT
}

func (t *IterableType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*IterableType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *IterableType) Generic() eval.PType {
	return NewIterableType(eval.GenericType(t.typ))
}

func (t *IterableType) Get(key string) (value eval.PValue, ok bool) {
	switch key {
	case `type`:
		return t.typ, true
	}
	return nil, false
}

func (t *IterableType) IsAssignable(o eval.PType, g eval.Guard) bool {
	var et eval.PType
	switch o.(type) {
	case *ArrayType:
		et = o.(*ArrayType).ElementType()
	case *BinaryType:
		et = NewIntegerType(0, 255)
	case *HashType:
		et = o.(*HashType).EntryType()
	case *StringType:
		et = ONE_CHAR_STRING_TYPE
	case *TupleType:
		return allAssignableTo(o.(*TupleType).types, t.typ, g)
	default:
		return false
	}
	return GuardedIsAssignable(t.typ, et, g)
}

func (t *IterableType) IsInstance(o eval.PValue, g eval.Guard) bool {
	if iv, ok := o.(eval.IterableValue); ok {
		return GuardedIsAssignable(t.typ, iv.ElementType(), g)
	}
	return false
}

func (t *IterableType) MetaType() eval.ObjectType {
	return Iterable_Type
}

func (t *IterableType) Name() string {
	return `Iterable`
}

func (t *IterableType) Parameters() []eval.PValue {
	if t.typ == DefaultAnyType() {
		return eval.EMPTY_VALUES
	}
	return []eval.PValue{t.typ}
}

func (t *IterableType) Resolve(c eval.Context) eval.PType {
	t.typ = resolve(c, t.typ)
	return t
}

func (t *IterableType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *IterableType) ElementType() eval.PType {
	return t.typ
}

func (t *IterableType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *IterableType) Type() eval.PType {
	return &TypeType{t}
}

var iterableType_DEFAULT = &IterableType{typ: DefaultAnyType()}
