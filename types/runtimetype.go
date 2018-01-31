package types

import (
	. "fmt"
	. "io"
	"reflect"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/utils"
	. "github.com/puppetlabs/go-parser/issue"
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

func DefaultRuntimeType() *RuntimeType {
	return runtimeType_DEFAULT
}

func NewRuntimeType(runtimeName string, name string, pattern *RegexpType) *RuntimeType {
	if runtimeName == `` && name == `` && pattern == nil {
		return DefaultRuntimeType()
	}
	if runtimeName == `go` && name != `` {
		panic(Error(EVAL_GO_RUNTIME_TYPE_WITHOUT_GO_TYPE, H{`name`: name}))
	}
	return &RuntimeType{runtime: runtimeName, name: name, pattern: pattern}
}

func NewRuntimeType2(args ...PValue) *RuntimeType {
	top := len(args)
	if top > 3 {
		panic(NewIllegalArgumentCount(`Runtime[]`, `0 - 3`, len(args)))
	}
	if top == 0 {
		return DefaultRuntimeType()
	}

	runtimeName, ok := args[0].(*StringValue)
	if !ok {
		panic(NewIllegalArgumentType2(`Runtime[]`, 0, `String`, args[0]))
	}

	var pattern *RegexpType
	var name PValue
	if top == 1 {
		name = EMPTY_STRING
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

func (t *RuntimeType) Accept(v Visitor, g Guard) {
	v(t)
}

func (t *RuntimeType) Default() PType {
	return runtimeType_DEFAULT
}

func (t *RuntimeType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*RuntimeType); ok && t.runtime == ot.runtime && t.name == ot.name {
		if t.pattern == nil {
			return ot.pattern == nil
		}
		return t.pattern.Equals(ot.pattern, g)
	}
	return false
}

func (t *RuntimeType) Generic() PType {
	return runtimeType_DEFAULT
}

func (t *RuntimeType) IsAssignable(o PType, g Guard) bool {
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

func (t *RuntimeType) IsInstance(o PValue, g Guard) bool {
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
	return t.name == Sprintf(`%T`, rt.Interface())
}

func (t *RuntimeType) Name() string {
	return `Runtime`
}

func (t *RuntimeType) Parameters() []PValue {
	if t.runtime == `` {
		return EMPTY_VALUES
	}
	ps := make([]PValue, 0, 2)
	ps = append(ps, WrapString(t.runtime))
	if t.name != `` {
		ps = append(ps, WrapString(t.name))
	}
	if t.pattern != nil {
		ps = append(ps, t.pattern)
	}
	return ps
}

func (t *RuntimeType) String() string {
	return ToString2(t, NONE)
}

func (t *RuntimeType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *RuntimeType) Type() PType {
	return &TypeType{t}
}

func WrapRuntime(value interface{}) *RuntimeValue {
	goType := reflect.TypeOf(value)
	return &RuntimeValue{&RuntimeType{runtime: `go`, name: goType.String(), goType: goType}, value}
}

func (rv *RuntimeValue) Equals(o interface{}, g Guard) bool {
	if ov, ok := o.(*RuntimeValue); ok {
		var re Equality
		if re, ok = rv.value.(Equality); ok {
			var oe Equality
			if oe, ok = ov.value.(Equality); ok {
				return re.Equals(oe, g)
			}
			return false
		}
		return reflect.DeepEqual(rv.value, ov.value)
	}
	return false
}

func (rv *RuntimeValue) String() string {
	return ToString2(rv, NONE)
}

func (rv *RuntimeValue) ToString(b Writer, s FormatContext, g RDetect) {
	PuppetQuote(b, Sprintf(`%v`, rv.value))
}

func (rv *RuntimeValue) Type() PType {
	return rv.puppetType
}

func (rv *RuntimeValue) Interface() interface{} {
	return rv.value
}
