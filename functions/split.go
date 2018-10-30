package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func init() {
	eval.NewGoFunction(`split`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				types.WrapRegexp(args[1].String()).Regexp()
				return args[0].(*types.StringValue).Split(types.WrapRegexp(args[1].String()).Regexp())
			})
		},

		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Param(`Regexp`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return args[0].(*types.StringValue).Split(args[1].(*types.RegexpValue).Regexp())
			})
		},

		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Param(`Type[Regexp]`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return args[0].(*types.StringValue).Split(args[1].(*types.RegexpType).Regexp())
			})
		},
	)
}
