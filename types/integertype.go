package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"reflect"
)

type (
	IntegerType struct {
		min int64
		max int64
	}

	// integerValue represents int64 as a value
	integerValue int64
)

var IntegerTypePositive = &IntegerType{0, math.MaxInt64}
var IntegerTypeZero = &IntegerType{0, 0}
var IntegerTypeOne = &IntegerType{1, 1}

var ZERO = integerValue(0)

var integerTypeDefault = &IntegerType{math.MinInt64, math.MaxInt64}
var integerType8 = &IntegerType{math.MinInt8, math.MaxInt8}
var integerType16 = &IntegerType{math.MinInt16, math.MaxInt16}
var integerType32 = &IntegerType{math.MinInt32, math.MaxInt32}
var integerTypeU8 = &IntegerType{0, math.MaxUint8}
var integerTypeU16 = &IntegerType{0, math.MaxUint16}
var integerTypeU32 = &IntegerType{0, math.MaxUint32}
var integerTypeU64 = IntegerTypePositive // MaxUInt64 isn't supported at this time

var IntegerMetaType eval.ObjectType

func init() {
	IntegerMetaType = newObjectType(`Pcore::IntegerType`,
		`Pcore::NumericType {
  attributes => {
    from => { type => Optional[Integer], value => undef },
    to => { type => Optional[Integer], value => undef }
  }
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
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
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				r := 10
				abs := false
				if len(args) > 1 {
					if radix, ok := args[1].(integerValue); ok {
						r = int(radix)
					}
					if len(args) > 2 {
						abs = args[2].(*BooleanValue).Bool()
					}
				}
				n := intFromConvertible(args[0], r)
				if abs && n < 0 {
					n = -n
				}
				return integerValue(n)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`NamedArgs`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				h := args[0].(*HashValue)
				r := 10
				abs := false
				if rx, ok := h.Get4(`radix`); ok {
					if radix, ok := rx.(integerValue); ok {
						r = int(radix)
					}
				}
				if ab, ok := h.Get4(`abs`); ok {
					abs = ab.(*BooleanValue).Bool()
				}
				n := intFromConvertible(h.Get5(`from`, _UNDEF), r)
				if abs && n < 0 {
					n = -n
				}
				return integerValue(n)
			})
		},
	)
}

func intFromConvertible(from eval.Value, radix int) int64 {
	switch from.(type) {
	case integerValue:
		return from.(integerValue).Int()
	case floatValue:
		return from.(floatValue).Int()
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
		panic(eval.Error(eval.EVAL_NOT_INTEGER, issue.H{`value`: from}))
	}
}

func DefaultIntegerType() *IntegerType {
	return integerTypeDefault
}

func PositiveIntegerType() *IntegerType {
	return IntegerTypePositive
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
			return IntegerTypeZero
		}
	} else if min == 1 && max == 1 {
		return IntegerTypeOne
	}
	if min > max {
		panic(errors.NewArgumentsError(`Integer[]`, `min is not allowed to be greater than max`))
	}
	return &IntegerType{min, max}
}

func NewIntegerType2(limits ...eval.Value) *IntegerType {
	argc := len(limits)
	if argc == 0 {
		return integerTypeDefault
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

func (t *IntegerType) Default() eval.Type {
	return integerTypeDefault
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

func (t *IntegerType) Generic() eval.Type {
	return integerTypeDefault
}

func (t *IntegerType) Get(key string) (eval.Value, bool) {
	switch key {
	case `from`:
		v := eval.UNDEF
		if t.min != math.MinInt64 {
			v = integerValue(t.min)
		}
		return v, true
	case `to`:
		v := eval.UNDEF
		if t.max != math.MaxInt64 {
			v = integerValue(t.max)
		}
		return v, true
	default:
		return nil, false
	}
}

func (t *IntegerType) IsAssignable(o eval.Type, g eval.Guard) bool {
	if it, ok := o.(*IntegerType); ok {
		return t.min <= it.min && t.max >= it.max
	}
	return false
}

func (t *IntegerType) IsInstance(o eval.Value, g eval.Guard) bool {
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
	return IntegerMetaType
}

func (t *IntegerType) Name() string {
	return `Integer`
}

func (t *IntegerType) Parameters() []eval.Value {
	if t.min == math.MinInt64 {
		if t.max == math.MaxInt64 {
			return eval.EMPTY_VALUES
		}
		return []eval.Value{WrapDefault(), integerValue(t.max)}
	}
	if t.max == math.MaxInt64 {
		return []eval.Value{integerValue(t.min)}
	}
	return []eval.Value{integerValue(t.min), integerValue(t.max)}
}

func (t *IntegerType) ReflectType(c eval.Context) (reflect.Type, bool) {
	return reflect.TypeOf(int64(0)), true
}

func (t *IntegerType) SizeParameters() []eval.Value {
	params := make([]eval.Value, 2)
	params[0] = integerValue(t.min)
	if t.max == math.MaxInt64 {
		params[1] = WrapDefault()
	} else {
		params[1] = integerValue(t.max)
	}
	return params
}

func (t *IntegerType) CanSerializeAsString() bool {
	return true
}

func (t *IntegerType) SerializationString() string {
	return t.String()
}

func (t *IntegerType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *IntegerType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *IntegerType) PType() eval.Type {
	return &TypeType{t}
}

func WrapInteger(val int64) eval.IntegerValue {
	return integerValue(val)
}

func (iv integerValue) Abs() int64 {
	if iv < 0 {
		return -int64(iv)
	}
	return int64(iv)
}

func (iv integerValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(integerValue); ok {
		return iv == ov
	}
	return false
}

func (iv integerValue) Float() float64 {
	return float64(iv)
}

func (iv integerValue) Int() int64 {
	return int64(iv)
}

func (iv integerValue) Reflect(c eval.Context) reflect.Value {
	return reflect.ValueOf(int64(iv))
}

func (iv integerValue) ReflectTo(c eval.Context, value reflect.Value) {
	if !value.CanSet() {
		panic(eval.Error(eval.EVAL_ATTEMPT_TO_SET_UNSETTABLE, issue.H{`kind`: reflect.Int.String()}))
	}
	ok := true
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(int64(iv))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value.SetUint(uint64(iv))
	case reflect.Interface:
		value.Set(reflect.ValueOf(int64(iv)))
	case reflect.Ptr:
		switch value.Type().Elem().Kind() {
		case reflect.Int64:
			v := int64(iv)
			value.Set(reflect.ValueOf(&v))
		case reflect.Int:
			v := int(iv)
			value.Set(reflect.ValueOf(&v))
		case reflect.Int8:
			v := int8(iv)
			value.Set(reflect.ValueOf(&v))
		case reflect.Int16:
			v := int16(iv)
			value.Set(reflect.ValueOf(&v))
		case reflect.Int32:
			v := int32(iv)
			value.Set(reflect.ValueOf(&v))
		case reflect.Uint:
			v := uint(iv)
			value.Set(reflect.ValueOf(&v))
		case reflect.Uint8:
			v := uint8(iv)
			value.Set(reflect.ValueOf(&v))
		case reflect.Uint16:
			v := uint16(iv)
			value.Set(reflect.ValueOf(&v))
		case reflect.Uint32:
			v := uint32(iv)
			value.Set(reflect.ValueOf(&v))
		case reflect.Uint64:
			v := uint64(iv)
			value.Set(reflect.ValueOf(&v))
		default:
			ok = false
		}
	default:
		ok = false
	}
	if !ok {
		panic(eval.Error(eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: reflect.Int.String(), `actual`: value.Kind().String()}))
	}
}

func (iv integerValue) String() string {
	return fmt.Sprintf(`%d`, int64(iv))
}

func (iv integerValue) ToKey(b *bytes.Buffer) {
	n := int64(iv)
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

func (iv integerValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), iv.PType())
	var err error
	switch f.FormatChar() {
	case 'x', 'X', 'o', 'd':
		_, err = fmt.Fprintf(b, f.OrigFormat(), int64(iv))
	case 'p', 'b', 'B':
		longVal := int64(iv)
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
			_, err = b.Write([]byte{' '})
			if err != nil {
				break
			}
		}
		if err != nil {
			break
		}

		_, err = io.WriteString(b, pfx)
		if err != nil {
			break
		}
		if zeroPad > 0 {
			padChar := []byte{'0'}
			if f.FormatChar() == 'p' {
				padChar = []byte{' '}
			}
			for ; zeroPad > 0; zeroPad-- {
				_, err = b.Write(padChar)
				if err != nil {
					break
				}
			}
		}
		if err == nil {
			_, err = io.WriteString(b, intString)
		}
	case 'e', 'E', 'f', 'g', 'G', 'a', 'A':
		floatValue(iv.Float()).ToString(b, eval.NewFormatContext(DefaultFloatType(), f, s.Indentation()), g)
	case 'c':
		bld := bytes.NewBufferString(``)
		bld.WriteRune(rune(int64(iv)))
		f.ApplyStringFlags(b, bld.String(), f.IsAlt())
	case 's':
		f.ApplyStringFlags(b, strconv.Itoa(int(int64(iv))), f.IsAlt())
	default:
		panic(s.UnsupportedFormat(iv.PType(), `dxXobBeEfgGaAspc`, f))
	}
	if err != nil {
		panic(err)
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

func (iv integerValue) PType() eval.Type {
	v := int64(iv)
	return &IntegerType{v, v}
}
