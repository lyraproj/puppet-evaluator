package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func anyIterator(c eval.Context, arg eval.IterableValue, block eval.Lambda) eval.Value {
	return types.WrapBoolean(arg.Iterator().Any(func(v eval.Value) bool { return eval.IsTruthy(block.Call(c, nil, v)) }))
}

func anyIndexIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.Value {
	index := int64(-1)
	return types.WrapBoolean(iter.Iterator().Any(func(v eval.Value) bool {
		index++
		return eval.IsTruthy(block.Call(c, nil, types.WrapInteger(index), v))
	}))
}

func anyHashIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.Value {
	return types.WrapBoolean(iter.Iterator().Any(func(v eval.Value) bool {
		vi := v.(eval.List)
		return eval.IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}))
}

func init() {
	eval.NewGoFunction(`any`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return anyIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return anyHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return anyIterator(c, args[0].(eval.IterableValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				iter := args[0].(eval.IterableValue)
				if iter.IsHashStyle() {
					return anyHashIterator(c, iter, block)
				}
				return anyIndexIterator(c, iter, block)
			})
		},
	)
}
