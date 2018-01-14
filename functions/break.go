package functions

import (
	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

func init() {
	NewGoFunction(`break`,
		func(d Dispatch) {
			d.Function(func(c EvalContext, args []PValue) PValue {
				panic(NewStopIteration(c.StackTop()))
			})
		},
	)
}
