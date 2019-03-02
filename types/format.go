package types

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/utils"
)

type (
	format struct {
		alt              bool
		left             bool
		zeroPad          bool
		formatChar       byte
		plus             byte
		precision        int
		width            int
		leftDelimiter    byte
		separator        string
		separator2       string
		origFmt          string
		containerFormats eval.FormatMap
	}

	formatContext struct {
		indentation eval.Indentation
		formatMap   eval.FormatMap
		properties  map[string]string
	}

	indentation struct {
		first     bool
		indenting bool
		level     int
		padding   string
	}
)

func (f *format) Equals(other interface{}, guard eval.Guard) bool {
	if of, ok := other.(*format); ok {
		return f.alt == of.alt &&
			f.left == of.left &&
			f.zeroPad == of.zeroPad &&
			f.formatChar == of.formatChar &&
			f.plus == of.plus &&
			f.precision == of.precision &&
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

func (f *format) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	utils.WriteString(bld, f.origFmt)
}

func (f *format) PType() eval.Type {
	return WrapRuntime(f).PType()
}

var DefaultProgramFormat = simpleFormat('p')

var DefaultAnyFormat = simpleFormat('s')

var PrettyProgramFormat = newFormat(`%#p`)
var PrettyContainerFormats = eval.FormatMap(WrapHash([]*HashEntry{WrapHashEntry(DefaultAnyType(), PrettyProgramFormat)}))
var PrettyArrayFormat = basicAltFormat('a', `,`, '[', PrettyContainerFormats)
var PrettyHashFormat = basicAltFormat('h', ` => `, '{', PrettyContainerFormats)
var PrettyObjectFormat = basicAltFormat('p', ` => `, '(', PrettyContainerFormats)

var PrettyIndentation = newIndentation(true, 0)

func init() {
	DefaultArrayFormat.(*format).containerFormats = DefaultContainerFormats
	DefaultHashFormat.(*format).containerFormats = DefaultContainerFormats
	DefaultObjectFormat.(*format).containerFormats = DefaultContainerFormats
	DefaultArrayContainerFormat.(*format).containerFormats = DefaultContainerFormats
	DefaultHashContainerFormat.(*format).containerFormats = DefaultContainerFormats
	DefaultObjectContainerFormat.(*format).containerFormats = DefaultContainerFormats

	eval.DefaultFormat = DefaultAnyFormat
	eval.DefaultFormatContext = None
	eval.Pretty = newFormatContext2(PrettyIndentation, eval.FormatMap(WrapHash([]*HashEntry{
		WrapHashEntry(DefaultObjectType(), PrettyObjectFormat),
		WrapHashEntry(DefaultTypeType(), PrettyObjectFormat),
		WrapHashEntry(DefaultFloatType(), simpleFormat('f')),
		WrapHashEntry(DefaultNumericType(), simpleFormat('d')),
		WrapHashEntry(DefaultStringType(), PrettyProgramFormat),
		WrapHashEntry(DefaultUriType(), PrettyProgramFormat),
		WrapHashEntry(DefaultSemVerType(), PrettyProgramFormat),
		WrapHashEntry(DefaultSemVerRangeType(), PrettyProgramFormat),
		WrapHashEntry(DefaultTimestampType(), PrettyProgramFormat),
		WrapHashEntry(DefaultTimespanType(), PrettyProgramFormat),
		WrapHashEntry(DefaultArrayType(), PrettyArrayFormat),
		WrapHashEntry(DefaultHashType(), PrettyHashFormat),
		WrapHashEntry(DefaultBinaryType(), simpleFormat('B')),
		WrapHashEntry(DefaultAnyType(), DefaultAnyFormat),
	})), map[string]string{})

	eval.NewFormatContext = newFormatContext
	eval.NewFormatContext2 = newFormatContext2
	eval.NewFormatContext3 = newFormatContext3
	eval.NewIndentation = newIndentation
	eval.NewFormat = newFormat

	eval.PrettyExpanded = eval.Pretty.WithProperties(map[string]string{`expanded`: `true`})
}

var DefaultArrayFormat = basicFormat('a', `,`, '[', nil)
var DefaultHashFormat = basicFormat('h', ` => `, '{', nil)
var DefaultObjectFormat = basicFormat('p', ` => `, '(', nil)

var DefaultArrayContainerFormat = basicFormat('p', `,`, '[', nil)
var DefaultHashContainerFormat = basicFormat('p', ` => `, '{', nil)
var DefaultObjectContainerFormat = basicFormat('p', ` => `, '(', nil)

var DefaultIndentation = newIndentation(false, 0)

var DefaultFormats = eval.FormatMap(WrapHash([]*HashEntry{
	WrapHashEntry(DefaultObjectType(), DefaultObjectFormat),
	WrapHashEntry(DefaultTypeType(), DefaultObjectFormat),
	WrapHashEntry(DefaultFloatType(), simpleFormat('f')),
	WrapHashEntry(DefaultNumericType(), simpleFormat('d')),
	WrapHashEntry(DefaultArrayType(), DefaultArrayFormat),
	WrapHashEntry(DefaultHashType(), DefaultHashFormat),
	WrapHashEntry(DefaultBinaryType(), simpleFormat('B')),
	WrapHashEntry(DefaultAnyType(), DefaultAnyFormat),
}))

var DefaultContainerFormats = eval.FormatMap(WrapHash([]*HashEntry{
	WrapHashEntry(DefaultObjectType(), DefaultObjectContainerFormat),
	WrapHashEntry(DefaultTypeType(), DefaultObjectContainerFormat),
	WrapHashEntry(DefaultFloatType(), DefaultProgramFormat),
	WrapHashEntry(DefaultNumericType(), DefaultProgramFormat),
	WrapHashEntry(DefaultArrayType(), DefaultArrayContainerFormat),
	WrapHashEntry(DefaultHashType(), DefaultHashContainerFormat),
	WrapHashEntry(DefaultBinaryType(), DefaultProgramFormat),
	WrapHashEntry(DefaultAnyType(), DefaultProgramFormat),
}))

var delimiters = []byte{'[', '{', '(', '<', '|'}
var delimiterPairs = map[byte][2]byte{
	'[': {'[', ']'},
	'{': {'{', '}'},
	'(': {'(', ')'},
	'<': {'<', '>'},
	'|': {'|', '|'},
	' ': {0, 0},
	0:   {'[', ']'},
}

var None = newFormatContext2(DefaultIndentation, DefaultFormats, nil)

var Expanded = newFormatContext2(DefaultIndentation, DefaultFormats, map[string]string{`expanded`: `true`})

var Program = newFormatContext2(DefaultIndentation, eval.FormatMap(SingletonHash(DefaultAnyType(), DefaultObjectFormat)), nil)

func newFormatContext(t eval.Type, format eval.Format, indentation eval.Indentation) eval.FormatContext {
	return &formatContext{indentation, WrapHash([]*HashEntry{WrapHashEntry(t, format)}), nil}
}

func newFormatContext2(indentation eval.Indentation, formatMap eval.FormatMap, properties map[string]string) eval.FormatContext {
	return &formatContext{indentation, formatMap, properties}
}

var typeStringFormat = NewVariantType(DefaultStringType(), DefaultDefaultType(), DefaultHashType())

func newFormatContext3(value eval.Value, format eval.Value) (context eval.FormatContext, err error) {
	eval.AssertInstance(`String format`, typeStringFormat, format)

	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(issue.Reported); !ok {
				panic(r)
			}
		}
	}()

	switch format.(type) {
	case stringValue:
		context = eval.NewFormatContext(value.PType(), newFormat(format.String()), DefaultIndentation)
	case *DefaultValue:
		context = eval.DefaultFormatContext
	default:
		context = newFormatContext2(DefaultIndentation, mergeFormats(DefaultFormats, NewFormatMap(format.(*HashValue))), nil)
	}
	return
}

