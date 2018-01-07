package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type NotUndefType struct {
	typ PType
}

func DefaultNotUndefType() *NotUndefType {
	return notUndefType_DEFAULT
}

func NewNotUndefType(containedType PType) *NotUndefType {
	if containedType == nil || containedType == anyType_DEFAULT {
		return DefaultNotUndefType()
	}
	return &NotUndefType{containedType}
}

func NewNotUndefType2(args ...PValue) *NotUndefType {
	switch len(args) {
	case 0:
		return DefaultNotUndefType()
	case 1:
		if containedType, ok := args[0].(PType); ok {
			return NewNotUndefType(containedType)
		}
		if containedType, ok := args[0].(*StringValue); ok {
			return NewNotUndefType3(containedType.String())
		}
		panic(NewIllegalArgumentType2(`NotUndef[]`, 0, `Variant[Type,String]`, args[0]))
	default:
		panic(NewIllegalArgumentCount(`NotUndef[]`, `0 - 1`, len(args)))
	}
}

func NewNotUndefType3(str string) *NotUndefType {
	return &NotUndefType{NewStringType(nil, str)}
}

func (t *NotUndefType) ContainedType() PType {
	return t.typ
}

func (t *NotUndefType) Default() PType {
	return notUndefType_DEFAULT
}

func (t *NotUndefType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*NotUndefType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *NotUndefType) Generic() PType {
	return NewNotUndefType(GenericType(t.typ))
}

func (t *NotUndefType) IsAssignable(o PType, g Guard) bool {
	return !GuardedIsAssignable(o, undefType_DEFAULT, g) && GuardedIsAssignable(t.typ, o, g)
}

func (t *NotUndefType) IsInstance(o PValue, g Guard) bool {
	return o != _UNDEF && GuardedIsInstance(t.typ, o, g)
}

func (t *NotUndefType) Name() string {
	return `NotUndef`
}

func (t *NotUndefType) Parameters() []PValue {
	if t.typ == DefaultAnyType() {
		return EMPTY_VALUES
	}
	if str, ok := t.typ.(*StringType); ok && str.value != `` {
		return []PValue{WrapString(str.value)}
	}
	return []PValue{t.typ}
}

func (t *NotUndefType) String() string {
	return ToString2(t, NONE)
}

func (t *NotUndefType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *NotUndefType) Type() PType {
	return &TypeType{t}
}

var notUndefType_DEFAULT = &NotUndefType{typ: anyType_DEFAULT}
