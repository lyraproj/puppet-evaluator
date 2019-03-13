package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/evaluator"
)

func init() {
	px.NewGoFunction(`keys`,
		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return args[0].(*types.Hash).Keys()
			})
		},

		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				evaluator.WrapIterable(args[0].(*types.Hash).Keys()).Each(func(v px.Value) { block.Call(c, nil, v) })
				return px.Undef
			})
		},
	)
}
