package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
)

type (
	IntegerType struct {
		min int64
		max int64
	}

	// IntegerValue represents IntegerType as a value
	IntegerValue IntegerType
)

var IntegerType_POSITIVE = &IntegerType{0, math.MaxInt64}
var IntegerType_ZERO = &IntegerType{0, 0}
var IntegerType_ONE = &IntegerType{1, 1}
var IntegerType_TWO = &IntegerType{2, 2}

var Integer_ZERO = &IntegerValue{0, 0}
var Integer_ONE = &IntegerValue{1, 1}
var Integer_TWO = &IntegerValue{2, 2}

var integerType_DEFAULT = &IntegerType{math.MinInt64, math.MaxInt64}
var integerType_8 = &IntegerType{math.MinInt8, math.MaxInt8}
var integerType_16 = &IntegerType{math.MinInt16, math.MaxInt16}
var integerType_32 = &IntegerType{math.MinInt32, math.MaxInt32}
var integerType_u8 = &IntegerType{0, math.MaxUint8}
var integerType_u16 = &IntegerType{0, math.MaxUint16}
var integerType_u32 = &IntegerType{0, math.MaxUint32}
var integerType_u64 = IntegerType_POSITIVE // MaxUInt64 isn't supported at this time
var ZERO = (*IntegerValue)(IntegerType_ZERO)
var MIN_INT = WrapInteger(math.MinInt64)
var MAX_INT = WrapInteger(math.MaxInt64)

var Integer_Type eval.ObjectType

func init() {
	Integer_Type = newObjectType(`Pcore::IntegerType`,
		`Pcore::NumericType {
  attributes => {
    from => { type => Optional[Integer], value => undef },
    to => { type => Optional[Integer], value => undef }
  }
}`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return NewIntegerType2(args...)
		})

	newGoConstructor2(`Integer`,
		func(t eval.LocalTypes) {
			t.Type(`Radix`, `Variant[Default, Integer[2,2], Integer[8,8], Integer[10,10], Integer[16,16]]`)
			t.Type(`Convertible`, `Variant[Numeric, Boolean, Pattern[/`+INTEGER_PATTERN+`/], Timespan, Timestamp]`)
			t.Type(`NamedArgs`, `Struct[{from => Convertible, Optional[radix] => Radix, Optional[abs] => Boolean}]`)
		},

		func(d eval.Dispatch) {
			d.Param(`Convertible`)
			d.OptionalParam(`Radix`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				r := 10
				abs := false
				if len(args) > 1 {
					if radix, ok := args[1].(*IntegerValue); ok {
						r = int(radix.Int())
					}
					if len(args) > 2 {
						abs = args[2].(*BooleanValue).Bool()
					}
				}
				n := intFromConvertible(c, args[0], r)
				if abs && n < 0 {
					n = -n
				}
				return WrapInteger(n)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`NamedArgs`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				h := args[0].(*HashValue)
				r := 10
				abs := false
				if rx, ok := h.Get4(`radix`); ok {
					if radix, ok := rx.(*IntegerValue); ok {
						r = int(radix.Int())
					}
				}
				if ab, ok := h.Get4(`abs`); ok {
					abs = ab.(*BooleanValue).Bool()
				}
				n := intFromConvertible(c, h.Get5(`from`, _UNDEF), r)
				if abs && n < 0 {
					n = -n
				}
				return WrapInteger(n)
			})
		},
	)
}

func intFromConvertible(c eval.Context, from eval.PValue, radix int) int64 {
	switch from.(type) {
	case *IntegerValue:
		return from.(*IntegerValue).Int()
	case *FloatValue:
		return from.(*FloatValue).Int()
	case *TimestampValue:
		return from.(*TimestampValue).Int()
	case *TimespanValue:
		return from.(*TimespanValue).Int()
	case *BooleanValue:
		return from.(*BooleanValue).Int()
	default:
		i, err := strconv.ParseInt(from.String(), radix, 64)
		if err == nil {
			return i
		}
		panic(eval.Error(c, eval.EVAL_NOT_INTEGER, issue.H{`value`: from}))
	}
}

func DefaultIntegerType() *IntegerType {
	return integerType_DEFAULT
}

func PositiveIntegerType() *IntegerType {
	return IntegerType_POSITIVE
}

