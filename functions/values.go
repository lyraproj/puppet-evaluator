package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func init() {
	eval.NewGoFunction(`values`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				return args[0].(*types.HashValue).Values()
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				args[0].(*types.HashValue).Values().Iterator().Each(func(v eval.PValue) { block.Call(c, nil, v) })
				return eval.UNDEF
			})
		},
	)
}
