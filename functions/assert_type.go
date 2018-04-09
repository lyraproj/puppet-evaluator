package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-parser/issue"
)

func assertType(c eval.Context, t eval.PType, v eval.PValue, b eval.Lambda) eval.PValue {
	if eval.IsInstance(c, t, v) {
		return v
	}
	vt := eval.DetailedValueType(v)
	if b == nil {
		panic(eval.Error(c, eval.EVAL_TYPE_MISMATCH, issue.H{`detail`: eval.DescribeMismatch(`assert_type():`, t, vt)}))
	}
	return b.Call(c, nil, t, vt)
}

func init() {
	eval.NewGoFunction(`assert_type`,
		func(d eval.Dispatch) {
			d.Param(`String[1]`)
			d.Param(`Any`)
			d.OptionalBlock(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return assertType(c, c.ParseType(args[0]), args[1], block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Type`)
			d.Param(`Any`)
			d.OptionalBlock(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return assertType(c, args[0].(eval.PType), args[1], block)
			})
		},
	)
}
