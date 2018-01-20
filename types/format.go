package types

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"github.com/puppetlabs/go-evaluator/utils"
	"unicode/utf8"
	"github.com/puppetlabs/go-parser/issue"
)

type (
	format struct {
		alt              bool
		left             bool
		zeroPad          bool
		formatChar       byte
		plus             byte
		prec             int
		width            int
		leftDelimiter    byte
		separator        string
		separator2       string
		origFmt          string
		containerFormats FormatMap
	}

	formatContext struct {
		indentation Indentation
		formatMap   FormatMap
	}

	indentation struct {
		first     bool
		indenting bool
		level     int
		padding   string
	}
)

func (f *format) Equals(other interface{}, guard Guard) bool {
	if of, ok := other.(*format); ok {
		return f.alt == of.alt &&
			f.left == of.left &&
			f.zeroPad == of.zeroPad &&
			f.formatChar == of.formatChar &&
			f.plus == of.plus &&
			f.prec == of.prec &&
			f.width == of.width &&
			f.leftDelimiter == of.leftDelimiter &&
			f.separator == of.separator &&
			f.separator2 == of.separator2 &&
			f.origFmt == of.origFmt &&
			f.containerFormats.Equals(of.containerFormats, nil)
	}
	return false
}

func (f *format) String() string {
	return f.origFmt
}

func (f *format) ToString(bld io.Writer, format FormatContext, g RDetect) {
	io.WriteString(bld, f.origFmt)
}

func (f *format) Type() PType {
	return WrapRuntime(f).Type()
}

var DEFAULT_CONTAINER_FORMATS = FormatMap(WrapHash([]*HashEntry{WrapHashEntry(DefaultAnyType(), simpleFormat('p'))}))

var DEFAULT_ANY_FORMAT = simpleFormat('s')

func init() {
	DEFAULT_FORMAT = DEFAULT_ANY_FORMAT
	DEFAULT_FORMAT_CONTEXT = NONE

	NewFormatContext = newFormatContext
	NewFormatContext2 = newFormatContext2
	NewFormatContext3 = newFormatContext3
	NewIndentation = newIndentation
	NewFormat = newFormat
}

var DEFAULT_ARRAY_FORMAT = basicFormat('a', `,`, '[', DEFAULT_CONTAINER_FORMATS)
var DEFAULT_HASH_FORMAT = basicFormat('h', ` => `, '{', DEFAULT_CONTAINER_FORMATS)

var DEFAULT_INDENTATION = newIndentation(false, 0)

var DEFAULT_FORMATS = FormatMap(WrapHash([]*HashEntry{
	WrapHashEntry(DefaultObjectType(), simpleFormat('p')),
	WrapHashEntry(DefaultTypeType(), simpleFormat('p')),
	WrapHashEntry(DefaultFloatType(), simpleFormat('f')),
	WrapHashEntry(DefaultNumericType(), simpleFormat('d')),
	WrapHashEntry(DefaultArrayType(), DEFAULT_ARRAY_FORMAT),
	WrapHashEntry(DefaultHashType(), DEFAULT_HASH_FORMAT),
	WrapHashEntry(DefaultBinaryType(), simpleFormat('B')),
	WrapHashEntry(DefaultAnyType(), DEFAULT_ANY_FORMAT),
}))

var delimiters = []byte{'[', '{', '(', '<', '|'}
var delimiterPairs = map[byte][2]byte{
	'[': [2]byte{'[', ']'},
	'{': [2]byte{'{', '}'},
	'(': [2]byte{'(', ')'},
	'<': [2]byte{'<', '>'},
	'|': [2]byte{'|', '|'},
	' ': [2]byte{0, 0},
	0:   [2]byte{'[', ']'},
}

var NONE = newFormatContext2(DEFAULT_INDENTATION, DEFAULT_FORMATS)

//TODO
var EXPANDED = newFormatContext2(DEFAULT_INDENTATION, DEFAULT_FORMATS)

func newFormatContext(t PType, format Format, indentation Indentation) FormatContext {
	return &formatContext{indentation, WrapHash([]*HashEntry{WrapHashEntry(t, format)})}
}

func newFormatContext2(indentation Indentation, formatMap FormatMap) FormatContext {
	return &formatContext{indentation, formatMap}
}


var TYPE_STRING_FORMAT = NewVariantType([]PType{DefaultStringType(), DefaultDefaultType(), DefaultHashType()})

