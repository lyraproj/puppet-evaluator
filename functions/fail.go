package functions

import (
	"bytes"

	"github.com/lyraproj/pcore/px"
)

func init() {
	px.NewGoFunction(`fail`,
		func(d px.Dispatch) {
			d.RepeatedParam(`Any`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				w := bytes.NewBufferString(``)
				for ix, arg := range args {
					if ix > 0 {
						w.WriteByte(' ')
					}
					px.ToString3(arg, w)
				}
				panic(c.Fail(w.String()))
			})
		},
	)
}
