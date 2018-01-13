package loader

import . "github.com/puppetlabs/go-evaluator/evaluator"

type dependencyLoader struct {
	basicLoader
	loaders []ModuleLoader
	index map[string]ModuleLoader
}

func newDependencyLoader(loaders []ModuleLoader) Loader {
	index := make(map[string]ModuleLoader, len(loaders))
	for _, ml := range loaders {
		index[ml.ModuleName()] = ml
	}
	return &dependencyLoader{
		basicLoader: basicLoader{namedEntries: make(map[string]Entry, 32)},
		loaders: loaders,
		index: index}
}

func init() {
	NewDependencyLoader = newDependencyLoader
}

func (l *dependencyLoader) LoadEntry(name TypedName) Entry {
	entry := l.basicLoader.LoadEntry(name)
	if entry == nil {
		entry = l.find(name)
		if entry == nil {
			entry = &loaderEntry{nil, ``}
		}
		l.SetEntry(name, entry)
	}
	return entry
}

func (l *dependencyLoader) LoaderFor(moduleName string) ModuleLoader {
	return l.index[moduleName]
}

func (l *dependencyLoader) find(name TypedName) Entry {
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
