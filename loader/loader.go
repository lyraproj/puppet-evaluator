package loader

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"reflect"
	"sort"
	"sync"
)

type (
	loaderEntry struct {
		value  interface{}
		origin issue.Location
	}

	basicLoader struct {
		lock         sync.RWMutex
		namedEntries map[string]eval.LoaderEntry
	}

	parentedLoader struct {
		basicLoader
		parent eval.Loader
	}

	typeSetLoader struct {
		parentedLoader
		typeSet eval.TypeSet
	}
)

var staticLoader = &basicLoader{namedEntries: make(map[string]eval.LoaderEntry, 64)}

func init() {
	sh := staticLoader.namedEntries
	types.EachCoreType(func(t eval.Type) {
		sh[types.NewTypedName(eval.NsType, t.Name()).MapKey()] = &loaderEntry{t, nil}
	})

	eval.StaticLoader = func() eval.Loader {
		return staticLoader
	}

	eval.NewParentedLoader = func(parent eval.Loader) eval.DefiningLoader {
		return &parentedLoader{basicLoader{namedEntries: make(map[string]eval.LoaderEntry, 64)}, parent}
	}

	eval.NewTypeSetLoader = func(parent eval.Loader, typeSet eval.Type) eval.TypeSetLoader {
		return &typeSetLoader{parentedLoader{basicLoader{namedEntries: make(map[string]eval.LoaderEntry, 64)}, parent}, typeSet.(eval.TypeSet)}
	}

	eval.NewLoaderEntry = func(value interface{}, origin issue.Location) eval.LoaderEntry {
		return &loaderEntry{value, origin}
	}

	eval.Load = load
}

func (e *loaderEntry) Origin() issue.Location {
	return e.origin
}

func (e *loaderEntry) Value() interface{} {
	return e.value
}

func load(c eval.Context, name eval.TypedName) (interface{}, bool) {
	l := c.Loader()
	if name.Authority() != l.NameAuthority() {
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

func (l *basicLoader) Discover(c eval.Context, predicate func(tn eval.TypedName) bool) []eval.TypedName {
	found := make([]eval.TypedName, 0)
	for k, _ := range l.namedEntries {
		tn := eval.TypedNameFromMapKey(k)
		if predicate(tn) {
			found = append(found, tn)
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].MapKey() < found[j].MapKey() })
	return found
}

func (l *basicLoader) LoadEntry(c eval.Context, name eval.TypedName) eval.LoaderEntry {
	return l.GetEntry(name)
}

func (l *basicLoader) GetEntry(name eval.TypedName) eval.LoaderEntry {
	l.lock.RLock()
	v := l.namedEntries[name.MapKey()]
	l.lock.RUnlock()
	return v
}

func (l *basicLoader) HasEntry(name eval.TypedName) bool {
	l.lock.RLock()
	e, found := l.namedEntries[name.MapKey()]
	l.lock.RUnlock()
	return found && e.Value() != nil
}

func (l *basicLoader) SetEntry(name eval.TypedName, entry eval.LoaderEntry) eval.LoaderEntry {
	l.lock.Lock()
	if old, ok := l.namedEntries[name.MapKey()]; ok && old.Value() != nil {
		l.lock.Unlock()
		if reflect.ValueOf(old.Value()).Pointer() == reflect.ValueOf(entry.Value()).Pointer() {
			return old
		}
		panic(eval.Error(eval.EVAL_ATTEMPT_TO_REDEFINE, issue.H{`name`: name}))
	}
	l.namedEntries[name.MapKey()] = entry
	l.lock.Unlock()
	return entry
}

func (l *basicLoader) NameAuthority() eval.URI {
	return eval.RUNTIME_NAME_AUTHORITY
}

func (l *parentedLoader) Discover(c eval.Context, predicate func(tn eval.TypedName) bool) []eval.TypedName {
	found := l.parent.Discover(c, predicate)
	added := false
	for k, _ := range l.namedEntries {
		tn := eval.TypedNameFromMapKey(k)
		if !l.parent.HasEntry(tn) {
			if predicate(tn) {
				found = append(found, tn)
				added = true
			}
		}
	}
	if added {
		sort.Slice(found, func(i, j int) bool { return found[i].MapKey() < found[j].MapKey() })
	}
	return found
}

func (l *parentedLoader) HasEntry(name eval.TypedName) bool {
	return l.parent.HasEntry(name) || l.basicLoader.HasEntry(name)
}

func (l *parentedLoader) LoadEntry(c eval.Context, name eval.TypedName) eval.LoaderEntry {
	entry := l.parent.LoadEntry(c, name)
	if entry == nil || entry.Value() == nil {
		entry = l.basicLoader.LoadEntry(c, name)
	}
	return entry
}

func (l *parentedLoader) NameAuthority() eval.URI {
	return l.parent.NameAuthority()
}

func (l *parentedLoader) Parent() eval.Loader {
	return l.parent
}

func (l *typeSetLoader) Discover(c eval.Context, predicate func(tn eval.TypedName) bool) []eval.TypedName {
	found := make([]eval.TypedName, 0)
	ts := l.typeSet.Types()
	ts.EachKey(func(v eval.Value) {
		tn := v.(eval.TypedName)
		if predicate(tn) {
			found = append(found, tn)
		}
	})

	pf := l.parentedLoader.Discover(c, func(tn eval.TypedName) bool { return !ts.IncludesKey(tn) && predicate(tn) })
	if len(pf) > 0 {
		found = append(found, pf...)
		sort.Slice(found, func(i, j int) bool { return found[i].MapKey() < found[j].MapKey() })
	}
	return found
}

func (l *typeSetLoader) HasEntry(name eval.TypedName) bool {
	if _, ok := l.typeSet.GetType(name); ok {
		return true
	}
	if l.parentedLoader.HasEntry(name) {
		return true
	}
	if child, ok := name.RelativeTo(l.typeSet.TypedName()); ok {
		return l.HasEntry(child)
	}
	return false
}

func (l *typeSetLoader) LoadEntry(c eval.Context, name eval.TypedName) eval.LoaderEntry {
	if tp, ok := l.typeSet.GetType(name); ok {
		return &loaderEntry{tp, nil}
	}
	entry := l.parentedLoader.LoadEntry(c, name)
	if entry == nil {
		if child, ok := name.RelativeTo(l.typeSet.TypedName()); ok {
			return l.LoadEntry(c, child)
		}
		entry = &loaderEntry{nil, nil}
		l.parentedLoader.SetEntry(name, entry)
	}
	return entry
}

func (l *typeSetLoader) SetEntry(name eval.TypedName, entry eval.LoaderEntry) eval.LoaderEntry {
	return l.parent.(eval.DefiningLoader).SetEntry(name, entry)
}

func (l *typeSetLoader) TypeSet() eval.Type {
	return l.typeSet
}
