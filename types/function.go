package types

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
)

var TYPE_FUNCTION_TYPE = NewTypeType(DefaultCallableType())

var TYPE_FUNCTION = NewStructType([]*StructElement{
	NewStructElement2(KEY_TYPE, TYPE_FUNCTION_TYPE),
	NewStructElement(NewOptionalType3(KEY_FINAL), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_OVERRIDE), DefaultBooleanType()),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS),
})

type function struct {
	annotatedMember
}

func newFunction(c eval.Context, name string, container *objectType, initHash *HashValue) eval.ObjFunc {
	f := &function{}
	f.initialize(c, name, container, initHash)
	return f
}

func (f *function) initialize(c eval.Context, name string, container *objectType, initHash *HashValue) {
	eval.AssertInstance(c, func() string { return fmt.Sprintf(`initializer function for %s[%s]`, container.Label(), name) }, TYPE_FUNCTION, initHash)
	f.annotatedMember.initialize(c, `function`, name, container, initHash)
}

func (a *function) Call(c eval.Context, receiver eval.PValue, block eval.Lambda, args []eval.PValue) eval.PValue {
	if a.CallableType().(*CallableType).CallableWith(c, args, block) {
		if co, ok := receiver.(eval.CallableObject); ok {
			if result, ok := co.Call(c, a.name, args, block); ok {
				return result
			}
		}
		panic(eval.Error(c, eval.EVAL_INSTANCE_DOES_NOT_RESPOND, issue.H{`instance`: receiver, `message`: a.name}))
	}
	panic(eval.Error(c, eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeSignatures(
		[]eval.Signature{a.CallableType().(*CallableType)}, NewTupleType2(args...), block)}))
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

func (f *function) CallableType() eval.PType {
	return f.typ.(*CallableType)
}

func (f *function) InitHash() eval.KeyedValue {
	return WrapStringPValue(f.initHash())
}
