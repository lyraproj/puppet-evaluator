package types

import (
	"fmt"
	"io"
	"reflect"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-issues/issue"
)

type (
	RuntimeType struct {
		name    string
		runtime string
		pattern *RegexpType
		goType  reflect.Type
	}

	// RuntimeValue Captures values of all types unknown to Puppet
	RuntimeValue struct {
		puppetType *RuntimeType
		value      interface{}
	}
)

var runtimeType_DEFAULT = &RuntimeType{``, ``, nil, nil}

var Runtime_Type eval.ObjectType

func init() {
	Runtime_Type = newObjectType(`Pcore::RuntimeType`,
		`Pcore::AnyType {
	attributes => {
		runtime => {
      type => Optional[String[1]],
      value => undef
    },
		name_or_pattern => {
      type => Variant[Undef,String[1],Tuple[Regexp,String[1]]],
      value => undef
    }
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewRuntimeType2(args...)
		})
}

func DefaultRuntimeType() *RuntimeType {
	return runtimeType_DEFAULT
}

func NewRuntimeType(runtimeName string, name string, pattern *RegexpType) *RuntimeType {
	if runtimeName == `` && name == `` && pattern == nil {
		return DefaultRuntimeType()
	}
	if runtimeName == `go` && name != `` {
		panic(eval.Error(eval.EVAL_GO_RUNTIME_TYPE_WITHOUT_GO_TYPE, issue.H{`name`: name}))
	}
	return &RuntimeType{runtime: runtimeName, name: name, pattern: pattern}
}

func NewRuntimeType2(args ...eval.Value) *RuntimeType {
	top := len(args)
	if top > 3 {
		panic(errors.NewIllegalArgumentCount(`Runtime[]`, `0 - 3`, len(args)))
	}
	if top == 0 {
		return DefaultRuntimeType()
	}

	runtimeName, ok := args[0].(*StringValue)
	if !ok {
		panic(NewIllegalArgumentType2(`Runtime[]`, 0, `String`, args[0]))
	}

	var pattern *RegexpType
	var name eval.Value
	if top == 1 {
		name = eval.EMPTY_STRING
	} else {
		var rv *StringValue
		rv, ok = args[1].(*StringValue)
		if !ok {
			panic(NewIllegalArgumentType2(`Runtime[]`, 1, `String`, args[1]))
		}
		name = rv

		if top == 2 {
			pattern = nil
		} else {
			pattern, ok = args[2].(*RegexpType)
			if !ok {
				panic(NewIllegalArgumentType2(`Runtime[]`, 2, `Type[Regexp]`, args[2]))
			}
		}
	}
	return NewRuntimeType(runtimeName.String(), name.String(), pattern)
}

// NewGoRuntimeType creates a Go runtime by extracting the element type of the given slice or array.
// The reason the argument must be a slice or array as opposed to an element is that there is
// no way to create elements of an interface. An empty array with an interface element type
// is however possible.
//
// To create an Runtime for the interface Foo, pass []Foo{} to this method.
func NewGoRuntimeType(array interface{}) *RuntimeType {
	goType := reflect.TypeOf(array).Elem()
	return &RuntimeType{runtime: `go`, name: goType.String(), goType: goType}
}

func (t *RuntimeType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *RuntimeType) Default() eval.Type {
	return runtimeType_DEFAULT
}

func (t *RuntimeType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*RuntimeType); ok && t.runtime == ot.runtime && t.name == ot.name {
		if t.pattern == nil {
			return ot.pattern == nil
		}
		return t.pattern.Equals(ot.pattern, g)
	}
	return false
}

func (t *RuntimeType) Generic() eval.Type {
	return runtimeType_DEFAULT
}

func (t *RuntimeType) Get(key string) (eval.Value, bool) {
	switch key {
	case `runtime`:
		if t.runtime == `` {
			return _UNDEF, true
		}
		return WrapString(t.runtime), true
	case `name_or_pattern`:
		if t.pattern != nil {
			return t.pattern, true
		}
		if t.name != `` {
			return WrapString(t.name), true
		}
		return _UNDEF, true
	default:
		return nil, false
	}
}

func (t *RuntimeType) IsAssignable(o eval.Type, g eval.Guard) bool {
	if rt, ok := o.(*RuntimeType); ok {
		if t.goType != nil && rt.goType != nil {
			return rt.goType.AssignableTo(t.goType)
		}
		if t.runtime == `` {
			return true
		}
		if t.runtime != rt.runtime {
			return false
		}
		if t.name == `` {
			return true
		}
		if t.pattern != nil {
			return t.name == rt.name && rt.pattern != nil && t.pattern.patternString == rt.pattern.patternString
		}
		if t.name == rt.name {
			return true
		}
	}
	return false
}

func (t *RuntimeType) IsInstance(o eval.Value, g eval.Guard) bool {
	rt, ok := o.(*RuntimeValue)
	if !ok {
		return false
	}
	if t.goType != nil {
		return reflect.ValueOf(rt.Interface()).Type().AssignableTo(t.goType)
	}
	if t.runtime == `` {
		return true
	}
	if o == nil || t.runtime != `go` || t.pattern != nil {
		return false
	}
	if t.name == `` {
		return true
	}
	return t.name == fmt.Sprintf(`%T`, rt.Interface())
}

func (t *RuntimeType) MetaType() eval.ObjectType {
	return Runtime_Type
}

func (t *RuntimeType) Name() string {
	return `Runtime`
}

func (t *RuntimeType) Parameters() []eval.Value {
	if t.runtime == `` {
		return eval.EMPTY_VALUES
	}
	ps := make([]eval.Value, 0, 2)
	ps = append(ps, WrapString(t.runtime))
	if t.name != `` {
		ps = append(ps, WrapString(t.name))
	}
	if t.pattern != nil {
		ps = append(ps, t.pattern)
	}
	return ps
}

func (t *RuntimeType) ReflectType() (reflect.Type, bool) {
	return t.goType, t.goType != nil
}

func (t *RuntimeType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *RuntimeType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *RuntimeType) Type() eval.Type {
	return &TypeType{t}
}

func WrapRuntime(value interface{}) *RuntimeValue {
	goType := reflect.TypeOf(value)
	return &RuntimeValue{&RuntimeType{runtime: `go`, name: goType.String(), goType: goType}, value}
}

func (rv *RuntimeValue) Equals(o interface{}, g eval.Guard) bool {
	if ov, ok := o.(*RuntimeValue); ok {
		var re eval.Equality
		if re, ok = rv.value.(eval.Equality); ok {
			var oe eval.Equality
			if oe, ok = ov.value.(eval.Equality); ok {
				return re.Equals(oe, g)
			}
			return false
		}
		return reflect.DeepEqual(rv.value, ov.value)
	}
	return false
}

func (rv *RuntimeValue) Reflect(c eval.Context) reflect.Value {
	gt := rv.puppetType.goType
	if gt == nil {
		panic(eval.Error(eval.EVAL_INVALID_SOURCE_FOR_GET, issue.H{`type`: rv.Type().String()}))
	}
	return reflect.ValueOf(rv.value)
}

func (rv *RuntimeValue) ReflectTo(c eval.Context, dest reflect.Value) {
	gt := rv.puppetType.goType
	if gt == nil {
		panic(eval.Error(eval.EVAL_INVALID_SOURCE_FOR_GET, issue.H{`type`: rv.Type().String()}))
	}
	if !gt.AssignableTo(dest.Type()) {
		panic(eval.Error(eval.EVAL_ATTEMPT_TO_SET_WRONG_KIND, issue.H{`expected`: gt.String(), `actual`: dest.Type().String()}))
	}
	dest.Set(reflect.ValueOf(rv.value))
}

func (rv *RuntimeValue) String() string {
	return eval.ToString2(rv, NONE)
}

func (rv *RuntimeValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	utils.PuppetQuote(b, fmt.Sprintf(`%v`, rv.value))
}

func (rv *RuntimeValue) Type() eval.Type {
	return rv.puppetType
}

func (rv *RuntimeValue) Interface() interface{} {
	return rv.value
}
