package functions

import "github.com/puppetlabs/go-evaluator/eval"

func init() {
	eval.NewGoFunction(`convert_to`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Param(`Type`)
			d.OptionalBlock(`Callable`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				result := c.Call(`new`, []eval.PValue{args[1], args[0]}, nil)
				if block != nil {
					result = block.Call(c, nil, result)
				}
				return result
			})
		},
	)
}
