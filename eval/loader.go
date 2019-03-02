package eval

import (
	"github.com/lyraproj/issue/issue"
	"regexp"
)

type (
	PathType string

	LoaderEntry interface {
		Value() interface{}

		Origin() issue.Location
	}

	Loader interface {
		// LoadEntry returns the requested entry or nil if no such entry can be found
		LoadEntry(c Context, name TypedName) LoaderEntry

		// NameAuthority returns the name authority
		NameAuthority() URI

		// Discover iterates over all entries accessible to this loader and its parents
		// and returns a slice of all entries for which the provided function returns
		// true
		Discover(c Context, predicate func(tn TypedName) bool) []TypedName

		// HasEntry returns true if this loader has an entry that maps to the give name
		// The status of the entry is determined without actually loading it.
		HasEntry(name TypedName) bool
	}

	DefiningLoader interface {
		Loader

		SetEntry(name TypedName, entry LoaderEntry) LoaderEntry
	}

	ParentedLoader interface {
		Loader

		// Parent returns the parent loader
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
	PuppetDataTypePath = PathType(`puppetDataType`)
	PuppetFunctionPath = PathType(`puppetFunction`)
	PlanPath           = PathType(`plan`)
	TaskPath           = PathType(`task`)
)

var moduleNameRX = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func IsValidModuleName(moduleName string) bool {
	return moduleNameRX.MatchString(moduleName)
}

var Load func(c Context, name TypedName) (interface{}, bool)
var NewLoaderEntry func(value interface{}, origin issue.Location) LoaderEntry
var StaticLoader func() Loader
var NewParentedLoader func(parent Loader) DefiningLoader
var NewFileBasedLoader func(parent Loader, path, moduleName string, pathTypes ...PathType) ModuleLoader
var NewDependencyLoader func(depLoaders []ModuleLoader) Loader
var RegisterGoFunction func(function ResolvableFunction)
var RegisterResolvableType func(rt ResolvableType)
var NewTypeSetLoader func(parent Loader, typeSet Type) TypeSetLoader
