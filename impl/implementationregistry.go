package impl

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
)

type implRegistry struct {
	reflectToObjectType map[string]string
	objectTypeToReflect map[string]reflect.Type
}

type parentedImplRegistry struct {
	eval.ImplementationRegistry
	implRegistry
}

func newImplementationRegistry() eval.ImplementationRegistry {
	return &implRegistry{make(map[string]string, 7), make(map[string]reflect.Type, 7)}
}

func newParentedImplementationRegistry(parent eval.ImplementationRegistry) eval.ImplementationRegistry {
	return &parentedImplRegistry{parent, implRegistry{make(map[string]string, 7), make(map[string]reflect.Type, 7)}}
}

func (ir *implRegistry) RegisterType(c eval.Context, t string, r reflect.Type) {
	r = assertUnregisteredStruct(c, ir, t, r)
	ir.addTypeMapping(t, r)
}

func (ir *implRegistry) PTypeToReflected(t string) (reflect.Type, bool) {
	rt, ok := ir.objectTypeToReflect[t]
	return rt, ok
}

func (ir *implRegistry) ReflectedToPtype(t reflect.Type) (string, bool) {
	if t = structType(t); t != nil {
		pt, ok := ir.reflectToObjectType[t.String()]
		return pt, ok
	}
	return ``, false
}

func (ir *implRegistry) addTypeMapping(t string, r reflect.Type) {
	ir.objectTypeToReflect[t] = r
	ir.reflectToObjectType[r.String()] = t
}

func (pr *parentedImplRegistry) RegisterType(c eval.Context, t string, r reflect.Type) {
	r = assertUnregisteredStruct(c, pr, t, r)
	pr.addTypeMapping(t, r)
}

func (pr *parentedImplRegistry) PTypeToReflected(t string) (reflect.Type, bool) {
	rt, ok := pr.ImplementationRegistry.PTypeToReflected(t)
	if !ok {
		rt, ok = pr.implRegistry.PTypeToReflected(t)
	}
	return rt, ok
}

func (pr *parentedImplRegistry) ReflectedToPtype(t reflect.Type) (string, bool) {
	if t = structType(t); t != nil {
		pt, ok := pr.ImplementationRegistry.ReflectedToPtype(t)
		if !ok {
			pt, ok = pr.implRegistry.ReflectedToPtype(t)
		}
		return pt, ok
	}
	return ``, false
}

func assertUnregisteredStruct(c eval.Context, ir eval.ImplementationRegistry, t string, r reflect.Type) reflect.Type {
	if rt, ok := ir.PTypeToReflected(t); ok {
		if r.String() != rt.String() {
			panic(eval.Error(eval.EVAL_IMPL_ALREDY_REGISTERED, issue.H{`type`: t}))
		}
	}
	if tn, ok := ir.ReflectedToPtype(r); ok {
		if tn != t {
			panic(eval.Error(eval.EVAL_IMPL_ALREDY_REGISTERED, issue.H{`type`: r.String()}))
		}
	}
	if st := structType(r); st != nil {
		return st
	}
	panic(eval.Error(eval.EVAL_IMPL_IS_NOT_STRUCT, issue.H{`type`: r.String()}))
}

func structType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return t
	}
	return nil
}
