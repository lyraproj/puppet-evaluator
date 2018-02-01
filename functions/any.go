package functions

import (
	. "github.com/puppetlabs/go-evaluator/eval"
	. "github.com/puppetlabs/go-evaluator/types"
)

func anyIterator(c EvalContext, arg IterableValue, block Lambda) PValue {
	return WrapBoolean(arg.Iterator().Any(func(v PValue) bool { return IsTruthy(block.Call(c, nil, v)) }))
}

func anyIndexIterator(c EvalContext, iter IterableValue, block Lambda) PValue {
	index := int64(-1)
	return WrapBoolean(iter.Iterator().Any(func(v PValue) bool {
		index++
		return IsTruthy(block.Call(c, nil, WrapInteger(index), v))
	}))
}

func anyHashIterator(c EvalContext, iter IterableValue, block Lambda) PValue {
	return WrapBoolean(iter.Iterator().Any(func(v PValue) bool {
		vi := v.(IndexedValue)
		return IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}))
}

func init() {
	NewGoFunction(`any`,
		func(d Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return anyIterator(c, args[0].(*HashValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return anyHashIterator(c, args[0].(*HashValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return anyIterator(c, args[0].(IterableValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				iter := args[0].(IterableValue)
				if iter.IsHashStyle() {
					return anyHashIterator(c, iter, block)
				}
				return anyIndexIterator(c, iter, block)
			})
		},
	)
}