func newFormatContext3(value PValue, format PValue) (context FormatContext, err error) {
	AssertInstance(`String format`, TYPE_STRING_FORMAT, format)

	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(*issue.ReportedIssue); !ok {
				panic(r)
			}
		}
	}()

	switch format.(type) {
	case *StringValue:
		context = NewFormatContext(value.Type(), newFormat(format.String()), DEFAULT_INDENTATION)
	case *DefaultValue:
		context = DEFAULT_FORMAT_CONTEXT
	default:
		context = newFormatContext2(DEFAULT_INDENTATION, mergeFormats(DEFAULT_FORMATS, NewFormatMap(format.(*HashValue))))
	}
	return
}

func mergeFormats(lower FormatMap, higher FormatMap) FormatMap {
	if lower == nil || lower.Len() == 0 {
		return higher
	}
	if higher == nil || higher.Len() == 0 {
		return lower
	}

	higherKeys := higher.Keys()
	normLower := WrapHash2(Reject2(lower.Entries(), func(lev PValue) bool {
		le := lev.(*HashEntry)
		return Any2(higherKeys, func(hk PValue) bool {
			return !hk.Equals(le.Key(), nil) && IsAssignable(hk.(PType), le.Key().(PType))
		})
	}).Elements())

	merged := make([]*HashEntry, 0, 8)
	allKeys := make([]PValue, normLower.Len(), normLower.Len()+higher.Len())
	copy(allKeys, normLower.Keys().Elements())
	allKeys = append(allKeys, higherKeys.Elements()...)
	for _, k := range UniqueValues(allKeys) {
		if low, ok := normLower.Get(k); ok {
			if high, ok := higher.Get(k); ok {
				merged = append(merged, WrapHashEntry(k, merge(low.(Format), high.(Format))))
			} else {
				merged = append(merged, WrapHashEntry(k, low))
			}
		} else {
			if high, ok := higher.Get(k); ok {
				merged = append(merged, WrapHashEntry(k, high))
			}
		}
	}

	sort.Slice(merged, func(ax, bx int) bool {
		a := merged[ax].Key().(PType)
		b := merged[bx].Key().(PType)
		if a.Equals(b, nil) {
			return false
		}
		ab := IsAssignable(b, a)
		ba := IsAssignable(a, b)
		if ab && !ba {
			return true
		}
		if !ab && ba {
			return false
		}
		ra := typeRank(a)
		rb := typeRank(b)
		if ra < rb {
			return true
		}
		if ra > rb {
			return false
		}
		return strings.Compare(a.String(), b.String()) < 0
	})
	return FormatMap(WrapHash(merged))
}

func merge(low Format, high Format) Format {
	sep := high.Separator(NO_STRING)
	if sep == NO_STRING {
		sep = low.Separator(NO_STRING)
	}
	sep2 := high.Separator2(NO_STRING)
	if sep2 == NO_STRING {
		sep2 = low.Separator2(NO_STRING)
	}

	return &format{
		origFmt:          high.OrigFormat(),
		alt:              high.IsAlt(),
		leftDelimiter:    high.LeftDelimiter(),
		formatChar:       high.FormatChar(),
		zeroPad:          high.IsZeroPad(),
		prec:             high.Precision(),
		left:             high.IsLeft(),
		plus:             high.Plus(),
		width:            high.Width(),
		separator2:       sep2,
		separator:        sep,
		containerFormats: mergeFormats(low.ContainerFormats(), high.ContainerFormats()),
	}
}

func typeRank(pt PType) int {
	switch pt.(type) {
	case *NumericType, *IntegerType, *FloatType:
		return 13
	case *StringType:
		return 12
	case *EnumType:
		return 11
	case *PatternType:
		return 10
	case *ArrayType:
		return 4
	case *TupleType:
		return 3
	case *HashType:
		return 2
	case *StructType:
		return 1
	}
	return 0
}

var TYPE_STRING_FORMAT_TYPE_HASH = NewHashType(DefaultTypeType(), NewVariantType([]PType{DefaultStringType(), DefaultHashType()}), nil)

