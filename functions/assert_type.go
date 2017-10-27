package functions

import (
	"fmt"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

func assertType(c EvalContext, t PType, v PValue, b Lambda) PValue {
	if IsInstance(t, v) {
		return v
	}
	if b == nil {
		// TODO: Use a proper TypeMismatchDescriber once it's been implemented
		panic(fmt.Sprintf(`Type mismatch. Expected %s, got %s`, t.String(), DetailedType(v)))
	}
	return b.Call(c, nil, t, v)
}

func init() {
	NewGoFunction(`assert_type`,
		func(d Dispatch) {
			d.Param(`String[1]`)
			d.Param(`Any`)
			d.OptionalBlock(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return assertType(c, c.ParseType(args[0]), args[1], block)
			})
		},

		func(d Dispatch) {
			d.Param(`Type`)
			d.Param(`Any`)
			d.OptionalBlock(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return assertType(c, args[0].(PType), args[1], block)
			})
		},
	)
}
