package types

import (
	"bytes"
	"io"
	"regexp"
	"sync"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/utils"
	"reflect"
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
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewRegexpType2(args...)
		})

	newGoConstructor(`Regexp`,
		func(d eval.Dispatch) {
			d.Param(`Variant[String,Regexp]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				arg := args[0]
				if s, ok := arg.(*RegexpValue); ok {
					return s
				}
				return WrapRegexp(arg.String())
			})
		},
	)
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

func NewRegexpType2(args ...eval.Value) *RegexpType {
	switch len(args) {
	case 0:
		return regexpType_DEFAULT
	case 1:
		rx := args[0]
		if str, ok := rx.(stringValue); ok {
			return NewRegexpType(string(str))
		}
		if rt, ok := rx.(*RegexpValue); ok {
			return rt.PType().(*RegexpType)
		}
		panic(NewIllegalArgumentType2(`Regexp[]`, 0, `Variant[Regexp,String]`, args[0]))
	default:
		panic(errors.NewIllegalArgumentCount(`Regexp[]`, `0 - 1`, len(args)))
	}
}

func (t *RegexpType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *RegexpType) Default() eval.Type {
	return regexpType_DEFAULT
}

func (t *RegexpType) Equals(o interface{}, g eval.Guard) bool {
	ot, ok := o.(*RegexpType)
	return ok && t.patternString == ot.patternString
}

func (t *RegexpType) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `pattern`:
		if t.patternString == `` {
			return _UNDEF, true
		}
		return stringValue(t.patternString), true
	}
	return nil, false
}

func (t *RegexpType) IsAssignable(o eval.Type, g eval.Guard) bool {
	rx, ok := o.(*RegexpType)
	return ok && (t.patternString == `` || t.patternString == rx.patternString)
}

func (t *RegexpType) IsInstance(o eval.Value, g eval.Guard) bool {
	rx, ok := o.(*RegexpValue)
	return ok && (t.patternString == `` || t.patternString == rx.PatternString())
}

func (t *RegexpType) MetaType() eval.ObjectType {
	return Regexp_Type
}

func (t *RegexpType) Name() string {
	return `Regexp`
}

func (t *RegexpType) Parameters() []eval.Value {
	if t.patternString == `` {
		return eval.EMPTY_VALUES
	}
	return []eval.Value{WrapRegexp(t.patternString)}
}

func (t *RegexpType) ReflectType(c eval.Context) (reflect.Type, bool) {
	return reflect.TypeOf(regexpType_DEFAULT.pattern), true
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

func (t *RegexpType) CanSerializeAsString() bool {
	return true
}

func (t *RegexpType) SerializationString() string {
	return t.String()
}

func (t *RegexpType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *RegexpType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *RegexpType) PType() eval.Type {
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

func (r *RegexpValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*RegexpValue); ok {
		return r.String() == ov.String()
	}
	return false
}

func (r *RegexpValue) Match(s string) []string {
	return (*RegexpType)(r).Regexp().FindStringSubmatch(s)
}

func (r *RegexpValue) Regexp() *regexp.Regexp {
	return (*RegexpType)(r).Regexp()
}

func (r *RegexpValue) PatternString() string {
	return r.patternString
}

func (r *RegexpValue) Reflect(c eval.Context) reflect.Value {
	return reflect.ValueOf(r.pattern)
}

func (r *RegexpValue) ReflectTo(c eval.Context, dest reflect.Value) {
	rv := r.Reflect(c).Elem()
	if !rv.Type().AssignableTo(dest.Type()) {
		panic(eval.Error(eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: rv.Type().String(), `actual`: dest.Type().String()}))
	}
	dest.Set(rv)
}

func (r *RegexpValue) String() string {
	return eval.ToString2(r, NONE)
}

func (r *RegexpValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HK_REGEXP)
	b.Write([]byte(r.patternString))
}

func (r *RegexpValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	utils.RegexpQuote(b, r.patternString)
}

func (r *RegexpValue) PType() eval.Type {
	return (*RegexpType)(r)
}
