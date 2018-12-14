package functions

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

func init() {
	eval.NewGoFunction(`name`,
		func(d eval.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				v := args[0]
				if n, ok := v.(issue.Named); ok {
					return types.WrapString(n.Name())
				}
				panic(eval.Error(eval.EVAL_UNKNOWN_FUNCTION, issue.H{`name`: `name`}))
			})
		})
}
