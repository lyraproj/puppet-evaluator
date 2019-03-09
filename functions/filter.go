package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/evaluator"
)

func selectIterator(c px.Context, arg px.Indexed, block px.Lambda) px.List {
	return evaluator.WrapIterable(arg).Select(func(v px.Value) bool { return px.IsTruthy(block.Call(c, nil, v)) }).AsArray()
}

func selectIndexIterator(c px.Context, iter px.Indexed, block px.Lambda) px.List {
	index := int64(-1)
	return evaluator.WrapIterable(iter).Select(func(v px.Value) bool {
		index++
		return px.IsTruthy(block.Call(c, nil, types.WrapInteger(index), v))
	}).AsArray()
}

func selectHashIterator(c px.Context, iter px.Indexed, block px.Lambda) px.List {
	return evaluator.WrapIterable(iter).Select(func(v px.Value) bool {
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
				return selectIterator(c, args[0].(*types.Hash), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return selectHashIterator(c, args[0].(*types.Hash), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return selectIterator(c, args[0].(px.Indexed), block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				iter := args[0].(px.Indexed)
				if iter.IsHashStyle() {
					return selectHashIterator(c, iter, block)
				}
				return selectIndexIterator(c, iter, block)
			})
		},
	)
}
