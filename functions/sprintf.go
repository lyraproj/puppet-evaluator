package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

func init() {
	eval.NewGoFunction(`sprintf`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return types.WrapString(types.PuppetSprintf(args[0].String(), args[1:]...))
			})
		})
}
