package functions

import (
	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`next`,
		func(d eval.Dispatch) {
			d.OptionalParam(`Any`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				arg := eval.UNDEF
				if len(args) > 0 {
					arg = args[0]
				}
				panic(errors.NewNextIteration(c.StackTop(), arg))
			})
		},
	)
}
