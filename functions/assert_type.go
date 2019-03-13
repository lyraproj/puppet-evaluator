package functions

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

func assertType(c px.Context, t px.Type, v px.Value, b px.Lambda) px.Value {
	if px.IsInstance(t, v) {
		return v
	}
	vt := px.DetailedValueType(v)
	if b == nil {
		panic(px.Error(px.TypeMismatch, issue.H{`detail`: px.DescribeMismatch(`assert_type():`, t, vt)}))
	}
	return b.Call(c, nil, t, vt)
}

func init() {
	px.NewGoFunction(`assert_type`,
		func(d px.Dispatch) {
			d.Param(`String[1]`)
			d.Param(`Any`)
			d.OptionalBlock(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return assertType(c, c.ParseTypeValue(args[0]), args[1], block)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Type`)
			d.Param(`Any`)
			d.OptionalBlock(`Callable[2,2]`)
			d.Function2(func(c px.Context, args []px.Value, block px.Lambda) px.Value {
				return assertType(c, args[0].(px.Type), args[1], block)
			})
		},
	)
}
