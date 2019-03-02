package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"time"

	"reflect"
	"regexp"
	"strconv"
	"sync"
	"unicode"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/utils"
)

type (
	TimespanType struct {
		min time.Duration
		max time.Duration
	}

	// TimespanValue represents time.Duration as an eval.Value
	TimespanValue time.Duration
)

const (
	NsecsPerUsec = 1000
	NsecsPerMsec = NsecsPerUsec * 1000
	NsecsPerSec  = NsecsPerMsec * 1000
	NsecsPerMin  = NsecsPerSec * 60
	NsecsPerHour = NsecsPerMin * 60
	NsecsPerDay  = NsecsPerHour * 24

	KeyString       = `string`
	KeyFormat       = `format`
	KeyNegative     = `negative`
	KeyDays         = `days`
	KeyHours        = `hours`
	KeyMinutes      = `minutes`
	KeySeconds      = `seconds`
	KeyMilliseconds = `milliseconds`
	KeyMicroseconds = `microseconds`
	KeyNanoseconds  = `nanoseconds`
)

var TimespanMin = time.Duration(math.MinInt64)
var TimespanMax = time.Duration(math.MaxInt64)

var timespanTypeDefault = &TimespanType{TimespanMin, TimespanMax}

var TimespanMetaType eval.ObjectType
var DefaultTimespanFormatParser *TimespanFormatParser

