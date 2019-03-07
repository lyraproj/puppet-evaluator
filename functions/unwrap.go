package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func init() {
	px.NewGoFunction(`unwrap`,
		func(d px.Dispatch) {
			d.Param(`Sensitive`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return args[0].(*types.SensitiveValue).Unwrap()
			})
		},

		func(d px.Dispatch) {
			d.Param(`Sensitive`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return block.Call(c, nil, args[0].(*types.SensitiveValue).Unwrap())
			})
		})
}
