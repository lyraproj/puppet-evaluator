package impl

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
	"github.com/puppetlabs/go-evaluator/types"
)

type implRegistry struct {
	reflectToObjectType map[string]eval.Type
	objectTypeToReflect map[string]reflect.Type
}

type parentedImplRegistry struct {
	eval.ImplementationRegistry
	implRegistry
}

func newImplementationRegistry() eval.ImplementationRegistry {
	return &implRegistry{make(map[string]eval.Type, 7), make(map[string]reflect.Type, 7)}
}

func newParentedImplementationRegistry(parent eval.ImplementationRegistry) eval.ImplementationRegistry {
	return &parentedImplRegistry{parent, implRegistry{make(map[string]eval.Type, 7), make(map[string]reflect.Type, 7)}}
}

func (ir *implRegistry) RegisterType(c eval.Context, t eval.Type, r reflect.Type) {
	r = types.NormalizeType(r)
	r = assertUnregistered(c, ir, t, r)
	ir.addTypeMapping(t, r)
}

func (ir *implRegistry) TypeToReflected(t eval.Type) (reflect.Type, bool) {
	rt, ok := ir.objectTypeToReflect[t.Name()]
	return rt, ok
}

func (ir *implRegistry) ReflectedNameToType(tn string) (eval.Type, bool) {
	pt, ok := ir.reflectToObjectType[tn]
	return pt, ok
}

func (ir *implRegistry) ReflectedToType(t reflect.Type) (eval.Type, bool) {
	return ir.ReflectedNameToType(types.NormalizeType(t).String())
}

func (ir *implRegistry) addTypeMapping(t eval.Type, r reflect.Type) {
	ir.objectTypeToReflect[t.Name()] = r
	ir.reflectToObjectType[r.String()] = t
}

func (pr *parentedImplRegistry) RegisterType(c eval.Context, t eval.Type, r reflect.Type) {
	r = types.NormalizeType(r)
	r = assertUnregistered(c, pr, t, r)
	pr.addTypeMapping(t, r)
}

func (pr *parentedImplRegistry) TypeToReflected(t eval.Type) (reflect.Type, bool) {
	rt, ok := pr.ImplementationRegistry.TypeToReflected(t)
	if !ok {
		rt, ok = pr.implRegistry.TypeToReflected(t)
	}
	return rt, ok
}

func (pr *parentedImplRegistry) ReflectedNameToType(tn string) (eval.Type, bool) {
	pt, ok := pr.ImplementationRegistry.ReflectedNameToType(tn)
	if !ok {
		pt, ok = pr.implRegistry.ReflectedNameToType(tn)
	}
	return pt, ok
}

func (pr *parentedImplRegistry) ReflectedToType(t reflect.Type) (eval.Type, bool) {
	return pr.ReflectedNameToType(types.NormalizeType(t).String())
}

func assertUnregistered(c eval.Context, ir eval.ImplementationRegistry, t eval.Type, r reflect.Type) reflect.Type {
	if rt, ok := ir.TypeToReflected(t); ok {
		if r.String() != rt.String() {
			panic(eval.Error(eval.EVAL_IMPL_ALREDY_REGISTERED, issue.H{`type`: t}))
		}
	}
	if tn, ok := ir.ReflectedToType(r); ok {
		if tn != t {
			panic(eval.Error(eval.EVAL_IMPL_ALREDY_REGISTERED, issue.H{`type`: r.String()}))
		}
	}
	return r
}
