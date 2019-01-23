package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/puppet-evaluator/yaml"
)

func init() {
	eval.NewGoFunction(`parse_yaml`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return yaml.Unmarshal(c, []byte(args[0].String()))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Binary`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return yaml.Unmarshal(c, args[0].(*types.BinaryValue).Bytes())
			})
		})
}
