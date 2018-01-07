package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type OptionalType struct {
	typ PType
}

func DefaultOptionalType() *OptionalType {
	return optionalType_DEFAULT
}

func NewOptionalType(containedType PType) *OptionalType {
	if containedType == nil || containedType == anyType_DEFAULT {
		return DefaultOptionalType()
	}
	return &OptionalType{containedType}
}

func NewOptionalType2(args ...PValue) *OptionalType {
	switch len(args) {
	case 0:
		return DefaultOptionalType()
	case 1:
		if containedType, ok := args[0].(PType); ok {
			return NewOptionalType(containedType)
		}
		if containedType, ok := args[0].(*StringValue); ok {
			return NewOptionalType3(containedType.String())
		}
		panic(NewIllegalArgumentType2(`Optional[]`, 0, `Variant[Type,String]`, args[0]))
	default:
		panic(NewIllegalArgumentCount(`Optional[]`, `0 - 1`, len(args)))
	}
}

func NewOptionalType3(str string) *OptionalType {
	return &OptionalType{NewStringType(nil, str)}
}

func (t *OptionalType) ContainedType() PType {
	return t.typ
}

func (t *OptionalType) Default() PType {
	return optionalType_DEFAULT
}

func (t *OptionalType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*OptionalType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *OptionalType) Generic() PType {
	return NewOptionalType(GenericType(t.typ))
}

func (t *OptionalType) IsAssignable(o PType, g Guard) bool {
	return GuardedIsAssignable(o, undefType_DEFAULT, g) || GuardedIsAssignable(t.typ, o, g)
}

func (t *OptionalType) IsInstance(o PValue, g Guard) bool {
	return o == _UNDEF || GuardedIsInstance(t.typ, o, g)
}

func (t *OptionalType) Name() string {
	return `Optional`
}

func (t *OptionalType) Parameters() []PValue {
	if t.typ == DefaultAnyType() {
		return EMPTY_VALUES
	}
	if str, ok := t.typ.(*StringType); ok && str.value != `` {
		return []PValue{WrapString(str.value)}
	}
	return []PValue{t.typ}
}

func (t *OptionalType) String() string {
	return ToString2(t, NONE)
}

func (t *OptionalType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *OptionalType) Type() PType {
	return &TypeType{t}
}

var optionalType_DEFAULT = &OptionalType{typ: anyType_DEFAULT}