func mergeFormats(lower eval.FormatMap, higher eval.FormatMap) eval.FormatMap {
	if lower == nil || lower.Len() == 0 {
		return higher
	}
	if higher == nil || higher.Len() == 0 {
		return lower
	}

	higherKeys := higher.Keys()
	normLower := WrapHash2(eval.Reject2(lower.Entries(), func(lev eval.Value) bool {
		le := lev.(*HashEntry)
		return eval.Any2(higherKeys, func(hk eval.Value) bool {
			return !hk.Equals(le.Key(), nil) && eval.IsAssignable(hk.(eval.Type), le.Key().(eval.Type))
		})
	}))

	merged := make([]*HashEntry, 0, 8)
	normLower.Keys().AddAll(higherKeys).Unique().Each(func(k eval.Value) {
		if low, ok := normLower.Get(k); ok {
			if high, ok := higher.Get(k); ok {
				merged = append(merged, WrapHashEntry(k, merge(low.(eval.Format), high.(eval.Format))))
			} else {
				merged = append(merged, WrapHashEntry(k, low))
			}
		} else {
			if high, ok := higher.Get(k); ok {
				merged = append(merged, WrapHashEntry(k, high))
			}
		}
	})

	sort.Slice(merged, func(ax, bx int) bool {
		a := merged[ax].Key().(eval.Type)
		b := merged[bx].Key().(eval.Type)
		if a.Equals(b, nil) {
			return false
		}
		ab := eval.IsAssignable(b, a)
		ba := eval.IsAssignable(a, b)
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
	return eval.FormatMap(WrapHash(merged))
}

func merge(low eval.Format, high eval.Format) eval.Format {
	sep := high.Separator(NoString)
	if sep == NoString {
		sep = low.Separator(NoString)
	}
	sep2 := high.Separator2(NoString)
	if sep2 == NoString {
		sep2 = low.Separator2(NoString)
	}

	return &format{
		origFmt:          high.OrigFormat(),
		alt:              high.IsAlt(),
		leftDelimiter:    high.LeftDelimiter(),
		formatChar:       high.FormatChar(),
		zeroPad:          high.IsZeroPad(),
		precision:        high.Precision(),
		left:             high.IsLeft(),
		plus:             high.Plus(),
		width:            high.Width(),
		separator2:       sep2,
		separator:        sep,
		containerFormats: mergeFormats(low.ContainerFormats(), high.ContainerFormats()),
	}
}

func typeRank(pt eval.Type) int {
	switch pt.(type) {
	case *NumericType, *IntegerType, *FloatType:
		return 13
	case *stringType, *vcStringType, *scStringType:
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

var typeStringFormatTypeHash = NewHashType(DefaultTypeType(), NewVariantType(DefaultStringType(), DefaultHashType()), nil)

func NewFormatMap(h *HashValue) eval.FormatMap {
	eval.AssertInstance(`String format type hash`, typeStringFormatTypeHash, h)
	result := make([]*HashEntry, h.Len())
	h.EachWithIndex(func(elem eval.Value, idx int) {
		entry := elem.(*HashEntry)
		pt := entry.Key().(eval.Type)
		v := entry.Value()
		if s, ok := v.(stringValue); ok {
			result[idx] = WrapHashEntry(pt, newFormat(s.String()))
		} else {
			result[idx] = WrapHashEntry(pt, FormatFromHash(v.(*HashValue)))
		}
	})
	return eval.FormatMap(WrapHash(result))
}

var typeStringFormatHash = NewStructType([]*StructElement{
	newStructElement2(`format`, DefaultStringType()),
	NewStructElement(newOptionalType3(`separator`), DefaultStringType()),
	NewStructElement(newOptionalType3(`separator2`), DefaultStringType()),
	NewStructElement(newOptionalType3(`string_formats`), DefaultHashType()),
})

func FormatFromHash(h *HashValue) eval.Format {
	eval.AssertInstance(`String format hash`, typeStringFormatHash, h)

	stringArg := func(key string, required bool) string {
		v := h.Get5(key, undef)
		switch v.(type) {
		case stringValue:
			return v.String()
		default:
			return NoString
		}
	}

	var cf eval.FormatMap
	if v := h.Get5(`string_formats`, undef); !eval.Equals(v, undef) {
		cf = NewFormatMap(v.(*HashValue))
	}
	return parseFormat(stringArg(`format`, true), stringArg(`separator`, false), stringArg(`separator2`, false), cf)
}

func (c *formatContext) Indentation() eval.Indentation {
	return c.indentation
}

func (c *formatContext) FormatMap() eval.FormatMap {
	return c.formatMap
}

func (c *formatContext) Property(key string) (string, bool) {
	if c.properties != nil {
		pv, ok := c.properties[key]
		return pv, ok
	}
	return ``, false
}

func (c *formatContext) Properties() map[string]string {
	return c.properties
}

func (c *formatContext) SetProperty(key, value string) {
	if c.properties == nil {
		c.properties = map[string]string{key: value}
	} else {
		c.properties[key] = value
	}
}

func (c *formatContext) Subsequent() eval.FormatContext {
	si := c.Indentation()
	if si.Breaks() {
		// Never break between the type and the start array marker
		return newFormatContext2(newIndentation(si.IsIndenting(), si.Level()), c.FormatMap(), c.Properties())
	}
	return c
}

func (c *formatContext) UnsupportedFormat(t eval.Type, supportedFormats string, actualFormat eval.Format) error {
	return eval.Error(eval.UnsupportedStringFormat, issue.H{`format`: actualFormat.FormatChar(), `type`: t.Name(), `supported_formats`: supportedFormats})
}

func (c *formatContext) WithProperties(properties map[string]string) eval.FormatContext {
	if c.properties != nil {
		merged := make(map[string]string, len(c.properties)+len(properties))
		for k, v := range c.properties {
			merged[k] = v
		}
		for k, v := range properties {
			merged[k] = v
		}
		properties = merged
	}
	return newFormatContext2(c.indentation, c.formatMap, properties)
}

func newIndentation(indenting bool, level int) eval.Indentation {
	return newIndentation2(true, indenting, level)
}

func newIndentation2(first bool, indenting bool, level int) eval.Indentation {
	return &indentation{first, indenting, level, strings.Repeat(`  `, level)}
}

func (i *indentation) Breaks() bool {
	return i.indenting && i.level > 0 && !i.first
}

func (i *indentation) Level() int {
	return i.level
}

func (i *indentation) Increase(indenting bool) eval.Indentation {
	return newIndentation2(true, indenting, i.level+1)
}

func (i *indentation) Indenting(indenting bool) eval.Indentation {
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

func (i *indentation) Subsequent() eval.Indentation {
	if i.first {
		return &indentation{false, i.indenting, i.level, i.padding}
	}
	return i
}

// NewFormat parses a format string into a Format
func newFormat(format string) eval.Format {
	return parseFormat(format, NoString, NoString, nil)
}

func simpleFormat(formatChar byte) eval.Format {
	return basicFormat(formatChar, NoString, '[', nil)
}

func basicFormat(formatChar byte, sep2 string, leftDelimiter byte, containerFormats eval.FormatMap) eval.Format {
	return &format{
		formatChar:       formatChar,
		precision:        -1,
		width:            -1,
		origFmt:          `%` + string(formatChar),
		separator:        `,`,
		separator2:       sep2,
		leftDelimiter:    leftDelimiter,
		containerFormats: containerFormats,
	}
}

func basicAltFormat(formatChar byte, sep2 string, leftDelimiter byte, containerFormats eval.FormatMap) eval.Format {
	return &format{
		formatChar:       formatChar,
		alt:              true,
		precision:        -1,
		width:            -1,
		origFmt:          `%` + string(formatChar),
		separator:        `,`,
		separator2:       sep2,
		leftDelimiter:    leftDelimiter,
		containerFormats: containerFormats,
	}
}

func parseFormat(origFmt string, separator string, separator2 string, containerFormats eval.FormatMap) eval.Format {
	group := eval.FormatPattern.FindStringSubmatch(origFmt)
	if group == nil {
		panic(eval.Error(eval.InvalidStringFormatSpec, issue.H{`format`: origFmt}))
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
				panic(eval.Error(eval.InvalidStringFormatDelimiter, issue.H{`delimiter`: foundDelim}))
			}
			foundDelim = delim
		}
	}

	if foundDelim == 0 && plus == ' ' {
		foundDelim = plus
	}

	width := -1
	prc := -1
	if tmp := group[2]; tmp != `` {
		width, _ = strconv.Atoi(tmp)
	}
	if tmp := group[3]; tmp != `` {
		prc, _ = strconv.Atoi(tmp)
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
		precision:        prc,
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
	if f.precision >= 0 {
		b.Write([]byte{'.'})
		b.WriteString(strconv.Itoa(f.precision))
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
				panic(eval.Error(eval.InvalidStringFormatRepeatedFlag, issue.H{`format`: format}))
			}
			found = true
		}
	}
	return found
}

func (f *format) HasStringFlags() bool {
	return f.left || f.width >= 0 || f.precision >= 0
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
			utils.WriteString(bld, strconv.Itoa(f.Width()))
		}
		if f.Precision() >= 0 {
			utils.WriteByte(bld, '.')
			utils.WriteString(bld, strconv.Itoa(f.Precision()))
		}
		bld.WriteByte('s')
		utils.Fprintf(b, bld.String(), str)
	} else {
		if quoted {
			utils.PuppetQuote(b, str)
		} else {
			utils.WriteString(b, str)
		}
	}
}

