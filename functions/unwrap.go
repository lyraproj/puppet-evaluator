package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func init() {
	eval.NewGoFunction(`unwrap`,
		func(d eval.Dispatch) {
			d.Param(`Sensitive`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return args[0].(*types.SensitiveValue).Unwrap()
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Sensitive`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return block.Call(c, nil, args[0].(*types.SensitiveValue).Unwrap())
			})
		})
}
