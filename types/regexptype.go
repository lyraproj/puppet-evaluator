package types

import (
	"bytes"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/utils"
	"io"
	"reflect"
	"regexp"
)

type (
	RegexpType struct {
		pattern *regexp.Regexp
	}

	// RegexpValue represents RegexpType as a value
	RegexpValue RegexpType
)

var regexpTypeDefault = &RegexpType{pattern: regexp.MustCompile(``)}

var RegexpMetaType eval.ObjectType

func init() {
	RegexpMetaType = newObjectType(`Pcore::RegexpType`,
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
	return regexpTypeDefault
}

func NewRegexpType(patternString string) *RegexpType {
	if patternString == `` {
		return DefaultRegexpType()
	}
	pattern, err := regexp.Compile(patternString)
	if err != nil {
		panic(eval.Error(eval.EVAL_INVALID_REGEXP, issue.H{`pattern`: patternString, `detail`: err.Error()}))
	}
	return &RegexpType{pattern: pattern}
}

func NewRegexpTypeR(pattern *regexp.Regexp) *RegexpType {
	if pattern.String() == `` {
		return DefaultRegexpType()
	}
	return &RegexpType{pattern: pattern}
}

func NewRegexpType2(args ...eval.Value) *RegexpType {
	switch len(args) {
	case 0:
		return regexpTypeDefault
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
	return regexpTypeDefault
}

func (t *RegexpType) Equals(o interface{}, g eval.Guard) bool {
	ot, ok := o.(*RegexpType)
	return ok && t.pattern.String() == ot.pattern.String()
}

func (t *RegexpType) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `pattern`:
		if t.String() == `` {
			return _UNDEF, true
		}
		return stringValue(t.pattern.String()), true
	}
	return nil, false
}

func (t *RegexpType) IsAssignable(o eval.Type, g eval.Guard) bool {
	rx, ok := o.(*RegexpType)
	return ok && (t.pattern.String() == `` || t.pattern.String() == rx.PatternString())
}

func (t *RegexpType) IsInstance(o eval.Value, g eval.Guard) bool {
	rx, ok := o.(*RegexpValue)
	return ok && (t.pattern.String() == `` || t.pattern.String() == rx.PatternString())
}

func (t *RegexpType) MetaType() eval.ObjectType {
	return RegexpMetaType
}

func (t *RegexpType) Name() string {
	return `Regexp`
}

func (t *RegexpType) Parameters() []eval.Value {
	if t.pattern.String() == `` {
		return eval.EMPTY_VALUES
	}
	return []eval.Value{WrapRegexp2(t.pattern)}
}

func (t *RegexpType) ReflectType(c eval.Context) (reflect.Type, bool) {
	return reflect.TypeOf(regexpTypeDefault.pattern), true
}

func (t *RegexpType) PatternString() string {
	return t.pattern.String()
}

func (t *RegexpType) Regexp() *regexp.Regexp {
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
		key := regexpType.String()
		if !exists[key] {
			exists[key] = true
			result = append(result, regexpType)
		}
	}
	return result
}

func WrapRegexp(str string) *RegexpValue {
	pattern, err := regexp.Compile(str)
	if err != nil {
		panic(eval.Error(eval.EVAL_INVALID_REGEXP, issue.H{`pattern`: str, `detail`: err.Error()}))
	}
	return &RegexpValue{pattern}
}

func WrapRegexp2(pattern *regexp.Regexp) *RegexpValue {
	return &RegexpValue{pattern}
}

func (r *RegexpValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*RegexpValue); ok {
		return r.pattern.String() == ov.pattern.String()
	}
	return false
}

func (r *RegexpValue) Match(s string) []string {
	return r.pattern.FindStringSubmatch(s)
}

func (r *RegexpValue) Regexp() *regexp.Regexp {
	return r.pattern
}

func (r *RegexpValue) PatternString() string {
	return r.pattern.String()
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
	b.Write([]byte(r.pattern.String()))
}

func (r *RegexpValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	utils.RegexpQuote(b, r.pattern.String())
}

func (r *RegexpValue) PType() eval.Type {
	rt := RegexpType(*r)
	return &rt
}
