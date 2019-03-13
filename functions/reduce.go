package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/puppet-evaluator/evaluator"
)

func init() {
	px.NewGoFunction(`reduce`,
		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return evaluator.WrapIterable(args[0].(px.Indexed)).Reduce(
					func(v1 px.Value, v2 px.Value) px.Value { return block.Call(c, nil, v1, v2) })
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Param(`Any`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return evaluator.WrapIterable(args[0].(px.Indexed)).Reduce2(
					args[1], func(v1 px.Value, v2 px.Value) px.Value { return block.Call(c, nil, v1, v2) })
			})
		},
	)
}
