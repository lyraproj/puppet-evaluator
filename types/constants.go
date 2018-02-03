package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
)

var _EMPTY_ARRAY = WrapArray([]eval.PValue{})
var _EMPTY_MAP = WrapHash([]*HashEntry{})
var _EMPTY_STRING = WrapString(``)
var _UNDEF = WrapUndef()

func init() {
	eval.EMPTY_ARRAY = _EMPTY_ARRAY
	eval.EMPTY_MAP = _EMPTY_MAP
	eval.EMPTY_STRING = _EMPTY_STRING
	eval.UNDEF = _UNDEF
}
