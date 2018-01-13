package loader

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"fmt"
	"github.com/puppetlabs/go-evaluator/errors"
	"path/filepath"
	"os"
	"strings"
	"io/ioutil"
	"github.com/puppetlabs/go-evaluator/types"
)

type (
	ContentProvidingLoader interface {
		Loader

		GetContent(path string) []byte
	}

	fileBasedLoader struct {
		parentedLoader
		path string
		moduleName string
		initPlanName TypedName
		initTaskName TypedName
		initTypeSetName TypedName
		paths map[Namespace][]SmartPath
		index map[string][]string
	}
)

func init() {
	NewFilebasedLoader = newFileBasedLoader
}

func newFileBasedLoader(parent Loader, path, moduleName string, loadables ...PathType) ModuleLoader {
	paths := make(map[Namespace][]SmartPath, len(loadables))
	loader := &fileBasedLoader{
		parentedLoader: parentedLoader{
			basicLoader: basicLoader{namedEntries: make(map[string]Entry, 64)},
			parent: parent},
		path: path,
		initPlanName: NewTypedName2(PLAN, `init`, parent.NameAuthority()),
		initTaskName: NewTypedName2(TASK, `init`, parent.NameAuthority()),
		initTypeSetName: NewTypedName2(TYPE, `init_typeset`, parent.NameAuthority()),
		moduleName: moduleName, paths: paths}

	for _, p := range loadables {
		path := loader.newSmartPath(p, !(moduleName == `` || moduleName == `environment`))
		if sa, ok := paths[path.Namespace()]; ok {
			paths[path.Namespace()] = append(sa, path)
		} else {
			paths[path.Namespace()] = []SmartPath{path}
		}
		loader.addToIndex(path)
	}
	return loader
}

func (l *fileBasedLoader) newSmartPath(pathType PathType, moduleNameRelative bool) SmartPath {
	switch pathType {
	case PUPPET_FUNCTION_PATH:
		return l.newPuppetFunctionPath(moduleNameRelative)
	case PUPPET_DATA_TYPE_PATH:
		return l.newPuppetTypePath(moduleNameRelative)
	case PLAN_PATH:
		return l.newPlanPath(moduleNameRelative)
	case TASK_PATH:
		return l.newTaskPath(moduleNameRelative)
	default:
		panic(errors.NewIllegalArgument(`newSmartPath`, 1, fmt.Sprintf(`Unknown path type '%s'`, pathType)))
	}
}

func (l *fileBasedLoader) newPuppetFunctionPath(moduleNameRelative bool) SmartPath {
	return &smartPath{
		relativePath: `functions`,
		loader: l,
		namespace: FUNCTION,
		extension: `.pp`,
		moduleNameRelative: moduleNameRelative,
		matchMany: false,
		instantiator: InstantiatePuppetFunction,
	}
}

func (l *fileBasedLoader) newPlanPath(moduleNameRelative bool) SmartPath {
	return &smartPath{
		relativePath: `plans`,
		loader: l,
		namespace: PLAN,
		extension: `.pp`,
		moduleNameRelative: moduleNameRelative,
		matchMany: false,
		instantiator: InstantiatePuppetTask,
	}
}

func (l *fileBasedLoader) newPuppetTypePath(moduleNameRelative bool) SmartPath {
	return &smartPath{
		relativePath: `types`,
		loader: l,
		namespace: TYPE,
		extension: `.pp`,
		moduleNameRelative: moduleNameRelative,
		matchMany: false,
		instantiator: InstantiatePuppetType,
	}
}

func (l *fileBasedLoader) newTaskPath(moduleNameRelative bool) SmartPath {
	return &smartPath{
		relativePath: `tasks`,
		loader: l,
		namespace: TASK,
		extension: ``,
		moduleNameRelative: moduleNameRelative,
		matchMany: true,
		instantiator: InstantiatePuppetTask,
	}
}

func (l *fileBasedLoader) LoadEntry(name TypedName) Entry {
	entry := l.parentedLoader.LoadEntry(name)
	if entry == nil {
		entry = l.find(name)
		if entry == nil {
			entry = &loaderEntry{nil, ``}
			l.SetEntry(name, entry)
		}
	}
	return entry
}

func (l *fileBasedLoader) ModuleName() string {
	return l.moduleName
}

