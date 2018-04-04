package loader

import (
	"fmt"
	"sync"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
)

type (
	loaderEntry struct {
		value  interface{}
		origin issue.Location
	}

	basicLoader struct {
		lock         sync.RWMutex
		namedEntries map[string]eval.Entry
	}

	parentedLoader struct {
		basicLoader
		parent eval.Loader
	}

	typeSetLoader struct {
		parentedLoader
		typeSet *types.TypeSetType
	}
)

var staticLoader = &basicLoader{namedEntries: make(map[string]eval.Entry, 64)}
var staticResourceLoader = &parentedLoader{basicLoader:basicLoader{namedEntries: make(map[string]eval.Entry, 64)}, parent:staticLoader}
var resolvableFunctions = make([]eval.ResolvableFunction, 0, 16)
var resolvableFunctionsLock sync.Mutex

func init() {
	eval.StaticLoader = func() eval.Loader {
		return staticLoader
	}

	eval.StaticResourceLoader = func() eval.Loader {
		return staticResourceLoader
	}

	eval.NewParentedLoader = func(parent eval.Loader) eval.DefiningLoader {
		return &parentedLoader{basicLoader{namedEntries: make(map[string]eval.Entry, 64)}, parent}
	}

	eval.NewTypeSetLoader = func(parent eval.Loader, typeSet eval.PType) eval.TypeSetLoader {
		return &typeSetLoader{parentedLoader{basicLoader{namedEntries: make(map[string]eval.Entry, 64)}, parent}, typeSet.(*types.TypeSetType)}
	}

	eval.RegisterGoFunction = func(function eval.ResolvableFunction) {
		resolvableFunctionsLock.Lock()
		resolvableFunctions = append(resolvableFunctions, function)
		resolvableFunctionsLock.Unlock()
	}

	eval.NewLoaderEntry = func(value interface{}, origin issue.Location) eval.Entry {
		return &loaderEntry{value, origin}
	}

	eval.Load = load
}

func popDeclaredGoFunctions() (funcs []eval.ResolvableFunction) {
	resolvableFunctionsLock.Lock()
	funcs = resolvableFunctions
	if len(funcs) > 0 {
		resolvableFunctions = make([]eval.ResolvableFunction, 0, 16)
	}
	resolvableFunctionsLock.Unlock()
	return
}

func (e *loaderEntry) Origin() issue.Location {
	return e.origin
}

func (e *loaderEntry) Value() interface{} {
	return e.value
}

func (l *basicLoader) ResolveResolvables(c eval.EvalContext) {
	ts := types.PopDeclaredTypes()
	for _, t := range ts {
		l.SetEntry(eval.NewTypedName(eval.TYPE, t.Name()), &loaderEntry{t, nil})
	}

	for _, t := range ts {
		t.(eval.ResolvableType).Resolve(c)
	}

	ctors := types.PopDeclaredConstructors()
	for _, ct := range ctors {
		rf := eval.BuildFunction(ct.Name, ct.LocalTypes, ct.Creators)
		l.SetEntry(eval.NewTypedName(eval.CONSTRUCTOR, rf.Name()), &loaderEntry{rf.Resolve(c), nil})
	}

	funcs := popDeclaredGoFunctions()
	for _, rf := range funcs {
		l.SetEntry(eval.NewTypedName(eval.FUNCTION, rf.Name()), &loaderEntry{rf.Resolve(c), nil})
	}
}

func load(c eval.EvalContext, name eval.TypedName) (interface{}, bool) {
	l := c.Loader()
	if name.NameAuthority() != l.NameAuthority() {
		return nil, false
	}
	entry := l.LoadEntry(c, name)
	if entry == nil {
		if dl, ok := l.(eval.DefiningLoader); ok {
			dl.SetEntry(name, &loaderEntry{nil, nil})
		}
		return nil, false
	}
	if entry.Value() == nil {
		return nil, false
	}
	return entry.Value(), true
}

func (l *basicLoader) LoadEntry(c eval.EvalContext, name eval.TypedName) eval.Entry {
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

func (l *parentedLoader) LoadEntry(c eval.EvalContext, name eval.TypedName) eval.Entry {
	entry := l.parent.LoadEntry(c, name)
	if entry == nil || entry.Value() == nil {
		entry = l.basicLoader.LoadEntry(c, name)
	}
	return entry
}

func (l *parentedLoader) NameAuthority() eval.URI {
	return l.parent.NameAuthority()
}

func (l *typeSetLoader) TypeSet() eval.PType {
	return l.typeSet
}


func (l *typeSetLoader) LoadEntry(c eval.EvalContext, name eval.TypedName) eval.Entry {
	entry := l.parentedLoader.LoadEntry(c, name)
	if entry == nil {
		entry = l.find(c, name)
		if entry == nil {
			entry = &loaderEntry{nil, nil}
			l.parentedLoader.SetEntry(name, entry)
		}
	}
	return entry
}

func (l *typeSetLoader) SetEntry(name eval.TypedName, entry eval.Entry) eval.Entry {
	return l.parent.(eval.DefiningLoader).SetEntry(name, entry)
}

func (l *typeSetLoader) find(c eval.EvalContext, name eval.TypedName) eval.Entry {
	if tp, ok := l.typeSet.GetType(name); ok {
		return l.SetEntry(name, &loaderEntry{tp, nil})
	}
	return nil
}
