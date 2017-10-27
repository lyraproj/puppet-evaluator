package types

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

func toFloat(v PValue) (float64, bool) {
	if iv, ok := v.(*FloatValue); ok {
		return iv.Float(), true
	}
	return 0.0, false
}

func toInt(v PValue) (int64, bool) {
	if iv, ok := v.(*IntegerValue); ok {
		return iv.Int(), true
	}
	return 0, false
}

func init() {
	ToInt = toInt
	ToFloat = toFloat
}
