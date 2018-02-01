package types

import (
	"fmt"
	. "io"
	. "math"

	"strings"

	"bytes"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/eval"
)

type (
	FloatType struct {
		min float64
		max float64
	}

	// FloatValue represents FloatType as a value
	FloatValue FloatType
)

var floatType_DEFAULT = &FloatType{-MaxFloat64, MaxFloat64}

func DefaultFloatType() *FloatType {
	return floatType_DEFAULT
}

func NewFloatType(min float64, max float64) *FloatType {
	if min == -MaxFloat64 && max == MaxFloat64 {
		return DefaultFloatType()
	}
	if min > max {
		panic(NewArgumentsError(`Float[]`, `min is not allowed to be greater than max`))
	}
	return &FloatType{min, max}
}

func NewFloatType2(limits ...PValue) *FloatType {
	argc := len(limits)
	if argc == 0 {
		return floatType_DEFAULT
	}
	min, ok := toFloat(limits[0])
	if !ok {
		if _, ok = limits[0].(*DefaultValue); !ok {
			panic(NewIllegalArgumentType2(`Float[]`, 0, `Float`, limits[0]))
		}
		min = -MaxFloat64
	}

	var max float64
	switch argc {
	case 1:
		max = MaxFloat64
	case 2:
		if max, ok = toFloat(limits[1]); !ok {
			if _, ok = limits[1].(*DefaultValue); !ok {
				panic(NewIllegalArgumentType2(`Float[]`, 1, `Float`, limits[1]))
			}
			max = MaxFloat64
		}
	default:
		panic(NewIllegalArgumentCount(`Float`, `0 - 2`, len(limits)))
	}
	return NewFloatType(min, max)
}

func (t *FloatType) Accept(v Visitor, g Guard) {
	v(t)
}

func (t *FloatType) Default() PType {
	return floatType_DEFAULT
}

func (t *FloatType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*FloatType); ok {
		return t.min == ot.min && t.max == ot.max
	}
	return false
}

func (t *FloatType) Generic() PType {
	return floatType_DEFAULT
}

func (t *FloatType) IsAssignable(o PType, g Guard) bool {
	if ft, ok := o.(*FloatType); ok {
		return t.min <= ft.min && t.max >= ft.max
	}
	return false
}

func (t *FloatType) IsInstance(o PValue, g Guard) bool {
	if n, ok := toFloat(o); ok {
		return t.min <= n && n <= t.max
	}
	return false
}

func (t *FloatType) Min() float64 {
	return t.min
}

func (t *FloatType) Max() float64 {
	return t.max
}

func (t *FloatType) Name() string {
	return `Float`
}

func (t *FloatType) Parameters() []PValue {
	if t.min == -MaxFloat64 {
		if t.max == MaxFloat64 {
			return EMPTY_VALUES
		}
		return []PValue{WrapDefault(), WrapFloat(t.max)}
	}
	if t.max == MaxFloat64 {
		return []PValue{WrapFloat(t.min)}
	}
	return []PValue{WrapFloat(t.min), WrapFloat(t.max)}
}

func (t *FloatType) String() string {
	return ToString2(t, NONE)
}

func (t *FloatType) IsUnbounded() bool {
	return t.min == -MaxFloat64 && t.max == MaxFloat64
}

func (t *FloatType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *FloatType) Type() PType {
	return &TypeType{t}
}

func WrapFloat(val float64) *FloatValue {
	return (*FloatValue)(NewFloatType(val, val))
}

func (fv *FloatValue) Abs() NumericValue {
	if fv.Float() < 0 {
		return WrapFloat(-fv.Float())
	}
	return fv
}

func (fv *FloatValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*FloatValue); ok {
		return fv.Float() == ov.Float()
	}
	return false
}

func (fv *FloatValue) Float() float64 {
	return fv.min
}

func (fv *FloatValue) Int() int64 {
	return int64(fv.Float())
}

