package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func init() {
	eval.NewGoFunction(`call`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return eval.Call(c, args[0].(*types.StringValue).String(), args[1:], block)
			})
		},
		func(d eval.Dispatch) {
			d.Param(`Deferred`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				return args[0].(types.Deferred).Resolve(c)
			})
		},
	)
}
