package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/puppet-evaluator/errors"
)

func init() {
	px.NewGoFunction(`break`,
		func(d px.Dispatch) {
			d.Function(func(c px.Context, args []px.Value) px.Value {
				panic(errors.NewStopIteration(c.StackTop()))
			})
		},
	)
}
