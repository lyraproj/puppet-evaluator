package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`with`,
		func(d eval.Dispatch) {
			d.RepeatedParam(`Any`)
			d.Block(`Callable`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return block.Call(c, nil, args...)
			})
		},
	)
}
