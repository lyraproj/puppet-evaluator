package eval

import (
	"testing"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/resource"
	"github.com/puppetlabs/go-pspec/pspec"
)

func TestPSpecs(t *testing.T) {
	pspec.RunPspecTests(t, `testdata`, func() eval.DefiningLoader {
		eval.NewGoFunction(`get_resource`,
			func(d eval.Dispatch) {
				d.Param(`Variant[Type[Resource],String]`)
				d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
					if node, ok := c.Evaluator().(resource.Evaluator).Node(c, args[0]); ok && node.Resolved(c) {
						return node.Value(c)
					}
					return eval.UNDEF
				})
			},
		)

		eval.NewGoFunction(`get_edges`,
			func(d eval.Dispatch) {
				d.Param(`Variant[Resource,Type[Resource],String]`)
				d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
					re := c.Evaluator().(resource.Evaluator)
					if from, ok := re.Node(c, args[0]); ok {
						return re.Edges(from);
					}
					return eval.EMPTY_ARRAY
				})
			},
		)

		eval.NewGoFunction(`get_resources`,
			func(d eval.Dispatch) {
				d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
					return c.Evaluator().(resource.Evaluator).Nodes().Select(func(node eval.PValue) bool {
						return node.(resource.Node).Resolved(c)
					}).Map(func(node eval.PValue) eval.PValue {
						return node.(resource.Node).Value(c)
					})
				})
			},
		)
		return eval.StaticResourceLoader().(eval.DefiningLoader)
	})
}