func (l *fileBasedLoader) isGlobal() bool {
	return l.moduleName == `` || l.moduleName == `environment`
}

func (l *fileBasedLoader) find(name TypedName) Entry {
	if name.IsQualified() {
		// The name is in a name space.
		if l.moduleName != name.NameParts()[0] {
			// Then entity cannot possible be in this module unless the name starts with the module name.
			// Note: If "module" represents a "global component", the module_name is empty and cannot match which is
			// ok since such a "module" cannot have namespaced content).
			return nil
		}
	} else {
		// The name is in the global name space.
		switch name.Namespace() {
		case FUNCTION:
			// Can be defined in module using a global name. No action required
		case PLAN:
			if !l.isGlobal() {
				// Global name must be the name of the module
				if l.moduleName != name.NameParts()[0] {
					// Global name must be the name of the module
					return nil
				}

				// Look for special 'init' plan
				origins, smartPath := l.findExistingPath(l.initPlanName)
				if smartPath == nil {
					return nil
				}
				return l.instantiate(smartPath, name, origins)
			}
		case TASK:
			if !l.isGlobal() {
				// Global name must be the name of the module
				if l.moduleName != name.NameParts()[0] {
					// Global name must be the name of the module
					return nil
				}

				// Look for special 'init' task
				origins, smartPath := l.findExistingPath(l.initTaskName)
				if smartPath == nil {
					return nil
				}
				return l.instantiate(smartPath, name, origins)
			}
		case TYPE:
			if !l.isGlobal() {
				// Global name must be the name of the module
				if l.moduleName != name.NameParts()[0] {
					// Global name must be the name of the module
					return nil
				}

				// Look for special 'init_typeset' TypeSet
				origins, smartPath := l.findExistingPath(l.initTypeSetName)
				if smartPath == nil {
					return nil
				}
				smartPath.Instantiator()(l, name, origins)
				entry := l.GetEntry(name)
				if entry != nil {
					if _, ok := entry.Value().(*types.TypeSetType); ok {
						return entry
					}
				}
				GeneralFailure(fmt.Sprintf(`The code loaded from %s does not define the TypeSet '%s'`, origins[0], l.moduleName))
			}
		default:
			return nil
		}
	}

	origins, smartPath := l.findExistingPath(name)
	if smartPath != nil {
		return l.instantiate(smartPath, name, origins)
	}

	if !(name.Namespace() == TYPE && name.IsQualified()) {
		return nil
	}

	// Search for TypeSet using parent name
	tsName := name.Parent()
	for tsName != nil {
		tse := l.GetEntry(tsName)
		if tse == nil {
			tse = l.find(tsName)
		}
		if tse != nil && tse.Value() != nil {
			if ts, ok := tse.Value().(*types.TypeSetType); ok {
				ts.Resolve(l)
				te := l.GetEntry(name)
				if te != nil {
					return te
				}
			}
		}
		tsName = tsName.Parent()
	}
	return nil
}

func (l *fileBasedLoader) findExistingPath(name TypedName) (origins []string, smartPath SmartPath) {
	if paths, ok := l.paths[name.Namespace()]; ok {
		for _, sm := range paths {
			if paths, ok := l.index[name.MapKey()]; ok {
				return paths, sm
			}
		}
	}
	return nil, nil
}

func (l *fileBasedLoader) instantiate(smartPath SmartPath, name TypedName, origins []string) Entry {
	smartPath.Instantiator()(l, name, origins)
	return l.GetEntry(name)
}

func (l *fileBasedLoader) GetContent(path string) []byte {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		GeneralFailure(err.Error())
	}
	return content
}

func (l *fileBasedLoader) addToIndex(smartPath SmartPath) {
	if l.index == nil {
		l.index = make(map[string][]string, 64)
	}
	ext := smartPath.Extension()
	noExtension := ext == ``

	generic := smartPath.GenericPath()
	filepath.Walk(generic, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if noExtension || strings.HasSuffix(path, ext) {
				rel, err :=  filepath.Rel(generic, path)
				if err == nil {
					tn := smartPath.TypedName(l.NameAuthority(), rel)
					if tn != nil {
						if paths, ok := l.index[tn.MapKey()]; ok {
							l.index[tn.MapKey()] = append(paths, path)
						} else {
							l.index[tn.MapKey()] = []string{path}
						}
					}
				}
			}
		}
		return nil
	})
}