package types

import (
	. "io"
	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

var sensitiveType_DEFAULT = &SensitiveType{typ: anyType_DEFAULT}

type (
	SensitiveType struct{
		typ PType
	}

	SensitiveValue struct {
		value PValue
	}
)

func DefaultSensitiveType() *SensitiveType {
	return sensitiveType_DEFAULT
}

func NewSensitiveType(containedType PType) *SensitiveType {
	if containedType == nil || containedType == anyType_DEFAULT {
		return DefaultSensitiveType()
	}
	return &SensitiveType{containedType}
}

func NewSensitiveType2(args ...PValue) *SensitiveType {
	switch len(args) {
	case 0:
		return DefaultSensitiveType()
	case 1:
		if containedType, ok := args[0].(PType); ok {
			return NewSensitiveType(containedType)
		}
		panic(NewIllegalArgumentType2(`Sensitive[]`, 0, `Type`, args[0]))
	default:
		panic(NewIllegalArgumentCount(`Sensitive[]`, `0 or 1`, len(args)))
	}
}

func (t *SensitiveType) ContainedType() PType {
	return t.typ
}

func (t *SensitiveType) Accept(v Visitor, g Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *SensitiveType) Default() PType {
	return DefaultSensitiveType()
}

func (t *SensitiveType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*SensitiveType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *SensitiveType) Generic() PType {
	return NewSensitiveType(GenericType(t.typ))
}

func (t *SensitiveType) IsAssignable(o PType, g Guard) bool {
	if ot, ok := o.(*SensitiveType); ok {
		return GuardedIsAssignable(t.typ, ot.typ, g)
	}
	return false
}

func (t *SensitiveType) IsInstance(o PValue, g Guard) bool {
	if sv, ok := o.(*SensitiveValue); ok {
		return GuardedIsInstance(t.typ, sv.Unwrap(), g)
	}
	return false
}

func (t *SensitiveType) Name() string {
	return `Sensitive`
}

func (t *SensitiveType) Parameters() []PValue {
	if t.typ == DefaultAnyType() {
		return EMPTY_VALUES
	}
	return []PValue{t.typ}
}

func (t *SensitiveType) String() string {
	return ToString2(t, NONE)
}

func (t *SensitiveType) Type() PType {
	return &SensitiveType{t}
}

func (t *SensitiveType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func WrapSensitive(val PValue) *SensitiveValue {
	return &SensitiveValue{val}
}

func (s *SensitiveValue) Equals(o interface{}, g Guard) bool {
	return false
}

func (s *SensitiveValue) String() string {
	return ToString2(s, NONE)
}

func (s *SensitiveValue) ToString(b Writer, f FormatContext, g RDetect) {
	WriteString(b, `Sensitive [value redacted]`)
}

func (s *SensitiveValue) Type() PType {
	return NewSensitiveType(s.Unwrap().Type())
}

func (s *SensitiveValue) Unwrap() PValue {
	return s.value
}
