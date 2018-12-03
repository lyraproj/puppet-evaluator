package types

import (
	"bytes"
	"io"
	"math"
	"time"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
	"sync"
)

type (
	TimestampType struct {
		min time.Time
		max time.Time
	}

	// TimestampValue represents TimestampType as a value
	TimestampValue TimestampType
)

// MAX_UNIX_SECS is an offset of 62135596800 seconds to sec that
// represents the number of seconds from 1970-01-01:00:00:00 UTC. This offset
// must be retracted from the MaxInt64 value in order for it to end up
// as that value internally.
const MAX_UNIX_SECS = math.MaxInt64 - 62135596800

var MIN_TIME = time.Time{}
var MAX_TIME = time.Unix(MAX_UNIX_SECS, 999999999)
var timestampType_DEFAULT = &TimestampType{MIN_TIME, MAX_TIME}

var Timestamp_Type eval.ObjectType

var DEFAULT_TIMESTAMP_FORMATS_WO_TZ []*TimestampFormat
var DEFAULT_TIMESTAMP_FORMATS []*TimestampFormat
var DefaultTimestampFormatParser *TimestampFormatParser

func init() {
	Timestamp_Type = newObjectType(`Pcore::TimestampType`,
		`Pcore::ScalarType {
	attributes => {
		from => { type => Optional[Timestamp], value => undef },
		to => { type => Optional[Timestamp], value => undef }
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewTimestampType2(args...)
		})

	tp := NewTimestampFormatParser()
	DefaultTimestampFormatParser = tp

	DEFAULT_TIMESTAMP_FORMATS_WO_TZ = []*TimestampFormat{
		tp.ParseFormat(`%FT%T.%N`),
		tp.ParseFormat(`%FT%T`),
		tp.ParseFormat(`%F %T.%N`),
		tp.ParseFormat(`%F %T`),
		tp.ParseFormat(`%F`),
	}

	DEFAULT_TIMESTAMP_FORMATS = []*TimestampFormat{
		tp.ParseFormat(`%FT%T.%N %Z`),
		tp.ParseFormat(`%FT%T %Z`),
		tp.ParseFormat(`%F %T.%N %Z`),
		tp.ParseFormat(`%F %T %Z`),
		tp.ParseFormat(`%F %Z`),
	}
	DEFAULT_TIMESTAMP_FORMATS = append(DEFAULT_TIMESTAMP_FORMATS, DEFAULT_TIMESTAMP_FORMATS_WO_TZ...)

	newGoConstructor2(`Timestamp`,

		func(t eval.LocalTypes) {
			t.Type(`Formats`, `Variant[String[2],Array[String[2], 1]]`)
		},

		func(d eval.Dispatch) {
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return WrapTimestamp(time.Now())
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Variant[Integer,Float]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				arg := args[0]
				if i, ok := arg.(*IntegerValue); ok {
					return WrapTimestamp(time.Unix(i.Int(), 0))
				}
				s, f := math.Modf(arg.(*FloatValue).Float())
				return WrapTimestamp(time.Unix(int64(s), int64(f*1000000000.0)))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`String[1]`)
			d.OptionalParam(`Formats`)
			d.OptionalParam(`String[1]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				formats := DEFAULT_TIMESTAMP_FORMATS
				tz := ``
				if len(args) > 1 {
					formats = toTimestampFormats(args[1])
					if len(args) > 2 {
						tz = args[2].String()
					}
				}
				return ParseTimestamp(args[0].String(), formats, tz)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Struct[string => String[1],Optional[format] => Formats,Optional[timezone] => String[1]]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				hash := args[0].(*HashValue)
				str := hash.Get5(`string`, _EMPTY_STRING).String()
				formats := toTimestampFormats(hash.Get5(`format`, eval.UNDEF))
				tz := hash.Get5(`timezone`, _EMPTY_STRING).String()
				return ParseTimestamp(str, formats, tz)
			})
		})
}

func DefaultTimestampType() *TimestampType {
	return timestampType_DEFAULT
}

func NewTimestampType(min time.Time, max time.Time) *TimestampType {
	return &TimestampType{min, max}
}

func TimeFromHash(value *HashValue) (time.Time, bool) {
	// TODO
	return time.Time{}, false
}

func TimeFromString(value string) time.Time {
	return ParseTimestamp(value, DEFAULT_TIMESTAMP_FORMATS, ``).Time()
}

