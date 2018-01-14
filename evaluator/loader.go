package evaluator

import "regexp"

type (
	PathType string

	Entry interface {
		Value() interface{}

		Origin() string
	}

	Loader interface {
		LoadEntry(name TypedName) Entry

		NameAuthority() URI
	}

	DefiningLoader interface {
		Loader

		ResolveGoFunctions(c EvalContext)

		SetEntry(name TypedName, entry Entry) Entry
	}

	ModuleLoader interface {
		Loader

		ModuleName() string
	}

	DependencyLoaer interface {
		Loader

		LoaderFor(key string) ModuleLoader
	}
)

const (
	PUPPET_DATA_TYPE_PATH = PathType(`puppetDataType`)
	PUPPET_FUNCTION_PATH  = PathType(`puppetFunction`)
	PLAN_PATH             = PathType(`plan`)
	TASK_PATH             = PathType(`task`)
)

var moduleNameRX = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func IsValidModuleName(moduleName string) bool {
	return moduleNameRX.MatchString(moduleName)
}

var Load func(loader Loader, name TypedName) (interface{}, bool)
var NewLoaderEntry func(value interface{}, origin string) Entry
var StaticLoader func() Loader
var NewParentedLoader func(parent Loader) DefiningLoader
var NewFilebasedLoader func(parent Loader, path, moduleName string, pathTypes ...PathType) ModuleLoader
var NewDependencyLoader func(depLoaders []ModuleLoader) Loader
var RegisterGoFunction func(function ResolvableFunction)
var RegisterGoConstructor func(function ResolvableFunction)
