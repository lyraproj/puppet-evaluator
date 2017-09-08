package values

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/eval/values/api"
)

type (
	BooleanType struct{}

	// BooleanValue keeps only the value because the type is known and not parameterized
	BooleanValue struct {
		value bool
	}
)

var booleanType_DEFAULT = &BooleanType{}

func DefaultBooleanType() *BooleanType {
	return booleanType_DEFAULT
}

func (t *BooleanType) Equals(o interface{}, g Guard) bool {
	_, ok := o.(*BooleanType)
	return ok
}

func (t *BooleanType) Name() string {
	return `Boolean`
}

func (t *BooleanType) String() string {
	return `Boolean`
}

func (t *BooleanType) IsAssignable(o PType, g Guard) bool {
	_, ok := o.(*BooleanType)
	return ok
}

func (t *BooleanType) IsInstance(o PValue, g Guard) bool {
	_, ok := o.(*BooleanValue)
	return ok
}

func (t *BooleanType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *BooleanType) Type() PType {
	return &TypeType{t}
}

func WrapBoolean(val bool) *BooleanValue {
	return &BooleanValue{val}
}

func (bv *BooleanValue) Bool() bool {
	return bv.value
}

func (bv *BooleanValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*BooleanValue); ok {
		return bv.value == ov.value
	}
	return false
}

func (bv *BooleanValue) String() string {
	if bv.value {
		return `true`
	}
	return `false`
}

func (bv *BooleanValue) ToString(b Writer, s FormatContext, g RDetect) {
	f := GetFormat(s.FormatMap(), bv.Type())
	switch f.FormatChar() {
	case 't':
		f.ApplyStringFlags(b, bv.stringVal(f.IsAlt(), `true`, `false`), false)
	case 'T':
		f.ApplyStringFlags(b, bv.stringVal(f.IsAlt(), `True`, `False`), false)
	case 'y':
		f.ApplyStringFlags(b, bv.stringVal(f.IsAlt(), `yes`, `no`), false)
	case 'Y':
		f.ApplyStringFlags(b, bv.stringVal(f.IsAlt(), `Yes`, `No`), false)
	case 'd', 'x', 'X', 'o', 'b', 'B':
		WrapInteger(bv.intVal()).ToString(b, NewFormatContext(DefaultIntegerType(), f, s.Indentation()), g)
	case 'e', 'E', 'f', 'g', 'G', 'a', 'A':
		WrapFloat(bv.floatVal()).ToString(b, NewFormatContext(DefaultFloatType(), f, s.Indentation()), g)
	case 's', 'p':
		f.ApplyStringFlags(b, bv.stringVal(false, `true`, `false`), f.IsAlt())
	default:
		panic(s.UnsupportedFormat(bv.Type(), `tTyYdxXobBeEfgGaAsp`, f))
	}
}

func (bv *BooleanValue) intVal() int64 {
	if bv.value {
		return 1
	}
	return 0
}

func (bv *BooleanValue) floatVal() float64 {
	if bv.value {
		return 1.0
	}
	return 0.0
}

func (bv *BooleanValue) stringVal(alt bool, yes string, no string) string {
	str := no
	if bv.value {
		str = yes
	}
	if alt {
		str = str[:1]
	}
	return str
}

func (bv *BooleanValue) ToKey() HashKey {
	if bv.value {
		return "\x01t"
	}
	return "\x01f"
}

func (bv *BooleanValue) Type() PType {
	return DefaultBooleanType()
}
