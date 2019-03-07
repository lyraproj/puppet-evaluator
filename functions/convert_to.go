package functions

import "github.com/lyraproj/pcore/px"

func init() {
	px.NewGoFunction(`convert_to`,
		func(d px.Dispatch) {
			d.Param(`Any`)
			d.Param(`Type`)
			d.OptionalBlock(`Callable`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				result := px.Call(c, `new`, []px.Value{args[1], args[0]}, nil)
				if block != nil {
					result = block.Call(c, nil, result)
				}
				return result
			})
		},
	)
}
