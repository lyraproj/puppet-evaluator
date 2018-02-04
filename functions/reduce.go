package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`reduce`,
		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				return args[0].(eval.IterableValue).Iterator().Reduce(
					func(v1 eval.PValue, v2 eval.PValue) eval.PValue { return block.Call(c, nil, v1, v2) })
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Param(`Any`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				return args[0].(eval.IterableValue).Iterator().Reduce2(
					args[1], func(v1 eval.PValue, v2 eval.PValue) eval.PValue { return block.Call(c, nil, v1, v2) })
			})
		},
	)
}
