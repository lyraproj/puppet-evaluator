package functions

import (
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`break`,
		func(d eval.Dispatch) {
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				panic(errors.NewStopIteration(c.StackTop()))
			})
		},
	)
}
