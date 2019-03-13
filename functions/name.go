package functions

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

func init() {
	px.NewGoFunction(`name`,
		func(d px.Dispatch) {
			d.Param(`Any`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				v := args[0]
				if n, ok := v.(issue.Named); ok {
					return types.WrapString(n.Name())
				}
				panic(px.Error(px.UnknownFunction, issue.H{`name`: `name`}))
			})
		})
}
