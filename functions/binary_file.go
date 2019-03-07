package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func init() {
	px.NewGoFunction(`binary_file`,
		func(d px.Dispatch) {
			d.Param(`String`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return types.BinaryFromFile(args[0].String())
			})
		})
}
