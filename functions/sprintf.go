package functions

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
)

func init() {
		NewGoFunction(`sprintf`,
			func(d Dispatch) {
				d.Param(`String`)
				d.RepeatedParam(`Any`)
				d.Function(func(c EvalContext, args []PValue) PValue {
					return WrapString(PuppetSprintf(args[0].String(), args[1:]...))
				})
			})
}

