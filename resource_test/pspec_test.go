package eval

import (
	"testing"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/resource"
	"github.com/puppetlabs/go-pspec/pspec"
)

func TestPSpecs(t *testing.T) {
	pspec.RunPspecTests(t, `testdata`)
}

func init() {
	eval.NewGoFunction(`get_resource`,
		func(d eval.Dispatch) {
			d.Param(`Variant[Type[Resource],String]`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				if node, ok := c.Evaluator().(resource.Evaluator).Node(args[0]); ok && node.Resolved() {
					return node.Value()
				}
				return eval.UNDEF
			})
		},
	)

	eval.NewGoFunction(`get_resources`,
		func(d eval.Dispatch) {
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				return c.Evaluator().(resource.Evaluator).Nodes().Select(func(node eval.PValue) bool {
					return node.(resource.Node).Resolved()
				}).Map(func(node eval.PValue) eval.PValue {
					return node.(resource.Node).Value()
				})
			})
		},
	)
}
