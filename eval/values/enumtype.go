package values

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/eval/utils"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
)

type EnumType struct {
	values []string
}

func DefaultEnumType() *EnumType {
	return enumType_DEFAULT
}

func NewEnumType(enums []string) *EnumType {
	return &EnumType{enums}
}

func NewEnumType2(args ...PValue) *EnumType {
	if len(args) == 0 {
		return DefaultEnumType()
	}
	var enums []string
	top := len(args)
	if top == 1 {
		first := args[0]
		switch first.(type) {
		case IndexedValue:
			return NewEnumType2(first.(IndexedValue).Elements()...)
		case *StringValue:
			enums = []string{first.String()}
		default:
			panic(NewIllegalArgumentType2(`Enum[]`, 0, `String or Array[String]`, args[0]))
		}
	} else {
		enums = make([]string, top)
		for idx, arg := range args {
			str, ok := arg.(*StringValue)
			if !ok {
				panic(NewIllegalArgumentType2(`Enum[]`, idx, `String`, arg))
			}
			enums[idx] = str.String()
		}
	}
	return NewEnumType(enums)
}

func (t *EnumType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*EnumType); ok {
		return len(t.values) == len(ot.values) && ContainsAllStrings(t.values, ot.values)
	}
	return false
}

func (t *EnumType) Generic() PType {
	return enumType_DEFAULT
}

func (t *EnumType) IsAssignable(o PType, g Guard) bool {
	if len(t.values) == 0 {
		switch o.(type) {
		case *StringType, *EnumType, *PatternType:
			return true
		}
		return false
	}

	if st, ok := o.(*StringType); ok {
		return ContainsString(t.values, st.value)
	}

	if en, ok := o.(*EnumType); ok {
		oEnums := en.values
		return len(oEnums) > 0 && ContainsAllStrings(t.values, oEnums)
	}
	return false
}

func (t *EnumType) IsInstance(o PValue, g Guard) bool {
	str, ok := o.(*StringValue)
	return ok && (len(t.values) == 0 || ContainsString(t.values, str.String()))
}

func (t *EnumType) Name() string {
	return `Enum`
}

func (t *EnumType) String() string {
	return ToString2(t, NONE)
}

func (t *EnumType) ToString(bld Writer, format FormatContext, g RDetect) {
	WriteString(bld, `Enum`)
	top := len(t.values)
	if top > 0 {
		WriteByte(bld, '[')
		PuppetQuote(bld, t.values[0])
		for idx := 1; idx < top; idx++ {
			WriteByte(bld, ',')
			PuppetQuote(bld, t.values[idx])
		}
		WriteByte(bld, ']')
	}
}

func (t *EnumType) Type() PType {
	return &TypeType{t}
}

var enumType_DEFAULT = &EnumType{[]string{}}
