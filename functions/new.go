package functions

import "github.com/lyraproj/pcore/eval"

func init() {
	eval.NewGoFunction(`new`,
		func(d eval.Dispatch) {
			d.Param(`Variant[Type,String]`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return eval.NewWithBlock(c, args[0], args[1:], block)
			})
		},
	)
}
