package functions

import (
	. "github.com/puppetlabs/go-evaluator/eval"
	. "github.com/puppetlabs/go-evaluator/types"
)

func mapIterator(c EvalContext, arg IterableValue, block Lambda) PValue {
	return arg.Iterator().Map(block.Signature().ReturnType(), func(v PValue) PValue { return block.Call(c, nil, v) })
}

func mapIndexIterator(c EvalContext, iter IterableValue, block Lambda) PValue {
	index := int64(-1)
	return iter.Iterator().Map(block.Signature().ReturnType(), func(v PValue) PValue {
		index++
		return block.Call(c, nil, WrapInteger(index), v)
	})
}

func mapHashIterator(c EvalContext, iter IterableValue, block Lambda) PValue {
	return iter.Iterator().Map(block.Signature().ReturnType(), func(v PValue) PValue {
		vi := v.(IndexedValue)
		return block.Call(c, nil, vi.At(0), vi.At(1))
	})
}

func init() {
	NewGoFunction(`map`,
		func(d Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return mapIterator(c, args[0].(*HashValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return mapHashIterator(c, args[0].(*HashValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				return mapIterator(c, args[0].(IterableValue), block)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				iter := args[0].(IterableValue)
				if iter.IsHashStyle() {
					return mapHashIterator(c, iter, block)
				}
				return mapIndexIterator(c, iter, block)
			})
		},
	)
}