func init() {
	tp := NewTimespanFormatParser()
	DefaultTimespanFormats = []*TimespanFormat{
		tp.ParseFormat(`%D-%H:%M:%S.%-N`),
		tp.ParseFormat(`%H:%M:%S.%-N`),
		tp.ParseFormat(`%M:%S.%-N`),
		tp.ParseFormat(`%S.%-N`),
		tp.ParseFormat(`%D-%H:%M:%S`),
		tp.ParseFormat(`%H:%M:%S`),
		tp.ParseFormat(`%D-%H:%M`),
		tp.ParseFormat(`%S`),
	}
	DefaultTimespanFormatParser = tp

	TimespanMetaType = newObjectType(`Pcore::TimespanType`,
		`Pcore::ScalarType{
	attributes => {
		from => { type => Optional[Timespan], value => undef },
		to => { type => Optional[Timespan], value => undef }
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return newTimespanType2(args...)
		})

	newGoConstructor2(`Timespan`,
		func(t eval.LocalTypes) {
			t.Type(`Formats`, `Variant[String[2],Array[String[2], 1]]`)
		},

		func(d eval.Dispatch) {
			d.Param(`Variant[Integer,Float]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				arg := args[0]
				if i, ok := arg.(integerValue); ok {
					return WrapTimespan(time.Duration(i * NsecsPerSec))
				}
				return WrapTimespan(time.Duration(arg.(floatValue) * NsecsPerSec))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`String[1]`)
			d.OptionalParam(`Formats`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				formats := DefaultTimespanFormats
				if len(args) > 1 {
					formats = toTimespanFormats(args[1])
				}

				return ParseTimespan(args[0].String(), formats)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Integer`)
			d.Param(`Integer`)
			d.Param(`Integer`)
			d.Param(`Integer`)
			d.OptionalParam(`Integer`)
			d.OptionalParam(`Integer`)
			d.OptionalParam(`Integer`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				days := args[0].(integerValue).Int()
				hours := args[1].(integerValue).Int()
				minutes := args[2].(integerValue).Int()
				seconds := args[3].(integerValue).Int()
				argc := len(args)
				var milliseconds, microseconds, nanoseconds int64
				if argc > 4 {
					milliseconds = args[4].(integerValue).Int()
					if argc > 5 {
						microseconds = args[5].(integerValue).Int()
						if argc > 6 {
							nanoseconds = args[6].(integerValue).Int()
						}
					}
				}
				return WrapTimespan(fromFields(false, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Struct[string => String[1], Optional[format] => Formats]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				hash := args[0].(*HashValue)
				str := hash.Get5(`string`, emptyString)
				formats := toTimespanFormats(hash.Get5(`format`, undef))
				return ParseTimespan(str.String(), formats)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Struct[Optional[negative] => Boolean,
        Optional[days] => Integer,
        Optional[hours] => Integer,
        Optional[minutes] => Integer,
        Optional[seconds] => Integer,
        Optional[milliseconds] => Integer,
        Optional[microseconds] => Integer,
        Optional[nanoseconds] => Integer]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return WrapTimespan(fromFieldsHash(args[0].(*HashValue)))
			})
		})
}

func DefaultTimespanType() *TimespanType {
	return timespanTypeDefault
}

func NewTimespanType(min time.Duration, max time.Duration) *TimespanType {
	return &TimespanType{min, max}
}

func newTimespanType2(args ...eval.Value) *TimespanType {
	argc := len(args)
	if argc > 2 {
		panic(errors.NewIllegalArgumentCount(`Timespan[]`, `0 or 2`, argc))
	}
	if argc == 0 {
		return timespanTypeDefault
	}
	convertArg := func(args []eval.Value, argNo int) time.Duration {
		arg := args[argNo]
		var (
			t  time.Duration
			ok bool
		)
		switch arg := arg.(type) {
		case *TimespanValue:
			t, ok = arg.Duration(), true
		case *HashValue:
			t, ok = fromHash(arg)
		case stringValue:
			t, ok = parseDuration(arg.String(), DefaultTimespanFormats)
		case integerValue:
			t, ok = time.Duration(arg*1000000000), true
		case floatValue:
			t, ok = time.Duration(arg*1000000000.0), true
		case *DefaultValue:
			if argNo == 0 {
				t, ok = TimespanMin, true
			} else {
				t, ok = TimespanMax, true
			}
		default:
			t, ok = time.Duration(0), false
		}
		if ok {
			return t
		}
		panic(NewIllegalArgumentType(`Timestamp[]`, 0, `Variant[Hash,String,Integer,Float,Default]`, args[0]))
	}

	min := convertArg(args, 0)
	if argc == 2 {
		return &TimespanType{min, convertArg(args, 1)}
	} else {
		return &TimespanType{min, TimespanMax}
	}
}

func (t *TimespanType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *TimespanType) Default() eval.Type {
	return timespanTypeDefault
}

func (t *TimespanType) Equals(other interface{}, guard eval.Guard) bool {
	if ot, ok := other.(*TimespanType); ok {
		return t.min == ot.min && t.max == ot.max
	}
	return false
}

func (t *TimespanType) Get(key string) (eval.Value, bool) {
	switch key {
	case `from`:
		v := eval.Undef
		if t.min != TimespanMin {
			v = WrapTimespan(t.min)
		}
		return v, true
	case `to`:
		v := eval.Undef
		if t.max != TimespanMax {
			v = WrapTimespan(t.max)
		}
		return v, true
	default:
		return nil, false
	}
}

func (t *TimespanType) MetaType() eval.ObjectType {
	return TimespanMetaType
}

func (t *TimespanType) Parameters() []eval.Value {
	if t.max == math.MaxInt64 {
		if t.min == math.MinInt64 {
			return eval.EmptyValues
		}
		return []eval.Value{stringValue(t.min.String())}
	}
	if t.min == math.MinInt64 {
		return []eval.Value{WrapDefault(), stringValue(t.max.String())}
	}
	return []eval.Value{stringValue(t.min.String()), stringValue(t.max.String())}
}

func (t *TimespanType) ReflectType(c eval.Context) (reflect.Type, bool) {
	return reflect.TypeOf(time.Duration(0)), true
}

func (t *TimespanType) CanSerializeAsString() bool {
	return true
}

func (t *TimespanType) SerializationString() string {
	return t.String()
}

func (t *TimespanType) String() string {
	return eval.ToString2(t, None)
}

func (t *TimespanType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TimespanType) PType() eval.Type {
	return &TypeType{t}
}

func (t *TimespanType) IsInstance(o eval.Value, g eval.Guard) bool {
	return t.IsAssignable(o.PType(), g)
}

func (t *TimespanType) IsAssignable(o eval.Type, g eval.Guard) bool {
	if ot, ok := o.(*TimespanType); ok {
		return t.min <= ot.min && t.max >= ot.max
	}
	return false
}

func (t *TimespanType) Name() string {
	return `Timespan`
}

func WrapTimespan(val time.Duration) TimespanValue {
	return TimespanValue(val)
}

func ParseTimespan(str string, formats []*TimespanFormat) TimespanValue {
	if d, ok := parseDuration(str, formats); ok {
		return WrapTimespan(d)
	}
	fs := bytes.NewBufferString(``)
	for i, f := range formats {
		if i > 0 {
			fs.WriteByte(',')
		}
		fs.WriteString(f.fmt)
	}
	panic(eval.Error(eval.CannotBeParsed, issue.H{`str`: str, `formats`: fs.String()}))
}

func fromFields(negative bool, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds int64) time.Duration {
	ns := (((((days*24+hours)*60+minutes)*60+seconds)*1000+milliseconds)*1000+microseconds)*1000 + nanoseconds
	if negative {
		ns = -ns
	}
	return time.Duration(ns)
}

func fromFieldsHash(hash *HashValue) time.Duration {
	intArg := func(key string) int64 {
		if v, ok := hash.Get4(key); ok {
			if i, ok := v.(integerValue); ok {
				return int64(i)
			}
		}
		return 0
	}
	boolArg := func(key string) bool {
		if v, ok := hash.Get4(key); ok {
			if b, ok := v.(booleanValue); ok {
				return b.Bool()
			}
		}
		return false
	}
	return fromFields(
		boolArg(KeyNegative),
		intArg(KeyDays),
		intArg(KeyHours),
		intArg(KeyMinutes),
		intArg(KeySeconds),
		intArg(KeyMilliseconds),
		intArg(KeyMicroseconds),
		intArg(KeyNanoseconds))
}

func fromStringHash(hash *HashValue) (time.Duration, bool) {
	str := hash.Get5(KeyString, emptyString)
	fmtStrings := hash.Get5(KeyFormat, nil)
	var formats []*TimespanFormat
	if fmtStrings == nil {
		formats = DefaultTimespanFormats
	} else {
		if fs, ok := fmtStrings.(stringValue); ok {
			formats = []*TimespanFormat{DefaultTimespanFormatParser.ParseFormat(string(fs))}
		} else {
			if fsa, ok := fmtStrings.(*ArrayValue); ok {
				formats = make([]*TimespanFormat, fsa.Len())
				fsa.EachWithIndex(func(fs eval.Value, i int) {
					formats[i] = DefaultTimespanFormatParser.ParseFormat(fs.String())
				})
			}
		}
	}
	return parseDuration(str.String(), formats)
}

func fromHash(hash *HashValue) (time.Duration, bool) {
	if hash.IncludesKey2(KeyString) {
		return fromStringHash(hash)
	}
	return fromFieldsHash(hash), true
}

func parseDuration(str string, formats []*TimespanFormat) (time.Duration, bool) {
	for _, f := range formats {
		if ts, ok := f.parse(str); ok {
			return ts, true
		}
	}
	return 0, false
}

func (tv TimespanValue) Abs() eval.NumericValue {
	if tv < 0 {
		return TimespanValue(-tv)
	}
	return tv
}

// Hours returns a positive integer denoting the number of days
func (tv TimespanValue) Days() int64 {
	return tv.totalDays()
}

func (tv TimespanValue) Duration() time.Duration {
	return time.Duration(tv)
}

// Hours returns a positive integer, 0 - 23 denoting hours of day
func (tv TimespanValue) Hours() int64 {
	return tv.totalHours() % 24
}

func (tv TimespanValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(TimespanValue); ok {
		return tv.Int() == ov.Int()
	}
	return false
}

// Float returns the number of seconds with fraction
func (tv TimespanValue) Float() float64 {
	return float64(tv.totalNanoseconds()) / float64(NsecsPerSec)
}

func (tv TimespanValue) Format(format string) string {
	return DefaultTimespanFormatParser.ParseFormat(format).format(tv)
}

// Int returns the total number of seconds
func (tv TimespanValue) Int() int64 {
	return tv.totalSeconds()
}

// Minutes returns a positive integer, 0 - 59 denoting minutes of hour
func (tv TimespanValue) Minutes() int64 {
	return tv.totalMinutes() % 60
}

func (tv TimespanValue) Reflect(c eval.Context) reflect.Value {
	return reflect.ValueOf(time.Duration(tv))
}

func (tv TimespanValue) ReflectTo(c eval.Context, dest reflect.Value) {
	rv := tv.Reflect(c)
	if !rv.Type().AssignableTo(dest.Type()) {
		panic(eval.Error(eval.AttemptToSetWrongKind, issue.H{`expected`: rv.Type().String(), `actual`: dest.Type().String()}))
	}
	dest.Set(rv)
}

// Seconds returns a positive integer, 0 - 59 denoting seconds of minute
func (tv TimespanValue) Seconds() int64 {
	return tv.totalSeconds() % 60
}

// Seconds returns a positive integer, 0 - 999 denoting milliseconds of second
func (tv TimespanValue) Milliseconds() int64 {
	return tv.totalMilliseconds() % 1000
}

func (tv TimespanValue) CanSerializeAsString() bool {
	return true
}

func (tv TimespanValue) SerializationString() string {
	return tv.String()
}

func (tv TimespanValue) String() string {
	return fmt.Sprintf(`%d`, tv.Int())
}

func (tv TimespanValue) ToKey(b *bytes.Buffer) {
	n := tv.Int()
	b.WriteByte(1)
	b.WriteByte(HkTimespan)
	b.WriteByte(byte(n >> 56))
	b.WriteByte(byte(n >> 48))
	b.WriteByte(byte(n >> 40))
	b.WriteByte(byte(n >> 32))
	b.WriteByte(byte(n >> 24))
	b.WriteByte(byte(n >> 16))
	b.WriteByte(byte(n >> 8))
	b.WriteByte(byte(n))
}

func (tv TimespanValue) totalDays() int64 {
	return time.Duration(tv).Nanoseconds() / NsecsPerDay
}

func (tv TimespanValue) totalHours() int64 {
	return time.Duration(tv).Nanoseconds() / NsecsPerHour
}

func (tv TimespanValue) totalMinutes() int64 {
	return time.Duration(tv).Nanoseconds() / NsecsPerMin
}

func (tv TimespanValue) totalSeconds() int64 {
	return time.Duration(tv).Nanoseconds() / NsecsPerSec
}

func (tv TimespanValue) totalMilliseconds() int64 {
	return time.Duration(tv).Nanoseconds() / NsecsPerMsec
}

func (tv TimespanValue) totalNanoseconds() int64 {
	return time.Duration(tv).Nanoseconds()
}

func (tv TimespanValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	DefaultTimespanFormats[0].format2(b, tv)
}

func (tv TimespanValue) PType() eval.Type {
	t := time.Duration(tv)
	return &TimespanType{t, t}
}

type (
	TimespanFormat struct {
		rx       *regexp.Regexp
		fmt      string
		segments []segment
	}

	TimespanFormatParser struct {
		lock    sync.Mutex
		formats map[string]*TimespanFormat
	}

	segment interface {
		appendRegexp(buffer *bytes.Buffer)

		appendTo(buffer io.Writer, ts TimespanValue)

		multiplier() int

		nanoseconds(group string, multiplier int) int64

		ordinal() int

		setUseTotal()
	}

	literalSegment struct {
		literal string
	}

	valueSegment struct {
		useTotal     bool
		padChar      rune
		width        int
		defaultWidth int
		format       string
	}

	daySegment struct {
		valueSegment
	}

	hourSegment struct {
		valueSegment
	}

	minuteSegment struct {
		valueSegment
	}

	secondSegment struct {
		valueSegment
	}

	fragmentSegment struct {
		valueSegment
	}

	millisecondSegment struct {
		fragmentSegment
	}

	nanosecondSegment struct {
		fragmentSegment
	}
)

const (
	nsecMax = 0
	msecMax = 1
	secMax  = 2
	minMax  = 3
	hourMax = 4
	dayMax  = 5

	// States used by the #internal_parser function

	stateLiteral = 0 // expects literal or '%'
	statePad     = 1 // expects pad, width, or format character
	stateWidth   = 2 // expects width, or format character
)

var trimTrailingZeroes = regexp.MustCompile(`\A([0-9]+?)0*\z`)
var digitsOnly = regexp.MustCompile(`\A[0-9]+\z`)
var DefaultTimespanFormats []*TimespanFormat

func NewTimespanFormatParser() *TimespanFormatParser {
	return &TimespanFormatParser{formats: make(map[string]*TimespanFormat, 17)}
}

func (p *TimespanFormatParser) ParseFormat(format string) *TimespanFormat {
	p.lock.Lock()
	defer p.lock.Unlock()

	if f, ok := p.formats[format]; ok {
		return f
	}
	f := p.parse(format)
	p.formats[format] = f
	return f
}

func (p *TimespanFormatParser) parse(str string) *TimespanFormat {
	bld := make([]segment, 0, 7)
	highest := -1
	state := stateLiteral
	padChar := '0'
	width := -1
	formatStart := 0

	for pos, c := range str {
		if state == stateLiteral {
			if c == '%' {
				state = statePad
				formatStart = pos
				padChar = '0'
				width = -1
			} else {
				bld = appendLiteral(bld, c)
			}
			continue
		}

		switch c {
		case '%':
			bld = appendLiteral(bld, c)
			state = stateLiteral
		case '-':
			if state != statePad {
				panic(badFormatSpecifier(str, formatStart, pos))
			}
			padChar = 0
			state = stateWidth
		case '_':
			if state != statePad {
				panic(badFormatSpecifier(str, formatStart, pos))
			}
			padChar = ' '
			state = stateWidth
		case 'D':
			highest = dayMax
			bld = append(bld, newDaySegment(padChar, width))
			state = stateLiteral
		case 'H':
			if highest < hourMax {
				highest = hourMax
			}
			bld = append(bld, newHourSegment(padChar, width))
			state = stateLiteral
		case 'M':
			if highest < minMax {
				highest = minMax
			}
			bld = append(bld, newMinuteSegment(padChar, width))
			state = stateLiteral
		case 'S':
			if highest < secMax {
				highest = secMax
			}
			bld = append(bld, newSecondSegment(padChar, width))
			state = stateLiteral
		case 'L':
			if highest < msecMax {
				highest = msecMax
			}
			bld = append(bld, newMillisecondSegment(padChar, width))
			state = stateLiteral
		case 'N':
			if highest < nsecMax {
				highest = nsecMax
			}
			bld = append(bld, newNanosecondSegment(padChar, width))
			state = stateLiteral
		default:
			if c < '0' || c > '9' {
				panic(badFormatSpecifier(str, formatStart, pos))
			}
			if state == statePad && c == '0' {
				padChar = '0'
			} else {
				n := int(c) - 0x30
				if width == -1 {
					width = n
				} else {
					width = width*10 + n
				}
			}
			state = stateWidth
		}
	}

	if state != stateLiteral {
		panic(badFormatSpecifier(str, formatStart, len(str)))
	}

	if highest != -1 {
		for _, s := range bld {
			if s.ordinal() == highest {
				s.setUseTotal()
			}
		}
	}
	return newTimespanFormat(str, bld)
}

func appendLiteral(bld []segment, c rune) []segment {
	s := string(c)
	lastIdx := len(bld) - 1
	if lastIdx >= 0 {
		if li, ok := bld[lastIdx].(*literalSegment); ok {
			li.literal += s
			return bld
		}
	}
	return append(bld, newLiteralSegment(s))
}

func badFormatSpecifier(str string, start, pos int) issue.Reported {
	return eval.Error(eval.TimespanBadFormatSpec, issue.H{`expression`: str[start:pos], `format`: str, `position`: pos})
}

func newTimespanFormat(format string, segments []segment) *TimespanFormat {
	return &TimespanFormat{fmt: format, segments: segments}
}

func (f *TimespanFormat) format(ts TimespanValue) string {
	b := bytes.NewBufferString(``)
	f.format2(b, ts)
	return b.String()
}

func (f *TimespanFormat) format2(b io.Writer, ts TimespanValue) {
	for _, s := range f.segments {
		s.appendTo(b, ts)
	}
}

func (f *TimespanFormat) parse(str string) (time.Duration, bool) {
	md := f.regexp().FindStringSubmatch(str)
	if md == nil {
		return 0, false
	}
	nanoseconds := int64(0)
	for idx, group := range md[1:] {
		segment := f.segments[idx]
		if _, ok := segment.(*literalSegment); ok {
			continue
		}
		in := 0
		for i, c := range group {
			if !unicode.IsSpace(c) {
				break
			}
			in = i
		}
		if in > 0 {
			group = group[in:]
		}
		if !digitsOnly.MatchString(group) {
			return 0, false
		}
		nanoseconds += segment.nanoseconds(group, segment.multiplier())
	}
	return time.Duration(nanoseconds), true
}

func (f *TimespanFormat) regexp() *regexp.Regexp {
	if f.rx == nil {
		b := bytes.NewBufferString(`\A-?`)
		for _, s := range f.segments {
			s.appendRegexp(b)
		}
		b.WriteString(`\z`)
		rx, err := regexp.Compile(b.String())
		if err != nil {
			panic(`Internal error while compiling Timespan format regexp: ` + err.Error())
		}
		f.rx = rx
	}
	return f.rx
}

func newLiteralSegment(literal string) segment {
	return &literalSegment{literal}
}

func (s *literalSegment) appendRegexp(buffer *bytes.Buffer) {
	buffer.WriteByte('(')
	buffer.WriteString(regexp.QuoteMeta(s.literal))
	buffer.WriteByte(')')
}

func (s *literalSegment) appendTo(buffer io.Writer, ts TimespanValue) {
	_, err := io.WriteString(buffer, s.literal)
	if err != nil {
		panic(err)
	}
}

func (s *literalSegment) multiplier() int {
	return 0
}

func (s *literalSegment) nanoseconds(group string, multiplier int) int64 {
	return 0
}

func (s *literalSegment) ordinal() int {
	return -1
}

func (s *literalSegment) setUseTotal() {}

func (s *valueSegment) initialize(padChar rune, width int, defaultWidth int) {
	s.useTotal = false
	s.padChar = padChar
	s.width = width
	s.defaultWidth = defaultWidth
}

func (s *valueSegment) appendRegexp(buffer *bytes.Buffer) {
	var err error
	if s.width < 0 {
		switch s.padChar {
		case 0, '0':
			if s.useTotal {
				buffer.WriteString(`([0-9]+)`)
			} else {
				_, err = fmt.Fprintf(buffer, `([0-9]{1,%d})`, s.defaultWidth)
			}
		default:
			if s.useTotal {
				buffer.WriteString(`\s*([0-9]+)`)
			} else {
				_, err = fmt.Fprintf(buffer, `([0-9\\s]{1,%d})`, s.defaultWidth)
			}
		}
	} else {
		switch s.padChar {
		case 0:
			_, err = fmt.Fprintf(buffer, `([0-9]{1,%d})`, s.width)
		case '0':
			_, err = fmt.Fprintf(buffer, `([0-9]{%d})`, s.width)
		default:
			_, err = fmt.Fprintf(buffer, `([0-9\\s]{%d})`, s.width)
		}
	}
	if err != nil {
		panic(err)
	}
}

func (s *valueSegment) appendValue(buffer io.Writer, n int64) {
	_, err := fmt.Fprintf(buffer, s.format, n)
	if err != nil {
		panic(err)
	}
}

func (s *valueSegment) createFormat() string {
	if s.padChar == 0 {
		return `%d`
	}
	w := s.width
	if w < 0 {
		w = s.defaultWidth
	}
	if s.padChar == ' ' {
		return fmt.Sprintf(`%%%dd`, w)
	}
	return fmt.Sprintf(`%%%c%dd`, s.padChar, w)
}

func (s *valueSegment) nanoseconds(group string, multiplier int) int64 {
	ns, err := strconv.ParseInt(group, 10, 64)
	if err != nil {
		ns = 0
	}
	return ns * int64(multiplier)
}

func (s *valueSegment) setUseTotal() {
	s.useTotal = true
}

func newDaySegment(padChar rune, width int) segment {
	s := &daySegment{}
	s.initialize(padChar, width, 1)
	s.format = s.createFormat()
	return s
}

func (s *daySegment) appendTo(buffer io.Writer, ts TimespanValue) {
	s.appendValue(buffer, ts.Days())
}

func (s *daySegment) multiplier() int {
	return NsecsPerDay
}

func (s *daySegment) ordinal() int {
	return dayMax
}

func newHourSegment(padChar rune, width int) segment {
	s := &hourSegment{}
	s.initialize(padChar, width, 2)
	s.format = s.createFormat()
	return s
}

func (s *hourSegment) appendTo(buffer io.Writer, ts TimespanValue) {
	var v int64
	if s.useTotal {
		v = ts.totalHours()
	} else {
		v = ts.Hours()
	}
	s.appendValue(buffer, v)
}

func (s *hourSegment) multiplier() int {
	return NsecsPerHour
}

func (s *hourSegment) ordinal() int {
	return hourMax
}

func newMinuteSegment(padChar rune, width int) segment {
	s := &minuteSegment{}
	s.initialize(padChar, width, 2)
	s.format = s.createFormat()
	return s
}

func (s *minuteSegment) appendTo(buffer io.Writer, ts TimespanValue) {
	var v int64
	if s.useTotal {
		v = ts.totalMinutes()
	} else {
		v = ts.Minutes()
	}
	s.appendValue(buffer, v)
}

func (s *minuteSegment) multiplier() int {
	return NsecsPerMin
}

func (s *minuteSegment) ordinal() int {
	return minMax
}

func newSecondSegment(padChar rune, width int) segment {
	s := &secondSegment{}
	s.initialize(padChar, width, 2)
	s.format = s.createFormat()
	return s
}

func (s *secondSegment) appendTo(buffer io.Writer, ts TimespanValue) {
	var v int64
	if s.useTotal {
		v = ts.totalSeconds()
	} else {
		v = ts.Seconds()
	}
	s.appendValue(buffer, v)
}

func (s *secondSegment) multiplier() int {
	return NsecsPerSec
}

func (s *secondSegment) ordinal() int {
	return secMax
}

func (s *fragmentSegment) appendValue(buffer io.Writer, n int64) {
	if !(s.useTotal || s.padChar == '0') {
		n, _ = strconv.ParseInt(trimTrailingZeroes.ReplaceAllString(strconv.FormatInt(n, 10), `$1`), 10, 64)
	}
	s.valueSegment.appendValue(buffer, n)
}

func (s *fragmentSegment) createFormat() string {
	if s.padChar == 0 {
		return `%d`
	}
	w := s.width
	if w < 0 {
		w = s.defaultWidth
	}
	return fmt.Sprintf(`%%-%dd`, w)
}

func (s *fragmentSegment) nanoseconds(group string, multiplier int) int64 {
	if s.useTotal {
		panic(eval.Error(eval.TimespanFormatSpecNotHigher, issue.NO_ARGS))
	}
	n := s.valueSegment.nanoseconds(group, multiplier)
	p := int64(9 - len(group))
	if p <= 0 {
		return n
	}
	return utils.Int64Pow(n*10, p)
}

func newMillisecondSegment(padChar rune, width int) segment {
	s := &millisecondSegment{}
	s.initialize(padChar, width, 3)
	s.format = s.createFormat()
	return s
}

func (s *millisecondSegment) appendTo(buffer io.Writer, ts TimespanValue) {
	var v int64
	if s.useTotal {
		v = ts.totalMilliseconds()
	} else {
		v = ts.Milliseconds()
	}
	s.appendValue(buffer, v)
}

func (s *millisecondSegment) multiplier() int {
	return NsecsPerMsec
}

func (s *millisecondSegment) ordinal() int {
	return msecMax
}

func newNanosecondSegment(padChar rune, width int) segment {
	s := &nanosecondSegment{}
	s.initialize(padChar, width, 9)
	s.format = s.createFormat()
	return s
}

func (s *nanosecondSegment) appendTo(buffer io.Writer, ts TimespanValue) {
	v := ts.totalNanoseconds()
	w := s.width
	if w < 0 {
		w = s.defaultWidth
	}
	if w < 9 {
		// Truncate digits to the right, i.e. let %6N reflect microseconds
		v /= utils.Int64Pow(10, int64(9-w))
		if !s.useTotal {
			v %= utils.Int64Pow(10, int64(w))
		}
	} else {
		if !s.useTotal {
			v %= NsecsPerSec
		}
	}
	s.appendValue(buffer, v)
}

func (s *nanosecondSegment) multiplier() int {
	w := s.width
	if w < 0 {
		w = s.defaultWidth
	}
	if w < 9 {
		return int(utils.Int64Pow(10, int64(9-w)))
	}
	return 1
}

func (s *nanosecondSegment) ordinal() int {
	return nsecMax
}

func toTimespanFormats(f eval.Value) []*TimespanFormat {
	fs := DefaultTimespanFormats
	switch f := f.(type) {
	case *ArrayValue:
		fs = make([]*TimespanFormat, f.Len())
		f.EachWithIndex(func(f eval.Value, i int) {
			fs[i] = DefaultTimespanFormatParser.ParseFormat(f.String())
		})
	case stringValue:
		fs = []*TimespanFormat{DefaultTimespanFormatParser.ParseFormat(f.String())}
	}
	return fs
}
