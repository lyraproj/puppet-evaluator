package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`then`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				if eval.Undef.Equals(args[0], nil) {
					return eval.Undef
				}
				return block.Call(c, nil, args[0])
			})
		})
}