func NewFormatMap(h *HashValue) FormatMap {
	AssertInstance(`String format type hash`, TYPE_STRING_FORMAT_TYPE_HASH, h)
	entries := h.EntriesSlice()
	result := make([]*HashEntry, len(entries))
	for idx, entry := range entries {
		pt := entry.Key().(PType)
		v := entry.Value()
		if s, ok := v.(*StringValue); ok {
			result[idx] = WrapHashEntry(pt, newFormat(s.String()))
		} else {
			result[idx] = WrapHashEntry(pt, FormatFromHash(v.(*HashValue)))
		}
	}
	return FormatMap(WrapHash(result))
}

func NewFormatMap2(t PType, tf Format, fm FormatMap) FormatMap {
	return mergeFormats(fm, FormatMap(WrapHash([]*HashEntry{&HashEntry{t, tf}})))
}

var TYPE_STRING_FORMAT_HASH = NewStructType([]*StructElement {
	NewStructElement2(`format`, DefaultStringType()),
	NewStructElement(NewOptionalType3(`separator`), DefaultStringType()),
	NewStructElement(NewOptionalType3(`separator2`), DefaultStringType()),
	NewStructElement(NewOptionalType3(`string_formats`), DefaultHashType()),
})

func FormatFromHash(h *HashValue) Format {
	AssertInstance(`String format hash`, TYPE_STRING_FORMAT_HASH, h)

	stringArg := func(key string, required bool) string {
		v := h.Get2(key, _UNDEF)
		switch v.(type) {
		case *StringValue:
			return v.String()
		default:
			return NO_STRING
		}
	}

	var cf FormatMap
	cf = nil
	if v := h.Get2(`string_formats`, _UNDEF); !Equals(v, _UNDEF) {
		cf = NewFormatMap(v.(*HashValue))
	}
	return parseFormat(stringArg(`format`, true), stringArg(`separator`, false), stringArg(`separator2`, false), cf)
}

func (c *formatContext) Indentation() Indentation {
	return c.indentation
}

func (c *formatContext) FormatMap() FormatMap {
	return c.formatMap
}

func (c *formatContext) UnsupportedFormat(t PType, supportedFormats string, actualFormat Format) error {
	return Error(EVAL_UNSUPPORTED_STRING_FORMAT, issue.H{`format`: actualFormat.FormatChar(), `type`: t.Name(), `supported_formats`: supportedFormats})
}

func newIndentation(indenting bool, level int) Indentation {
	return newIndentation2(true, indenting, level)
}

func newIndentation2(first bool, indenting bool, level int) Indentation {
	return &indentation{first, indenting, level, strings.Repeat(`  `, level)}
}

func (i *indentation) Breaks() bool {
	return i.indenting && i.level > 0 && !i.first
}

func (i *indentation) Level() int {
	return i.level
}

func (i *indentation) Increase(indenting bool) Indentation {
	return newIndentation2(true, indenting, i.level+1)
}

func (i *indentation) Indenting(indenting bool) Indentation {
	if i.indenting == indenting {
		return i
	}
	return &indentation{i.first, indenting, i.level, i.padding}
}

func (i *indentation) IsFirst() bool {
	return i.first
}

func (i *indentation) IsIndenting() bool {
	return i.indenting
}

func (i *indentation) Padding() string {
	return i.padding
}

func (i *indentation) Subsequent() Indentation {
	if i.first {
		return &indentation{false, i.indenting, i.level, i.padding}
	}
	return i
}

// NewFormat parses a format string into a Format
func newFormat(format string) Format {
	return parseFormat(format, NO_STRING, NO_STRING, nil)
}

func simpleFormat(formatChar byte) Format {
	return basicFormat(formatChar, NO_STRING, '[', nil)
}

func basicFormat(formatChar byte, sep2 string, leftDelimiter byte, containerFormats FormatMap) Format {
	return &format{
		formatChar:       formatChar,
		prec:             -1,
		width:            -1,
		origFmt:          `%` + string(formatChar),
		separator:        `,`,
		separator2:       sep2,
		leftDelimiter:    leftDelimiter,
		containerFormats: containerFormats,
	}
}

