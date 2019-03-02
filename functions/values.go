package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

func init() {
	eval.NewGoFunction(`values`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return args[0].(*types.HashValue).Values()
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				args[0].(*types.HashValue).Values().Iterator().Each(func(v eval.Value) { block.Call(c, nil, v) })
				return eval.Undef
			})
		},
	)
}
