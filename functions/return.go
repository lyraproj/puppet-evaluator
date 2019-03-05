package functions

import (
	"github.com/lyraproj/pcore/errors"
	"github.com/lyraproj/pcore/eval"
)

func init() {
	eval.NewGoFunction(`return`,
		func(d eval.Dispatch) {
			d.OptionalParam(`Any`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				arg := eval.Undef
				if len(args) > 0 {
					arg = args[0]
				}
				panic(errors.NewReturn(c.StackTop(), arg))
			})
		},
	)
}
