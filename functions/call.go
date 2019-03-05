package functions

import (
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/types"
)

func init() {
	eval.NewGoFunction(`call`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return eval.Call(c, args[0].String(), args[1:], block)
			})
		},
		func(d eval.Dispatch) {
			d.Param(`Deferred`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return args[0].(types.Deferred).Resolve(c)
			})
		},
	)
}
