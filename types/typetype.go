package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
)

type TypeType struct {
	typ eval.PType
}

var typeType_DEFAULT = &TypeType{typ: anyType_DEFAULT}

func DefaultTypeType() *TypeType {
	return typeType_DEFAULT
}

func NewTypeType(containedType eval.PType) *TypeType {
	if containedType == nil || containedType == anyType_DEFAULT {
		return DefaultTypeType()
	}
	return &TypeType{containedType}
}

func NewTypeType2(args ...eval.PValue) *TypeType {
	switch len(args) {
	case 0:
		return DefaultTypeType()
	case 1:
		if containedType, ok := args[0].(eval.PType); ok {
			return NewTypeType(containedType)
		}
		panic(NewIllegalArgumentType2(`Type[]`, 0, `Type`, args[0]))
	default:
		panic(errors.NewIllegalArgumentCount(`Type[]`, `0 or 1`, len(args)))
	}
}

func (t *TypeType) ContainedType() eval.PType {
	return t.typ
}

func (t *TypeType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.typ.Accept(v, g)
}

func (t *TypeType) Default() eval.PType {
	return typeType_DEFAULT
}

func (t *TypeType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*TypeType); ok {
		return t.typ.Equals(ot.typ, g)
	}
	return false
}

func (t *TypeType) Generic() eval.PType {
	return NewTypeType(eval.GenericType(t.typ))
}

func (t *TypeType) IsAssignable(o eval.PType, g eval.Guard) bool {
	if ot, ok := o.(*TypeType); ok {
		return GuardedIsAssignable(t.typ, ot.typ, g)
	}
	return false
}

func (t *TypeType) IsInstance(o eval.PValue, g eval.Guard) bool {
	if ot, ok := o.(eval.PType); ok {
		return GuardedIsAssignable(t.typ, ot, g)
	}
	return false
}

func (t *TypeType) Name() string {
	return `Type`
}

func (t *TypeType) Parameters() []eval.PValue {
	if t.typ == DefaultAnyType() {
		return eval.EMPTY_VALUES
	}
	return []eval.PValue{t.typ}
}

func (t *TypeType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *TypeType) Type() eval.PType {
	return &TypeType{t}
}

func (t *TypeType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}
