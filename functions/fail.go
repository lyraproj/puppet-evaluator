package functions

import (
	"bytes"

	"github.com/puppetlabs/go-evaluator/eval"
)

func init() {
	eval.NewGoFunction(`fail`,
		func(d eval.Dispatch) {
			d.RepeatedParam(`Any`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				w := bytes.NewBufferString(``)
				for ix, arg := range args {
					if ix > 0 {
						w.WriteByte(' ')
					}
					eval.ToString3(arg, w)
				}
				panic(c.Fail(w.String()))
			})
		},
	)
}
