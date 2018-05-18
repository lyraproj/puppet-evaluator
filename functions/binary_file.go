package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
)

func init() {
	eval.NewGoFunction(`binary_file`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				return types.BinaryFromFile(c, args[0].String())
			})
		})
}
