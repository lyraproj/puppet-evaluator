package functions

import (
	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

func init() {
	NewGoFunction(`return`,
		func(d Dispatch) {
			d.OptionalParam(`Any`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				arg := UNDEF
				if len(args) > 0 {
					arg = args[0]
				}
				panic(NewReturn(c.StackTop(), arg))
			})
		},
	)
}
