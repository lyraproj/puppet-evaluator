package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func init() {
	px.NewGoFunction(`strftime`,
		func(d px.Dispatch) {
			d.Param(`Timespan`)
			d.Param(`String`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return types.WrapString(args[0].(types.Timespan).Format(args[1].String()))
			})
		},

		func(d px.Dispatch) {
			d.Param(`Timestamp`)
			d.Param(`String`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return types.WrapString(args[0].(*types.Timestamp).Format(args[1].String()))
			})
		},

		func(d px.Dispatch) {
			d.Param(`Timestamp`)
			d.Param(`String`)
			d.Param(`String`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return types.WrapString(args[0].(*types.Timestamp).Format2(args[1].String(), args[2].String()))
			})
		})
}
