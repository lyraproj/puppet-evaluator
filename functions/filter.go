package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func selectIterator(c eval.Context, arg eval.IterableValue, block eval.Lambda) eval.IndexedValue {
	return arg.Iterator().Select(func(v eval.PValue) bool { return eval.IsTruthy(block.Call(c, nil, v)) }).AsArray()
}

func selectIndexIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.IndexedValue {
	index := int64(-1)
	return iter.Iterator().Select(func(v eval.PValue) bool {
		index++
		return eval.IsTruthy(block.Call(c, nil, types.WrapInteger(index), v))
	}).AsArray()
}

func selectHashIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.IndexedValue {
	return iter.Iterator().Select(func(v eval.PValue) bool {
		vi := v.(eval.IndexedValue)
		return eval.IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}).AsArray()
}

func init() {
	eval.NewGoFunction(`filter`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return selectIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return selectHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return selectIterator(c, args[0].(eval.IterableValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				iter := args[0].(eval.IterableValue)
				if iter.IsHashStyle() {
					return selectHashIterator(c, iter, block)
				}
				return selectIndexIterator(c, iter, block)
			})
		},
	)
}
