package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func anyIterator(c eval.Context, arg eval.IterableValue, block eval.Lambda) eval.PValue {
	return types.WrapBoolean(arg.Iterator().Any(func(v eval.PValue) bool { return eval.IsTruthy(block.Call(c, nil, v)) }))
}

func anyIndexIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.PValue {
	index := int64(-1)
	return types.WrapBoolean(iter.Iterator().Any(func(v eval.PValue) bool {
		index++
		return eval.IsTruthy(block.Call(c, nil, types.WrapInteger(index), v))
	}))
}

func anyHashIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.PValue {
	return types.WrapBoolean(iter.Iterator().Any(func(v eval.PValue) bool {
		vi := v.(eval.IndexedValue)
		return eval.IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}))
}

func init() {
	eval.NewGoFunction(`any`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return anyIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return anyHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return anyIterator(c, args[0].(eval.IterableValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				iter := args[0].(eval.IterableValue)
				if iter.IsHashStyle() {
					return anyHashIterator(c, iter, block)
				}
				return anyIndexIterator(c, iter, block)
			})
		},
	)
}