func (f *format) Width() int {
	return f.width
}

func (f *format) Precision() int {
	return f.precision
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

func (f *format) ContainerFormats() eval.FormatMap {
	return f.containerFormats
}

func (f *format) Separator(dflt string) string {
	if f.separator == NoString {
		return dflt
	}
	return f.separator
}

func (f *format) Separator2(dflt string) string {
	if f.separator2 == NoString {
		return dflt
	}
	return f.separator2
}

func (f *format) OrigFormat() string {
	return f.origFmt
}

func (f *format) ReplaceFormatChar(c byte) eval.Format {
	nf := &format{}
	*nf = *f
	nf.formatChar = c
	nf.origFmt = nf.unParse()
	return nf
}

func (f *format) WithoutWidth() eval.Format {
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

// PuppetSprintf is like fmt.Fprintf but using named arguments accessed with %{key} formatting instructions
// and using Puppet StringFormatter for evaluating formatting specifications
func PuppetSprintf(s string, args ...eval.Value) string {
	buf := bytes.NewBufferString(``)
	fprintf(buf, `sprintf`, s, args...)
	return buf.String()
}

// PuppetFprintf is like fmt.Fprintf but using named arguments accessed with %{key} formatting instructions
// and using Puppet StringFormatter for evaluating formatting specifications
func PuppetFprintf(buf io.Writer, s string, args ...eval.Value) {
	fprintf(buf, `fprintf`, s, args...)
}

func fprintf(buf io.Writer, callerName string, s string, args ...eval.Value) {
	// Transform the map into a slice of values and a map that maps a key to the position
	// of its value in the slice.
	// Transform all %{key} to %[pos]
	var c rune
	var ok bool
	rdr := &stringReader{0, s}

	consumeAndApplyPattern := func(v eval.Value) {
		f := bytes.NewBufferString(`%`)
		for ok {
			f.WriteRune(c)
			if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' {
				c, ok = rdr.Next()
				break
			}
			c, ok = rdr.Next()
		}
		ctx, err := eval.NewFormatContext3(v, stringValue(f.String()))
		if err != nil {
			panic(errors.NewIllegalArgument(callerName, 1, err.Error()))
		}
		eval.ToString4(v, ctx, buf)
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
				panic(errors.NewArgumentsError(callerName, `keyed and positional format specifications cannot be mixed`))
			}
			if pos >= top {
				panic(errors.NewArgumentsError(callerName, `unbalanced format versus arguments`))
			}
			consumeAndApplyPattern(args[pos])
			pos++
			continue
		}

		if pos > 0 {
			panic(errors.NewArgumentsError(callerName, `keyed and positional format specifications cannot be mixed`))
		}

		if hashArg == nil {
			if top == 1 {
				hashArg, _ = args[0].(*HashValue)
			}
			if hashArg == nil {
				panic(errors.NewArgumentsError(callerName, `keyed format specifications requires one hash argument`))
			}
		}

		b := c
		keyStart := rdr.i
		c, ok = rdr.Next()
		for ok {
			if c == e {
				keyEnd := rdr.i - 1 // Safe since '}' is below RuneSelf
				key := s[keyStart:keyEnd]
				if value, keyFound := hashArg.Get(stringValue(key)); keyFound {
					c, ok = rdr.Next()
					if b == '{' {
						eval.ToString4(value, None, buf)
					} else {
						consumeAndApplyPattern(value)
					}
					continue nextChar
				}
				panic(errors.NewIllegalArgument(callerName, 1, fmt.Sprintf("key%c%s%c not found", b, key, c)))
			}
			c, ok = rdr.Next()
		}
		panic(errors.NewArgumentsError(callerName, fmt.Sprintf(`unterminated %%%c`, b)))
	}
}
