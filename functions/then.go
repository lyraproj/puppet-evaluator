package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`then`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				if eval.UNDEF.Equals(args[0], nil) {
					return eval.UNDEF
				}
				return block.Call(c, nil, args[0])
			})
		})
}
