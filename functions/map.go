package functions

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

func mapIterableOne(c EvalContext, arg IndexedValue, block Lambda) PValue {
	return mapIteratorOne(c, arg.Iterator(), block)
}

func mapIterableTwo(c EvalContext, arg IndexedValue, block Lambda) PValue {
	return mapIteratorTwo(c, arg.Iterator(), block)
}

func mapIteratorOne(c EvalContext, arg Iterator, block Lambda) PValue {
	return arg.Map(block.Signature().ReturnType(), func(v PValue) PValue { return block.Call(c, nil, v) })
}

func mapIteratorTwo(c EvalContext, arg Iterator, block Lambda) PValue {
	return arg.Map(block.Signature().ReturnType(), func(v PValue) PValue {
		if vi, ok := v.(IndexedValue); ok {
			return block.Call(c, nil, vi.At(0), vi.At(1))
		}
		return block.Call(c, nil, v, UNDEF)
	})
}

func init() {
	NewGoFunction(`map`,
		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return mapIterableOne(c, args[0].(IndexedValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return mapIterableTwo(c, args[0].(IndexedValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterator`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return mapIteratorOne(c, args[0].(IteratorValue).DynamicValue(), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterator`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return mapIteratorTwo(c, args[0].(IteratorValue).DynamicValue(), block)
			})
		},
	)
}
