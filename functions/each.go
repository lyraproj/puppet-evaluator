package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

func eachIterator(c eval.Context, arg eval.IterableValue, block eval.Lambda) {
	arg.Iterator().Each(func(v eval.Value) { block.Call(c, nil, v) })
}

func eachIndexIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) {
	iter.Iterator().EachWithIndex(func(idx eval.Value, v eval.Value) {
		block.Call(c, nil, idx, v)
	})
}

func eachHashIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) {
	iter.Iterator().Each(func(v eval.Value) {
		vi := v.(eval.List)
		block.Call(c, nil, vi.At(0), vi.At(1))
	})
}

func init() {
	eval.NewGoFunction(`each`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				hash := args[0].(*types.HashValue)
				eachIterator(c, hash, block)
				return hash
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				hash := args[0].(*types.HashValue)
				eachHashIterator(c, hash, block)
				return hash
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				iter := args[0].(eval.IterableValue)
				eachIterator(c, iter, block)
				return iter.(eval.Value)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				iter := args[0].(eval.IterableValue)
				if iter.IsHashStyle() {
					eachHashIterator(c, iter, block)
				} else {
					eachIndexIterator(c, iter, block)
				}
				return iter.(eval.Value)
			})
		},
	)
}
