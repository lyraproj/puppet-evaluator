package values

import (
	"fmt"
	. "io"
	. "math"
	. "time"

	. "github.com/puppetlabs/go-evaluator/eval/errors"
	. "github.com/puppetlabs/go-evaluator/eval/utils"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
)

type (
	TimespanType struct {
		min Duration
		max Duration
	}

	// TimespanValue represents TimespanType as a value
	TimespanValue TimespanType
)

var timespanType_DEFAULT = &TimespanType{Duration(MinInt64), Duration(MaxInt64)}

func DurationFromHash(value *HashValue) (Duration, bool) {
	// TODO
	return Duration(0), false
}

func DurationFromString(value string) (Duration, bool) {
	// TODO
	return Duration(0), false
}

func DefaultTimespanType() *TimespanType {
	return timespanType_DEFAULT
}

func NewTimespanType(min Duration, max Duration) *TimespanType {
	return &TimespanType{min, max}
}

func NewTimespanType2(args ...PValue) *TimespanType {
	argc := len(args)
	if argc > 2 {
		panic(NewIllegalArgumentCount(`Timespan[]`, `0 or 2`, argc))
	}
	if argc == 0 {
		return timespanType_DEFAULT
	}
	convertArg := func(args []PValue, argNo int) Duration {
		arg := args[argNo]
		var (
			t  Duration
			ok bool
		)
		switch arg.(type) {
		case *TimestampValue:
			t, ok = arg.(*TimespanValue).Duration(), true
		case *HashValue:
			t, ok = DurationFromHash(arg.(*HashValue))
		case *StringValue:
			t, ok = DurationFromString(arg.(*StringValue).value)
		case *IntegerValue:
			t, ok = Duration(arg.(*IntegerValue).Int()*1000000000), true
		case *FloatValue:
			t, ok = Duration(arg.(*FloatValue).Float()*1000000000.0), true
		case *DefaultValue:
			if argNo == 0 {
				t, ok = Duration(MinInt64), true
			} else {
				t, ok = Duration(MaxInt64), true
			}
		default:
			t, ok = Duration(0), false
		}
		if ok {
			return t
		}
		panic(NewIllegalArgumentType2(`Timestamp[]`, 0, `Variant[Hash,String,Integer,Float,Default]`, args[0]))
	}

	min := convertArg(args, 0)
	if argc == 2 {
		return &TimespanType{min, convertArg(args, 1)}
	} else {
		return &TimespanType{min, Duration(MaxInt64)}
	}
}

func (t *TimespanType) Equals(other interface{}, guard Guard) bool {
	if ot, ok := other.(*TimespanType); ok {
		return t.min == ot.min && t.max == ot.max
	}
	return false
}

func (t *TimespanType) String() string {
	return ToString2(t, NONE)
}

func (t *TimespanType) ToString(bld Writer, format FormatContext, g RDetect) {
	// TODO: Formatting
	WriteString(bld, `Timespan`)
	if t.max == MaxInt64 {
		if t.min == MinInt64 {
			return
		}
		WriteByte(bld, '[')
		PuppetQuote(bld, t.min.String())
	} else {
		WriteByte(bld, '[')
		if t.min == MinInt64 {
			WriteString(bld, `default`)
		} else {
			PuppetQuote(bld, t.String())
		}
		WriteString(bld, `, `)
		PuppetQuote(bld, t.max.String())
	}
	WriteByte(bld, ']')
}

func (t *TimespanType) Type() PType {
	return &TypeType{t}
}

func (t *TimespanType) IsInstance(o PValue, g Guard) bool {
	return t.IsAssignable(o.Type(), g)
}

func (t *TimespanType) IsAssignable(o PType, g Guard) bool {
	if ot, ok := o.(*TimespanType); ok {
		return t.min <= ot.min && t.max >= ot.max
	}
	return false
}

func (t *TimespanType) Name() string {
	return `Timespan`
}

func WrapTimespan(val Duration) *TimespanValue {
	return (*TimespanValue)(NewTimespanType(val, val))
}

func (tv *TimespanValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*TimespanValue); ok {
		return tv.Int() == ov.Int()
	}
	return false
}

func (tv *TimespanValue) Float() float64 {
	return float64(tv.min) / 1000000000.0
}

func (tv *TimespanValue) Duration() Duration {
	return tv.min
}

func (tv *TimespanValue) Int() int64 {
	return int64(tv.min) / 1000000000
}

func (tv *TimespanValue) String() string {
	return fmt.Sprintf(`%d`, tv.Int())
}

func (tv *TimespanValue) ToKey() HashKey {
	return HashKey(fmt.Sprintf("\x01d%d", tv.Int()))
}

func (tv *TimespanValue) ToString(b Writer, s FormatContext, g RDetect) {
	fmt.Fprintf(b, `%d`, tv.Int())
}

func (tv *TimespanValue) Type() PType {
	return (*TimespanType)(tv)
}
