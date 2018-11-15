package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
)

type (
	FloatType struct {
		min float64
		max float64
	}

	// FloatValue represents FloatType as a value
	FloatValue FloatType
)

var floatType_DEFAULT = &FloatType{-math.MaxFloat64, math.MaxFloat64}
var floatType_32 = &FloatType{-math.MaxFloat32, math.MaxFloat32}

var Float_Type eval.ObjectType

func init() {
	Float_Type = newObjectType(`Pcore::FloatType`,
		`Pcore::NumericType {
  attributes => {
    from => { type => Optional[Float], value => undef },
    to => { type => Optional[Float], value => undef }
  }
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewFloatType2(args...)
		})

	newGoConstructor2(`Float`,
		func(t eval.LocalTypes) {
			t.Type(`Convertible`, `Variant[Numeric, Boolean, Pattern[/`+FLOAT_PATTERN+`/], Timespan, Timestamp]`)
			t.Type(`NamedArgs`, `Struct[{from => Convertible, Optional[abs] => Boolean}]`)
		},

		func(d eval.Dispatch) {
			d.Param(`Convertible`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return numberFromPositionalArgs(args, false)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`NamedArgs`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return numberFromNamedArgs(args, false)
			})
		},
	)
}

func DefaultFloatType() *FloatType {
	return floatType_DEFAULT
}

func NewFloatType(min float64, max float64) *FloatType {
	if min == -math.MaxFloat64 && max == math.MaxFloat64 {
		return DefaultFloatType()
	}
	if min > max {
		panic(errors.NewArgumentsError(`Float[]`, `min is not allowed to be greater than max`))
	}
	return &FloatType{min, max}
}

func NewFloatType2(limits ...eval.Value) *FloatType {
	argc := len(limits)
	if argc == 0 {
		return floatType_DEFAULT
	}
	min, ok := toFloat(limits[0])
	if !ok {
		if _, ok = limits[0].(*DefaultValue); !ok {
			panic(NewIllegalArgumentType2(`Float[]`, 0, `Float`, limits[0]))
		}
		min = -math.MaxFloat64
	}

	var max float64
	switch argc {
	case 1:
		max = math.MaxFloat64
	case 2:
		if max, ok = toFloat(limits[1]); !ok {
			if _, ok = limits[1].(*DefaultValue); !ok {
				panic(NewIllegalArgumentType2(`Float[]`, 1, `Float`, limits[1]))
			}
			max = math.MaxFloat64
		}
	default:
		panic(errors.NewIllegalArgumentCount(`Float`, `0 - 2`, len(limits)))
	}
	return NewFloatType(min, max)
}

func (t *FloatType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *FloatType) Default() eval.Type {
	return floatType_DEFAULT
}

func (t *FloatType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*FloatType); ok {
		return t.min == ot.min && t.max == ot.max
	}
	return false
}

func (t *FloatType) Generic() eval.Type {
	return floatType_DEFAULT
}

func (t *FloatType) Get(key string) (eval.Value, bool) {
	switch key {
	case `from`:
		v := eval.UNDEF
		if t.min != -math.MaxFloat64 {
			v = WrapFloat(t.min)
		}
		return v, true
	case `to`:
		v := eval.UNDEF
		if t.max != math.MaxFloat64 {
			v = WrapFloat(t.max)
		}
		return v, true
	default:
		return nil, false
	}
}

func (t *FloatType) IsAssignable(o eval.Type, g eval.Guard) bool {
	if ft, ok := o.(*FloatType); ok {
		return t.min <= ft.min && t.max >= ft.max
	}
	return false
}

func (t *FloatType) IsInstance(o eval.Value, g eval.Guard) bool {
	if n, ok := toFloat(o); ok {
		return t.min <= n && n <= t.max
	}
	return false
}

func (t *FloatType) MetaType() eval.ObjectType {
	return Float_Type
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

func (t *FloatType) Parameters() []eval.Value {
	if t.min == -math.MaxFloat64 {
		if t.max == math.MaxFloat64 {
			return eval.EMPTY_VALUES
		}
		return []eval.Value{WrapDefault(), WrapFloat(t.max)}
	}
	if t.max == math.MaxFloat64 {
		return []eval.Value{WrapFloat(t.min)}
	}
	return []eval.Value{WrapFloat(t.min), WrapFloat(t.max)}
}

func (t *FloatType) ReflectType(c eval.Context) (reflect.Type, bool) {
	return reflect.TypeOf(float64(0.0)), true
}

func (t *FloatType) SerializationString() string {
	return t.String()
}

func (t *FloatType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *FloatType) IsUnbounded() bool {
	return t.min == -math.MaxFloat64 && t.max == math.MaxFloat64
}

func (t *FloatType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *FloatType) PType() eval.Type {
	return &TypeType{t}
}

func WrapFloat(val float64) *FloatValue {
	return (*FloatValue)(NewFloatType(val, val))
}

func (fv *FloatValue) Abs() eval.NumericValue {
	if fv.Float() < 0 {
		return WrapFloat(-fv.Float())
	}
	return fv
}

func (fv *FloatValue) Equals(o interface{}, g eval.Guard) bool {
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

func (fv *FloatValue) Reflect(c eval.Context) reflect.Value {
	return reflect.ValueOf(fv.Float())
}

func (fv *FloatValue) ReflectTo(c eval.Context, value reflect.Value) {
	switch value.Kind() {
	case reflect.Float64, reflect.Float32:
		value.SetFloat(fv.Float())
	case reflect.Interface:
		value.Set(reflect.ValueOf(fv.Float()))
	default:
		panic(eval.Error(eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: `Float`, `actual`: value.Kind().String()}))
	}
}

func (fv *FloatValue) String() string {
	return fmt.Sprintf(`%v`, fv.Float())
}

func (fv *FloatValue) ToKey(b *bytes.Buffer) {
	n := math.Float64bits(fv.Float())
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

func (fv *FloatValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), fv.PType())
	switch f.FormatChar() {
	case 'd', 'x', 'X', 'o', 'b', 'B':
		WrapInteger(fv.Int()).ToString(b, eval.NewFormatContext(DefaultIntegerType(), f, s.Indentation()), g)
	case 'p':
		f.ApplyStringFlags(b, floatGFormat(DEFAULT_P_FORMAT, fv.Float()), false)
	case 'e', 'E', 'f':
		fmt.Fprintf(b, f.OrigFormat(), fv.Float())
	case 'g', 'G':
		io.WriteString(b, floatGFormat(f, fv.Float()))
	case 's':
		f.ApplyStringFlags(b, floatGFormat(DEFAULT_S_FORMAT, fv.Float()), f.IsAlt())
	case 'a', 'A':
		// TODO: Implement this or list as limitation?
		panic(s.UnsupportedFormat(fv.PType(), `dxXobBeEfgGaAsp`, f))
	default:
		panic(s.UnsupportedFormat(fv.PType(), `dxXobBeEfgGaAsp`, f))
	}
}

func floatGFormat(f eval.Format, value float64) string {
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

func (fv *FloatValue) PType() eval.Type {
	return (*FloatType)(fv)
}