func (fv *FloatValue) String() string {
	return fmt.Sprintf(`%v`, fv.Float())
}

func (fv *FloatValue) ToKey(b *bytes.Buffer) {
	n := Float64bits(fv.Float())
	b.WriteByte(1)
	b.WriteByte(HK_FLOAT)
	b.WriteByte(byte(n >> 56))
	b.WriteByte(byte(n >> 48))
	b.WriteByte(byte(n >> 40))
	b.WriteByte(byte(n >> 32))
	b.WriteByte(byte(n >> 24))
	b.WriteByte(byte(n >> 16))
	b.WriteByte(byte(n >> 8))
	b.WriteByte(byte(n))
}

var DEFAULT_P_FORMAT = newFormat(`%g`)
var DEFAULT_S_FORMAT = newFormat(`%#g`)

func (fv *FloatValue) ToString(b Writer, s FormatContext, g RDetect) {
	f := GetFormat(s.FormatMap(), fv.Type())
	switch f.FormatChar() {
	case 'd', 'x', 'X', 'o', 'b', 'B':
		WrapInteger(fv.Int()).ToString(b, NewFormatContext(DefaultIntegerType(), f, s.Indentation()), g)
	case 'p':
		f.ApplyStringFlags(b, floatGFormat(DEFAULT_P_FORMAT, fv.Float()), false)
	case 'e', 'E', 'f':
		fmt.Fprintf(b, f.OrigFormat(), fv.Float())
	case 'g', 'G':
		WriteString(b, floatGFormat(f, fv.Float()))
	case 's':
		f.ApplyStringFlags(b, floatGFormat(DEFAULT_S_FORMAT, fv.Float()), f.IsAlt())
	case 'a', 'A':
		// TODO: Implement this or list as limitation?
		panic(s.UnsupportedFormat(fv.Type(), `dxXobBeEfgGaAsp`, f))
	default:
		panic(s.UnsupportedFormat(fv.Type(), `dxXobBeEfgGaAsp`, f))
	}
}

func floatGFormat(f Format, value float64) string {
	str := fmt.Sprintf(f.WithoutWidth().OrigFormat(), value)
	sc := byte('e')
	if f.FormatChar() == 'G' {
		sc = 'E'
	}
	if strings.IndexByte(str, sc) >= 0 {
		// Scientific notation in use.
		return str
	}

	// Go might strip both trailing zeroes and decimal point when using '%g'. The
	// decimal point and trailing zeroes are restored here
	totLen := len(str)
	prec := f.Precision()
	if prec < 0 && !f.IsAlt() {
		prec = 6
	}

	dotIndex := strings.IndexByte(str, '.')
	missing := 0
	if prec >= 0 {
		if dotIndex >= 0 {
			missing = prec - (totLen - 1)
		} else {
			missing = prec - totLen
			if missing == 0 {
				// Impossible to add a fraction part. Force scientific notation
				return fmt.Sprintf(f.ReplaceFormatChar(sc).OrigFormat(), value)
			}
		}
	}

	b := bytes.NewBufferString(``)

	padByte := byte(' ')
	if f.IsZeroPad() {
		padByte = '0'
	}
	pad := 0
	if f.Width() > 0 {
		pad = f.Width() - (totLen + missing + 1)
	}

	if !f.IsLeft() {
		for ; pad > 0; pad-- {
			b.WriteByte(padByte)
		}
	}

	b.WriteString(str)
	if dotIndex < 0 {
		b.WriteByte('.')
		if missing == 0 {
			b.WriteByte('0')
		}
	}
	for missing > 0 {
		b.WriteByte('0')
		missing--
	}

	if f.IsLeft() {
		for ; pad > 0; pad-- {
			b.WriteByte(padByte)
		}
	}
	return b.String()
}

func (fv *FloatValue) Type() PType {
	return (*FloatType)(fv)
}
