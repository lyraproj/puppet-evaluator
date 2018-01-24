package functions


import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-parser/issue"
)

func init() {
	NewGoFunction(`load_task`,
		func(d Dispatch) {
			d.Param(`String`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				taskName := args[0].String()
				if task, ok := Load(c.Loader(), NewTypedName(TASK, taskName)); ok {
					return task.(PValue)
				}
				panic(Error(EVAL_UNKNOWN_TASK, H{`name`: taskName}))
			})
		})
}
