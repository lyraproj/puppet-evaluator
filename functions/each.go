package functions

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
)

func eachIterator(c EvalContext, arg IterableValue, block Lambda) {
	arg.Iterator().Each(func(v PValue) { block.Call(c, nil, v) })
}

func eachIndexIterator(c EvalContext, iter IterableValue, block Lambda) {
	iter.Iterator().EachWithIndex(func(idx PValue, v PValue) {
		block.Call(c, nil, idx, v)
	})
}

func eachHashIterator(c EvalContext, iter IterableValue, block Lambda) {
	iter.Iterator().Each(func(v PValue) {
		vi := v.(IndexedValue)
		block.Call(c, nil, vi.At(0), vi.At(1))
	})
}

func init() {
	NewGoFunction(`each`,
		func(d Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				hash := args[0].(*HashValue)
				eachIterator(c, hash, block)
				return hash
			})
		},

		func(d Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				hash := args[0].(*HashValue)
				eachHashIterator(c, hash, block)
				return hash
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				iter := args[0].(IterableValue)
				eachIterator(c, iter, block)
				return iter.(PValue)
			})
		},

		func(d Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c EvalContext, args []PValue, block Lambda) PValue {
				iter := args[0].(IterableValue)
				if iter.IsHashStyle() {
					eachHashIterator(c, iter, block)
				} else {
					eachIndexIterator(c, iter, block)
				}
				return iter.(PValue)
			})
		},
	)
}
