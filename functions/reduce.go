package functions

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

func init() {
	NewGoFunction(`reduce`,
		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return args[0].(IterableValue).Iterator().Reduce(
					func(v1 PValue, v2 PValue) PValue { return block.Call(c, nil, v1, v2) })
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Param(`Any`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return args[0].(IterableValue).Iterator().Reduce2(
					args[1], func(v1 PValue, v2 PValue) PValue { return block.Call(c, nil, v1, v2) })
			})
		},
	)
}
