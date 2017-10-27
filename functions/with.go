package functions

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

func init() {
	NewGoFunction(`with`,
		func(d Dispatch) {
			d.RepeatedParam(`Any`)
			d.Block(`Callable`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return block.Call(c, nil, args...)
			})
		},
	)
}