func parseFormat(origFmt string, separator string, separator2 string, containerFormats FormatMap) Format {
	group := FORMAT_PATTERN.FindStringSubmatch(origFmt)
	if group == nil {
		panic(Error(EVAL_INVALID_STRING_FORMAT_SPEC, issue.H{`format`: origFmt}))
	}

	flags := group[1]

	plus := byte(0)
	if hasDelimOnce(flags, origFmt, ' ') {
		plus = ' '
	} else if hasDelimOnce(flags, origFmt, '+') {
		plus = '+'
	}

	foundDelim := byte(0)
	for _, delim := range delimiters {
		if hasDelimOnce(flags, origFmt, delim) {
			if foundDelim != 0 {
				panic(Error(EVAL_INVALID_STRING_FORMAT_DELIMITER, issue.H{`delimiter`: foundDelim}))
			}
			foundDelim = delim
		}
	}

	if foundDelim == 0 && plus == ' ' {
		foundDelim = plus
	}

	width := -1
	prec := -1
	if tmp := group[2]; tmp != `` {
		width, _ = strconv.Atoi(tmp)
	}
	if tmp := group[3]; tmp != `` {
		prec, _ = strconv.Atoi(tmp)
	}
	return &format{
		origFmt:          origFmt,
		formatChar:       group[4][0],
		left:             hasDelimOnce(flags, origFmt, '-'),
		alt:              hasDelimOnce(flags, origFmt, '#'),
		zeroPad:          hasDelimOnce(flags, origFmt, '0'),
		plus:             plus,
		leftDelimiter:    foundDelim,
		width:            width,
		prec:             prec,
		separator:        separator,
		separator2:       separator2,
		containerFormats: containerFormats,
	}
}

func (f *format) unParse() string {
	b := bytes.NewBufferString(`%`)
	if f.zeroPad {
		b.Write([]byte{'0'})
	}
	if f.plus != 0 {
		b.Write([]byte{f.plus})
	}
	if f.left {
		b.Write([]byte{'-'})
	}
	if f.leftDelimiter != 0 && f.leftDelimiter != f.plus {
		b.Write([]byte{f.leftDelimiter})
	}
	if f.width >= 0 {
		b.WriteString(strconv.Itoa(f.width))
	}
	if f.prec >= 0 {
		b.Write([]byte{'.'})
		b.WriteString(strconv.Itoa(f.prec))
	}
	if f.alt {
		b.Write([]byte{'#'})
	}
	b.Write([]byte{f.formatChar})
	return b.String()
}

func hasDelimOnce(flags string, format string, delim byte) bool {
	found := false
	for _, b := range flags {
		if byte(b) == delim {
			if found {
				panic(Error(EVAL_INVALID_STRING_FORMAT_REPEATED_FLAG, issue.H{`format`: format}))
			}
			found = true
		}
	}
	return found
}

func (f *format) HasStringFlags() bool {
	return f.left || f.width >= 0 || f.prec >= 0
}

func (f *format) ApplyStringFlags(b io.Writer, str string, quoted bool) {
	if f.HasStringFlags() {
		bld := bytes.NewBufferString(``)
		if quoted {
			utils.PuppetQuote(bld, str)
			str = bld.String()
			bld.Truncate(0)
		}
		bld.WriteByte('%')
		if f.IsLeft() {
			bld.WriteByte('-')
		}
		if f.Width() >= 0 {
			fmt.Fprintf(bld, `%d`, f.Width())
		}
		if f.Precision() >= 0 {
			fmt.Fprintf(bld, `.%d`, f.Precision())
		}
		bld.WriteByte('s')
		fmt.Fprintf(b, bld.String(), str)
	} else {
		if quoted {
			utils.PuppetQuote(b, str)
		} else {
			io.WriteString(b, str)
		}
	}
}

func (f *format) Width() int {
	return f.width
}

func (f *format) Precision() int {
	return f.prec
}

func (f *format) FormatChar() byte {
	return f.formatChar
}

func (f *format) Plus() byte {
	return f.plus
}

func (f *format) IsAlt() bool {
	return f.alt
}

func (f *format) IsLeft() bool {
	return f.left
}

func (f *format) IsZeroPad() bool {
	return f.zeroPad
}

func (f *format) LeftDelimiter() byte {
	return f.leftDelimiter
}

func (f *format) ContainerFormats() FormatMap {
	return f.containerFormats
}

func (f *format) Separator(dflt string) string {
	if f.separator == NO_STRING {
		return dflt
	}
	return f.separator
}

func (f *format) Separator2(dflt string) string {
	if f.separator2 == NO_STRING {
		return dflt
	}
	return f.separator2
}

func (f *format) OrigFormat() string {
	return f.origFmt
}

func (f *format) ReplaceFormatChar(c byte) Format {
	nf := &format{}
	*nf = *f
	nf.formatChar = c
	nf.origFmt = nf.unParse()
	return nf
}

