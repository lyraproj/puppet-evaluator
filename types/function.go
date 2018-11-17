package types

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
)

var TYPE_FUNCTION_TYPE = NewTypeType(DefaultCallableType())

const KEY_RETURNS_ERROR = `returns_error`

var TYPE_FUNCTION = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, TYPE_FUNCTION_TYPE),
	NewStructElement(NewOptionalType3(KEY_FINAL), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_OVERRIDE), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
	NewStructElement(NewOptionalType3(KEY_GONAME), DefaultStringType()),
	NewStructElement(NewOptionalType3(KEY_RETURNS_ERROR), NewBooleanType(true)),
})

type function struct {
	annotatedMember
	returnsError bool
	goName string
}

func newFunction(c eval.Context, name string, container *objectType, initHash *HashValue) eval.ObjFunc {
	f := &function{}
	f.initialize(c, name, container, initHash)
	return f
}

func (f *function) initialize(c eval.Context, name string, container *objectType, initHash *HashValue) {
	eval.AssertInstance(func() string { return fmt.Sprintf(`initializer function for %s[%s]`, container.Label(), name) }, TYPE_FUNCTION, initHash)
	f.annotatedMember.initialize(c, `function`, name, container, initHash)
	if gn, ok := initHash.Get4(KEY_GONAME); ok {
		f.goName = gn.String()
	}
	if re, ok := initHash.Get4(KEY_RETURNS_ERROR); ok {
		f.returnsError = re.(*BooleanValue).Bool()
	}
}

func (a *function) Call(c eval.Context, receiver eval.Value, block eval.Lambda, args []eval.Value) eval.Value {
	if a.CallableType().(*CallableType).CallableWith(args, block) {
		if co, ok := receiver.(eval.CallableObject); ok {
			if result, ok := co.Call(c, a, args, block); ok {
				return result
			}
		}

		panic(eval.Error(eval.EVAL_INSTANCE_DOES_NOT_RESPOND, issue.H{`instance`: receiver, `message`: a.name}))
	}
	types := make([]eval.Value, len(args))
	for i, a := range args {
		types[i] = a.PType()
	}
	panic(eval.Error(eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeSignatures(
		[]eval.Signature{a.CallableType().(*CallableType)}, NewTupleType2(types...), block)}))
}

func (f *function) CallGo(c eval.Context, receiver interface{}, args ...interface{}) []interface{} {
	rfArgs := make([]reflect.Value, 1 + len(args))
	rfArgs[0] = reflect.ValueOf(receiver)
	for i, arg := range args {
		rfArgs[i+1] = reflect.ValueOf(arg)
	}
	result := f.CallGoReflected(c, rfArgs)
	rs := make([]interface{}, len(result))
	for i, ret := range result {
		if ret.IsValid() {
			rs[i] = ret.Interface()
		}
	}
	return rs
}

func (f *function) CallGoReflected(c eval.Context, args []reflect.Value) []reflect.Value {
	rt := args[0].Type()
	m, ok := rt.MethodByName(f.goName)
	if !ok {
		panic(eval.Error(eval.EVAL_INSTANCE_DOES_NOT_RESPOND, issue.H{`instance`: rt.String(), `message`: f.goName}))
	}

	mt := m.Type
	maxReturn := 1
	if f.ReturnsError() {
		maxReturn = 2
	}
	pc := mt.NumIn()
	if mt.NumOut() > maxReturn || pc != len(args) {
		panic(eval.Error(eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeSignatures(
			[]eval.Signature{f.CallableType().(*CallableType)}, NewTupleType([]eval.Type{}, NewIntegerType(int64(pc-1), int64(pc-1))), nil)}))
	}
	result := m.Func.Call(args)
	oc := mt.NumOut()

	if f.ReturnsError() {
		oc--
		err := result[oc].Interface()
		if err != nil {
			panic(err)
		}
		result = result[:oc]
	}
	return result
}

func (f *function) GoName() string {
	return f.goName
}

func (f *function) ReturnsError() bool {
	return f.returnsError
}

func (f *function) Equals(other interface{}, g eval.Guard) bool {
	if of, ok := other.(*function); ok {
		return f.override == of.override && f.name == of.name && f.final == of.final && f.typ.Equals(of.typ, g)
	}
	return false
}

func (f *function) FeatureType() string {
	return `function`
}

func (f *function) Label() string {
	return fmt.Sprintf(`function %s[%s]`, f.container.Label(), f.Name())
}

func (f *function) CallableType() eval.Type {
	return f.typ.(*CallableType)
}

func (f *function) InitHash() eval.OrderedMap {
	return WrapStringPValue(f.initHash())
}
