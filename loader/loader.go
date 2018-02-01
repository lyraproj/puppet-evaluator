package loader

import (
	"fmt"
	"sync"

	. "github.com/puppetlabs/go-evaluator/eval"
	. "github.com/puppetlabs/go-evaluator/types"
)

type (
	loaderEntry struct {
		value  interface{}
		origin string
	}

	basicLoader struct {
		lock         sync.RWMutex
		namedEntries map[string]Entry
	}

	parentedLoader struct {
		basicLoader
		parent Loader
	}
)

var staticLoader = &basicLoader{namedEntries: make(map[string]Entry, 64)}
var resolvableConstructors = make([]ResolvableFunction, 0, 16)
var resolvableFunctions = make([]ResolvableFunction, 0, 16)
var resolvableFunctionsLock sync.Mutex

func init() {
	StaticLoader = func() Loader {
		return staticLoader
	}

	NewParentedLoader = func(parent Loader) DefiningLoader {
		return &parentedLoader{basicLoader{namedEntries: make(map[string]Entry, 64)}, parent}
	}

	RegisterGoFunction = func(function ResolvableFunction) {
		resolvableFunctionsLock.Lock()
		resolvableFunctions = append(resolvableFunctions, function)
		resolvableFunctionsLock.Unlock()
	}

	RegisterGoConstructor = func(function ResolvableFunction) {
		resolvableFunctionsLock.Lock()
		resolvableConstructors = append(resolvableConstructors, function)
		resolvableFunctionsLock.Unlock()
	}

	NewLoaderEntry = func(value interface{}, origin string) Entry {
		return &loaderEntry{value, origin}
	}

	Load = load
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

func (e *loaderEntry) Origin() string {
	return e.origin
}

func (e *loaderEntry) Value() interface{} {
	return e.value
}

func (l *basicLoader) ResolveResolvables(c EvalContext) {
	types := PopDeclaredTypes()
	for _, t := range types {
		l.SetEntry(NewTypedName(TYPE, t.Name()), &loaderEntry{t, ``})
	}

	for _, t := range types {
		t.Resolve(c)
	}

	funcs, ctors := popDeclaredGoFunctions()
	for _, rf := range funcs {
		l.SetEntry(NewTypedName(FUNCTION, rf.Name()), &loaderEntry{rf.Resolve(c), ``})
	}
	for _, ct := range ctors {
		l.SetEntry(NewTypedName(CONSTRUCTOR, ct.Name()), &loaderEntry{ct.Resolve(c), ``})
	}
}

func load(l Loader, name TypedName) (interface{}, bool) {
	if name.NameAuthority() != l.NameAuthority() {
		return nil, false
	}
	entry := l.LoadEntry(name)
	if entry == nil {
		if dl, ok := l.(DefiningLoader); ok {
			dl.SetEntry(name, &loaderEntry{nil, ``})
		}
		return nil, false
	}
	if entry.Value() == nil {
		return nil, false
	}
	return entry.Value(), true
}

func (l *basicLoader) LoadEntry(name TypedName) Entry {
	return l.GetEntry(name)
}

func (l *basicLoader) GetEntry(name TypedName) Entry {
	l.lock.RLock()
	v := l.namedEntries[name.MapKey()]
	l.lock.RUnlock()
	return v
}

func (l *basicLoader) SetEntry(name TypedName, entry Entry) Entry {
	l.lock.Lock()
	_, ok := l.namedEntries[name.MapKey()]
	if ok {
		l.lock.Unlock()
		panic(fmt.Sprintf(`Attempt to redefine %s`, name.String()))
	}
	l.namedEntries[name.MapKey()] = entry
	l.lock.Unlock()
	return entry
}

func (l *basicLoader) NameAuthority() URI {
	return RUNTIME_NAME_AUTHORITY
}

func (l *parentedLoader) LoadEntry(name TypedName) Entry {
	entry := l.parent.LoadEntry(name)
	if entry == nil || entry.Value() == nil {
		entry = l.basicLoader.LoadEntry(name)
	}
	return entry
}

func (l *parentedLoader) NameAuthority() URI {
	return l.parent.NameAuthority()
}
