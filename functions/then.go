package functions

import "github.com/lyraproj/pcore/px"

func init() {
	px.NewGoFunction(`then`,
		func(d px.Dispatch) {
			d.Param(`Any`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				if px.Undef.Equals(args[0], nil) {
					return px.Undef
				}
				return block.Call(c, nil, args[0])
			})
		})
}
