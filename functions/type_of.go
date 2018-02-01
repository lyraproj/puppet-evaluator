package functions

import (
	. "github.com/puppetlabs/go-evaluator/eval"
)

func typeOf(c EvalContext, v PValue, i string) PType {
	switch i {
	case `generalized`:
		return GenericValueType(v)
	case `reduced`:
		return v.Type()
	default:
		return DetailedValueType(v)
	}
}

func init() {
	NewGoFunction(`type`,
		func(d Dispatch) {
			d.Param(`Any`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				return typeOf(c, args[0], `generalized`)
			})
		},

		func(d Dispatch) {
			d.Param(`Any`)
			d.Param(`Enum[detailed, reduced, generalized]`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				return typeOf(c, args[0], args[1].String())
			})
		},
	)
}
