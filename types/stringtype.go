package types

import (
	"bytes"
	"fmt"
	. "io"
	. "math"

	"strings"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/utils"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type (
	StringType struct {
		size  *IntegerType
		value string
	}

	// StringValue represents StringType as a value
	StringValue StringType
)

var stringType_DEFAULT = &StringType{integerType_POSITIVE, ``}
var stringType_NOT_EMPTY = &StringType{NewIntegerType(1, MaxInt64), ``}

func DefaultStringType() *StringType {
	return stringType_DEFAULT
}

func NewStringType(rng *IntegerType, s string) *StringType {
	if s == `` {
		if rng == nil || *rng == *integerType_POSITIVE {
			return DefaultStringType()
		}
		return &StringType{rng, s}
	}
	sz := int64(len(s))
	return &StringType{NewIntegerType(sz, sz), s}
}

func NewStringType2(args ...PValue) *StringType {
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
			rng = NewIntegerType(min, MaxInt64)
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
		panic(NewIllegalArgumentCount(`String[]`, `0 - 2`, len(args)))
	}
	return NewStringType(rng, ``)
}

func (t *StringType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*StringType); ok {
		return t.value == ot.value && t.size.Equals(ot.size, g)
	}
	return false
}

func (t *StringType) IsAssignable(o PType, g Guard) bool {
	if st, ok := o.(*StringType); ok {
		if t.value == `` {
			return t.size.IsAssignable(st.size, g)
		}
		return t.value == st.value
	}

	if et, ok := o.(*EnumType); ok {
		if t.value == `` {
			if *t.size == *integerType_POSITIVE {
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

func (t *StringType) IsInstance(o PValue, g Guard) bool {
	str, ok := o.(*StringValue)
	return ok && t.size.IsInstance3(len(str.String())) && (t.value == `` || t.value == str.String())
}

func (t *StringType) Name() string {
	return `String`
}

func (t *StringType) Parameters() []PValue {
	if t.value != `` || *t.size == *integerType_POSITIVE {
		return EMPTY_VALUES
	}
	return t.size.Parameters()
}

func (t *StringType) String() string {
	return ToString2(t, NONE)
}

func (t *StringType) Size() *IntegerType {
	return t.size
}

func (t *StringType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *StringType) Type() PType {
	return &TypeType{t}
}

func (t *StringType) Value() string {
	return t.value
}

func WrapString(str string) *StringValue {
	return (*StringValue)(NewStringType(nil, str))
}
func (sv *StringValue) Add(v PValue) IndexedValue {
	if ov, ok := v.(*StringValue); ok {
		return WrapString(sv.String() + ov.String())
	}
	panic(fmt.Sprintf(`No auto conversion from %s to String`, v.Type().String()))
}

var ONE_CHAR_STRING_TYPE = NewStringType(NewIntegerType(1, 1), ``)

func (sv *StringValue) AddAll(tv IndexedValue) IndexedValue {
	if a, ok := tv.(*ArrayValue); ok {
		s := bytes.NewBufferString(sv.String())
		for _, e := range a.elements {
			var ev *StringValue
			if ev, ok = e.(*StringValue); ok {
				WriteString(s, ev.String())
				continue
			}
			panic(fmt.Sprintf(`No auto conversion from %s to String`, e.Type().String()))
		}
		return WrapString(s.String())
	}
	panic(`Operation not supported`)
}

func (sv *StringValue) At(i int) PValue {
	if i >= 0 && i < len(sv.String()) {
		return WrapString(sv.String()[i : i+1])
	}
	return _UNDEF
}

func (sv *StringValue) Elements() []PValue {
	str := sv.String()
	top := len(str)
	el := make([]PValue, top)
	for idx, c := range str {
		el[idx] = WrapString(string(c))
	}
	return el
}

func (sv *StringValue) ElementType() PType {
	return ONE_CHAR_STRING_TYPE
}

func (sv *StringValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*StringValue); ok {
		return sv.String() == ov.String()
	}
	return false
}

func (sv *StringValue) ToString(b Writer, s FormatContext, g RDetect) {
	f := GetFormat(s.FormatMap(), sv.Type())
	val := sv.value
	switch f.FormatChar() {
	case 's':
		fmt.Fprintf(b, f.OrigFormat(), val)
	case 'p':
		f.ApplyStringFlags(b, val, true)
	case 'c':
		val = CapitalizeSegment(val)
		f.ReplaceFormatChar('s').ApplyStringFlags(b, val, f.IsAlt())
	case 'C':
		val = CapitalizeSegments(val)
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
		panic(s.UnsupportedFormat(sv.Type(), `cCudspt`, f))
	}
}

func (sv *StringValue) Delete(v PValue) IndexedValue {
	panic(`Operation not supported`)
}

func (sv *StringValue) DeleteAll(tv IndexedValue) IndexedValue {
	panic(`Operation not supported`)
}

func (sv *StringValue) Iterator() Iterator {
	return &indexedIterator{ONE_CHAR_STRING_TYPE, -1, sv}
}

func (sv *StringValue) Len() int {
	return int((*StringType)(sv).Size().Min())
}

func (sv *StringValue) Slice(i int, j int) *StringValue {
	return WrapString(sv.String()[i:j])
}

func (sv *StringValue) String() string {
	return (*StringType)(sv).Value()
}

func (sv *StringValue) ToKey() HashKey {
	return HashKey(sv.String())
}

func (sv *StringValue) Type() PType {
	return (*StringType)(sv)
}
