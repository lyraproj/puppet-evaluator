package functions

import (
	"github.com/lyraproj/pcore/errors"
	"github.com/lyraproj/pcore/eval"
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
