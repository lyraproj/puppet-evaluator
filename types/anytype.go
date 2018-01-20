package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type AnyType struct{}

var Any_Type PType

func init() {
	Any_Type = newType(`AnyType`, `{}`)
}

func DefaultAnyType() *AnyType {
	return anyType_DEFAULT
}

func (t *AnyType) Accept(v Visitor, g Guard) {
	v(t)
}

func (t *AnyType) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*AnyType)
	return ok
}

func (t *AnyType) IsAssignable(o PType, g Guard) bool {
	return true
}

func (t *AnyType) IsInstance(v PValue, g Guard) bool {
	return true
}

func (t *AnyType) Name() string {
	return `Any`
}

func (t *AnyType) String() string {
	return `Any`
}

func (t *AnyType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *AnyType) Type() PType {
	return Any_Type
}

var anyType_DEFAULT = &AnyType{}
