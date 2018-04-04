package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func init() {
	eval.NewGoFunction(`sprintf`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				return types.WrapString(types.PuppetSprintf(args[0].String(), args[1:]...))
			})
		})
}
