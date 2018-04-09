package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"regexp"
	"strconv"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-evaluator/utils"
	"sync"
	"unicode"
)

type (
	TimespanType struct {
		min time.Duration
		max time.Duration
	}

	// TimespanValue represents TimespanType as a value
	TimespanValue TimespanType
)

const (
	NSECS_PER_USEC = 1000
	NSECS_PER_MSEC = NSECS_PER_USEC * 1000
	NSECS_PER_SEC  = NSECS_PER_MSEC * 1000
	NSECS_PER_MIN  = NSECS_PER_SEC  * 60
	NSECS_PER_HOUR = NSECS_PER_MIN  * 60
	NSECS_PER_DAY  = NSECS_PER_HOUR * 24

	KEY_STRING = `string`
	KEY_FORMAT = `format`
	KEY_NEGATIVE = `negative`
	KEY_DAYS = `days`
	KEY_HOURS = `hours`
	KEY_MINUTES = `minutes`
	KEY_SECONDS = `seconds`
	KEY_MILLISECONDS = `milliseconds`
	KEY_MICROSECONDS = `microseconds`
	KEY_NANOSECONDS = `nanoseconds`
)

var TIMESPAN_MIN = time.Duration(math.MinInt64)
var TIMESPAN_MAX = time.Duration(math.MaxInt64)

var timespanType_DEFAULT = &TimespanType{TIMESPAN_MIN, TIMESPAN_MAX}

var Timespan_Type eval.ObjectType
var DefaultTimespanFormatParser *TimespanFormatParser

