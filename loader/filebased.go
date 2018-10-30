package loader

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-issues/issue"
)

type (
	ContentProvidingLoader interface {
		eval.Loader

		GetContent(c eval.Context, path string) []byte
	}

	fileBasedLoader struct {
		parentedLoader
		path            string
		moduleName      string
		initPlanName    eval.TypedName
		initTaskName    eval.TypedName
		initTypeSetName eval.TypedName
		paths           map[eval.Namespace][]SmartPath
		index           map[string][]string
	}
)

func init() {
	eval.NewFilebasedLoader = newFileBasedLoader
}

func newFileBasedLoader(parent eval.Loader, path, moduleName string, loadables ...eval.PathType) eval.ModuleLoader {
	paths := make(map[eval.Namespace][]SmartPath, len(loadables))
	loader := &fileBasedLoader{
		parentedLoader: parentedLoader{
			basicLoader: basicLoader{namedEntries: make(map[string]eval.LoaderEntry, 64)},
			parent:      parent},
		path:            path,
		initPlanName:    eval.NewTypedName2(eval.PLAN, `init`, parent.NameAuthority()),
		initTaskName:    eval.NewTypedName2(eval.TASK, `init`, parent.NameAuthority()),
		initTypeSetName: eval.NewTypedName2(eval.TYPE, `init_typeset`, parent.NameAuthority()),
		moduleName:      moduleName,
		paths:           paths}

	for _, p := range loadables {
		path := loader.newSmartPath(p, !(moduleName == `` || moduleName == `environment`))
		if sa, ok := paths[path.Namespace()]; ok {
			paths[path.Namespace()] = append(sa, path)
		} else {
			paths[path.Namespace()] = []SmartPath{path}
		}
	}
	return loader
}

func (l *fileBasedLoader) newSmartPath(pathType eval.PathType, moduleNameRelative bool) SmartPath {
	switch pathType {
	case eval.PUPPET_FUNCTION_PATH:
		return l.newPuppetFunctionPath(moduleNameRelative)
	case eval.PUPPET_DATA_TYPE_PATH:
		return l.newPuppetTypePath(moduleNameRelative)
	case eval.PLAN_PATH:
		return l.newPlanPath(moduleNameRelative)
	case eval.TASK_PATH:
		return l.newTaskPath(moduleNameRelative)
	default:
		panic(errors.NewIllegalArgument(`newSmartPath`, 1, fmt.Sprintf(`Unknown path type '%s'`, pathType)))
	}
}

func (l *fileBasedLoader) newPuppetFunctionPath(moduleNameRelative bool) SmartPath {
	return &smartPath{
		relativePath:       `functions`,
		loader:             l,
		namespace:          eval.FUNCTION,
		extension:          `.pp`,
		moduleNameRelative: moduleNameRelative,
		matchMany:          false,
		instantiator:       InstantiatePuppetFunction,
	}
}

func (l *fileBasedLoader) newPlanPath(moduleNameRelative bool) SmartPath {
	return &smartPath{
		relativePath:       `plans`,
		loader:             l,
		namespace:          eval.PLAN,
		extension:          `.pp`,
		moduleNameRelative: moduleNameRelative,
		matchMany:          false,
		instantiator:       InstantiatePuppetPlan,
	}
}

func (l *fileBasedLoader) newPuppetTypePath(moduleNameRelative bool) SmartPath {
	return &smartPath{
		relativePath:       `types`,
		loader:             l,
		namespace:          eval.TYPE,
		extension:          `.pp`,
		moduleNameRelative: moduleNameRelative,
		matchMany:          false,
		instantiator:       InstantiatePuppetType,
	}
}

func (l *fileBasedLoader) newTaskPath(moduleNameRelative bool) SmartPath {
	return &smartPath{
		relativePath:       `tasks`,
		loader:             l,
		namespace:          eval.TASK,
		extension:          ``,
		moduleNameRelative: moduleNameRelative,
		matchMany:          true,
		instantiator:       InstantiatePuppetTask,
	}
}

func (l *fileBasedLoader) LoadEntry(c eval.Context, name eval.TypedName) eval.LoaderEntry {
	entry := l.parentedLoader.LoadEntry(c, name)
	if entry == nil {
		entry = l.find(c, name)
		if entry == nil {
			entry = &loaderEntry{nil, nil}
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

func (l *fileBasedLoader) find(c eval.Context, name eval.TypedName) eval.LoaderEntry {
	if name.IsQualified() {
		// The name is in a name space.
		if l.moduleName != name.NameParts()[0] {
			// Then entity cannot possible be in this module unless the name starts with the module name.
			// Note: If "module" represents a "global component", the module_name is empty and cannot match which is
			// ok since such a "module" cannot have namespaced content).
			return nil
		}
		if name.Namespace() == eval.TASK && len(name.NameParts()) > 2 {
			// Subdirectories beneath the tasks directory are currently not recognized
			return nil
		}
	} else {
		// The name is in the global name space.
		switch name.Namespace() {
		case eval.FUNCTION:
			// Can be defined in module using a global name. No action required
		case eval.PLAN:
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
				return l.instantiate(c, smartPath, name, origins)
			}
		case eval.TASK:
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
				return l.instantiate(c, smartPath, name, origins)
			}
		case eval.TYPE:
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
				smartPath.Instantiator()(c, l, name, origins)
				entry := l.GetEntry(name)
				if entry != nil {
					if _, ok := entry.Value().(eval.TypeSet); ok {
						return entry
					}
				}
				panic(eval.Error(eval.EVAL_NOT_EXPECTED_TYPESET, issue.H{`source`: origins[0], `name`: utils.CapitalizeSegment(l.moduleName)}))
			}
		default:
			return nil
		}
	}

	origins, smartPath := l.findExistingPath(name)
	if smartPath != nil {
		return l.instantiate(c, smartPath, name, origins)
	}

	if !(name.Namespace() == eval.TYPE && name.IsQualified()) {
		return nil
	}

	// Search for TypeSet using parent name
	tsName := name.Parent()
	for tsName != nil {
		tse := l.GetEntry(tsName)
		if tse == nil {
			tse = l.find(c, tsName)
		}
		if tse != nil && tse.Value() != nil {
			if ts, ok := tse.Value().(eval.TypeSet); ok {
				c.DoWithLoader(l, func() {
					ts.(eval.ResolvableType).Resolve(c)
				})
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

func (l *fileBasedLoader) findExistingPath(name eval.TypedName) (origins []string, smartPath SmartPath) {
	l.lock.Lock()
	defer l.lock.Unlock()

	if paths, ok := l.paths[name.Namespace()]; ok {
		for _, sm := range paths {
			l.ensureIndexed(sm)
			if paths, ok := l.index[name.MapKey()]; ok {
				return paths, sm
			}
		}
	}
	return nil, nil
}

func (l *fileBasedLoader) ensureIndexed(sp SmartPath) {
	if !sp.Indexed() {
		sp.SetIndexed()
		l.addToIndex(sp)
	}
}

func (l *fileBasedLoader) instantiate(c eval.Context, smartPath SmartPath, name eval.TypedName, origins []string) eval.LoaderEntry {
	smartPath.Instantiator()(c, l, name, origins)
	return l.GetEntry(name)
}

func (l *fileBasedLoader) GetContent(c eval.Context, path string) []byte {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		panic(eval.Error(eval.EVAL_UNABLE_TO_READ_FILE, issue.H{`path`: path, `detail`: err.Error()}))
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
				rel, err := filepath.Rel(generic, path)
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
