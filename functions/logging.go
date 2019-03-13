package functions

import "github.com/lyraproj/pcore/px"

func init() {
	for _, level := range px.LogLevels {
		px.NewGoFunction(string(level),
			func(d px.Dispatch) {
				d.RepeatedParam(`Any`)
				d.Function(func(c px.Context, args []px.Value) px.Value {
					c.Logger().Log(px.LogLevel(d.Name()), args...)
					return px.Undef
				})
			})
	}
}
