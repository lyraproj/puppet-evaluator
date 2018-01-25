package evaluator

import "github.com/puppetlabs/go-evaluator/semver"

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

const RUNTIME_NAME_AUTHORITY = URI(`http://puppet.com/2016.1/runtime`)
const PCORE_URI = URI(`http://puppet.com/2016.1/pcore`)

var PCORE_VERSION = semver.NewVersion4(1, 0, 0, ``, ``)

var Puppet Pcore = nil

func GetSetting(name string, dflt PValue) PValue {
  if Puppet == nil {
  	return dflt
  }
  return Puppet.Get(name, func() PValue { return dflt })
}
