package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/utils"
	"reflect"
	"regexp"
)

type (
	StringType struct {
		size  *IntegerType
		value string
	}

	// StringValue represents StringType as a value
	StringValue StringType
)

var stringType_DEFAULT = &StringType{IntegerType_POSITIVE, ``}
var stringType_NOT_EMPTY = &StringType{NewIntegerType(1, math.MaxInt64), ``}

var String_Type eval.ObjectType

func init() {
	String_Type = newObjectType(`Pcore::StringType`,
		`Pcore::ScalarDataType {
	attributes => {
		size_type_or_value => {
			type => Variant[Undef,String,Type[Integer]],
			value => undef
		},
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewStringType2(args...)
		})

	newGoConstructor2(`String`,
		func(t eval.LocalTypes) {
			t.Type2(`Format`, NewPatternType([]*RegexpType{NewRegexpTypeR(eval.FORMAT_PATTERN)}))
			t.Type(`ContainerFormat`, `Struct[{
          Optional[format]         => Format,
          Optional[separator]      => String,
          Optional[separator2]     => String,
          Optional[string_formats] => Hash[Type, Format]
        }]`)
			t.Type(`TypeMap`, `Hash[Type, Variant[Format, ContainerFormat]]`)
			t.Type(`Formats`, `Variant[Default, String[1], TypeMap]`)
		},

		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.OptionalParam(`Formats`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				fmt := NONE
				if len(args) > 1 {
					var err error
					fmt, err = eval.NewFormatContext3(args[0], args[1])
					if err != nil {
						panic(errors.NewIllegalArgument(`String`, 1, err.Error()))
					}
				}

				// Convert errors on first argument to argument errors
				defer func() {
					if r := recover(); r != nil {
						if ge, ok := r.(errors.GenericError); ok {
							panic(errors.NewIllegalArgument(`String`, 0, ge.Error()))
						}
						panic(r)
					}
				}()
				return WrapString(eval.ToString2(args[0], fmt))
			})
		},
	)
}

func DefaultStringType() *StringType {
	return stringType_DEFAULT
}

func NewStringType(rng *IntegerType, s string) *StringType {
	if s == `` {
		if rng == nil || *rng == *IntegerType_POSITIVE {
			return DefaultStringType()
		}
		return &StringType{rng, s}
	}
	sz := int64(len(s))
	return &StringType{NewIntegerType(sz, sz), s}
}

func NewStringType2(args ...eval.Value) *StringType {
	var rng *IntegerType
	var ok bool
	switch len(args) {
	case 0:
		return DefaultStringType()
	case 1:
		var value *StringValue
		if value, ok = args[0].(*StringValue); ok {
			return NewStringType(nil, value.String())
		}
		rng, ok = args[0].(*IntegerType)
		if !ok {
			var min int64
			min, ok = toInt(args[0])
			if !ok {
				panic(NewIllegalArgumentType2(`String[]`, 0, `String, Integer or Type[Integer]`, args[0]))
			}
			rng = NewIntegerType(min, math.MaxInt64)
		}
	case 2:
		var min, max int64
		min, ok = toInt(args[0])
		if !ok {
			panic(NewIllegalArgumentType2(`String[]`, 0, `Integer`, args[0]))
		}
		max, ok = toInt(args[1])
		if !ok {
			panic(NewIllegalArgumentType2(`String[]`, 1, `Integer`, args[1]))
		}
		rng = NewIntegerType(min, max)
	default:
		panic(errors.NewIllegalArgumentCount(`String[]`, `0 - 2`, len(args)))
	}
	return NewStringType(rng, ``)
}

func (t *StringType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *StringType) Default() eval.Type {
	return stringType_DEFAULT
}

func (t *StringType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*StringType); ok {
		return t.value == ot.value && t.size.Equals(ot.size, g)
	}
	return false
}

func (t *StringType) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `size_type_or_value`:
		if t.value == `` {
			return t.size, true
		}
		return WrapString(t.value), true
	}
	return nil, false
}

func (t *StringType) IsAssignable(o eval.Type, g eval.Guard) bool {
	if st, ok := o.(*StringType); ok {
		if t.value == `` {
			return t.size.IsAssignable(st.size, g)
		}
		return t.value == st.value
	}

	if et, ok := o.(*EnumType); ok {
		if t.value == `` {
			if *t.size == *IntegerType_POSITIVE {
				return true
			}
			for _, str := range et.values {
				if !t.size.IsInstance3(len(str)) {
					return false
				}
			}
			return true
		}
	}

	if _, ok := o.(*PatternType); ok {
		// Pattern is only assignable to the default string
		return *t == *stringType_DEFAULT
	}
	return false
}

func (t *StringType) IsInstance(o eval.Value, g eval.Guard) bool {
	str, ok := o.(*StringValue)
	return ok && t.size.IsInstance3(len(str.String())) && (t.value == `` || t.value == str.String())
}

func (t *StringType) MetaType() eval.ObjectType {
	return String_Type
}

func (t *StringType) Name() string {
	return `String`
}

func (t *StringType) Parameters() []eval.Value {
	if t.value != `` || *t.size == *IntegerType_POSITIVE {
		return eval.EMPTY_VALUES
	}
	return t.size.Parameters()
}

func (t *StringType) ReflectType(c eval.Context) (reflect.Type, bool) {
	return reflect.TypeOf(`x`), true
}

func (t *StringType)  CanSerializeAsString() bool {
  return true
}

func (t *StringType)  SerializationString() string {
	return t.String()
}


func (t *StringType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *StringType) Size() *IntegerType {
	return t.size
}

func (t *StringType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *StringType) PType() eval.Type {
	return &TypeType{t}
}

func (t *StringType) Value() string {
	return t.value
}

func WrapString(str string) *StringValue {
	return (*StringValue)(NewStringType(nil, str))
}
func (sv *StringValue) Add(v eval.Value) eval.List {
	if ov, ok := v.(*StringValue); ok {
		return WrapString(sv.String() + ov.String())
	}
	panic(fmt.Sprintf(`No auto conversion from %s to String`, v.PType().String()))
}

var ONE_CHAR_STRING_TYPE = NewStringType(NewIntegerType(1, 1), ``)

func (sv *StringValue) AddAll(tv eval.List) eval.List {
	s := bytes.NewBufferString(sv.String())
	tv.Each(func(e eval.Value) {
		ev, ok := e.(*StringValue)
		if !ok {
			panic(fmt.Sprintf(`No auto conversion from %s to String`, e.PType().String()))
		}
		io.WriteString(s, ev.String())
	})
	return WrapString(s.String())
}

func (sv *StringValue) All(predicate eval.Predicate) bool {
	for _, c := range sv.String() {
		if !predicate(WrapString(string(c))) {
			return false
		}
	}
	return true
}

func (sv *StringValue) Any(predicate eval.Predicate) bool {
	for _, c := range sv.String() {
		if predicate(WrapString(string(c))) {
			return true
		}
	}
	return false
}

func (sv *StringValue) AppendTo(slice []eval.Value) []eval.Value {
	for _, c := range sv.String() {
		slice = append(slice, WrapString(string(c)))
	}
	return slice
}

func (sv *StringValue) At(i int) eval.Value {
	if i >= 0 && i < len(sv.String()) {
		return WrapString(sv.String()[i : i+1])
	}
	return _UNDEF
}

func (sv *StringValue) Delete(v eval.Value) eval.List {
	panic(`Operation not supported`)
}

func (sv *StringValue) DeleteAll(tv eval.List) eval.List {
	panic(`Operation not supported`)
}

func (sv *StringValue) Elements() []eval.Value {
	str := sv.String()
	top := len(str)
	el := make([]eval.Value, top)
	for idx, c := range str {
		el[idx] = WrapString(string(c))
	}
	return el
}

func (sv *StringValue) Each(consumer eval.Consumer) {
	for _, c := range sv.String() {
		consumer(WrapString(string(c)))
	}
}

func (sv *StringValue) EachSlice(n int, consumer eval.SliceConsumer) {
	s := sv.String()
	top := len(s)
	for i := 0; i < top; i += n {
		e := i + n
		if e > top {
			e = top
		}
		consumer(WrapString(s[i:e]))
	}
}

func (sv *StringValue) EachWithIndex(consumer eval.IndexedConsumer) {
	for i, c := range sv.String() {
		consumer(WrapString(string(c)), i)
	}
}

func (sv *StringValue) ElementType() eval.Type {
	return ONE_CHAR_STRING_TYPE
}

func (sv *StringValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*StringValue); ok {
		return sv.String() == ov.String()
	}
	return false
}

func (sv *StringValue) Find(predicate eval.Predicate) (eval.Value, bool) {
	for _, c := range sv.String() {
		e := WrapString(string(c))
		if predicate(e) {
			return e, true
		}
	}
	return nil, false
}

func (sv *StringValue) Flatten() eval.List {
	return sv
}

func (sv *StringValue) IsEmpty() bool {
	return sv.Len() == 0
}

func (sv *StringValue) IsHashStyle() bool {
	return false
}

func (sv *StringValue) Iterator() eval.Iterator {
	return &indexedIterator{ONE_CHAR_STRING_TYPE, -1, sv}
}

func (sv *StringValue) Len() int {
	return int((*StringType)(sv).Size().Min())
}

func (sv *StringValue) Map(mapper eval.Mapper) eval.List {
	s := sv.String()
	mapped := make([]eval.Value, len(s))
	for i, c := range s {
		mapped[i] = mapper(WrapString(string(c)))
	}
	return WrapValues(mapped)
}

func (sv *StringValue) Reduce(redactor eval.BiMapper) eval.Value {
	s := sv.String()
	if len(s) == 0 {
		return _UNDEF
	}
	return reduceString(s[1:], sv.At(0), redactor)
}

func (sv *StringValue) Reduce2(initialValue eval.Value, redactor eval.BiMapper) eval.Value {
	return reduceString(sv.String(), initialValue, redactor)
}

func (sv *StringValue) Reflect(c eval.Context) reflect.Value {
	return reflect.ValueOf(sv.String())
}

func (sv *StringValue) ReflectTo(c eval.Context, value reflect.Value) {
	if value.Kind() == reflect.Interface {
		value.Set(sv.Reflect(c))
	} else {
		value.SetString(sv.String())
	}
}

func (sv *StringValue) Reject(predicate eval.Predicate) eval.List {
	selected := bytes.NewBufferString(``)
	for _, c := range sv.String() {
		if !predicate(WrapString(string(c))) {
			selected.WriteRune(c)
		}
	}
	return WrapString(selected.String())
}

func (sv *StringValue) Select(predicate eval.Predicate) eval.List {
	selected := bytes.NewBufferString(``)
	for _, c := range sv.String() {
		if predicate(WrapString(string(c))) {
			selected.WriteRune(c)
		}
	}
	return WrapString(selected.String())
}

func (sv *StringValue) Slice(i int, j int) eval.List {
	return WrapString(sv.String()[i:j])
}

func (sv *StringValue) Split(pattern *regexp.Regexp) *ArrayValue {
	strings := pattern.Split(sv.String(), -1)
	result := make([]eval.Value, len(strings))
	for i, s := range strings {
		result[i] = WrapString(s)
	}
	return WrapValues(result)
}

func (sv *StringValue) String() string {
	return (*StringType)(sv).Value()
}

func (sv *StringValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), sv.PType())
	val := sv.value
	switch f.FormatChar() {
	case 's':
		fmt.Fprintf(b, f.OrigFormat(), val)
	case 'p':
		f.ApplyStringFlags(b, val, true)
	case 'c':
		val = utils.CapitalizeSegment(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	case 'C':
		val = utils.CapitalizeSegments(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	case 'u':
		val = strings.ToUpper(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	case 'd':
		val = strings.ToLower(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	case 't':
		val = strings.TrimSpace(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	default:
		panic(s.UnsupportedFormat(sv.PType(), `cCudspt`, f))
	}
}

func (sv *StringValue) ToKey() eval.HashKey {
	return eval.HashKey(sv.String())
}

func (sv *StringValue) PType() eval.Type {
	return (*StringType)(sv)
}

func (sv *StringValue) Unique() eval.List {
	s := sv.String()
	top := len(s)
	if top < 2 {
		return sv
	}

	result := bytes.NewBufferString(``)
	exists := make(map[rune]bool, top)
	for _, c := range s {
		if !exists[c] {
			exists[c] = true
			result.WriteRune(c)
		}
	}
	if result.Len() == len(s) {
		return sv
	}
	return WrapString(result.String())
}

func reduceString(slice string, initialValue eval.Value, redactor eval.BiMapper) eval.Value {
	memo := initialValue
	for _, v := range slice {
		memo = redactor(memo, WrapString(string(v)))
	}
	return memo
}
