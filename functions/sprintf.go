package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func init() {
	px.NewGoFunction(`sprintf`,
		func(d px.Dispatch) {
			d.Param(`String`)
			d.RepeatedParam(`Any`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return types.WrapString(types.PuppetSprintf(args[0].String(), args[1:]...))
			})
		})
}
