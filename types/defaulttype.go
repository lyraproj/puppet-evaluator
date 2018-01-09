package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-parser/parser"
)

type (
	DefaultType struct{}

	// DefaultValue is an empty struct because both type and value are known
	DefaultValue struct{}
)

var defaultType_DEFAULT = &DefaultType{}

func DefaultDefaultType() *DefaultType {
	return defaultType_DEFAULT
}

func (t *DefaultType) Accept(v Visitor, g Guard) {
	v(t)
}

func (t *DefaultType) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*DefaultType)
	return ok
}

func (t *DefaultType) IsAssignable(o PType, g Guard) bool {
	return o == defaultType_DEFAULT
}

func (t *DefaultType) IsInstance(o PValue, g Guard) bool {
	_, ok := o.(*DefaultValue)
	return ok
}

func (t *DefaultType) Name() string {
	return `Default`
}

func (t *DefaultType) String() string {
	return ToString2(t, NONE)
}

func (t *DefaultType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *DefaultType) Type() PType {
	return &TypeType{t}
}

func WrapDefault() *DefaultValue {
	return &DefaultValue{}
}

func (dv *DefaultValue) DynamicValue() Default {
	return DEFAULT_INSTANCE
}

func (dv *DefaultValue) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*DefaultValue)
	return ok
}

func (dv *DefaultValue) ToKey() HashKey {
	return HashKey([]byte{1, HK_DEFAULT})
}

func (dv *DefaultValue) String() string {
	return `default`
}

func (dv *DefaultValue) ToString(b Writer, s FormatContext, g RDetect) {
	f := GetFormat(s.FormatMap(), dv.Type())
	switch f.FormatChar() {
	case 'd', 's', 'p':
		f.ApplyStringFlags(b, `default`, f.IsAlt())
	case 'D':
		f.ApplyStringFlags(b, `Default`, f.IsAlt())
	default:
		panic(s.UnsupportedFormat(dv.Type(), `dDsp`, f))
	}
}

func (dv *DefaultValue) Type() PType {
	return DefaultDefaultType()
}
