package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func anyIterator(c px.Context, arg px.IterableValue, block px.Lambda) px.Value {
	return types.WrapBoolean(arg.Iterator().Any(func(v px.Value) bool { return px.IsTruthy(block.Call(c, nil, v)) }))
}

func anyIndexIterator(c px.Context, iter px.IterableValue, block px.Lambda) px.Value {
	index := int64(-1)
	return types.WrapBoolean(iter.Iterator().Any(func(v px.Value) bool {
		index++
		return px.IsTruthy(block.Call(c, nil, types.WrapInteger(index), v))
	}))
}

func anyHashIterator(c px.Context, iter px.IterableValue, block px.Lambda) px.Value {
	return types.WrapBoolean(iter.Iterator().Any(func(v px.Value) bool {
		vi := v.(px.List)
		return px.IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}))
}

func init() {
	px.NewGoFunction(`any`,
		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return anyIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return anyHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return anyIterator(c, args[0].(px.IterableValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				iter := args[0].(px.IterableValue)
				if iter.IsHashStyle() {
					return anyHashIterator(c, iter, block)
				}
				return anyIndexIterator(c, iter, block)
			})
		},
	)
}
