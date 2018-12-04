package functions

import "github.com/lyraproj/puppet-evaluator/eval"

func init() {
	eval.NewGoFunction(`convert_to`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Param(`Type`)
			d.OptionalBlock(`Callable`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				result := eval.Call(c, `new`, []eval.Value{args[1], args[0]}, nil)
				if block != nil {
					result = block.Call(c, nil, result)
				}
				return result
			})
		},
	)
}
