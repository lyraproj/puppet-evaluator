package functions

import "github.com/lyraproj/pcore/px"

func init() {
	px.NewGoFunction(`new`,
		func(d px.Dispatch) {
			d.Param(`Variant[Type,String]`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return px.NewWithBlock(c, args[0], args[1:], block)
			})
		},
	)
}
