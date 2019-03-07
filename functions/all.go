package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func allIterator(c px.Context, arg px.IterableValue, block px.Lambda) px.Value {
	return types.WrapBoolean(arg.Iterator().All(func(v px.Value) bool { return px.IsTruthy(block.Call(c, nil, v)) }))
}

func allIndexIterator(c px.Context, iter px.IterableValue, block px.Lambda) px.Value {
	index := int64(-1)
	return types.WrapBoolean(iter.Iterator().All(func(v px.Value) bool {
		index++
		return px.IsTruthy(block.Call(c, nil, types.WrapInteger(index), v))
	}))
}

func allHashIterator(c px.Context, iter px.IterableValue, block px.Lambda) px.Value {
	return types.WrapBoolean(iter.Iterator().All(func(v px.Value) bool {
		vi := v.(px.List)
		return px.IsTruthy(block.Call(c, nil, vi.At(0), vi.At(1)))
	}))
}

func init() {
	px.NewGoFunction(`all`,
		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return allIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return allHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return allIterator(c, args[0].(px.IterableValue), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				iter := args[0].(px.IterableValue)
				if iter.IsHashStyle() {
					return allHashIterator(c, iter, block)
				}
				return allIndexIterator(c, iter, block)
			})
		},
	)
}
