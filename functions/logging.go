package functions

import (
	. "github.com/puppetlabs/go-evaluator/eval/evaluator"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
)

func init() {
	for _, level := range LOG_LEVELS {
		NewGoFunction(string(level),
			func(d Dispatch) {
				d.RepeatedParam(`Any`)
				d.Function(func(c EvalContext, args []PValue) PValue {
					c.Logger().Log(LogLevel(d.Name()), args...)
					return UNDEF
				})
			})
	}
}