func init() {
	tp := NewTimespanFormatParser()
	DEFAULT_TIMESPAN_FORMATS = []*TimespanFormat{
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

	Timespan_Type = newObjectType(`Pcore::TimespanType`,
		`Pcore::ScalarType{
	attributes => {
		from => { type => Optional[Timespan], value => undef },
		to => { type => Optional[Timespan], value => undef }
	}
}`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return NewTimespanType2(args...)
		})

	newGoConstructor2(`Timespan`,
		func(t eval.LocalTypes) {
			t.Type(`Formats`, `Variant[String[2],Array[String[2], 1]]`)
		},

		func(d eval.Dispatch) {
			d.Param(`Variant[Integer,Float]`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				arg := args[0]
				if i, ok := arg.(*IntegerValue); ok {
					return WrapTimespan(time.Duration(i.Int() * NSECS_PER_SEC))
				}
				return WrapTimespan(time.Duration(arg.(*FloatValue).Float() * NSECS_PER_SEC))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`String[1]`)
			d.OptionalParam(`Formats`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				formats := DEFAULT_TIMESPAN_FORMATS
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
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				days := args[0].(*IntegerValue).Int()
				hours := args[1].(*IntegerValue).Int()
				minutes := args[2].(*IntegerValue).Int()
				seconds := args[3].(*IntegerValue).Int()
				argc := len(args)
				var milliseconds, microseconds, nanoseconds int64
				if argc > 4 {
					milliseconds = args[4].(*IntegerValue).Int()
					if argc > 5 {
						microseconds = args[5].(*IntegerValue).Int()
						if argc > 6 {
							nanoseconds = args[6].(*IntegerValue).Int()
						}
					}
				}
				return WrapTimespan(fromFields(false, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Struct[string => String[1], Optional[format] => Formats]`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				hash := args[0].(*HashValue)
				str := hash.Get5(`string`, _EMPTY_STRING)
				formats := toTimespanFormats(hash.Get5(`format`, _UNDEF))
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
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				return WrapTimespan(fromFieldsHash(args[0].(*HashValue)))
			})
		})
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
			t, ok =fromHash(arg.(*HashValue))
		case *StringValue:
			t, ok = parseDuration(arg.(*StringValue).value, DEFAULT_TIMESPAN_FORMATS)
		case *IntegerValue:
			t, ok = time.Duration(arg.(*IntegerValue).Int()*1000000000), true
		case *FloatValue:
			t, ok = time.Duration(arg.(*FloatValue).Float()*1000000000.0), true
		case *DefaultValue:
			if argNo == 0 {
				t, ok = TIMESPAN_MIN, true
			} else {
				t, ok = TIMESPAN_MAX, true
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
		return &TimespanType{min, TIMESPAN_MAX}
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

func (t *TimespanType) Get(c eval.Context, key string) (eval.PValue, bool) {
	switch key {
	case `from`:
		v := eval.UNDEF
		if t.min != TIMESPAN_MIN {
			v = WrapTimespan(t.min)
		}
		return v, true
	case `to`:
		v := eval.UNDEF
		if t.max != TIMESPAN_MAX {
			v = WrapTimespan(t.max)
		}
		return v, true
	default:
		return nil, false
	}
}

func (t *TimespanType) MetaType() eval.ObjectType {
	return Timespan_Type
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

func (t *TimespanType) IsInstance(c eval.Context, o eval.PValue, g eval.Guard) bool {
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

func ParseTimespan(str string, formats []*TimespanFormat) *TimespanValue {
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
	panic(eval.Error(nil, eval.EVAL_TIMESPAN_CANNOT_BE_PARSED, issue.H{`str`: str, `formats`: fs.String()}))
}

func fromFields(negative bool, days, hours, minutes, seconds, milliseconds, microseconds, nanoseconds int64) time.Duration {
	ns := (((((days * 24 + hours) * 60 + minutes) * 60 + seconds) * 1000 + milliseconds) * 1000 + microseconds) * 1000 + nanoseconds
	if negative {
		ns = -ns
	}
	return time.Duration(ns)
}

func fromFieldsHash(hash *HashValue) time.Duration {
	intArg := func(key string) int64 {
		if v, ok := hash.Get4(key); ok {
			if i, ok := v.(*IntegerValue); ok {
				return i.Int()
			}
		}
		return 0
	}
	boolArg := func(key string) bool {
		if v, ok := hash.Get4(key); ok {
			if b, ok := v.(*BooleanValue); ok {
				return b.Bool()
			}
		}
		return false
	}
	return fromFields(
		boolArg(KEY_NEGATIVE),
		intArg(KEY_DAYS),
		intArg(KEY_HOURS),
		intArg(KEY_MINUTES),
		intArg(KEY_SECONDS),
		intArg(KEY_MILLISECONDS),
		intArg(KEY_MICROSECONDS),
		intArg(KEY_NANOSECONDS))
}

func fromStringHash(hash *HashValue) (time.Duration, bool) {
  str := hash.Get5(KEY_STRING, _EMPTY_STRING)
  fmtStrings := hash.Get5(KEY_FORMAT, nil)
  var formats []*TimespanFormat
  if fmtStrings == nil {
  	formats = DEFAULT_TIMESPAN_FORMATS
  } else {
  	if fs, ok := fmtStrings.(*StringValue); ok {
  		formats = []*TimespanFormat{DefaultTimespanFormatParser.ParseFormat(fs.String())}
	  } else {
	    if fsa, ok := fmtStrings.(*ArrayValue); ok {
		    formats = make([]*TimespanFormat, fsa.Len())
		    fsa.EachWithIndex(func(fs eval.PValue, i int) {
		      formats[i] = DefaultTimespanFormatParser.ParseFormat(fs.String())
			  })
		  }
	  }
  }
	return parseDuration(str.String(), formats)
}

func fromHash(hash *HashValue) (time.Duration, bool) {
	if hash.IncludesKey2(KEY_STRING) {
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

func (tv *TimespanValue) Abs() eval.NumericValue {
	if tv.min < 0 {
		return WrapTimespan(-tv.min)
	}
	return tv
}

// Hours returns a positive integer denoting the number of days
func (tv *TimespanValue) Days() int64 {
	return tv.totalDays()
}

func (tv *TimespanValue) Duration() time.Duration {
	return tv.min
}

// Hours returns a positive integer, 0 - 23 denoting hours of day
func (tv *TimespanValue) Hours() int64 {
	return tv.totalHours() % 24
}

func (tv *TimespanValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*TimespanValue); ok {
		return tv.Int() == ov.Int()
	}
	return false
}

// Float returns the number of seconds with fraction
func (tv *TimespanValue) Float() float64 {
	return float64(tv.totalNanoseconds()) / float64(NSECS_PER_SEC)
}

func (tv *TimespanValue) Format(format string) string {
	return DefaultTimespanFormatParser.ParseFormat(format).format(tv)
}

// Int returns the total number of seconds
func (tv *TimespanValue) Int() int64 {
	return tv.totalSeconds()
}

// Minutes returns a positive integer, 0 - 59 denoting minutes of hour
func (tv *TimespanValue) Minutes() int64 {
	return tv.totalMinutes() % 60
}

// Seconds returns a positive integer, 0 - 59 denoting seconds of minute
func (tv *TimespanValue) Seconds() int64 {
	return tv.totalSeconds() % 60
}

// Seconds returns a positive integer, 0 - 999 denoting milliseconds of second
func (tv *TimespanValue) Milliseconds() int64 {
	return tv.totalMilliseconds() % 1000
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

func (tv *TimespanValue) totalDays() int64 {
	return tv.min.Nanoseconds() / NSECS_PER_DAY
}

func (tv *TimespanValue) totalHours() int64 {
	return tv.min.Nanoseconds() / NSECS_PER_HOUR
}

func (tv *TimespanValue) totalMinutes() int64 {
	return tv.min.Nanoseconds() / NSECS_PER_MIN
}

func (tv *TimespanValue) totalSeconds() int64 {
	return tv.min.Nanoseconds() / NSECS_PER_SEC
}

func (tv *TimespanValue) totalMilliseconds() int64 {
	return tv.min.Nanoseconds() / NSECS_PER_MSEC
}

func (tv *TimespanValue) totalMicroseconds() int64 {
	return tv.min.Nanoseconds() / NSECS_PER_USEC
}

func (tv *TimespanValue) totalNanoseconds() int64 {
	return tv.min.Nanoseconds()
}

func (tv *TimespanValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	DEFAULT_TIMESPAN_FORMATS[0].format2(b, tv)
}

func (tv *TimespanValue) Type() eval.PType {
	return (*TimespanType)(tv)
}

type(
	TimespanFormat struct {
		rx *regexp.Regexp
		fmt string
		segments []segment
	}

	TimespanFormatParser struct {
		lock sync.Mutex
		formats map[string]*TimespanFormat
	}

	segment interface {
		appendRegexp(buffer *bytes.Buffer)

		appendTo(buffer io.Writer, ts *TimespanValue)

		multiplier() int

		nanoseconds(group string, multiplier int) int64

		ordinal() int

		setUseTotal()
	}

	literalSegment struct {
		literal string
	}

	valueSegment struct {
		useTotal bool
		padchar rune
		width int
		defaultWidth int
		format string
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

const(
	NSEC_MAX = 0
	MSEC_MAX = 1
	SEC_MAX = 2
	MIN_MAX = 3
	HOUR_MAX = 4
	DAY_MAX = 5

	// States used by the #internal_parser function

	STATE_LITERAL = 0 // expects literal or '%'
	STATE_PAD  = 1 // expects pad, width, or format character
	STATE_WIDTH = 2 // expects width, or format character
)

var trimTrailingZeroes = regexp.MustCompile(`\A([0-9]+?)0*\z`)
var digitsOnly = regexp.MustCompile(`\A[0-9]+\z`)
var DEFAULT_TIMESPAN_FORMATS []*TimespanFormat

func NewTimespanFormatParser() *TimespanFormatParser {
	return &TimespanFormatParser{formats:make(map[string]*TimespanFormat, 17)}
}

func (p *TimespanFormatParser) ParseFormat(format string) *TimespanFormat  {
	p.lock.Lock()
	defer p.lock.Unlock()

	if fmt, ok := p.formats[format]; ok {
		return fmt
	}
	fmt := p.parse(format)
	p.formats[format] = fmt
	return fmt
}

func (p *TimespanFormatParser) parse(str string) *TimespanFormat  {
	bld := make([]segment, 0, 7)
	highest := -1
	state := STATE_LITERAL
	padchar := '0'
	width := -1
	fstart := 0

	for pos, c := range str {
		if state == STATE_LITERAL {
			if c == '%' {
				state = STATE_PAD
				fstart = pos
				padchar = '0'
				width = -1
			} else {
				bld = appendLiteral(bld, c)
			}
			continue
		}

		switch c {
		case '%':
			bld = appendLiteral(bld, c)
			state = STATE_LITERAL
		case '-':
			if state != STATE_PAD {
				panic(badFormatSpecifier(str, fstart, pos))
			}
			padchar = 0
			state = STATE_WIDTH
		case '_':
			if state != STATE_PAD {
				panic(badFormatSpecifier(str, fstart, pos))
			}
			padchar = ' '
			state = STATE_WIDTH
		case 'D':
			highest = DAY_MAX
			bld = append(bld, newDaySegment(padchar, width))
			state = STATE_LITERAL
		case 'H':
			if highest < HOUR_MAX {
				highest = HOUR_MAX
			}
			bld = append(bld, newHourSegment(padchar, width))
			state = STATE_LITERAL
		case 'M':
			if highest < MIN_MAX {
				highest = MIN_MAX
			}
			bld = append(bld, newMinuteSegment(padchar, width))
			state = STATE_LITERAL
		case 'S':
			if highest < SEC_MAX {
				highest = SEC_MAX
			}
			bld = append(bld, newSecondSegment(padchar, width))
			state = STATE_LITERAL
		case 'L':
			if highest < MSEC_MAX {
				highest = MSEC_MAX
			}
			bld = append(bld, newMillisecondSegment(padchar, width))
			state = STATE_LITERAL
		case 'N':
			if highest < NSEC_MAX {
				highest = NSEC_MAX
			}
			bld = append(bld, newNanosecondSegment(padchar, width))
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
					width = width * 10 + n
				}
			}
			state = STATE_WIDTH
		}
	}

	if state != STATE_LITERAL {
		panic(badFormatSpecifier(str, fstart, len(str)))
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

func badFormatSpecifier(str string, start, pos int) *issue.Reported {
  return eval.Error(nil, eval.EVAL_TIMESPAN_BAD_FSPEC, issue.H{`expression`: str[start:pos], `format`: str, `position`: pos})
}

func newTimespanFormat(format string, segments []segment) *TimespanFormat {
	return &TimespanFormat{fmt:format, segments:segments}
}

func (f *TimespanFormat) format(ts *TimespanValue) string {
	b := bytes.NewBufferString(``)
	f.format2(b, ts)
	return b.String()
}

func (f *TimespanFormat) format2(b io.Writer, ts *TimespanValue) {
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

func (s *literalSegment) appendTo(buffer io.Writer, ts *TimespanValue) {
	io.WriteString(buffer, s.literal)
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

func (s *valueSegment) initialize(padchar rune, width int, defaultWidth int) {
	s.useTotal = false
	s.padchar = padchar
	s.width = width
	s.defaultWidth = defaultWidth
}

func (s *valueSegment) appendRegexp(buffer *bytes.Buffer) {
	if s.width < 0 {
		switch s.padchar {
		case 0, '0':
			if s.useTotal {
				buffer.WriteString(`([0-9]+)`)
			} else {
				fmt.Fprintf(buffer, `([0-9]{1,%d})`, s.defaultWidth)
			}
		default:
			if s.useTotal {
				buffer.WriteString(`\s*([0-9]+)`)
			} else {
				fmt.Fprintf(buffer, `([0-9\\s]{1,%d})`, s.defaultWidth)
			}
		}
	} else {
		switch s.padchar {
		case 0:
			fmt.Fprintf(buffer, `([0-9]{1,%d})`, s.width)
		case '0':
			fmt.Fprintf(buffer, `([0-9]{%d})`, s.width)
		default:
			fmt.Fprintf(buffer, `([0-9\\s]{%d})`, s.width)
		}
	}
}

func (s *valueSegment) appendValue(buffer io.Writer, n int64) {
	fmt.Fprintf(buffer, s.format, n)
}

func (s *valueSegment) createFormat() string {
	if s.padchar == 0 {
		return `%d`
	}
	w := s.width
	if w < 0 {
		w = s.defaultWidth
	}
	if s.padchar == ' ' {
		return fmt.Sprintf(`%%%dd`, w)
	}
	return fmt.Sprintf(`%%%c%dd`, s.padchar, w)
}

func (s *valueSegment) multiplier() int {
	return 0
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

func newDaySegment(padchar rune, width int) segment {
	s := &daySegment{}
	s.initialize(padchar, width, 1)
	s.format = s.createFormat()
	return s
}

func (s *daySegment) appendTo(buffer io.Writer, ts *TimespanValue) {
	s.appendValue(buffer, ts.Days())
}

func (s *daySegment) multiplier() int {
	return NSECS_PER_DAY
}

func (s *daySegment) ordinal() int {
	return DAY_MAX
}

func newHourSegment(padchar rune, width int) segment {
	s := &hourSegment{}
	s.initialize(padchar, width, 2)
	s.format = s.createFormat()
	return s
}

func (s *hourSegment) appendTo(buffer io.Writer, ts *TimespanValue) {
	var v int64
	if s.useTotal {
		v = ts.totalHours()
	} else {
		v = ts.Hours()
	}
	s.appendValue(buffer, v)
}

func (s *hourSegment) multiplier() int {
	return NSECS_PER_HOUR
}

func (s *hourSegment) ordinal() int {
	return HOUR_MAX
}

func newMinuteSegment(padchar rune, width int) segment {
	s := &minuteSegment{}
	s.initialize(padchar, width, 2)
	s.format = s.createFormat()
	return s
}

func (s *minuteSegment) appendTo(buffer io.Writer, ts *TimespanValue) {
	var v int64
	if s.useTotal {
		v = ts.totalMinutes()
	} else {
		v = ts.Minutes()
	}
	s.appendValue(buffer, v)
}

func (s *minuteSegment) multiplier() int {
	return NSECS_PER_MIN
}

func (s *minuteSegment) ordinal() int {
	return MIN_MAX
}

func newSecondSegment(padchar rune, width int) segment {
	s := &secondSegment{}
	s.initialize(padchar, width, 2)
	s.format = s.createFormat()
	return s
}

func (s *secondSegment) appendTo(buffer io.Writer, ts *TimespanValue) {
	var v int64
	if s.useTotal {
		v = ts.totalSeconds()
	} else {
		v = ts.Seconds()
	}
	s.appendValue(buffer, v)
}

func (s *secondSegment) multiplier() int {
	return NSECS_PER_SEC
}

func (s *secondSegment) ordinal() int {
	return SEC_MAX
}

func (s *fragmentSegment) appendValue(buffer io.Writer, n int64) {
	if !(s.useTotal || s.padchar == '0') {
		n, _ = strconv.ParseInt(trimTrailingZeroes.ReplaceAllString(strconv.FormatInt(n, 10), `$1`), 10, 64)
	}
	s.valueSegment.appendValue(buffer, n)
}

func (s *fragmentSegment) createFormat() string {
	if s.padchar == 0 {
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
		panic(eval.Error(nil, eval.EVAL_TIMESPAN_FSPEC_NOT_HIGHER, issue.NO_ARGS))
	}
	n := s.valueSegment.nanoseconds(group, multiplier)
	p := int64(9 - len(group))
	if p <= 0 {
		return n
	}
	return utils.Int64Pow(n * 10, p)
}

func newMillisecondSegment(padchar rune, width int) segment {
	s := &millisecondSegment{}
	s.initialize(padchar, width, 3)
	s.format = s.createFormat()
	return s
}

func (s *millisecondSegment) appendTo(buffer io.Writer, ts *TimespanValue) {
	var v int64
	if s.useTotal {
		v = ts.totalMilliseconds()
	} else {
		v = ts.Milliseconds()
	}
	s.appendValue(buffer, v)
}

func (s *millisecondSegment) multiplier() int {
	return NSECS_PER_MSEC
}

func (s *millisecondSegment) ordinal() int {
	return MSEC_MAX
}

func newNanosecondSegment(padchar rune, width int) segment {
	s := &nanosecondSegment{}
	s.initialize(padchar, width, 9)
	s.format = s.createFormat()
	return s
}

func (s *nanosecondSegment) appendTo(buffer io.Writer, ts *TimespanValue) {
	v := ts.totalNanoseconds()
  w := s.width
  if w < 0 {
  	w = s.defaultWidth
  }
  if w < 9 {
  	// Truncate digits to the right, i.e. let %6N reflect microseconds
  	v /= utils.Int64Pow(10, int64(9 - w))
  	if !s.useTotal {
		  v %= utils.Int64Pow(10, int64(w))
	  }
  } else {
  	if !s.useTotal {
		  v %= NSECS_PER_SEC
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
		return int(utils.Int64Pow(10, int64(9 - w)))
	}
	return 1
}

func (s *nanosecondSegment) ordinal() int {
	return NSEC_MAX
}

func toTimespanFormats(fmt eval.PValue) []*TimespanFormat {
	formats := DEFAULT_TIMESPAN_FORMATS
	switch fmt.(type) {
	case *ArrayValue:
		fa := fmt.(*ArrayValue)
		formats = make([]*TimespanFormat, fa.Len())
		fa.EachWithIndex(func(f eval.PValue, i int) {
			formats[i] = DefaultTimespanFormatParser.ParseFormat(f.String())
		})
	case *StringValue:
		formats = []*TimespanFormat{DefaultTimespanFormatParser.ParseFormat(fmt.String())}
	}
	return formats
}
