package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type (
	BooleanType struct{
		value int  // -1 == unset, 0 == false, 1 == true
	}

	// BooleanValue keeps only the value because the type is known and not parameterized
	BooleanValue BooleanType
)

var booleanType_DEFAULT = &BooleanType{-1}

func DefaultBooleanType() *BooleanType {
	return booleanType_DEFAULT
}

func NewBooleanType(value bool) *BooleanType {
	n := 0
	if value {
		n = 1
	}
	return &BooleanType{n}
}

func (t *BooleanType) Accept(v Visitor, g Guard) {
	v(t)
}

func (t *BooleanType) Default() PType {
	return booleanType_DEFAULT
}

func (t *BooleanType) Generic() PType {
	return booleanType_DEFAULT
}

func (t *BooleanType) Equals(o interface{}, g Guard) bool {
	if bo, ok := o.(*BooleanType); ok {
		return t.value == bo.value
	}
	return false
}

func (t *BooleanType) Name() string {
	return `Boolean`
}

func (t *BooleanType) String() string {
	return `Boolean`
}

func (t *BooleanType) IsAssignable(o PType, g Guard) bool {
	if bo, ok := o.(*BooleanType); ok {
		return t.value == -1 || t.value == bo.value
	}
	return false
}

func (t *BooleanType) IsInstance(o PValue, g Guard) bool {
	if bo, ok := o.(*BooleanValue); ok {
		return t.value == -1 || t.value == bo.value
	}
	return false
}

func (t *BooleanType) Parameters() []PValue {
  if t.value == -1 {
  	return EMPTY_VALUES
	}
	return []PValue{&BooleanValue{t.value}}
}

func (t *BooleanType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *BooleanType) Type() PType {
	return &TypeType{t}
}

func WrapBoolean(val bool) *BooleanValue {
	n := 0
	if val {
		n = 1
	}
	return &BooleanValue{n}
}

func (bv *BooleanValue) Bool() bool {
	return bv.value == 1
}

func (bv *BooleanValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*BooleanValue); ok {
		return bv.value == ov.value
	}
	return false
}

func (bv *BooleanValue) String() string {
	if bv.value == 1 {
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
	return int64(bv.value)
}

func (bv *BooleanValue) floatVal() float64 {
	return float64(bv.value)
}

func (bv *BooleanValue) stringVal(alt bool, yes string, no string) string {
	str := no
	if bv.value == 1{
		str = yes
	}
	if alt {
		str = str[:1]
	}
	return str
}

func (bv *BooleanValue) ToKey() HashKey {
	if bv.value == 1 {
		return HashKey([]byte{1, HK_BOOLEAN, 1})
	}
	return HashKey([]byte{1, HK_BOOLEAN, 0})
}

func (bv *BooleanValue) Type() PType {
	return DefaultBooleanType()
}
