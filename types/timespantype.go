package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
)

type (
	TimespanType struct {
		min time.Duration
		max time.Duration
	}

	// TimespanValue represents TimespanType as a value
	TimespanValue TimespanType
)

var timespanType_DEFAULT = &TimespanType{time.Duration(math.MinInt64), time.Duration(math.MaxInt64)}

func DurationFromHash(value *HashValue) (time.Duration, bool) {
	// TODO
	return time.Duration(0), false
}

func DurationFromString(value string) (time.Duration, bool) {
	// TODO
	return time.Duration(0), false
}

func DefaultTimespanType() *TimespanType {
	return timespanType_DEFAULT
}

func NewTimespanType(min time.Duration, max time.Duration) *TimespanType {
	return &TimespanType{min, max}
}

func NewTimespanType2(args ...eval.PValue) *TimespanType {
	argc := len(args)
	if argc > 2 {
		panic(errors.NewIllegalArgumentCount(`Timespan[]`, `0 or 2`, argc))
	}
	if argc == 0 {
		return timespanType_DEFAULT
	}
	convertArg := func(args []eval.PValue, argNo int) time.Duration {
		arg := args[argNo]
		var (
			t  time.Duration
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
			t, ok = time.Duration(arg.(*IntegerValue).Int()*1000000000), true
		case *FloatValue:
			t, ok = time.Duration(arg.(*FloatValue).Float()*1000000000.0), true
		case *DefaultValue:
			if argNo == 0 {
				t, ok = time.Duration(math.MinInt64), true
			} else {
				t, ok = time.Duration(math.MaxInt64), true
			}
		default:
			t, ok = time.Duration(0), false
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
		return &TimespanType{min, time.Duration(math.MaxInt64)}
	}
}

func (t *TimespanType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *TimespanType) Default() eval.PType {
	return timespanType_DEFAULT
}

func (t *TimespanType) Equals(other interface{}, guard eval.Guard) bool {
	if ot, ok := other.(*TimespanType); ok {
		return t.min == ot.min && t.max == ot.max
	}
	return false
}

func (t *TimespanType) Parameters() []eval.PValue {
	if t.max == math.MaxInt64 {
		if t.min == math.MinInt64 {
			return eval.EMPTY_VALUES
		}
		return []eval.PValue{WrapString(t.min.String())}
	}
	if t.min == math.MinInt64 {
		return []eval.PValue{WrapDefault(), WrapString(t.max.String())}
	}
	return []eval.PValue{WrapString(t.min.String()), WrapString(t.max.String())}
}

func (t *TimespanType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *TimespanType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TimespanType) Type() eval.PType {
	return &TypeType{t}
}

func (t *TimespanType) IsInstance(o eval.PValue, g eval.Guard) bool {
	return t.IsAssignable(o.Type(), g)
}

func (t *TimespanType) IsAssignable(o eval.PType, g eval.Guard) bool {
	if ot, ok := o.(*TimespanType); ok {
		return t.min <= ot.min && t.max >= ot.max
	}
	return false
}

func (t *TimespanType) Name() string {
	return `Timespan`
}

func WrapTimespan(val time.Duration) *TimespanValue {
	return (*TimespanValue)(NewTimespanType(val, val))
}

func (tv *TimespanValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*TimespanValue); ok {
		return tv.Int() == ov.Int()
	}
	return false
}

func (tv *TimespanValue) Float() float64 {
	return float64(tv.min) / 1000000000.0
}

func (tv *TimespanValue) Duration() time.Duration {
	return tv.min
}

func (tv *TimespanValue) Int() int64 {
	return int64(tv.min) / 1000000000
}

func (tv *TimespanValue) SerializationString() string {
	return tv.String()
}

func (tv *TimespanValue) String() string {
	return fmt.Sprintf(`%d`, tv.Int())
}

func (tv *TimespanValue) ToKey(b *bytes.Buffer) {
	n := tv.Int()
	b.WriteByte(1)
	b.WriteByte(HK_TIMESPAN)
	b.WriteByte(byte(n >> 56))
	b.WriteByte(byte(n >> 48))
	b.WriteByte(byte(n >> 40))
	b.WriteByte(byte(n >> 32))
	b.WriteByte(byte(n >> 24))
	b.WriteByte(byte(n >> 16))
	b.WriteByte(byte(n >> 8))
	b.WriteByte(byte(n))
}

func (tv *TimespanValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	fmt.Fprintf(b, `%d`, tv.Int())
}

func (tv *TimespanValue) Type() eval.PType {
	return (*TimespanType)(tv)
}
