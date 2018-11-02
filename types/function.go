package types

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
)

var TYPE_FUNCTION_TYPE = NewTypeType(DefaultCallableType())

var TYPE_FUNCTION = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, TYPE_FUNCTION_TYPE),
	NewStructElement(NewOptionalType3(KEY_FINAL), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_OVERRIDE), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
	NewStructElement(NewOptionalType3(KEY_GONAME), DefaultStringType()),
})

type function struct {
	annotatedMember
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

func (f *function) CallGo(c eval.Context, receiver interface{}, args ...interface{}) interface{} {
	rv := reflect.ValueOf(receiver)
	rt := rv.Type()
	m, ok := rt.MethodByName(f.goName)
	if !ok {
		panic(eval.Error(eval.EVAL_INSTANCE_DOES_NOT_RESPOND, issue.H{`instance`: rt.String(), `message`: f.goName}))
	}

	mt := m.Type
	pc := mt.NumIn()
	if mt.NumOut() > 1 || pc != 1 + len(args) {
		panic(eval.Error(eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeSignatures(
			[]eval.Signature{f.CallableType().(*CallableType)}, NewTupleType([]eval.Type{}, NewIntegerType(int64(pc), int64(pc))), nil)}))
	}

	rfArgs := make([]reflect.Value, pc)
	rfArgs[0] = rv
	for i, arg := range args {
		rfArgs[i + 1] = reflect.ValueOf(arg)
	}
	result := m.Func.Call(rfArgs)
	if mt.NumOut() == 1 {
		return result[0].Interface()
	}
	return nil
}

func (f *function) GoName() string {
	return f.goName
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
