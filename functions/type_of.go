package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
)

func typeOf(v eval.Value, i string) eval.Type {
	switch i {
	case `generalized`:
		return eval.GenericValueType(v)
	case `reduced`:
		return v.PType()
	default:
		return eval.DetailedValueType(v)
	}
}

func init() {
	eval.NewGoFunction(`type`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return typeOf(args[0], `generalized`)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Param(`Enum[detailed, reduced, generalized]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return typeOf(args[0], args[1].String())
			})
		},
	)
}
