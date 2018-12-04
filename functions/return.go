package functions

import (
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`return`,
		func(d eval.Dispatch) {
			d.OptionalParam(`Any`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				arg := eval.UNDEF
				if len(args) > 0 {
					arg = args[0]
				}
				panic(errors.NewReturn(c.StackTop(), arg))
			})
		},
	)
}
