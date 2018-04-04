package functions

import (
	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`break`,
		func(d eval.Dispatch) {
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				panic(errors.NewStopIteration(c.StackTop()))
			})
		},
	)
}
