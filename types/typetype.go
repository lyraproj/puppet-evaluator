package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/eval"
)

type TypeType struct {
	typ PType
}

var typeType_DEFAULT = &TypeType{typ: anyType_DEFAULT}

func DefaultTypeType() *TypeType {
	return typeType_DEFAULT
}

func NewTypeType(containedType PType) *TypeType {
	if containedType == nil || containedType == anyType_DEFAULT {
		return DefaultTypeType()
	}
	return &TypeType{containedType}
}

func NewTypeType2(args ...PValue) *TypeType {
	switch len(args) {
	case 0:
		return DefaultTypeType()
	case 1:
		if containedType, ok := args[0].(PType); ok {
			return NewTypeType(containedType)
		}
		panic(NewIllegalArgumentType2(`Type[]`, 0, `Type`, args[0]))
	default:
		panic(NewIllegalArgumentCount(`Type[]`, `0 or 1`, len(args)))
	}
}

func (t *TypeType) ContainedType() PType {
	return t.typ
}

func (t *TypeType) Accept(v Visitor, g Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *TypeType) Default() PType {
	return typeType_DEFAULT
}

func (t *TypeType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*TypeType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *TypeType) Generic() PType {
	return NewTypeType(GenericType(t.typ))
}

func (t *TypeType) IsAssignable(o PType, g Guard) bool {
	if ot, ok := o.(*TypeType); ok {
		return GuardedIsAssignable(t.typ, ot.typ, g)
	}
	return false
}

func (t *TypeType) IsInstance(o PValue, g Guard) bool {
	if ot, ok := o.(PType); ok {
		return GuardedIsAssignable(t.typ, ot, g)
	}
	return false
}

func (t *TypeType) Name() string {
	return `Type`
}

func (t *TypeType) Parameters() []PValue {
	if t.typ == DefaultAnyType() {
		return EMPTY_VALUES
	}
	return []PValue{t.typ}
}

func (t *TypeType) String() string {
	return ToString2(t, NONE)
}

func (t *TypeType) Type() PType {
	return &TypeType{t}
}

func (t *TypeType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}
