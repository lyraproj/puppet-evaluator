package evaluator

import "github.com/puppetlabs/go-evaluator/semver"

type (
	URI string

	Pcore interface {
		Loader() Loader
	}
)

const RUNTIME_NAME_AUTHORITY = URI(`http://puppet.com/2016.1/runtime`)
const PCORE_URI = URI(`http://puppet.com/2016.1/pcore`)

var PCORE_VERSION = semver.NewVersion4(1, 0, 0, ``, ``)
