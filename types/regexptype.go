package types

import (
	. "io"
	. "regexp"
	"sync"

	"bytes"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-parser/issue"
)

type (
	RegexpType struct {
		lock          sync.Mutex
		pattern       *Regexp
		patternString string
	}

	// RegexpValue represents RegexpType as a value
	RegexpValue RegexpType
)

var regexpType_DEFAULT_PATTERN = `.*`
var regexpType_DEFAULT = &RegexpType{pattern: MustCompile(regexpType_DEFAULT_PATTERN), patternString: regexpType_DEFAULT_PATTERN}

func DefaultRegexpType() *RegexpType {
	return regexpType_DEFAULT
}

func NewRegexpType(patternString string) *RegexpType {
	if patternString == regexpType_DEFAULT_PATTERN {
		return DefaultRegexpType()
	}
	return &RegexpType{patternString: patternString}
}

func NewRegexpTypeR(pattern *Regexp) *RegexpType {
	patternString := pattern.String()
	if patternString == regexpType_DEFAULT_PATTERN {
		return DefaultRegexpType()
	}
	return &RegexpType{pattern: pattern, patternString: patternString}
}

func NewRegexpType2(args ...PValue) *RegexpType {
	switch len(args) {
	case 0:
		return regexpType_DEFAULT
	case 1:
		rx := args[0]
		if str, ok := rx.(*StringValue); ok {
			return NewRegexpType(str.String())
		}
		if rt, ok := rx.(*RegexpValue); ok {
			return rt.Type().(*RegexpType)
		}
		panic(NewIllegalArgumentType2(`Regexp[]`, 0, `Variant[Regexp,String]`, args[0]))
	default:
		panic(NewIllegalArgumentCount(`Regexp[]`, `0 - 1`, len(args)))
	}
}

func (t *RegexpType) Accept(v Visitor, g Guard) {
	v(t)
}

func (t *RegexpType) Default() PType {
	return regexpType_DEFAULT
}

func (t *RegexpType) Equals(o interface{}, g Guard) bool {
	ot, ok := o.(*RegexpType)
	return ok && t.patternString == ot.patternString
}

func (t *RegexpType) IsAssignable(o PType, g Guard) bool {
	rx, ok := o.(*RegexpType)
	return ok && (t.patternString == regexpType_DEFAULT_PATTERN || t.patternString == rx.patternString)
}

func (t *RegexpType) IsInstance(o PValue, g Guard) bool {
	rx, ok := o.(*RegexpValue)
	return ok && (t.patternString == regexpType_DEFAULT_PATTERN || t.patternString == rx.PatternString())
}

func (t *RegexpType) Name() string {
	return `Regexp`
}

func (t *RegexpType) Parameters() []PValue {
	if t.patternString == regexpType_DEFAULT_PATTERN {
		return EMPTY_VALUES
	}
	return []PValue{WrapRegexp(t.patternString)}
}

func (t *RegexpType) PatternString() string {
	return t.patternString
}

func (t *RegexpType) Regexp() *Regexp {
	t.lock.Lock()
	if t.pattern == nil {
		pattern, err := Compile(t.patternString)
		if err != nil {
			t.lock.Unlock()
			panic(Error(EVAL_INVALID_REGEXP, issue.H{`pattern`: t.patternString, `detail`: err.Error()}))
		}
		t.pattern = pattern
	}
	t.lock.Unlock()
	return t.pattern
}

func (t *RegexpType) String() string {
	return ToString2(t, NONE)
}

func (t *RegexpType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *RegexpType) Type() PType {
	return &TypeType{t}
}

func MapToRegexps(regexpTypes []*RegexpType) []*Regexp {
	top := len(regexpTypes)
	result := make([]*Regexp, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = regexpTypes[idx].Regexp()
	}
	return result
}

func UniqueRegexps(regexpTypes []*RegexpType) []*RegexpType {
	top := len(regexpTypes)
	if top < 2 {
		return regexpTypes
	}

	result := make([]*RegexpType, 0, top)
	exists := make(map[string]bool, top)
	for _, regexpType := range regexpTypes {
		key := regexpType.patternString
		if !exists[key] {
			exists[key] = true
			result = append(result, regexpType)
		}
	}
	return result
}

func WrapRegexp(str string) *RegexpValue {
	return (*RegexpValue)(NewRegexpType(str))
}

func (sv *RegexpValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*RegexpValue); ok {
		return sv.String() == ov.String()
	}
	return false
}

func (sv *RegexpValue) Regexp() *Regexp {
	return (*RegexpType)(sv).Regexp()
}

func (sv *RegexpValue) PatternString() string {
	return sv.patternString
}

func (sv *RegexpValue) String() string {
	return ToString2(sv, NONE)
}

func (sv *RegexpValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HK_REGEXP)
	b.Write([]byte(sv.patternString))
}

func (sv *RegexpValue) ToString(b Writer, s FormatContext, g RDetect) {
	RegexpQuote(b, sv.patternString)
}

func (sv *RegexpValue) Type() PType {
	return (*RegexpType)(sv)
}
