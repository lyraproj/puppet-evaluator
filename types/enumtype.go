package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
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

func NewEnumType2(args ...eval.PValue) *EnumType {
	return NewEnumType3(WrapArray(args))
}

func NewEnumType3(args eval.IndexedValue) *EnumType {
	if args.Len() == 0 {
		return DefaultEnumType()
	}
	var enums []string
	top := args.Len()
	if top == 1 {
		first := args.At(0)
		switch first.(type) {
		case *StringValue:
			enums = []string{first.String()}
		case *ArrayValue:
			return NewEnumType3(first.(*ArrayValue))
		default:
			panic(NewIllegalArgumentType2(`Enum[]`, 0, `String or Array[String]`, args.At(0)))
		}
	} else {
		enums = make([]string, top)
		args.EachWithIndex(func(arg eval.PValue, idx int) {
			str, ok := arg.(*StringValue)
			if !ok {
				panic(NewIllegalArgumentType2(`Enum[]`, idx, `String`, arg))
			}
			enums[idx] = str.String()
		})
	}
	return NewEnumType(enums)
}

func (t *EnumType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *EnumType) Default() eval.PType {
	return enumType_DEFAULT
}

func (t *EnumType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*EnumType); ok {
		return len(t.values) == len(ot.values) && utils.ContainsAllStrings(t.values, ot.values)
	}
	return false
}

func (t *EnumType) Generic() eval.PType {
	return enumType_DEFAULT
}

func (t *EnumType) IsAssignable(o eval.PType, g eval.Guard) bool {
	if len(t.values) == 0 {
		switch o.(type) {
		case *StringType, *EnumType, *PatternType:
			return true
		}
		return false
	}

	if st, ok := o.(*StringType); ok {
		return utils.ContainsString(t.values, st.value)
	}

	if en, ok := o.(*EnumType); ok {
		oEnums := en.values
		return len(oEnums) > 0 && utils.ContainsAllStrings(t.values, oEnums)
	}
	return false
}

func (t *EnumType) IsInstance(o eval.PValue, g eval.Guard) bool {
	str, ok := o.(*StringValue)
	return ok && (len(t.values) == 0 || utils.ContainsString(t.values, str.String()))
}

func (t *EnumType) Name() string {
	return `Enum`
}

func (t *EnumType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *EnumType) Parameters() []eval.PValue {
	top := len(t.values)
	if top == 0 {
		return eval.EMPTY_VALUES
	}
	v := make([]eval.PValue, top)
	for idx, e := range t.values {
		v[idx] = WrapString(e)
	}
	return v
}

func (t *EnumType) ToString(b io.Writer, f eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, f, g)
}

func (t *EnumType) Type() eval.PType {
	return &TypeType{t}
}

var enumType_DEFAULT = &EnumType{[]string{}}
