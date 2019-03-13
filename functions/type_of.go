package functions

import "github.com/lyraproj/pcore/px"

func typeOf(v px.Value, i string) px.Type {
	switch i {
	case `generalized`:
		return px.GenericValueType(v)
	case `reduced`:
		return v.PType()
	default:
		return px.DetailedValueType(v)
	}
}

func init() {
	px.NewGoFunction(`type`,
		func(d px.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return typeOf(args[0], `generalized`)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Any`)
			d.Param(`Enum[detailed, reduced, generalized]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return typeOf(args[0], args[1].String())
			})
		},
	)
}
