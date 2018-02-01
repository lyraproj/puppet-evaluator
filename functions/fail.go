package functions

import (
	"bytes"
	. "github.com/puppetlabs/go-evaluator/eval"
)

func init() {
	NewGoFunction(`fail`,
		func(d Dispatch) {
			d.RepeatedParam(`Any`)
			d.Function(func(c EvalContext, args []PValue) PValue {
				w := bytes.NewBufferString(``)
				for ix, arg := range args {
					if ix > 0 {
						w.WriteByte(' ')
					}
					ToString3(arg, w)
				}
				panic(c.Fail(w.String()))
			})
		},
	)
}
