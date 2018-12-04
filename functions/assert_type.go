package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/issue/issue"
)

func assertType(c eval.Context, t eval.Type, v eval.Value, b eval.Lambda) eval.Value {
	if eval.IsInstance(t, v) {
		return v
	}
	vt := eval.DetailedValueType(v)
	if b == nil {
		panic(eval.Error(eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeMismatch(`assert_type():`, t, vt)}))
	}
	return b.Call(c, nil, t, vt)
}

func init() {
	eval.NewGoFunction(`assert_type`,
		func(d eval.Dispatch) {
			d.Param(`String[1]`)
			d.Param(`Any`)
			d.OptionalBlock(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return assertType(c, c.ParseType(args[0]), args[1], block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Type`)
			d.Param(`Any`)
			d.OptionalBlock(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return assertType(c, args[0].(eval.Type), args[1], block)
			})
		},
	)
}
