package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
)

type PatternType struct {
	regexps []*RegexpType
}

var Pattern_Type eval.ObjectType

func init() {
	Pattern_Type = newObjectType(`Pcore::PatternType`,
		`Pcore::ScalarDataType {
	attributes => {
		patterns => Array[Regexp]
	}
}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
			return NewPatternType2(args...)
		})
}

func DefaultPatternType() *PatternType {
	return patternType_DEFAULT
}

func NewPatternType(regexps []*RegexpType) *PatternType {
	return &PatternType{regexps}
}

func NewPatternType2(regexps ...eval.PValue) *PatternType {
	return NewPatternType3(WrapArray(regexps))
}

func NewPatternType3(regexps eval.IndexedValue) *PatternType {

	cnt := regexps.Len()
	switch cnt {
	case 0:
		return DefaultPatternType()
	case 1:
		if iv, ok := regexps.At(0).(eval.IndexedValue); ok {
			return NewPatternType3(iv)
		}
	}

	rs := make([]*RegexpType, cnt)
	regexps.EachWithIndex(func(arg eval.PValue, idx int) {
		switch arg.(type) {
		case *RegexpType:
			rs[idx] = arg.(*RegexpType)
		case *RegexpValue:
			rs[idx] = arg.(*RegexpValue).Type().(*RegexpType)
		case *StringValue:
			rs[idx] = NewRegexpType2(arg)
		default:
			panic(NewIllegalArgumentType2(`Pattern[]`, idx, `Type[Regexp], Regexp, or String`, arg))
		}
	})
	return NewPatternType(rs)
}

func (t *PatternType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	for _, rx := range t.regexps {
		rx.Accept(v, g)
	}
}

func (t *PatternType) Default() eval.PType {
	return patternType_DEFAULT
}

func (t *PatternType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*PatternType); ok {
		return len(t.regexps) == len(ot.regexps) && eval.GuardedIncludesAll(eval.EqSlice(t.regexps), eval.EqSlice(ot.regexps), g)
	}
	return false
}

func (t *PatternType) Get(key string) (value eval.PValue, ok bool) {
	switch key {
	case `patterns`:
		return WrapArray(t.Parameters()), true
	}
	return nil, false
}

func (t *PatternType) IsAssignable(o eval.PType, g eval.Guard) bool {
	if st, ok := o.(*StringType); ok {
		if len(t.regexps) == 0 {
			return true
		}
		str := st.value
		return str != `` && utils.MatchesString(MapToRegexps(t.regexps), str)
	}

	if et, ok := o.(*EnumType); ok {
		if len(t.regexps) == 0 {
			return true
		}
		enums := et.values
		return len(enums) > 0 && utils.MatchesAllStrings(MapToRegexps(t.regexps), enums)
	}
	return false
}

func (t *PatternType) IsInstance(o eval.PValue, g eval.Guard) bool {
	str, ok := o.(*StringValue)
	return ok && (len(t.regexps) == 0 || utils.MatchesString(MapToRegexps(t.regexps), str.String()))
}

func (t *PatternType) MetaType() eval.ObjectType {
	return Pattern_Type
}

func (t *PatternType) Name() string {
	return `Pattern`
}

func (t *PatternType) Parameters() []eval.PValue {
	top := len(t.regexps)
	if top == 0 {
		return eval.EMPTY_VALUES
	}
	rxs := make([]eval.PValue, top)
	for idx, rx := range t.regexps {
		rxs[idx] = WrapRegexp(rx.patternString)
	}
	return rxs
}

func (t *PatternType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *PatternType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *PatternType) Type() eval.PType {
	return &TypeType{t}
}

var patternType_DEFAULT = &PatternType{[]*RegexpType{}}
