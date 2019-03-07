package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func init() {
	px.NewGoFunction(`call`,
		func(d px.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return px.Call(c, args[0].String(), args[1:], block)
			})
		},
		func(d px.Dispatch) {
			d.Param(`Deferred`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return args[0].(types.Deferred).Resolve(c)
			})
		},
	)
}
