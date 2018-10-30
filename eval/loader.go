package eval

import (
	"github.com/puppetlabs/go-issues/issue"
	"regexp"
)

type (
	PathType string

	LoaderEntry interface {
		Value() interface{}

		Origin() issue.Location
	}

	Loader interface {
		LoadEntry(c Context, name TypedName) LoaderEntry

		NameAuthority() URI
	}

	DefiningLoader interface {
		Loader

		ResolveResolvables(c Context)

		SetEntry(name TypedName, entry LoaderEntry) LoaderEntry
	}

	ParentedLoader interface {
		Loader

		Parent() Loader
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

		TypeSet() Type
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

var Load func(c Context, name TypedName) (interface{}, bool)
var NewLoaderEntry func(value interface{}, origin issue.Location) LoaderEntry
var StaticLoader func() Loader
var StaticResourceLoader func() Loader
var NewParentedLoader func(parent Loader) DefiningLoader
var NewFilebasedLoader func(parent Loader, path, moduleName string, pathTypes ...PathType) ModuleLoader
var NewDependencyLoader func(depLoaders []ModuleLoader) Loader
var RegisterGoFunction func(function ResolvableFunction)
var RegisterResolvableType func(rt ResolvableType)
var NewTypeSetLoader func(parent Loader, typeSet Type) TypeSetLoader
