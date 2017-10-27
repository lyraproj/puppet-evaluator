package types

import (
	. "io"

	. "github.com/puppetlabs/go-evaluator/utils"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type PatternType struct {
	regexps []*RegexpType
}

func DefaultPatternType() *PatternType {
	return patternType_DEFAULT
}

func NewPatternType(regexps []*RegexpType) *PatternType {
	return &PatternType{regexps}
}

func NewPatternType2(regexps ...PValue) *PatternType {
	cnt := len(regexps)
	switch cnt {
	case 0:
		return DefaultPatternType()
	case 1:
		if iv, ok := regexps[0].(IndexedValue); ok {
			return NewPatternType2(iv.Elements()...)
		}
	}

	rs := make([]*RegexpType, cnt)
	for idx, arg := range regexps {
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
	}
	return NewPatternType(rs)
}

func (t *PatternType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*PatternType); ok {
		return len(t.regexps) == len(ot.regexps) && GuardedIncludesAll(EqSlice(t.regexps), EqSlice(ot.regexps), g)
	}
	return false
}

func (t *PatternType) IsAssignable(o PType, g Guard) bool {
	if st, ok := o.(*StringType); ok {
		if len(t.regexps) == 0 {
			return true
		}
		str := st.value
		return str != `` && MatchesString(MapToRegexps(t.regexps), str)
	}

	if et, ok := o.(*EnumType); ok {
		if len(t.regexps) == 0 {
			return true
		}
		enums := et.values
		return len(enums) > 0 && MatchesAllStrings(MapToRegexps(t.regexps), enums)
	}
	return false
}

func (t *PatternType) IsInstance(o PValue, g Guard) bool {
	str, ok := o.(*StringValue)
	return ok && (len(t.regexps) == 0 || MatchesString(MapToRegexps(t.regexps), str.String()))
}

func (t *PatternType) Name() string {
	return `Pattern`
}

func (t *PatternType) Parameters() []PValue {
	top := len(t.regexps)
	if top == 0 {
		return EMPTY_VALUES
	}
	rxs := make([]PValue, top)
	for idx, rx := range t.regexps {
		rxs[idx] = WrapRegexp(rx.patternString)
	}
	return rxs
}

func (t *PatternType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *PatternType) String() string {
	return ToString2(t, NONE)
}

func (t *PatternType) Type() PType {
	return &TypeType{t}
}

var patternType_DEFAULT = &PatternType{[]*RegexpType{}}
