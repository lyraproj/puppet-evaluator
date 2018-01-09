package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type IterableType struct {
	typ PType
}

func DefaultIterableType() *IterableType {
	return iterableType_DEFAULT
}

func NewIterableType(elementType PType) *IterableType {
	if elementType == nil || elementType == anyType_DEFAULT {
		return DefaultIterableType()
	}
	return &IterableType{elementType}
}

func NewIterableType2(args ...PValue) *IterableType {
	switch len(args) {
	case 0:
		return DefaultIterableType()
	case 1:
		containedType, ok := args[0].(PType)
		if !ok {
			panic(NewIllegalArgumentType2(`Iterable[]`, 0, `Type`, args[0]))
		}
		return NewIterableType(containedType)
	default:
		panic(NewIllegalArgumentCount(`Iterable[]`, `0 - 1`, len(args)))
	}
}

func (t *IterableType) Accept(v Visitor, g Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *IterableType) Default() PType {
	return iterableType_DEFAULT
}

func (t *IterableType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*IterableType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *IterableType) Generic() PType {
	return NewIterableType(GenericType(t.typ))
}

func (t *IterableType) IsAssignable(o PType, g Guard) bool {
	var et PType
	switch o.(type) {
	case *ArrayType:
		et = o.(*ArrayType).ElementType()
	case *BinaryType:
		et = NewIntegerType(0, 255)
	case *HashType:
		et = o.(*HashType).EntryType()
	case *StringType:
		et = ONE_CHAR_STRING_TYPE
	default:
		return false
	}
	return GuardedIsAssignable(t.typ, et, g)
}

func (t *IterableType) IsInstance(o PValue, g Guard) bool {
	if iv, ok := o.(IterableValue); ok {
		return GuardedIsAssignable(t.typ, iv.ElementType(), g)
	}
	return false
}

func (t *IterableType) Name() string {
	return `Iterable`
}

func (t *IterableType) Parameters() []PValue {
	if t.typ == DefaultAnyType() {
		return EMPTY_VALUES
	}
	return []PValue{t.typ}
}

func (t *IterableType) String() string {
	return ToString2(t, NONE)
}

func (t *IterableType) ElementType() PType {
	return t.typ
}

func (t *IterableType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *IterableType) Type() PType {
	return &TypeType{t}
}

var iterableType_DEFAULT = &IterableType{typ: DefaultAnyType()}
