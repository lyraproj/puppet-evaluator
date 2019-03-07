package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func selectIterator(c px.Context, arg px.IterableValue, block px.Lambda) px.List {
	return arg.Iterator().Select(func(v px.Value) bool { return px.IsTruthy(block.Call(c, nil, v)) }).AsArray()
}

func selectIndexIterator(c px.Context, iter px.IterableValue, block px.Lambda) px.List {
	index := int64(-1)
	return iter.Iterator().Select(func(v px.Value) bool {
		index++
		return px.IsTruthy(block.Call(c, nil, types.WrapInteger(index), v))
	}).AsArray()
}

func selectHashIterator(c px.Context, iter px.IterableValue, block px.Lambda) px.List {
	return iter.Iterator().Select(func(v px.Value) bool {
		vi := v.(px.List)
		return px.IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}).AsArray()
}

func init() {
	px.NewGoFunction(`filter`,
		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return selectIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return selectHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return selectIterator(c, args[0].(px.IterableValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				iter := args[0].(px.IterableValue)
				if iter.IsHashStyle() {
					return selectHashIterator(c, iter, block)
				}
				return selectIndexIterator(c, iter, block)
			})
		},
	)
}
