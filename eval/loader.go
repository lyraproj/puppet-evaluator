package eval

import (
	"fmt"
	"sync"

	. "github.com/puppetlabs/go-evaluator/eval/evaluator"
	. "github.com/puppetlabs/go-evaluator/eval/utils"
)

type (
	ResolvableFunction interface {
		Name() string
		Resolve(c EvalContext) Function
	}

	basicLoader struct {
		namedEntries map[string]interface{}
	}

	parentedLoader struct {
		basicLoader
		parent Loader
	}
)

var staticLoader = &basicLoader{make(map[string]interface{}, 64)}
var resolvableConstructors = make([]ResolvableFunction, 0, 16)
var resolvableFunctions = make([]ResolvableFunction, 0, 16)
var resolvableFunctionsLock sync.Mutex

func StaticLoader() Loader {
	return staticLoader
}

func RegisterGoFunction(function ResolvableFunction) {
	resolvableFunctionsLock.Lock()
	resolvableFunctions = append(resolvableFunctions, function)
	resolvableFunctionsLock.Unlock()
}

func RegisterGoConstructor(function ResolvableFunction) {
	resolvableFunctionsLock.Lock()
	resolvableConstructors = append(resolvableConstructors, function)
	resolvableFunctionsLock.Unlock()
}

func popDeclaredGoFunctions() (funcs []ResolvableFunction, ctors []ResolvableFunction) {
	resolvableFunctionsLock.Lock()
	funcs = resolvableFunctions
	if len(funcs) > 0 {
		resolvableFunctions = make([]ResolvableFunction, 0, 16)
	}
	ctors = resolvableConstructors
	if len(ctors) > 0 {
		resolvableConstructors = make([]ResolvableFunction, 0, 16)
	}
	resolvableFunctionsLock.Unlock()
	return
}

func (l *basicLoader) ResolveGoFunctions(c EvalContext) {
	funcs, ctors := popDeclaredGoFunctions()
	for _, rf := range funcs {
		l.SetEntry(NewTypedName(FUNCTION, rf.Name()), rf.Resolve(c))
	}
	for _, ct := range ctors {
		l.SetEntry(NewTypedName(CONSTRUCTOR, ct.Name()), ct.Resolve(c))
	}
}

func NewParentedLoader(parent Loader) DefiningLoader {
	return &parentedLoader{basicLoader{make(map[string]interface{}, 64)}, parent}
}

func (l *basicLoader) Load(name TypedName) (found interface{}, ok bool) {
	found, ok = l.namedEntries[name.MapKey()]
	return
}

func (l *basicLoader) SetEntry(name TypedName, value interface{}) {
	if _, ok := l.namedEntries[name.MapKey()]; ok {
		panic(fmt.Sprintf(`Attempt to redefine %s`, name.String()))
	}
	l.namedEntries[name.MapKey()] = value
}

func (l *basicLoader) NameAuthority() URI {
	return RUNTIME_NAME_AUTHORITY
}

func (l *parentedLoader) Load(name TypedName) (found interface{}, ok bool) {
	found, ok = l.parent.Load(name)
	if !ok {
		found, ok = l.basicLoader.Load(name)
	}
	return
}

func (l *parentedLoader) SetEntry(name TypedName, value interface{}) {
	l.basicLoader.SetEntry(name, value)
}

func (l *parentedLoader) NameAuthority() URI {
	return l.parent.NameAuthority()
}