func NewIntegerType(min int64, max int64) *IntegerType {
	if min == math.MinInt64 {
		if max == math.MaxInt64 {
			return DefaultIntegerType()
		}
	} else if min == 0 {
		if max == math.MaxInt64 {
			return PositiveIntegerType()
		} else if max == 0 {
			return IntegerType_ZERO
		}
	} else if min == 1 && max == 1 {
		return IntegerType_ONE
	}
	if min > max {
		panic(errors.NewArgumentsError(`Integer[]`, `min is not allowed to be greater than max`))
	}
	return &IntegerType{min, max}
}

func NewIntegerType2(limits ...eval.PValue) *IntegerType {
	argc := len(limits)
	if argc == 0 {
		return integerType_DEFAULT
	}
	min, ok := toInt(limits[0])
	if !ok {
		if _, ok = limits[0].(*DefaultValue); !ok {
			panic(NewIllegalArgumentType2(`Integer[]`, 0, `Integer`, limits[0]))
		}
		min = math.MinInt64
	}

	var max int64
	switch len(limits) {
	case 1:
		max = math.MaxInt64
	case 2:
		max, ok = toInt(limits[1])
		if !ok {
			if _, ok = limits[1].(*DefaultValue); !ok {
				panic(NewIllegalArgumentType2(`Integer[]`, 1, `Integer`, limits[1]))
			}
			max = math.MaxInt64
		}
	default:
		panic(errors.NewIllegalArgumentCount(`Integer[]`, `0 - 2`, len(limits)))
	}
	return NewIntegerType(min, max)
}

func (t *IntegerType) Default() eval.PType {
	return integerType_DEFAULT
}

func (t *IntegerType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *IntegerType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*IntegerType); ok {
		return t.min == ot.min && t.max == ot.max
	}
	return false
}

func (t *IntegerType) Generic() eval.PType {
	return integerType_DEFAULT
}

func (t *IntegerType) Get(c eval.Context, key string) (eval.PValue, bool) {
	switch key {
	case `from`:
		v := eval.UNDEF
		if t.min != math.MinInt64 {
			v = WrapInteger(t.min)
		}
		return v, true
	case `to`:
		v := eval.UNDEF
		if t.max != math.MaxInt64 {
			v = WrapInteger(t.max)
		}
		return v, true
	default:
		return nil, false
	}
}

func (t *IntegerType) IsAssignable(o eval.PType, g eval.Guard) bool {
	if it, ok := o.(*IntegerType); ok {
		return t.min <= it.min && t.max >= it.max
	}
	return false
}

func (t *IntegerType) IsInstance(c eval.Context, o eval.PValue, g eval.Guard) bool {
	if n, ok := toInt(o); ok {
		return t.IsInstance2(n)
	}
	return false
}

func (t *IntegerType) IsInstance2(n int64) bool {
	return t.min <= n && n <= t.max
}

func (t *IntegerType) IsInstance3(n int) bool {
	return t.IsInstance2(int64(n))
}

func (t *IntegerType) IsUnbounded() bool {
	return t.min == math.MinInt64 && t.max == math.MaxInt64
}

func (t *IntegerType) Min() int64 {
	return t.min
}

func (t *IntegerType) Max() int64 {
	return t.max
}

func (t *IntegerType) MetaType() eval.ObjectType {
	return Integer_Type
}

func (t *IntegerType) Name() string {
	return `Integer`
}

func (t *IntegerType) Parameters() []eval.PValue {
	if t.min == math.MinInt64 {
		if t.max == math.MaxInt64 {
			return eval.EMPTY_VALUES
		}
		return []eval.PValue{WrapDefault(), WrapInteger(t.max)}
	}
	if t.max == math.MaxInt64 {
		return []eval.PValue{WrapInteger(t.min)}
	}
	return []eval.PValue{WrapInteger(t.min), WrapInteger(t.max)}
}

func (t *IntegerType) ReflectType() (reflect.Type, bool) {
	return reflect.TypeOf(int64(0)), true
}

func (t *IntegerType) SizeParameters() []eval.PValue {
	params := make([]eval.PValue, 2)
	params[0] = WrapInteger(t.min)
	if t.max == math.MaxInt64 {
		params[1] = WrapDefault()
	} else {
		params[1] = WrapInteger(t.max)
	}
	return params
}

func (t *IntegerType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *IntegerType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *IntegerType) Type() eval.PType {
	return &TypeType{t}
}

func WrapInteger(val int64) *IntegerValue {
	return (*IntegerValue)(NewIntegerType(val, val))
}

