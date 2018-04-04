package functions

import (
	"fmt"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
)

func callNew(c eval.EvalContext, name string, args []eval.PValue, block eval.Lambda) eval.PValue {
	// Always make an attempt to load the named type
	// TODO: This should be a properly checked load but it currently isn't because some receivers in the PSpec
	// evaluator are not proper types yet.
	eval.Load(c, eval.NewTypedName(eval.TYPE, name))

	tn := eval.NewTypedName(eval.CONSTRUCTOR, name)
	if ctor, ok := eval.Load(c, tn); ok {
		r := ctor.(eval.Function).Call(c, nil, args...)
		if block != nil {
			r = block.Call(c, nil, r)
		}
		return r
	}
	panic(errors.NewArgumentsError(`new`, fmt.Sprintf(`Creation of new instance of type '%s' is not supported`, name)))
}

func init() {
	eval.NewGoFunction(`new`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				return callNew(c, args[0].String(), args[1:], block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Type`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				pt := args[0].(eval.PType)
				return assertType(c, pt, callNew(c, pt.Name(), args[1:], block), nil)
			})
		},
	)
}
