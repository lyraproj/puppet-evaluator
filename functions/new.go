package functions

import (
	"fmt"

	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
)

func callNew(c eval.Context, typ eval.Value, args []eval.Value, block eval.Lambda) eval.Value {
	// Always make an attempt to load the named type
	// TODO: This should be a properly checked load but it currently isn't because some receivers in the PSpec
	// evaluator are not proper types yet.
	var ctor eval.Function
	name := ``
	if ot, ok := typ.(eval.ObjectType); ok {
		ctor = ot.Constructor()
		name = ot.Name()
	} else {
		name = typ.String()
		if t, ok := eval.Load(c, eval.NewTypedName(eval.NsType, name)); ok {
			if ot, ok := t.(eval.ObjectType); ok {
				ctor = ot.Constructor()
			}
		}
		if ctor == nil {
			tn := eval.NewTypedName(eval.NsConstructor, name)
			if t, ok := eval.Load(c, tn); ok {
				ctor = t.(eval.Function)
			}
		}
	}

	if ctor == nil {
		panic(errors.NewArgumentsError(`new`, fmt.Sprintf(`Creation of new instance of type '%s' is not supported`, typ.String())))
	}

	r := ctor.(eval.Function).Call(c, nil, args...)
	if block != nil {
		r = block.Call(c, nil, r)
	}
	return r
}

func init() {
	eval.NewGoFunction(`new`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return callNew(c, args[0], args[1:], block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Type`)
			d.RepeatedParam(`Any`)
			d.OptionalBlock(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				pt := args[0].(eval.Type)
				return assertType(c, pt, callNew(c, pt, args[1:], block), nil)
			})
		},
	)
}
