package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`flatten`,
		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				arg := args[0]
				switch arg.(type) {
				case eval.IndexedValue:
					return arg.(eval.IndexedValue).Flatten()
				default:
					return arg.(eval.IterableValue).Iterator().AsArray().Flatten()
				}
			})
		},
	)
}
