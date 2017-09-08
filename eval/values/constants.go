package values

import (
	"github.com/puppetlabs/go-evaluator/eval/values/api"
)

var _EMPTY_ARRAY = WrapArray([]api.PValue{})
var _EMPTY_MAP = WrapHash([]*HashEntry{})
var _EMPTY_STRING = WrapString(``)
var _UNDEF = WrapUndef()

func init() {
	api.EMPTY_ARRAY = _EMPTY_ARRAY
	api.EMPTY_MAP = _EMPTY_MAP
	api.EMPTY_STRING = _EMPTY_STRING
	api.UNDEF = _UNDEF
}
