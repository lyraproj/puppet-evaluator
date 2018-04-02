package eval

import (
	"regexp"
	"github.com/puppetlabs/go-parser/issue"
)

type (
	PathType string

	Entry interface {
		Value() interface{}

		Origin() issue.Location
	}

	Loader interface {
		LoadEntry(name TypedName) Entry

		NameAuthority() URI
	}

	DefiningLoader interface {
		Loader

		ResolveResolvables(c EvalContext)

		SetEntry(name TypedName, entry Entry) Entry
	}

	ModuleLoader interface {
		Loader

		ModuleName() string
	}

	DependencyLoader interface {
		Loader

		LoaderFor(key string) ModuleLoader
	}

	TypeSetLoader interface {
		Loader

		TypeSet() PType
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
var NewLoaderEntry func(value interface{}, origin issue.Location) Entry
var StaticLoader func() Loader
var NewParentedLoader func(parent Loader) DefiningLoader
var NewFilebasedLoader func(parent Loader, path, moduleName string, pathTypes ...PathType) ModuleLoader
var NewDependencyLoader func(depLoaders []ModuleLoader) Loader
var RegisterGoFunction func(function ResolvableFunction)
var RegisterResolvableType func(rt ResolvableType)
var NewTypeSetLoader func(parent Loader, typeSet PType) TypeSetLoader
