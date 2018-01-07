package types

import (
	"fmt"
	. "io"
	. "math"
	. "time"

	"bytes"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type (
	TimestampType struct {
		min Time
		max Time
	}

	// TimestampValue represents TimestampType as a value
	TimestampValue TimestampType
)

// Unit(sec, nsec) adds an offset of 62135596800 seconds to sec that
// represents the number of seconds from 1970-01-01:00:00:00 UTC. This offset
// must be retracted from the MaxInt64 value in order for it to end up
// as that value internally.
const MAX_UNIX_SECS = MaxInt64 - 62135596800

var MIN_TIME = Time{}
var MAX_TIME = Unix(MAX_UNIX_SECS, 999999999)
var timestampType_DEFAULT = &TimestampType{MIN_TIME, MAX_TIME}

func DefaultTimestampType() *TimestampType {
	return timestampType_DEFAULT
}

func NewTimestampType(min Time, max Time) *TimestampType {
	return &TimestampType{min, max}
}

func TimeFromHash(value *HashValue) (Time, bool) {
	// TODO
	return Time{}, false
}

func TimeFromString(value string) (Time, bool) {
	// TODO
	return Time{}, false
}

func NewTimestampType2(args ...PValue) *TimestampType {
	argc := len(args)
	if argc > 2 {
		panic(NewIllegalArgumentCount(`Timestamp[]`, `0 or 2`, argc))
	}
	if argc == 0 {
		return timestampType_DEFAULT
	}
	convertArg := func(args []PValue, argNo int) Time {
		arg := args[argNo]
		var (
			t  Time
			ok bool
		)
		switch arg.(type) {
		case *TimestampValue:
			t, ok = arg.(*TimestampValue).Time(), true
		case *HashValue:
			t, ok = TimeFromHash(arg.(*HashValue))
		case *StringValue:
			t, ok = TimeFromString(arg.(*StringValue).value)
		case *IntegerValue:
			t, ok = Unix(arg.(*IntegerValue).Int(), 0), true
		case *FloatValue:
			s, f := Modf(arg.(*FloatValue).Float())
			t, ok = Unix(int64(s), int64(f*1000000000.0)), true
		case *DefaultValue:
			if argNo == 0 {
				t, ok = Time{}, true
			} else {
				t, ok = Unix(MAX_UNIX_SECS, 999999999), true
			}
		default:
			t, ok = Time{}, false
		}
		if ok {
			return t
		}
		panic(NewIllegalArgumentType2(`Timestamp[]`, 0, `Variant[Hash,String,Integer,Float,Default]`, args[0]))
	}

	min := convertArg(args, 0)
	if argc == 2 {
		return &TimestampType{min, convertArg(args, 1)}
	} else {
		return &TimestampType{min, MAX_TIME}
	}
}

func (t *TimestampType) Default() PType {
	return timestampType_DEFAULT
}

func (t *TimestampType) Equals(other interface{}, guard Guard) bool {
	if ot, ok := other.(*TimestampType); ok {
		return t.min.Equal(ot.min) && t.max.Equal(ot.max)
	}
	return false
}

func (t *TimestampType) IsInstance(o PValue, g Guard) bool {
	return t.IsAssignable(o.Type(), g)
}

func (t *TimestampType) IsAssignable(o PType, g Guard) bool {
	if ot, ok := o.(*TimestampType); ok {
		return (t.min.Before(ot.min) || t.min.Equal(ot.min)) && (t.max.After(ot.max) || t.max.Equal(ot.max))
	}
	return false
}

func (t *TimestampType) Parameters() []PValue {
	if t.max.Equal(MAX_TIME) {
		if t.min.Equal(MIN_TIME) {
			return EMPTY_VALUES
		}
		return []PValue{WrapString(t.min.String())}
	}
	if t.min.Equal(MIN_TIME) {
		return []PValue{WrapDefault(), WrapString(t.max.String())}
	}
	return []PValue{WrapString(t.min.String()), WrapString(t.max.String())}
}

func (t *TimestampType) String() string {
	return ToString2(t, NONE)
}

func (t *TimestampType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TimestampType) Type() PType {
	return &TypeType{t}
}

func (t *TimestampType) Name() string {
	return `Timestamp`
}

func WrapTimestamp(time Time) *TimestampValue {
	return (*TimestampValue)(NewTimestampType(time, time))
}

func (tv *TimestampValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*TimestampValue); ok {
		return tv.Int() == ov.Int()
	}
	return false
}

func (tv *TimestampValue) Float() float64 {
	y := tv.min.Year()
	// Timestamps that represent a date before the year 1678 or after 2262 can
	// be represented as nanoseconds in an int64.
	if 1678 < y && y < 2262 {
		return float64(float64(tv.min.UnixNano()) / 1000000000.0)
	}
	// Fall back to microsecond precision
	us := tv.min.Unix()*1000000 + int64(tv.min.Nanosecond())/1000
	return float64(us) / 1000000.0
}

func (tv *TimestampValue) Time() Time {
	return tv.min
}

func (tv *TimestampValue) Int() int64 {
	return tv.min.Unix()
}

func (tv *TimestampValue) String() string {
	return fmt.Sprintf(`%d`, tv.Int())
}

func (tv *TimestampValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HK_TIMESTAMP)
	n := tv.min.Unix()
	b.WriteByte(byte(n >> 56))
	b.WriteByte(byte(n >> 48))
	b.WriteByte(byte(n >> 40))
	b.WriteByte(byte(n >> 32))
	b.WriteByte(byte(n >> 24))
	b.WriteByte(byte(n >> 16))
	b.WriteByte(byte(n >> 8))
	b.WriteByte(byte(n))
	n = int64(tv.min.Nanosecond())
	b.WriteByte(byte(n >> 56))
	b.WriteByte(byte(n >> 48))
	b.WriteByte(byte(n >> 40))
	b.WriteByte(byte(n >> 32))
	b.WriteByte(byte(n >> 24))
	b.WriteByte(byte(n >> 16))
	b.WriteByte(byte(n >> 8))
	b.WriteByte(byte(n))
}

func (tv *TimestampValue) ToString(b Writer, s FormatContext, g RDetect) {
	fmt.Fprintf(b, `%d`, tv.Int())
}

func (tv *TimestampValue) Type() PType {
	return (*TimestampType)(tv)
}
