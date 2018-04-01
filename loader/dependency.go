package loader

import "github.com/puppetlabs/go-evaluator/eval"

type dependencyLoader struct {
	basicLoader
	loaders []eval.ModuleLoader
	index   map[string]eval.ModuleLoader
}

func newDependencyLoader(loaders []eval.ModuleLoader) eval.Loader {
	index := make(map[string]eval.ModuleLoader, len(loaders))
	for _, ml := range loaders {
		index[ml.ModuleName()] = ml
	}
	return &dependencyLoader{
		basicLoader: basicLoader{namedEntries: make(map[string]eval.Entry, 32)},
		loaders:     loaders,
		index:       index}
}

func init() {
	eval.NewDependencyLoader = newDependencyLoader
}

func (l *dependencyLoader) LoadEntry(name eval.TypedName) eval.Entry {
	entry := l.basicLoader.LoadEntry(name)
	if entry == nil {
		entry = l.find(name)
		if entry == nil {
			entry = &loaderEntry{nil, nil}
		}
		l.SetEntry(name, entry)
	}
	return entry
}

func (l *dependencyLoader) LoaderFor(moduleName string) eval.ModuleLoader {
	return l.index[moduleName]
}

func (l *dependencyLoader) find(name eval.TypedName) eval.Entry {
	if name.IsQualified() {
		if ml, ok := l.index[name.NameParts()[0]]; ok {
			return ml.LoadEntry(name)
		}
		return nil
	}

	for _, ml := range l.loaders {
		e := ml.LoadEntry(name)
		if !(e == nil || e.Value() == nil) {
			return e
		}
	}
	return nil
}
