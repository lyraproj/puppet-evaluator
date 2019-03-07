package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func init() {
	px.NewGoFunction(`values`,
		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return args[0].(*types.HashValue).Values()
			})
		},

		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				args[0].(*types.HashValue).Values().Iterator().Each(func(v px.Value) { block.Call(c, nil, v) })
				return px.Undef
			})
		},
	)
}
