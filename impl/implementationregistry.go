package impl

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"reflect"
)

type implRegistry struct {
	reflectToObjectType map[string]eval.ObjectType
	objectTypeToReflect map[string]reflect.Type
}

type parentedImplRegistry struct {
	eval.ImplementationRegistry
	implRegistry
}

func NewImplementationRegistry() eval.ImplementationRegistry {
	return &implRegistry{make(map[string]eval.ObjectType, 7), make(map[string]reflect.Type, 7)}
}

func NewParentedImplementationRegistry(parent eval.ImplementationRegistry) eval.ImplementationRegistry {
	return &parentedImplRegistry{parent, implRegistry{make(map[string]eval.ObjectType, 7), make(map[string]reflect.Type, 7)}}
}

func (ir *implRegistry) RegisterType2(c eval.Context, tn string, r reflect.Type) {
	ir.RegisterType(c, loadObjectType(c, tn), r)
}

func (ir *implRegistry) RegisterType(c eval.Context, t eval.ObjectType, r reflect.Type) {
	r = assertUnregisteredStruct(c, ir, t, r)
	ir.addTypeMapping(t, r)
}

func (ir *implRegistry) PTypeToReflected(t eval.ObjectType) (reflect.Type, bool) {
	rt, ok := ir.objectTypeToReflect[t.Name()]
	return rt, ok
}

func (ir *implRegistry) ReflectedToPtype(t reflect.Type) (eval.ObjectType, bool) {
	if t = structType(t); t != nil {
		pt, ok := ir.reflectToObjectType[t.Name()]
		return pt, ok
	}
	return nil, false
}

func (ir *implRegistry) addTypeMapping(t eval.ObjectType, r reflect.Type) {
	ir.objectTypeToReflect[t.Name()] = r
	ir.reflectToObjectType[r.Name()] = t
}

func (pr *parentedImplRegistry) RegisterType(c eval.Context, t eval.ObjectType, r reflect.Type) {
	r = assertUnregisteredStruct(c, pr, t, r)
	pr.addTypeMapping(t, r)
}

func (pr *parentedImplRegistry) RegisterType2(c eval.Context, tn string, r reflect.Type) {
	pr.RegisterType(c, loadObjectType(c, tn), r)
}

func (pr *parentedImplRegistry) PTypeToReflected(t eval.ObjectType) (reflect.Type, bool) {
	rt, ok := pr.ImplementationRegistry.PTypeToReflected(t)
	if !ok {
		rt, ok = pr.implRegistry.PTypeToReflected(t)
	}
	return rt, ok
}

func (pr *parentedImplRegistry) ReflectedToPtype(t reflect.Type) (eval.ObjectType, bool) {
	if t = structType(t); t != nil {
		pt, ok := pr.ImplementationRegistry.ReflectedToPtype(t)
		if !ok {
			pt, ok = pr.implRegistry.ReflectedToPtype(t)
		}
		return pt, ok
	}
	return nil, false
}

func loadObjectType(c eval.Context, tn string) eval.ObjectType {
	if lo, ok := eval.Load(c, eval.NewTypedName(eval.TYPE, tn)); ok {
		if t, ok := lo.(eval.ObjectType); ok {
			return t
		}
		panic(eval.Error(c, eval.EVAL_NOT_OBJECT_TYPE, issue.H{`type`: tn}))
	}
	panic(eval.Error(c, eval.EVAL_UNRESOLVED_TYPE, issue.H{`typeString`: tn}))
}

func assertUnregisteredStruct(c eval.Context, ir eval.ImplementationRegistry, t eval.ObjectType, r reflect.Type) reflect.Type {
	if _, ok := ir.PTypeToReflected(t); ok {
		panic(eval.Error(c, eval.EVAL_IMPL_ALREDY_REGISTERED, issue.H{`type`: t.Name()}))
	}
	if _, ok := ir.ReflectedToPtype(r); ok {
		panic(eval.Error(c, eval.EVAL_IMPL_ALREDY_REGISTERED, issue.H{`type`: r.Name()}))
	}
	if st := structType(r); st != nil {
		return st
	}
	panic(eval.Error(c, eval.EVAL_IMPL_IS_NOT_STRUCT, issue.H{`type`: r.Name()}))
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
