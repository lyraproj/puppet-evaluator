package eval

import (
	"github.com/puppetlabs/go-evaluator/semver"
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

	Pcore interface {
		Reset()

		SystemLoader() Loader

		EnvironmentLoader() Loader

		Loader(key string) Loader

		Get(key string, defaultProducer Producer) PValue

		Set(key string, value PValue)

		DefineSetting(key string, valueType PType, dflt PValue)
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
