package functions

import "github.com/lyraproj/pcore/eval"

func init() {
	for _, level := range eval.LogLevels {
		eval.NewGoFunction(string(level),
			func(d eval.Dispatch) {
				d.RepeatedParam(`Any`)
				d.Function(func(c eval.Context, args []eval.Value) eval.Value {
					c.Logger().Log(eval.LogLevel(d.Name()), args...)
					return eval.Undef
				})
			})
	}
}
