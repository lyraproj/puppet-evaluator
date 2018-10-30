package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func mapIterator(c eval.Context, arg eval.IterableValue, block eval.Lambda) eval.List {
	return arg.Iterator().Map(block.Signature().ReturnType(), func(v eval.Value) eval.Value { return block.Call(c, nil, v) }).AsArray()
}

func mapIndexIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.List {
	index := int64(-1)
	return iter.Iterator().Map(block.Signature().ReturnType(), func(v eval.Value) eval.Value {
		index++
		return block.Call(c, nil, types.WrapInteger(index), v)
	}).AsArray()
}

func mapHashIterator(c eval.Context, iter eval.IterableValue, block eval.Lambda) eval.List {
	return iter.Iterator().Map(block.Signature().ReturnType(), func(v eval.Value) eval.Value {
		vi := v.(eval.List)
		return block.Call(c, nil, vi.At(0), vi.At(1))
	}).AsArray()
}

func init() {
	eval.NewGoFunction(`map`,
		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return mapIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return mapHashIterator(c, args[0].(*types.HashValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[1,1]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				return mapIterator(c, args[0].(eval.IterableValue), block)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Iterable`)
			d.Block(`Callable[2,2]`)
			d.Function2(func(c eval.Context, args []eval.Value, block eval.Lambda) eval.Value {
				iter := args[0].(eval.IterableValue)
				if iter.IsHashStyle() {
					return mapHashIterator(c, iter, block)
				}
				return mapIndexIterator(c, iter, block)
			})
		},
	)
}
