package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func eachIterator(c px.Context, arg px.IterableValue, block px.Lambda) {
	arg.Iterator().Each(func(v px.Value) { block.Call(c, nil, v) })
}

func eachIndexIterator(c px.Context, iter px.IterableValue, block px.Lambda) {
	iter.Iterator().EachWithIndex(func(idx px.Value, v px.Value) {
		block.Call(c, nil, idx, v)
	})
}

func eachHashIterator(c px.Context, iter px.IterableValue, block px.Lambda) {
	iter.Iterator().Each(func(v px.Value) {
		vi := v.(px.List)
		block.Call(c, nil, vi.At(0), vi.At(1))
	})
}

func init() {
	px.NewGoFunction(`each`,
		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				hash := args[0].(*types.HashValue)
				eachIterator(c, hash, block)
				return hash
			})
		},

		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				hash := args[0].(*types.HashValue)
				eachHashIterator(c, hash, block)
				return hash
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				iter := args[0].(px.IterableValue)
				eachIterator(c, iter, block)
				return iter.(px.Value)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				iter := args[0].(px.IterableValue)
				if iter.IsHashStyle() {
					eachHashIterator(c, iter, block)
				} else {
					eachIndexIterator(c, iter, block)
				}
				return iter.(px.Value)
			})
		},
	)
}
