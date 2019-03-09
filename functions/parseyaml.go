package functions

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/yaml"
)

func init() {
	px.NewGoFunction(`parse_yaml`,
		func(d px.Dispatch) {
			d.Param(`String`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return yaml.Unmarshal(c, []byte(args[0].String()))
			})
		},

		func(d px.Dispatch) {
			d.Param(`Binary`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return yaml.Unmarshal(c, args[0].(*types.Binary).Bytes())
			})
		})
}
