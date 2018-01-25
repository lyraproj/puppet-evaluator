package functions


import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-parser/issue"
)

func init() {
	NewGoFunction(`load_plan`,
		func(d Dispatch) {
			d.Param(`String`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				planName := args[0].String()
				if plan, ok := Load(c.Loader(), NewTypedName(PLAN, planName)); ok {
					return WrapUnknown(plan)
				}
				panic(Error(EVAL_UNKNOWN_PLAN, H{`name`: planName}))
			})
		})
}
