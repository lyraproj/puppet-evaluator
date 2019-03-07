package functions

import "github.com/lyraproj/pcore/px"

func init() {
	px.NewGoFunction(`reduce`,
		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return args[0].(px.IterableValue).Iterator().Reduce(
					func(v1 px.Value, v2 px.Value) px.Value { return block.Call(c, nil, v1, v2) })
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Param(`Any`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return args[0].(px.IterableValue).Iterator().Reduce2(
					args[1], func(v1 px.Value, v2 px.Value) px.Value { return block.Call(c, nil, v1, v2) })
			})
		},
	)
}
