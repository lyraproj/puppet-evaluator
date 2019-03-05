package functions

import "github.com/lyraproj/pcore/eval"

func init() {
	eval.NewGoFunction(`flatten`,
		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				switch arg := args[0].(type) {
				case eval.List:
					return arg.Flatten()
				default:
					return arg.(eval.IterableValue).Iterator().AsArray().Flatten()
				}
			})
		},
	)
}
