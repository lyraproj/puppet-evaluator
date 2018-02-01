package types

import (
	. "github.com/puppetlabs/go-evaluator/eval"
)

var _EMPTY_ARRAY = WrapArray([]PValue{})
var _EMPTY_MAP = WrapHash([]*HashEntry{})
var _EMPTY_STRING = WrapString(``)
var _UNDEF = WrapUndef()

func init() {
	EMPTY_ARRAY = _EMPTY_ARRAY
	EMPTY_MAP = _EMPTY_MAP
	EMPTY_STRING = _EMPTY_STRING
	UNDEF = _UNDEF
}
