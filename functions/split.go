package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func init() {
	px.NewGoFunction(`split`,
		func(d px.Dispatch) {
			d.Param(`String`)
			d.Param(`String`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				types.WrapRegexp(args[1].String()).Regexp()
				return args[0].(px.StringValue).Split(types.WrapRegexp(args[1].String()).Regexp())
			})
		},

		func(d px.Dispatch) {
			d.Param(`String`)
			d.Param(`Regexp`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return args[0].(px.StringValue).Split(args[1].(*types.RegexpValue).Regexp())
			})
		},

		func(d px.Dispatch) {
			d.Param(`String`)
			d.Param(`Type[Regexp]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return args[0].(px.StringValue).Split(args[1].(*types.RegexpType).Regexp())
			})
		},
	)
}