func NewTimestampType2(args ...eval.Value) *TimestampType {
	argc := len(args)
	if argc > 2 {
		panic(errors.NewIllegalArgumentCount(`Timestamp[]`, `0 or 2`, argc))
	}
	if argc == 0 {
		return timestampType_DEFAULT
	}
	convertArg := func(args []eval.Value, argNo int) time.Time {
		arg := args[argNo]
		var (
			t  time.Time
			ok bool
		)
		switch arg.(type) {
		case *TimestampValue:
			t, ok = arg.(*TimestampValue).Time(), true
		case *HashValue:
			t, ok = TimeFromHash(arg.(*HashValue))
		case *StringValue:
			t, ok = TimeFromString(arg.(*StringValue).value), true
		case *IntegerValue:
			t, ok = time.Unix(arg.(*IntegerValue).Int(), 0), true
		case *FloatValue:
			s, f := math.Modf(arg.(*FloatValue).Float())
			t, ok = time.Unix(int64(s), int64(f*1000000000.0)), true
		case *DefaultValue:
			if argNo == 0 {
				t, ok = time.Time{}, true
			} else {
				t, ok = time.Unix(MAX_UNIX_SECS, 999999999), true
			}
		default:
			t, ok = time.Time{}, false
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

func (t *TimestampType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *TimestampType) Default() eval.Type {
	return timestampType_DEFAULT
}

func (t *TimestampType) Equals(other interface{}, guard eval.Guard) bool {
	if ot, ok := other.(*TimestampType); ok {
		return t.min.Equal(ot.min) && t.max.Equal(ot.max)
	}
	return false
}

func (t *TimestampType) Get(key string) (eval.Value, bool) {
	switch key {
	case `from`:
		v := eval.UNDEF
		if t.min != MIN_TIME {
			v = WrapTimestamp(t.min)
		}
		return v, true
	case `to`:
		v := eval.UNDEF
		if t.max != MAX_TIME {
			v = WrapTimestamp(t.max)
		}
		return v, true
	default:
		return nil, false
	}
}

func (t *TimestampType) IsInstance(o eval.Value, g eval.Guard) bool {
	return t.IsAssignable(o.PType(), g)
}

func (t *TimestampType) IsAssignable(o eval.Type, g eval.Guard) bool {
	if ot, ok := o.(*TimestampType); ok {
		return (t.min.Before(ot.min) || t.min.Equal(ot.min)) && (t.max.After(ot.max) || t.max.Equal(ot.max))
	}
	return false
}

func (t *TimestampType) MetaType() eval.ObjectType {
	return Timestamp_Type
}

func (t *TimestampType) Parameters() []eval.Value {
	if t.max.Equal(MAX_TIME) {
		if t.min.Equal(MIN_TIME) {
			return eval.EMPTY_VALUES
		}
		return []eval.Value{WrapString(t.min.String())}
	}
	if t.min.Equal(MIN_TIME) {
		return []eval.Value{WrapDefault(), WrapString(t.max.String())}
	}
	return []eval.Value{WrapString(t.min.String()), WrapString(t.max.String())}
}

func (t *TimestampType) ReflectType(c eval.Context) (reflect.Type, bool) {
	return reflect.TypeOf(time.Time{}), true
}

func (t *TimestampType)  CanSerializeAsString() bool {
  return true
}

func (t *TimestampType)  SerializationString() string {
	return t.String()
}


func (t *TimestampType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *TimestampType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TimestampType) PType() eval.Type {
	return &TypeType{t}
}

func (t *TimestampType) Name() string {
	return `Timestamp`
}

func WrapTimestamp(time time.Time) *TimestampValue {
	return (*TimestampValue)(NewTimestampType(time, time))
}

func ParseTimestamp(str string, formats []*TimestampFormat, tz string) *TimestampValue {
	if t, ok := parseTime(str, formats, tz); ok {
		return WrapTimestamp(t)
	}
	fs := bytes.NewBufferString(``)
	for i, f := range formats {
		if i > 0 {
			fs.WriteByte(',')
		}
		fs.WriteString(f.format)
	}
	panic(eval.Error(eval.EVAL_TIMESTAMP_CANNOT_BE_PARSED, issue.H{`str`: str, `formats`: fs.String()}))
}

func parseTime(str string, formats []*TimestampFormat, tz string) (time.Time, bool) {
	usedTz := tz
	if usedTz == `` {
		usedTz = `UTC`
	}
	loc := loadLocation(usedTz)

	for _, f := range formats {
		ts, err := time.ParseInLocation(f.layout, str, loc)
		if err == nil {
			if usedTz != ts.Location().String() {
				if tz != `` {
					panic(eval.Error(eval.EVAL_TIMESTAMP_TZ_AMBIGUITY, issue.H{`parsed`: ts.Location().String(), `given`: tz}))
				}
				// Golang does real weird things when the string contains a timezone that isn't equal
				// to the given timezone. Instead of loading the given zone, it creates a new location
				// with a similarly named zone but with offset 0. It doesn't matter if Parse or ParseInLocation
				// is used. Both has the same weird behavior. For this reason, a new loadLocation is performed
				// here followed by a reparse.
				loc, _ = time.LoadLocation(ts.Location().String())
				ts, err = time.ParseInLocation(f.layout, str, loc)
			}
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

func loadLocation(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		panic(eval.Error(eval.EVAL_INVALID_TIMEZONE, issue.H{`zone`: tz, `detail`: err.Error()}))
	}
	return loc
}

func (tv *TimestampValue) Equals(o interface{}, g eval.Guard) bool {
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

func (tv *TimestampValue) Format(format string) string {
	return DefaultTimestampFormatParser.ParseFormat(format).Format(tv)
}

func (tv *TimestampValue) Format2(format, tz string) string {
	return DefaultTimestampFormatParser.ParseFormat(format).Format2(tv, tz)
}

func (tv *TimestampValue) Reflect(c eval.Context) reflect.Value {
	return reflect.ValueOf(tv.Time())
}

func (tv *TimestampValue) ReflectTo(c eval.Context, dest reflect.Value) {
	rv := tv.Reflect(c)
	if !rv.Type().AssignableTo(dest.Type()) {
		panic(eval.Error(eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: rv.Type().String(), `actual`: dest.Type().String()}))
	}
	dest.Set(rv)
}

func (tv *TimestampValue) Time() time.Time {
	return tv.min
}

func (tv *TimestampValue) Int() int64 {
	return tv.min.Unix()
}

func (tv *TimestampValue)  CanSerializeAsString() bool {
  return true
}

func (tv *TimestampValue)  SerializationString() string {
	return tv.String()
}


func (tv *TimestampValue) String() string {
	return eval.ToString2(tv, NONE)
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

func (tv *TimestampValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	io.WriteString(b, tv.min.Format(DEFAULT_TIMESTAMP_FORMATS[0].layout))
}

func (tv *TimestampValue) PType() eval.Type {
	return (*TimestampType)(tv)
}

const (
	// Strings recognized as golang "layout" elements

	loLongMonth             = `January`
	loMonth                 = `Jan`
	loLongWeekDay           = `Monday`
	loWeekDay               = `Mon`
	loTZ                    = `MST`
	loZeroMonth             = `01`
	loZeroDay               = `02`
	loZeroHour12            = `03`
	loZeroMinute            = `04`
	loZeroSecond            = `05`
	loYear                  = `06`
	loHour                  = `15`
	loNumMonth              = `1`
	loLongYear              = `2006`
	loDay                   = `2`
	loUnderDay              = `_2`
	loHour12                = `3`
	loMinute                = `4`
	loSecond                = `5`
	loPM                    = `PM`
	lopm                    = `pm`
	loNumSecondsTz          = `-070000`
	loNumColonSecondsTZ     = `-07:00:00`
	loNumTZ                 = `-0700`
	loNumColonTZ            = `-07:00`
	loNumShortTZ            = `-07`
	loISO8601SecondsTZ      = `Z070000`
	loISO8601ColonSecondsTZ = `Z07:00:00`
	loISO8601TZ             = `Z0700`
	loISO8601ColonTZ        = `Z07:00`
	loISO8601ShortTz        = `Z07`
	loFracSecond0           = `.000`
	loFracSecond9           = `.999`
)

type (
	TimestampFormat struct {
		format string
		layout string
	}

	TimestampFormatParser struct {
		lock    sync.Mutex
		formats map[string]*TimestampFormat
	}
)

func NewTimestampFormatParser() *TimestampFormatParser {
	return &TimestampFormatParser{formats: make(map[string]*TimestampFormat, 17)}
}

func (p *TimestampFormatParser) ParseFormat(format string) *TimestampFormat {
	p.lock.Lock()
	defer p.lock.Unlock()

	if fmt, ok := p.formats[format]; ok {
		return fmt
	}
	bld := bytes.NewBufferString(``)
	strftimeToLayout(bld, format)
	fmt := &TimestampFormat{format, bld.String()}
	p.formats[format] = fmt
	return fmt
}

func (f *TimestampFormat) Format(t *TimestampValue) string {
	return t.min.Format(f.layout)
}

func (f *TimestampFormat) Format2(t *TimestampValue, tz string) string {
	return t.min.In(loadLocation(tz)).Format(f.layout)
}

func strftimeToLayout(bld *bytes.Buffer, str string) {
	state := STATE_LITERAL
	colons := 0
	padchar := '0'
	width := -1
	fstart := 0
	upper := false

	for pos, c := range str {
		if state == STATE_LITERAL {
			if c == '%' {
				state = STATE_PAD
				fstart = pos
				padchar = '0'
				width = -1
				upper = false
				colons = 0
			} else {
				bld.WriteRune(c)
			}
			continue
		}

		switch c {
		case '-':
			if state != STATE_PAD {
				panic(badFormatSpecifier(str, fstart, pos))
			}
			padchar = 0
			state = STATE_WIDTH
		case '^':
			if state != STATE_PAD {
				panic(badFormatSpecifier(str, fstart, pos))
			}
			upper = true
		case '_':
			if state != STATE_PAD {
				panic(badFormatSpecifier(str, fstart, pos))
			}
			padchar = ' '
			state = STATE_WIDTH
		case 'Y':
			bld.WriteString(loLongYear)
			state = STATE_LITERAL
		case 'y':
			bld.WriteString(loYear)
			state = STATE_LITERAL
		case 'C':
			panic(notSupportedByGoTimeLayout(str, fstart, pos, `century`))
		case 'm':
			switch padchar {
			case 0:
				bld.WriteString(loNumMonth)
			case '0':
				bld.WriteString(loZeroMonth)
			case ' ':
				panic(notSupportedByGoTimeLayout(str, fstart, pos, `space padded month`))
			}
			state = STATE_LITERAL
		case 'B':
			if upper {
				panic(notSupportedByGoTimeLayout(str, fstart, pos, `upper cased month`))
			}
			bld.WriteString(loLongMonth)
			state = STATE_LITERAL
		case 'b', 'h':
			if upper {
				panic(notSupportedByGoTimeLayout(str, fstart, pos, `upper cased short month`))
			}
			bld.WriteString(loMonth)
			state = STATE_LITERAL
		case 'd':
			switch padchar {
			case 0:
				bld.WriteString(loDay)
			case '0':
				bld.WriteString(loZeroDay)
			case ' ':
				bld.WriteString(loUnderDay)
			}
			state = STATE_LITERAL
		case 'e':
			bld.WriteString(loUnderDay)
			state = STATE_LITERAL
		case 'j':
			panic(notSupportedByGoTimeLayout(str, fstart, pos, `year of the day`))
		case 'H':
			switch padchar {
			case ' ':
				panic(notSupportedByGoTimeLayout(str, fstart, pos, `blank padded 24 hour`))
			case 0:
				panic(notSupportedByGoTimeLayout(str, fstart, pos, `short 24 hour`))
			default:
				bld.WriteString(loHour)
			}
			state = STATE_LITERAL
		case 'k':
			panic(notSupportedByGoTimeLayout(str, fstart, pos, `blank padded 24 hour`))
		case 'I':
			bld.WriteString(loZeroHour12)
			state = STATE_LITERAL
		case 'l':
			bld.WriteString(loHour12)
			state = STATE_LITERAL
		case 'P':
			bld.WriteString(lopm)
			state = STATE_LITERAL
		case 'p':
			bld.WriteString(loPM)
			state = STATE_LITERAL
		case 'M':
			switch padchar {
			case ' ':
				panic(notSupportedByGoTimeLayout(str, fstart, pos, `blank padded minute`))
			case 0:
				bld.WriteString(loMinute)
			default:
				bld.WriteString(loZeroMinute)
			}
			state = STATE_LITERAL
		case 'S':
			switch padchar {
			case ' ':
				panic(notSupportedByGoTimeLayout(str, fstart, pos, `blank padded second`))
			case 0:
				bld.WriteString(loSecond)
			default:
				bld.WriteString(loZeroSecond)
			}
			state = STATE_LITERAL
		case 'L':
			if fstart == 0 || str[fstart-1] != '.' {
				panic(notSupportedByGoTimeLayout(str, fstart, pos, `fraction not preceded by dot in format`))
			}
			if padchar == '0' {
				bld.WriteString(`000`)
			} else {
				bld.WriteString(`999`)
			}
			state = STATE_LITERAL
		case 'N':
			if fstart == 0 || str[fstart-1] != '.' {
				panic(notSupportedByGoTimeLayout(str, fstart, pos, `fraction not preceded by dot in format`))
			}
			digit := byte('9')
			if padchar == '0' {
				digit = '0'
			}
			w := width
			if width == -1 {
				w = 9
			}
			for i := 0; i < w; i++ {
				bld.WriteByte(digit)
			}
			state = STATE_LITERAL
		case 'z':
			switch colons {
			case 0:
				bld.WriteString(loNumTZ)
			case 1:
				bld.WriteString(loNumColonTZ)
			case 2:
				bld.WriteString(loNumColonSecondsTZ)
			default:
				// Not entirely correct since loosely defined num TZ not supported in Go
				bld.WriteString(loNumShortTZ)
			}
			state = STATE_LITERAL
		case 'Z':
			bld.WriteString(loTZ)
			state = STATE_LITERAL
		case 'A':
			bld.WriteString(loLongWeekDay)
			state = STATE_LITERAL
		case 'a':
			bld.WriteString(loWeekDay)
			state = STATE_LITERAL
		case 'u', 'w':
			panic(notSupportedByGoTimeLayout(str, fstart, pos, `numeric week day`))
		case 'G', 'g':
			panic(notSupportedByGoTimeLayout(str, fstart, pos, `week based year`))
		case 'V':
			panic(notSupportedByGoTimeLayout(str, fstart, pos, `week number of the based year`))
		case 's':
			panic(notSupportedByGoTimeLayout(str, fstart, pos, `seconds since epoch`))
		case 'Q':
			panic(notSupportedByGoTimeLayout(str, fstart, pos, `milliseconds since epoch`))
		case 't':
			bld.WriteString("\t")
			state = STATE_LITERAL
		case 'n':
			bld.WriteString("\n")
			state = STATE_LITERAL
		case '%':
			bld.WriteByte('%')
			state = STATE_LITERAL
		case 'c':
			strftimeToLayout(bld, `%a %b %-d %T %Y`)
			state = STATE_LITERAL
		case 'D', 'x':
			strftimeToLayout(bld, `%m/%d/%y`)
			state = STATE_LITERAL
		case 'F':
			strftimeToLayout(bld, `%Y-%m-%d`)
			state = STATE_LITERAL
		case 'r':
			strftimeToLayout(bld, `%I:%M:%S %p`)
			state = STATE_LITERAL
		case 'R':
			strftimeToLayout(bld, `%H:%M`)
			state = STATE_LITERAL
		case 'X', 'T':
			strftimeToLayout(bld, `%H:%M:%S`)
			state = STATE_LITERAL
		case '+':
			strftimeToLayout(bld, `%a %b %-d %H:%M:%S %Z %Y`)
			state = STATE_LITERAL
		default:
			if c < '0' || c > '9' {
				panic(badFormatSpecifier(str, fstart, pos))
			}
			if state == STATE_PAD && c == '0' {
				padchar = '0'
			} else {
				n := int(c) - 0x30
				if width == -1 {
					width = n
				} else {
					width = width*10 + n
				}
			}
			state = STATE_WIDTH
		}
	}

	if state != STATE_LITERAL {
		panic(badFormatSpecifier(str, fstart, len(str)))
	}
}

func notSupportedByGoTimeLayout(str string, start, pos int, description string) issue.Reported {
	return eval.Error(eval.EVAL_NOT_SUPPORTED_BY_GO_TIME_LAYOUT, issue.H{`format_specifier`: str[start : pos+1], `description`: description})
}

func toTimestampFormats(fmt eval.Value) []*TimestampFormat {
	formats := DEFAULT_TIMESTAMP_FORMATS
	switch fmt.(type) {
	case *ArrayValue:
		fa := fmt.(*ArrayValue)
		formats = make([]*TimestampFormat, fa.Len())
		fa.EachWithIndex(func(f eval.Value, i int) {
			formats[i] = DefaultTimestampFormatParser.ParseFormat(f.String())
		})
	case *StringValue:
		formats = []*TimestampFormat{DefaultTimestampFormatParser.ParseFormat(fmt.String())}
	}
	return formats
}