func (iv *IntegerValue) Abs() eval.NumericValue {
	if iv.Int() < 0 {
		return WrapInteger(-iv.Int())
	}
	return iv
}

func (iv *IntegerValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*IntegerValue); ok {
		return iv.Int() == ov.Int()
	}
	return false
}

func (iv *IntegerValue) Float() float64 {
	return float64(iv.Int())
}

func (iv *IntegerValue) Int() int64 {
	return iv.min
}

func (iv *IntegerValue) Reflect(c eval.Context) reflect.Value {
	return reflect.ValueOf(iv.Int())
}

func (iv *IntegerValue) ReflectTo(c eval.Context, value reflect.Value) {
	if !value.CanSet() {
		panic(eval.Error(c, eval.EVAL_ATTEMPT_TO_SET_UNSETTABLE, issue.H{`kind`: reflect.Int.String()}))
	}
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(iv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value.SetUint(uint64(iv.Int()))
	case reflect.Interface:
		value.Set(reflect.ValueOf(iv.Int()))
	default:
		panic(eval.Error(c, eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: reflect.Int.String(), `actual`: value.Kind().String()}))
	}
}

func (iv *IntegerValue) String() string {
	return fmt.Sprintf(`%d`, iv.Int())
}

func (iv *IntegerValue) ToKey(b *bytes.Buffer) {
	n := iv.Int()
	b.WriteByte(1)
	b.WriteByte(HK_INTEGER)
	b.WriteByte(byte(n >> 56))
	b.WriteByte(byte(n >> 48))
	b.WriteByte(byte(n >> 40))
	b.WriteByte(byte(n >> 32))
	b.WriteByte(byte(n >> 24))
	b.WriteByte(byte(n >> 16))
	b.WriteByte(byte(n >> 8))
	b.WriteByte(byte(n))
}

func (iv *IntegerValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), iv.Type())
	switch f.FormatChar() {
	case 'x', 'X', 'o', 'd':
		fmt.Fprintf(b, f.OrigFormat(), iv.Int())
	case 'p', 'b', 'B':
		longVal := iv.Int()
		intString := strconv.FormatInt(longVal, integerRadix(f.FormatChar()))
		totWidth := 0
		if f.Width() > 0 {
			totWidth = f.Width()
		}
		numWidth := 0
		if f.Precision() > 0 {
			numWidth = f.Precision()
		}

		if numWidth > 0 && numWidth < len(intString) && f.FormatChar() == 'p' {
			intString = intString[:numWidth]
		}

		zeroPad := numWidth - len(intString)

		pfx := ``
		if f.IsAlt() && longVal != 0 && !(f.FormatChar() == 'o' && zeroPad > 0) {
			pfx = integerPrefixRadix(f.FormatChar())
		}
		computedFieldWidth := len(pfx) + intMax(numWidth, len(intString))

		for spacePad := totWidth - computedFieldWidth; spacePad > 0; spacePad-- {
			b.Write([]byte{' '})
		}

		io.WriteString(b, pfx)
		if zeroPad > 0 {
			padChar := []byte{'0'}
			if f.FormatChar() == 'p' {
				padChar = []byte{' '}
			}
			for ; zeroPad > 0; zeroPad-- {
				b.Write(padChar)
			}
		}
		io.WriteString(b, intString)
	case 'e', 'E', 'f', 'g', 'G', 'a', 'A':
		WrapFloat(iv.Float()).ToString(b, eval.NewFormatContext(DefaultFloatType(), f, s.Indentation()), g)
	case 'c':
		bld := bytes.NewBufferString(``)
		bld.WriteRune(rune(iv.Int()))
		f.ApplyStringFlags(b, bld.String(), f.IsAlt())
	case 's':
		f.ApplyStringFlags(b, strconv.Itoa(int(iv.Int())), f.IsAlt())
	default:
		panic(s.UnsupportedFormat(iv.Type(), `dxXobBeEfgGaAspc`, f))
	}
}

func intMax(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func integerRadix(c byte) int {
	switch c {
	case 'b', 'B':
		return 2
	case 'o':
		return 8
	case 'x', 'X':
		return 16
	default:
		return 10
	}
}

func integerPrefixRadix(c byte) string {
	switch c {
	case 'x':
		return `0x`
	case 'X':
		return `0X`
	case 'o':
		return `0`
	case 'b':
		return `0b`
	case 'B':
		return `0B`
	default:
		return ``
	}
}

func (iv *IntegerValue) Type() eval.PType {
	return (*IntegerType)(iv)
}
