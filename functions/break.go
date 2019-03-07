package functions

import (
	"github.com/lyraproj/pcore/errors"
	"github.com/lyraproj/pcore/px"
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
