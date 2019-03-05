package functions

import (
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/types"
)

func selectIterator(c eval.Context, arg eval.IterableValue, block eval.Lambda) eval.List {
	return arg.Iterator().Select(func(v eval.Value) bool { return eval.IsTruthy(block.Call(c, nil, v)) }).AsArray()
}

func selectIndexIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.List {
	index := int64(-1)
	return iter.Iterator().Select(func(v eval.Value) bool {
		index++
		return eval.IsTruthy(block.Call(c, nil, types.WrapInteger(index), v))
	}).AsArray()
}

func selectHashIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.List {
	return iter.Iterator().Select(func(v eval.Value) bool {
		vi := v.(eval.List)
		return eval.IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}).AsArray()
}

func init() {
	eval.NewGoFunction(`filter`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return selectIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return selectHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return selectIterator(c, args[0].(eval.IterableValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				iter := args[0].(eval.IterableValue)
				if iter.IsHashStyle() {
					return selectHashIterator(c, iter, block)
				}
				return selectIndexIterator(c, iter, block)
			})
		},
	)
}
