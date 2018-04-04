package eval

import (
	"github.com/puppetlabs/go-evaluator/semver"
	"github.com/puppetlabs/go-parser/validator"
)

type (
	URI string

	Setting interface {
		Name() string
		Get() PValue
		IsSet() bool
		Reset()
		Set(value PValue)
		Type() PType
	}

	// Pcore is the interface to the evaluator runtime. The runtime is
	// normally available using the global variable Puppet.
	Pcore interface {
		// Reset clears all settings and loaders, except the system loader
		Reset()

		// SystemLoader returs the loader that finds all built-ins
		SystemLoader() Loader

		// EnvironmentLoader returs the loader that finds things declared
		// in the environment and its modules. This loader is parented
		// by the SystemLoader
		EnvironmentLoader() Loader

		// Loader returns a loader for module.
		Loader(moduleName string) Loader

		// Logger returns the logger that this instance was created with
		Logger() Logger

		// Get returns a setting or calls the given defaultProducer
		// function if the setting does not exist
		Get(key string, defaultProducer Producer) PValue

		// Set changes a setting
		Set(key string, value PValue)

		// Produce executes a given function with an unparented initialized Context instance
		// and returns the result
		Produce(func(EvalContext) PValue) PValue

		// Do executes a given function with an unparented initialized Context instance
		Do(func(EvalContext))

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

		// Resolve types, constructions, or functions that has been recently added
		ResolveResolvables(loader DefiningLoader)
	}
)

const(
	KEY_PCORE_URI = `pcore_uri`
	KEY_PCORE_VERSION = `pcore_version`

	RUNTIME_NAME_AUTHORITY = URI(`http://puppet.com/2016.1/runtime`)
	PCORE_URI = URI(`http://puppet.com/2016.1/pcore`)
)

var PCORE_VERSION = semver.NewVersion4(1, 0, 0, ``, ``)
var PARSABLE_PCORE_VERSIONS, _ = semver.ParseVersionRange(`1.x`)

var Puppet Pcore = nil

func GetSetting(name string, dflt PValue) PValue {
	if Puppet == nil {
		return dflt
	}
	return Puppet.Get(name, func() PValue { return dflt })
}
