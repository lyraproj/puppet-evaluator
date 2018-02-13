package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`lest`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Block(`Callable[0,0]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				if eval.UNDEF.Equals(args[0], nil) {
					return block.Call(c, nil)
				}
				return args[0]
			})
		})
}
