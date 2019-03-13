package functions

import "github.com/lyraproj/pcore/px"

func init() {
	px.NewGoFunction(`lest`,
		func(d px.Dispatch) {
			d.Param(`Any`)
			d.Block(`Callable[0,0]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				if px.Undef.Equals(args[0], nil) {
					return block.Call(c, nil)
				}
				return args[0]
			})
		})
}
