package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

func init() {
	eval.NewGoFunction(`strftime`,
		func(d eval.Dispatch) {
			d.Param(`Timespan`)
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return types.WrapString(args[0].(*types.TimespanValue).Format(args[1].String()))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Timestamp`)
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return types.WrapString(args[0].(*types.TimestampValue).Format(args[1].String()))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Timestamp`)
			d.Param(`String`)
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return types.WrapString(args[0].(*types.TimestampValue).Format2(args[1].String(), args[2].String()))
			})
		})
}
