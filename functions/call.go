package functions

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
)

func init() {
	NewGoFunction(`call`,
		func(d Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return c.Call(args[0].(*StringValue).String(), args[1:], block)
			})
		},
	)
}
