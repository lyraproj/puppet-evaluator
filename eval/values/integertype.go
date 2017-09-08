package values

import (
	"bytes"
	. "fmt"
	. "io"
	. "math"
	"strconv"

	. "github.com/puppetlabs/go-evaluator/eval/errors"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
)

type (
	IntegerType struct {
		min int64
		max int64
	}

	// IntegerValue represents IntegerType as a value
	IntegerValue IntegerType
)

var integerType_DEFAULT = &IntegerType{MinInt64, MaxInt64}
var integerType_POSITIVE = &IntegerType{0, MaxInt64}
var integerType_ZERO = &IntegerType{0, 0}
var integerType_ONE = &IntegerType{1, 1}
var ZERO = (*IntegerValue)(integerType_ZERO)

func DefaultIntegerType() *IntegerType {
	return integerType_DEFAULT
}

func PositiveIntegerType() *IntegerType {
	return integerType_POSITIVE
}

func NewIntegerType(min int64, max int64) *IntegerType {
	if min == MinInt64 {
		if max == MaxInt64 {
			return DefaultIntegerType()
		}
	} else if min == 0 {
		if max == MaxInt64 {
			return PositiveIntegerType()
		} else if max == 0 {
			return integerType_ZERO
		}
	} else if min == 1 && max == 1 {
		return integerType_ONE
	}
	if min > max {
		panic(NewArgumentsError(`Integer[]`, `min is not allowed to be greater than max`))
	}
	return &IntegerType{min, max}
}

func NewIntegerType2(limits ...PValue) *IntegerType {
	argc := len(limits)
	if argc == 0 {
		return integerType_DEFAULT
	}
	min, ok := toInt(limits[0])
	if !ok {
		if _, ok = limits[0].(*DefaultValue); !ok {
			panic(NewIllegalArgumentType2(`Integer[]`, 0, `Integer`, limits[0]))
		}
		min = MinInt64
	}

	var max int64
	switch len(limits) {
	case 1:
		max = MaxInt64
	case 2:
		max, ok = toInt(limits[1])
		if !ok {
			if _, ok = limits[1].(*DefaultValue); !ok {
				panic(NewIllegalArgumentType2(`Integer[]`, 1, `Integer`, limits[1]))
			}
			max = MaxInt64
		}
	default:
		panic(NewIllegalArgumentCount(`Integer[]`, `0 - 2`, len(limits)))
	}
	return NewIntegerType(min, max)
}
func (t *IntegerType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*IntegerType); ok {
		return t.min == ot.min && t.max == ot.max
	}
	return false
}

func (t *IntegerType) Generic() PType {
	return integerType_DEFAULT
}

func (t *IntegerType) IsAssignable(o PType, g Guard) bool {
	if it, ok := o.(*IntegerType); ok {
		return t.min <= it.min && t.max >= it.max
	}
	return false
}

func (t *IntegerType) IsInstance(o PValue, g Guard) bool {
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
	return t.min == MinInt64 && t.max == MaxInt64
}

func (t *IntegerType) Min() int64 {
	return t.min
}

func (t *IntegerType) Max() int64 {
	return t.max
}

func (t *IntegerType) Name() string {
	return `Integer`
}

func (t *IntegerType) String() string {
	return ToString2(t, NONE)
}

func (t *IntegerType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *IntegerType) Parameters() []PValue {
	params := make([]PValue, 0)
	if t.min == MinInt64 {
		if t.max == MaxInt64 {
			return params
		}
		params = append(params, WrapDefault())
	} else {
		params = append(params, WrapInteger(t.min))
	}
	if t.max != MaxInt64 {
		params = append(params, WrapInteger(t.max))
	}
	return params
}

func (t *IntegerType) SizeParameters() []PValue {
	params := make([]PValue, 2)
	params[0] = WrapInteger(t.min)
	if t.max == MaxInt64 {
		params[1] = WrapDefault()
	} else {
		params[1] = WrapInteger(t.max)
	}
	return params
}

func (t *IntegerType) Type() PType {
	return &TypeType{t}
}

func WrapInteger(val int64) *IntegerValue {
	return (*IntegerValue)(NewIntegerType(val, val))
}

func (iv *IntegerValue) Abs() NumericValue {
	if iv.Int() < 0 {
		return WrapInteger(-iv.Int())
	}
	return iv
}

func (iv *IntegerValue) Equals(o interface{}, g Guard) bool {
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

func (iv *IntegerValue) String() string {
	return Sprintf(`%d`, iv.Int())
}

func (iv *IntegerValue) ToKey() HashKey {
	return HashKey(Sprintf("\x01i%d", iv.Int()))
}

func (iv *IntegerValue) ToString(b Writer, s FormatContext, g RDetect) {
	f := GetFormat(s.FormatMap(), iv.Type())
	switch f.FormatChar() {
	case 'x', 'X', 'o', 'd':
		Fprintf(b, f.OrigFormat(), iv.Int())
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

		WriteString(b, pfx)
		if zeroPad > 0 {
			padChar := []byte{'0'}
			if f.FormatChar() == 'p' {
				padChar = []byte{' '}
			}
			for ; zeroPad > 0; zeroPad-- {
				b.Write(padChar)
			}
		}
		WriteString(b, intString)
	case 'e', 'E', 'f', 'g', 'G', 'a', 'A':
		WrapFloat(iv.Float()).ToString(b, NewFormatContext(DefaultFloatType(), f, s.Indentation()), g)
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

func (iv *IntegerValue) Type() PType {
	return (*IntegerType)(iv)
}
