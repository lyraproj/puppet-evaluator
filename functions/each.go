package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func eachIterator(c eval.EvalContext, arg eval.IterableValue, block eval.Lambda) {
	arg.Iterator().Each(func(v eval.PValue) { block.Call(c, nil, v) })
}

func eachIndexIterator(c eval.EvalContext, iter eval.IterableValue, block eval.Lambda) {
	iter.Iterator().EachWithIndex(func(idx eval.PValue, v eval.PValue) {
		block.Call(c, nil, idx, v)
	})
}

func eachHashIterator(c eval.EvalContext, iter eval.IterableValue, block eval.Lambda) {
	iter.Iterator().Each(func(v eval.PValue) {
		vi := v.(eval.IndexedValue)
		block.Call(c, nil, vi.At(0), vi.At(1))
	})
}

func init() {
	eval.NewGoFunction(`each`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				hash := args[0].(*types.HashValue)
				eachIterator(c, hash, block)
				return hash
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				hash := args[0].(*types.HashValue)
				eachHashIterator(c, hash, block)
				return hash
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				iter := args[0].(eval.IterableValue)
				eachIterator(c, iter, block)
				return iter.(eval.PValue)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.EvalContext, args []eval.PValue, block eval.Lambda) eval.PValue {
				iter := args[0].(eval.IterableValue)
				if iter.IsHashStyle() {
					eachHashIterator(c, iter, block)
				} else {
					eachIndexIterator(c, iter, block)
				}
				return iter.(eval.PValue)
			})
		},
	)
}
