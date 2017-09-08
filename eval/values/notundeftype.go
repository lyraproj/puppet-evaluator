package values

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/eval/errors"
	. "github.com/puppetlabs/go-evaluator/eval/utils"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
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
		containedType, ok := args[0].(PType)
		if !ok {
			panic(NewIllegalArgumentType2(`NotUndef[]`, 0, `Type`, args[0]))
		}
		return NewNotUndefType(containedType)
	default:
		panic(NewIllegalArgumentCount(`NotUndef[]`, `0 - 1`, len(args)))
	}
}
func (t *NotUndefType) ContainedType() PType {
	return t.typ
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

func (t *NotUndefType) String() string {
	return ToString2(t, NONE)
}

func (t *NotUndefType) ToString(bld Writer, format FormatContext, g RDetect) {
	WriteString(bld, `NotUndef`)
	if t.typ != anyType_DEFAULT {
		WriteByte(bld, '[')
		t.typ.ToString(bld, format, g)
		WriteByte(bld, ']')
	}
}

func (t *NotUndefType) Type() PType {
	return &TypeType{t}
}

var notUndefType_DEFAULT = &NotUndefType{typ: anyType_DEFAULT}
