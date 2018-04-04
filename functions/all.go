package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func allIterator(c eval.Context, arg eval.IterableValue, block eval.Lambda) eval.PValue {
	return types.WrapBoolean(arg.Iterator().All(func(v eval.PValue) bool { return eval.IsTruthy(block.Call(c, nil, v)) }))
}

func allIndexIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.PValue {
	index := int64(-1)
	return types.WrapBoolean(iter.Iterator().All(func(v eval.PValue) bool {
		index++
		return eval.IsTruthy(block.Call(c, nil, types.WrapInteger(index), v))
	}))
}

func allHashIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.PValue {
	return types.WrapBoolean(iter.Iterator().All(func(v eval.PValue) bool {
		vi := v.(eval.IndexedValue)
		return eval.IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}))
}

func init() {
	eval.NewGoFunction(`all`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return allIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return allHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				return allIterator(c, args[0].(eval.IterableValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.PValue, block eval.Lambda) eval.PValue {
				iter := args[0].(eval.IterableValue)
				if iter.IsHashStyle() {
					return allHashIterator(c, iter, block)
				}
				return allIndexIterator(c, iter, block)
			})
		},
	)
}
