package functions

import (
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/types"
)

func allIterator(c eval.Context, arg eval.IterableValue, block eval.Lambda) eval.Value {
	return types.WrapBoolean(arg.Iterator().All(func(v eval.Value) bool { return eval.IsTruthy(block.Call(c, nil, v)) }))
}

func allIndexIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.Value {
	index := int64(-1)
	return types.WrapBoolean(iter.Iterator().All(func(v eval.Value) bool {
		index++
		return eval.IsTruthy(block.Call(c, nil, types.WrapInteger(index), v))
	}))
}

func allHashIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.Value {
	return types.WrapBoolean(iter.Iterator().All(func(v eval.Value) bool {
		vi := v.(eval.List)
		return eval.IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}))
}

func init() {
	eval.NewGoFunction(`all`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return allIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return allHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return allIterator(c, args[0].(eval.IterableValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				iter := args[0].(eval.IterableValue)
				if iter.IsHashStyle() {
					return allHashIterator(c, iter, block)
				}
				return allIndexIterator(c, iter, block)
			})
		},
	)
}
