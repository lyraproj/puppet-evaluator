package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`lest`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Block(`Callable[0,0]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				if eval.Undef.Equals(args[0], nil) {
					return block.Call(c, nil)
				}
				return args[0]
			})
		})
}
