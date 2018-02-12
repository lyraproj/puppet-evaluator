package types

import (
	"bytes"
	"io"
	"regexp"
	"sync"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-parser/issue"
)

type (
	RegexpType struct {
		lock          sync.Mutex
		pattern       *regexp.Regexp
		patternString string
	}

	// RegexpValue represents RegexpType as a value
	RegexpValue RegexpType
)

var regexpType_DEFAULT = &RegexpType{pattern: regexp.MustCompile(``), patternString: ``}

var Regexp_Type eval.ObjectType

func init() {
	Regexp_Type = newObjectType(`Pcore::RegexpType`,
		`Pcore::ScalarType {
	attributes => {
		pattern => {
      type => Variant[Undef,String,Regexp],
      value => undef
    }
	}
}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
			return NewRegexpType2(args...)
		})
}

func DefaultRegexpType() *RegexpType {
	return regexpType_DEFAULT
}

func NewRegexpType(patternString string) *RegexpType {
	if patternString == `` {
		return DefaultRegexpType()
	}
	return &RegexpType{patternString: patternString}
}

func NewRegexpTypeR(pattern *regexp.Regexp) *RegexpType {
	patternString := pattern.String()
	if patternString == `` {
		return DefaultRegexpType()
	}
	return &RegexpType{pattern: pattern, patternString: patternString}
}

func NewRegexpType2(args ...eval.PValue) *RegexpType {
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
		panic(errors.NewIllegalArgumentCount(`Regexp[]`, `0 - 1`, len(args)))
	}
}

func (t *RegexpType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *RegexpType) Default() eval.PType {
	return regexpType_DEFAULT
}

func (t *RegexpType) Equals(o interface{}, g eval.Guard) bool {
	ot, ok := o.(*RegexpType)
	return ok && t.patternString == ot.patternString
}

func (t *RegexpType) Get(key string) (value eval.PValue, ok bool) {
	switch key {
	case `pattern`:
		if t.patternString == `` {
			return _UNDEF, true
		}
		return WrapString(t.patternString), true
	}
	return nil, false
}

func (t *RegexpType) IsAssignable(o eval.PType, g eval.Guard) bool {
	rx, ok := o.(*RegexpType)
	return ok && (t.patternString == `` || t.patternString == rx.patternString)
}

func (t *RegexpType) IsInstance(o eval.PValue, g eval.Guard) bool {
	rx, ok := o.(*RegexpValue)
	return ok && (t.patternString == `` || t.patternString == rx.PatternString())
}

func (t *RegexpType) MetaType() eval.ObjectType {
	return Regexp_Type
}

func (t *RegexpType) Name() string {
	return `Regexp`
}

func (t *RegexpType) Parameters() []eval.PValue {
	if t.patternString == `` {
		return eval.EMPTY_VALUES
	}
	return []eval.PValue{WrapRegexp(t.patternString)}
}

func (t *RegexpType) PatternString() string {
	return t.patternString
}

func (t *RegexpType) Regexp() *regexp.Regexp {
	t.lock.Lock()
	if t.pattern == nil {
		pattern, err := regexp.Compile(t.patternString)
		if err != nil {
			t.lock.Unlock()
			panic(eval.Error(eval.EVAL_INVALID_REGEXP, issue.H{`pattern`: t.patternString, `detail`: err.Error()}))
		}
		t.pattern = pattern
	}
	t.lock.Unlock()
	return t.pattern
}

func (t *RegexpType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *RegexpType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *RegexpType) Type() eval.PType {
	return &TypeType{t}
}

func MapToRegexps(regexpTypes []*RegexpType) []*regexp.Regexp {
	top := len(regexpTypes)
	result := make([]*regexp.Regexp, top)
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

func WrapRegexp2(pattern *regexp.Regexp) *RegexpValue {
	return (*RegexpValue)(NewRegexpTypeR(pattern))
}

func (sv *RegexpValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*RegexpValue); ok {
		return sv.String() == ov.String()
	}
	return false
}

func (sv *RegexpValue) Regexp() *regexp.Regexp {
	return (*RegexpType)(sv).Regexp()
}

func (sv *RegexpValue) PatternString() string {
	return sv.patternString
}

func (sv *RegexpValue) String() string {
	return eval.ToString2(sv, NONE)
}

func (sv *RegexpValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HK_REGEXP)
	b.Write([]byte(sv.patternString))
}

func (sv *RegexpValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	utils.RegexpQuote(b, sv.patternString)
}

func (sv *RegexpValue) Type() eval.PType {
	return (*RegexpType)(sv)
}
