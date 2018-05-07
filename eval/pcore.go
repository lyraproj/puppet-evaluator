package eval

import (
	"github.com/puppetlabs/go-semver/semver"
	"github.com/puppetlabs/go-parser/validator"
	"context"
)

type (
	URI string

	// Pcore is the interface to the evaluator runtime. The runtime is
	// a singleton available using the global variable Puppet.
	Pcore interface {
		// Reset clears all settings and loaders, except the static loaders
		Reset()

		// SystemLoader returs the loader that finds all built-ins. It's parented
		// by a static loader. The choice of parent is depending on the 'tasks'
		// setting. When 'tasks' is enabled, all resources, including the Resource
		// type, are excluded.
		SystemLoader() Loader

		// EnvironmentLoader returs the loader that finds things declared
		// in the environment and its modules. This loader is parented
		// by the SystemLoader
		EnvironmentLoader() Loader

		// Loader returns a loader for module.
		Loader(moduleName string) Loader

		// Logger returns the logger that this instance was created with
		Logger() Logger

		// RootContext returns a new Context that is parented by the context.Background()
		// and is initialized with a loader that is parented by the EnvironmentLoader.
		//
		RootContext() Context

		// Get returns a setting or calls the given defaultProducer
		// function if the setting does not exist
		Get(key string, defaultProducer Producer) PValue

		// Set changes a setting
		Set(key string, value PValue)

		// Do executes a given function with an initialized Context instance. The
		// Context will be parented by the Go context returned by context.Background()
		Do(func(Context) error) error

		// DoWithParent executes a given function with an initialized Context instance. The
		// context will be parented by the given Go context
		DoWithParent(context.Context, func(Context) error) error

		// NewEvaluator creates a new evaluator instance that will be initialized
		// with a loader parented by the EnvironmenLoader and the logger configured
		// for this instance.
		NewEvaluator() Evaluator

		// NewEvaluatorWithLogger is like NewEvaluator but with a given logger. This
		// method is primarily intended for test purposes where it is necessary to
		// collect log output.
		NewEvaluatorWithLogger(logger Logger) Evaluator

		// NewParser returns a parser and validator that corresponds to the current
		// settings of the receiver.
		//
		// At present, the only setting that has an impact on the parser/validator is
		// the "tasks" setting. If it is enabled, the parser will recognize
		// the keyword "plan" and the validator will not accept resource expressions.
		NewParser() validator.ParserValidator

		// DefineSetting defines a new setting with a given valueType and default
		// value.
		DefineSetting(key string, valueType PType, dflt PValue)
	}
)

const(
	KEY_PCORE_URI = `pcore_uri`
	KEY_PCORE_VERSION = `pcore_version`

	RUNTIME_NAME_AUTHORITY = URI(`http://puppet.com/2016.1/runtime`)
	PCORE_URI = URI(`http://puppet.com/2016.1/pcore`)
)

var PCORE_VERSION, _ = semver.NewVersion3(1, 0, 0, ``, ``)
var PARSABLE_PCORE_VERSIONS, _ = semver.ParseVersionRange(`1.x`)

var Puppet Pcore = nil

func GetSetting(name string, dflt PValue) PValue {
	if Puppet == nil {
		return dflt
	}
	return Puppet.Get(name, func() PValue { return dflt })
}
