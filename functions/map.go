package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func mapIterator(c px.Context, arg px.IterableValue, block px.Lambda) px.List {
	return arg.Iterator().Map(block.Signature().ReturnType(), func(v px.Value) px.Value { return block.Call(c, nil, v) }).AsArray()
}

func mapIndexIterator(c px.Context, iter px.IterableValue, block px.Lambda) px.List {
	index := int64(-1)
	return iter.Iterator().Map(block.Signature().ReturnType(), func(v px.Value) px.Value {
		index++
		return block.Call(c, nil, types.WrapInteger(index), v)
	}).AsArray()
}

func mapHashIterator(c px.Context, iter px.IterableValue, block px.Lambda) px.List {
	return iter.Iterator().Map(block.Signature().ReturnType(), func(v px.Value) px.Value {
		vi := v.(px.List)
		return block.Call(c, nil, vi.At(0), vi.At(1))
	}).AsArray()
}

func init() {
	px.NewGoFunction(`map`,
		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return mapIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return mapHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return mapIterator(c, args[0].(px.IterableValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				iter := args[0].(px.IterableValue)
				if iter.IsHashStyle() {
					return mapHashIterator(c, iter, block)
				}
				return mapIndexIterator(c, iter, block)
			})
		},
	)
}
