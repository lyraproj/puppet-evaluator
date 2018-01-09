package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type TypeReferenceType struct {
	typeString string
}

func DefaultTypeReferenceType() *TypeReferenceType {
	return typeReferenceType_DEFAULT
}

func NewTypeReferenceType(typeString string) *TypeReferenceType {
	return &TypeReferenceType{typeString}
}

func NewTypeReferenceType2(args ...PValue) *TypeReferenceType {
	switch len(args) {
	case 0:
		return DefaultTypeReferenceType()
	case 1:
		if str, ok := args[0].(*StringValue); ok {
			return &TypeReferenceType{str.String()}
		}
		panic(NewIllegalArgumentType2(`TypeReference[]`, 0, `String`, args[0]))
	default:
		panic(NewIllegalArgumentCount(`TypeReference[]`, `0 - 1`, len(args)))
	}
}

func (t *TypeReferenceType) Accept(v Visitor, g Guard) {
	v(t)
}

func (t *TypeReferenceType) Default() PType {
	return typeReferenceType_DEFAULT
}

func (t *TypeReferenceType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*TypeReferenceType); ok {
		return t.typeString == ot.typeString
	}
	return false
}

func (t *TypeReferenceType) IsAssignable(o PType, g Guard) bool {
	tr, ok := o.(*TypeReferenceType)
	return ok && t.typeString == tr.typeString
}

func (t *TypeReferenceType) IsInstance(o PValue, g Guard) bool {
	return false
}

func (t *TypeReferenceType) Name() string {
	return `TypeReference`
}

func (t *TypeReferenceType) String() string {
	return ToString2(t, NONE)
}

func (t *TypeReferenceType) Parameters() []PValue {
	if *t == *typeReferenceType_DEFAULT {
		return EMPTY_VALUES
	}
	return []PValue{WrapString(t.typeString)}
}

func (t *TypeReferenceType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TypeReferenceType) Type() PType {
	return &TypeType{t}
}

func (t *TypeReferenceType) TypeString() string {
	return t.typeString
}

var typeReferenceType_DEFAULT = &TypeReferenceType{`UnresolvedReference`}
