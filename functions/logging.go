package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
)

func init() {
	for _, level := range eval.LOG_LEVELS {
		eval.NewGoFunction(string(level),
			func(d eval.Dispatch) {
				d.RepeatedParam(`Any`)
				d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
					c.Logger().Log(eval.LogLevel(d.Name()), args...)
					return eval.UNDEF
				})
			})
	}
}
