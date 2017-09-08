package values

import (
	. "fmt"
	. "io"
	"reflect"

	. "github.com/puppetlabs/go-evaluator/eval/errors"
	. "github.com/puppetlabs/go-evaluator/eval/utils"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
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
	return &RuntimeType{runtimeName, name, pattern, nil}
}

func NewRuntimeType2(args ...PValue) *RuntimeType {
	top := len(args)
	if top > 3 {
		panic(NewIllegalArgumentCount(`Runtime[]`, `0 - 3`, len(args)))
	}
	if top == 0 {
		return DefaultRuntimeType()
	}

	name, ok := args[0].(*StringValue)
	if !ok {
		panic(NewIllegalArgumentType2(`Runtime[]`, 0, `String`, args[0]))
	}

	var runtimeName string
	if top == 1 {
		runtimeName = ``
	} else {
		var rv *StringValue
		rv, ok = args[1].(*StringValue)
		if !ok {
			panic(NewIllegalArgumentType2(`Runtime[]`, 1, `String`, args[1]))
		}
		runtimeName = rv.String()
	}

	var pattern *RegexpType
	if top == 2 {
		pattern = nil
	} else {
		pattern, ok = args[2].(*RegexpType)
		if !ok {
			panic(NewIllegalArgumentType2(`Runtime[]`, 2, `Type[Regexp]`, args[2]))
		}
	}
	return NewRuntimeType(runtimeName, name.String(), pattern)
}

func NewRuntimeType3(goType reflect.Type) *RuntimeType {
	return &RuntimeType{`go`, goType.Name(), nil, goType}
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

func (t *RuntimeType) IsAssignable(o PType, g Guard) bool {
	if rt, ok := o.(*RuntimeType); ok {
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
		// There is no way to turn a string into a Type and then check assignability in Go
	}
	return false
}

func (t *RuntimeType) IsInstance(o PValue, g Guard) bool {
	rt, ok := o.(*RuntimeValue)
	if !ok {
		return false
	}
	if t.goType != nil && reflect.ValueOf(rt.Interface()).Type().AssignableTo(t.goType) {
		return true
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

func (t *RuntimeType) String() string {
	return ToString2(t, NONE)
}

func (t *RuntimeType) ToString(bld Writer, format FormatContext, g RDetect) {
	WriteString(bld, `Runtime`)
	if t.runtime != `` {
		WriteByte(bld, '[')
		WriteString(bld, t.runtime)
		if t.name != `` {
			WriteByte(bld, ',')
			PuppetQuote(bld, t.name)
		}
		if t.pattern != nil {
			WriteByte(bld, ',')
			RegexpQuote(bld, t.pattern.patternString)
		}
		WriteByte(bld, ']')
	}
}

func (t *RuntimeType) Type() PType {
	return &TypeType{t}
}

func WrapRuntime(value interface{}) *RuntimeValue {
	return &RuntimeValue{NewRuntimeType(`go`, Sprintf("%T", value), nil), value}
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