func (f *format) WithoutWidth() Format {
	nf := &format{}
	*nf = *f
	nf.width = -1
	nf.left = false
	nf.zeroPad = false
	nf.alt = false
	nf.origFmt = nf.unParse()
	return nf
}

type stringReader struct {
	i    int
	text string
}

func (r *stringReader) Next() (rune, bool) {
	if r.i >= len(r.text) {
		return 0, false
	}
	c := rune(r.text[r.i])
	if c < utf8.RuneSelf {
		r.i++
		return c, true
	}
	c, size := utf8.DecodeRuneInString(r.text[r.i:])
	if c == utf8.RuneError {
		panic(`invalid unicode character`)
	}
	r.i += size
	return c, true
}

// Like fmt.Fprintf but using named arguments accessed with %{key} formatting instructions
// and using Puppet StringFormatter for evaluating formatting specifications
func PuppetSprintf(s string, args ...PValue) string {
	buf := bytes.NewBufferString(``)
	fprintf(buf, `sprintf`, s, args...)
	return buf.String()
}

// Like fmt.Fprintf but using named arguments accessed with %{key} formatting instructions
// and using Puppet StringFormatter for evaluating formatting specifications
func PuppetFprintf(buf io.Writer, s string, args ...PValue) {
	fprintf(buf, `fprintf`, s, args...)
}

// Like fmt.Fprintf but using named arguments accessed with %{key} formatting instructions
// and using Puppet StringFormatter for evaluating formatting specifications
func fprintf(buf io.Writer, callerName string, s string, args ...PValue) {
	// Transform the map into a slice of values and a map that maps a key to the position
	// of its value in the slice.
	// Transform all %{key} to %[pos]
	var c rune
	var ok bool
	rdr := &stringReader{0, s}

	consumeAndApplyPattern := func(v PValue) {
		f := bytes.NewBufferString(`%`)
		for ok {
			f.WriteRune(c)
			if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' {
				c, ok = rdr.Next()
				break
			}
			c, ok = rdr.Next()
		}
		ctx, err := NewFormatContext3(v, WrapString(f.String()))
		if err != nil {
			panic(NewIllegalArgument(callerName, 1, err.Error()))
		}
		ToString4(v, ctx, buf)
	}

	var hashArg *HashValue

	pos := 0
	top := len(args)
	c, ok = rdr.Next()
nextChar:
	for ok {
		if c != '%' {
			utils.WriteRune(buf, c)
			c, ok = rdr.Next()
			continue
		}

		c, ok = rdr.Next()
		if c == '%' {
			// %% means % verbatim
			utils.WriteRune(buf, c)
			c, ok = rdr.Next()
			continue
		}

		// Both %<key> and %{key} are allowed
		e := rune(0)
		if c == '{' {
			e = '}'
		} else if c == '<' {
			e = '>'
		}

		if e == 0 {
			// This is a positional argument. It is allowed but there can only be one (for the
			// hash as a whole)
			if hashArg != nil {
				panic(NewArgumentsError(callerName, `keyed and positional format specifications cannot be mixed`))
			}
			if pos >= top {
				panic(NewArgumentsError(callerName, `unbalanced format versus arguments`))
			}
			consumeAndApplyPattern(args[pos])
			pos++
			continue
		}

		if pos > 0 {
			panic(NewArgumentsError(callerName, `keyed and positional format specifications cannot be mixed`))
		}

		if hashArg == nil {
			ok = false
			if top == 1 {
				hashArg, ok = args[0].(*HashValue)
			}
			if !ok {
				panic(NewArgumentsError(callerName, `keyed format specifications requires one hash argument`))
			}
		}

		b := c
		keyStart := rdr.i
		c, ok = rdr.Next()
		for ok {
			if c == e {
				keyEnd := rdr.i - 1 // Safe since '}' is below RuneSelf
				key := s[keyStart:keyEnd]
				if value, keyFound := hashArg.Get(WrapString(key)); keyFound {
					c, ok = rdr.Next()
					if b == '{' {
						ToString4(value, NONE, buf)
					} else {
						consumeAndApplyPattern(value)
					}
					continue nextChar
				}
				panic(NewIllegalArgument(callerName, 1, fmt.Sprintf("key%c%s%c not found", b, key, c)))
			}
			c, ok = rdr.Next()
		}
		panic(NewArgumentsError(callerName, fmt.Sprintf(`unterminated %%c`, b)))
	}
}
