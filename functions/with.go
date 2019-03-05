package functions

import "github.com/lyraproj/pcore/eval"

func init() {
	eval.NewGoFunction(`with`,
		func(d eval.Dispatch) {
			d.RepeatedParam(`Any`)
			d.Block(`Callable`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return block.Call(c, nil, args...)
			})
		},
	)
}
