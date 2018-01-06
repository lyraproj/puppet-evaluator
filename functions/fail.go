package functions

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"bytes"
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
				c.Fail(w.String())
				return UNDEF
			})
		},
	)
}
