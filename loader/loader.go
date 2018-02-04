package loader

import (
	"fmt"
	"sync"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

type (
	loaderEntry struct {
		value  interface{}
		origin string
	}

	basicLoader struct {
		lock         sync.RWMutex
		namedEntries map[string]eval.Entry
	}

	parentedLoader struct {
		basicLoader
		parent eval.Loader
	}
)

var staticLoader = &basicLoader{namedEntries: make(map[string]eval.Entry, 64)}
var resolvableConstructors = make([]eval.ResolvableFunction, 0, 16)
var resolvableFunctions = make([]eval.ResolvableFunction, 0, 16)
var resolvableFunctionsLock sync.Mutex

func init() {
	eval.StaticLoader = func() eval.Loader {
		return staticLoader
	}

	eval.NewParentedLoader = func(parent eval.Loader) eval.DefiningLoader {
		return &parentedLoader{basicLoader{namedEntries: make(map[string]eval.Entry, 64)}, parent}
	}

	eval.RegisterGoFunction = func(function eval.ResolvableFunction) {
		resolvableFunctionsLock.Lock()
		resolvableFunctions = append(resolvableFunctions, function)
		resolvableFunctionsLock.Unlock()
	}

	eval.RegisterGoConstructor = func(function eval.ResolvableFunction) {
		resolvableFunctionsLock.Lock()
		resolvableConstructors = append(resolvableConstructors, function)
		resolvableFunctionsLock.Unlock()
	}

	eval.NewLoaderEntry = func(value interface{}, origin string) eval.Entry {
		return &loaderEntry{value, origin}
	}

	eval.Load = load
}

func popDeclaredGoFunctions() (funcs []eval.ResolvableFunction, ctors []eval.ResolvableFunction) {
	resolvableFunctionsLock.Lock()
	funcs = resolvableFunctions
	if len(funcs) > 0 {
		resolvableFunctions = make([]eval.ResolvableFunction, 0, 16)
	}
	ctors = resolvableConstructors
	if len(ctors) > 0 {
		resolvableConstructors = make([]eval.ResolvableFunction, 0, 16)
	}
	resolvableFunctionsLock.Unlock()
	return
}

func (e *loaderEntry) Origin() string {
	return e.origin
}

func (e *loaderEntry) Value() interface{} {
	return e.value
}

func (l *basicLoader) ResolveResolvables(c eval.EvalContext) {
	types := types.PopDeclaredTypes()
	for _, t := range types {
		l.SetEntry(eval.NewTypedName(eval.TYPE, t.Name()), &loaderEntry{t, ``})
	}

	for _, t := range types {
		t.(eval.ResolvableType).Resolve(c)
	}

	funcs, ctors := popDeclaredGoFunctions()
	for _, rf := range funcs {
		l.SetEntry(eval.NewTypedName(eval.FUNCTION, rf.Name()), &loaderEntry{rf.Resolve(c), ``})
	}
	for _, ct := range ctors {
		l.SetEntry(eval.NewTypedName(eval.CONSTRUCTOR, ct.Name()), &loaderEntry{ct.Resolve(c), ``})
	}
}

func load(l eval.Loader, name eval.TypedName) (interface{}, bool) {
	if name.NameAuthority() != l.NameAuthority() {
		return nil, false
	}
	entry := l.LoadEntry(name)
	if entry == nil {
		if dl, ok := l.(eval.DefiningLoader); ok {
			dl.SetEntry(name, &loaderEntry{nil, ``})
		}
		return nil, false
	}
	if entry.Value() == nil {
		return nil, false
	}
	return entry.Value(), true
}

func (l *basicLoader) LoadEntry(name eval.TypedName) eval.Entry {
	return l.GetEntry(name)
}

func (l *basicLoader) GetEntry(name eval.TypedName) eval.Entry {
	l.lock.RLock()
	v := l.namedEntries[name.MapKey()]
	l.lock.RUnlock()
	return v
}

func (l *basicLoader) SetEntry(name eval.TypedName, entry eval.Entry) eval.Entry {
	l.lock.Lock()
	if old, ok := l.namedEntries[name.MapKey()]; ok && old.Value() != nil {
		l.lock.Unlock()
		panic(fmt.Sprintf(`Attempt to redefine %s`, name.String()))
	}
	l.namedEntries[name.MapKey()] = entry
	l.lock.Unlock()
	return entry
}

func (l *basicLoader) NameAuthority() eval.URI {
	return eval.RUNTIME_NAME_AUTHORITY
}

func (l *parentedLoader) LoadEntry(name eval.TypedName) eval.Entry {
	entry := l.parent.LoadEntry(name)
	if entry == nil || entry.Value() == nil {
		entry = l.basicLoader.LoadEntry(name)
	}
	return entry
}

func (l *parentedLoader) NameAuthority() eval.URI {
	return l.parent.NameAuthority()
}
