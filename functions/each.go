package functions

import (
	. "github.com/puppetlabs/go-evaluator/eval/evaluator"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
)

func eachIterableOne(c EvalContext, arg IndexedValue, block Lambda) PValue {
	eachIteratorOne(c, arg.Iterator(), block)
	return arg
}

func eachIterableTwo(c EvalContext, arg IndexedValue, block Lambda) PValue {
	eachIteratorTwo(c, arg.Iterator(), block)
	return arg
}

func eachIteratorOne(c EvalContext, arg Iterator, block Lambda) PValue {
	arg.Each(func(v PValue) { block.Call(c, nil, v) })
	return UNDEF
}

func eachIteratorTwo(c EvalContext, arg Iterator, block Lambda) PValue {
	arg.Each(func(v PValue) {
		if vi, ok := v.(IndexedValue); ok {
			block.Call(c, nil, vi.At(0), vi.At(1))
		} else {
			block.Call(c, nil, v, UNDEF)
		}
	})
	return UNDEF
}

func init() {
	NewGoFunction(`each`,
		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return eachIterableOne(c, args[0].(IndexedValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return eachIterableTwo(c, args[0].(IndexedValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterator`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return eachIteratorOne(c, args[0].(IteratorValue).DynamicValue(), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterator`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return eachIteratorTwo(c, args[0].(IteratorValue).DynamicValue(), block)
			})
		},
	)
}
