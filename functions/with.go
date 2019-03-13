package functions

import "github.com/lyraproj/pcore/px"

func init() {
	px.NewGoFunction(`with`,
		func(d px.Dispatch) {
			d.RepeatedParam(`Any`)
			d.Block(`Callable`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return block.Call(c, nil, args...)
			})
		},
	)
}
