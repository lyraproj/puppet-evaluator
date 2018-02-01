package functions

import (
	. "github.com/puppetlabs/go-evaluator/eval"
	. "github.com/puppetlabs/go-evaluator/types"
)

func allIterator(c EvalContext, arg IterableValue, block Lambda) PValue {
	return WrapBoolean(arg.Iterator().All(func(v PValue) bool { return IsTruthy(block.Call(c, nil, v)) }))
}

func allIndexIterator(c EvalContext, iter IterableValue, block Lambda) PValue {
	index := int64(-1)
	return WrapBoolean(iter.Iterator().All(func(v PValue) bool {
		index++
		return IsTruthy(block.Call(c, nil, WrapInteger(index), v))
	}))
}

func allHashIterator(c EvalContext, iter IterableValue, block Lambda) PValue {
	return WrapBoolean(iter.Iterator().All(func(v PValue) bool {
		vi := v.(IndexedValue)
		return IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}))
}

func init() {
	NewGoFunction(`all`,
		func(d Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return allIterator(c, args[0].(*HashValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return allHashIterator(c, args[0].(*HashValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return allIterator(c, args[0].(IterableValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				iter := args[0].(IterableValue)
				if iter.IsHashStyle() {
					return allHashIterator(c, iter, block)
				}
				return allIndexIterator(c, iter, block)
			})
		},
	)
}
