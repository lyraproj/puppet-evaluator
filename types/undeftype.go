package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type (
	UndefType struct{}

	// UndefValue is an empty struct because both type and value are known
	UndefValue struct{}
)

var undefType_DEFAULT = &UndefType{}

func DefaultUndefType() *UndefType {
	return undefType_DEFAULT
}

func (t *UndefType) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*UndefType)
	return ok
}

func (t *UndefType) IsAssignable(o PType, g Guard) bool {
	_, ok := o.(*UndefType)
	return ok
}

func (t *UndefType) IsInstance(o PValue, g Guard) bool {
	return o == _UNDEF
}

func (t *UndefType) Name() string {
	return `Undef`
}

func (t *UndefType) String() string {
	return `Undef`
}

func (t *UndefType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *UndefType) Type() PType {
	return &TypeType{t}
}

func WrapUndef() *UndefValue {
	return &UndefValue{}
}

func (uv *UndefValue) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*UndefValue)
	return ok
}

func (uv *UndefValue) String() string {
	return `undef`
}

func (uv *UndefValue) ToKey() HashKey {
	return HashKey([]byte{1, HK_UNDEF})
}

func (uv *UndefValue) ToString(b Writer, s FormatContext, g RDetect) {
	WriteString(b, `undef`)
}

func (uv *UndefValue) Type() PType {
	return DefaultUndefType()
}
